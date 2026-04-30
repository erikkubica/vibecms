package uploads

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

// Sentinel errors for predictable handling at the HTTP and MCP layers.
var (
	ErrNotFound         = errors.New("upload token not found")
	ErrExpired          = errors.New("upload token expired")
	ErrAlreadyUploaded  = errors.New("upload already uploaded")
	ErrAlreadyFinalized = errors.New("upload already finalized")
	ErrKindMismatch     = errors.New("upload kind mismatch")
	ErrSHA256Mismatch   = errors.New("sha256 mismatch")
	ErrInvalidKind      = errors.New("invalid upload kind")
	ErrSizeExceedsCap   = errors.New("upload exceeds max_bytes")
	ErrNotUploaded      = errors.New("upload not yet uploaded")
)

// IssueOptions describes a new upload row about to be issued. UserID and Kind
// are mandatory; everything else is metadata the finalize step may use.
type IssueOptions struct {
	Kind     Kind
	UserID   int64
	Filename string
	MimeType string
	MaxBytes int64
	TTL      time.Duration
}

// Store wraps the pending_uploads table and the on-disk staging directory
// (data/pending/<token>.bin). All flows run through a Store so the SQL state
// machine stays the single source of truth for what's allowed when.
type Store struct {
	db      *gorm.DB
	tempDir string
	now     func() time.Time
}

// NewStore constructs a Store. tempDir is created if it does not exist —
// callers don't need to mkdir before constructing.
func NewStore(db *gorm.DB, tempDir string) (*Store, error) {
	if tempDir == "" {
		tempDir = "data/pending"
	}
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return nil, fmt.Errorf("create pending dir: %w", err)
	}
	// Best-effort schema sync for SQLite-backed tests; production runs the
	// SQL migration. AutoMigrate is idempotent and matches columns to tags.
	if err := db.AutoMigrate(&PendingUpload{}); err != nil {
		return nil, fmt.Errorf("migrate pending_uploads: %w", err)
	}
	return &Store{db: db, tempDir: tempDir, now: time.Now}, nil
}

// SetClock overrides the clock for tests (cleanup, expiry).
func (s *Store) SetClock(now func() time.Time) { s.now = now }

// TempDir is the on-disk staging directory.
func (s *Store) TempDir() string { return s.tempDir }

// Issue allocates a new pending row with a fresh token and returns it.
func (s *Store) Issue(opts IssueOptions) (*PendingUpload, error) {
	if !opts.Kind.Valid() {
		return nil, ErrInvalidKind
	}
	if opts.MaxBytes <= 0 {
		return nil, fmt.Errorf("max_bytes must be positive")
	}
	if opts.TTL <= 0 {
		opts.TTL = 15 * time.Minute
	}
	tok, err := newToken()
	if err != nil {
		return nil, fmt.Errorf("token: %w", err)
	}
	row := &PendingUpload{
		Token:     tok,
		Kind:      string(opts.Kind),
		UserID:    opts.UserID,
		Filename:  opts.Filename,
		MimeType:  opts.MimeType,
		MaxBytes:  opts.MaxBytes,
		State:     string(StatePending),
		ExpiresAt: s.now().Add(opts.TTL),
	}
	if err := s.db.Create(row).Error; err != nil {
		return nil, fmt.Errorf("insert pending_upload: %w", err)
	}
	return row, nil
}

// Lookup fetches a row by token. ErrNotFound when missing.
func (s *Store) Lookup(token string) (*PendingUpload, error) {
	var row PendingUpload
	if err := s.db.Where("token = ?", token).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

// ValidateForPUT checks a row is usable for a fresh PUT: not expired, still
// in 'pending' state. Single-use semantics — a second PUT to the same token
// returns ErrAlreadyUploaded.
func (s *Store) ValidateForPUT(row *PendingUpload) error {
	if row.State == string(StateFinalized) {
		return ErrAlreadyFinalized
	}
	if row.State == string(StateUploaded) {
		return ErrAlreadyUploaded
	}
	if s.now().After(row.ExpiresAt) {
		return ErrExpired
	}
	return nil
}

// MarkUploaded transitions pending → uploaded after the temp file has been
// fully written. Stores the size + sha256 + temp_path on the row so finalize
// can verify and locate the bytes.
func (s *Store) MarkUploaded(token string, size int64, sha string, tempPath string) error {
	res := s.db.Model(&PendingUpload{}).
		Where("token = ? AND state = ?", token, string(StatePending)).
		Updates(map[string]interface{}{
			"state":      string(StateUploaded),
			"size_bytes": size,
			"sha256":     sha,
			"temp_path":  tempPath,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// Row was concurrently modified or no longer pending. Re-read so the
		// caller can return a precise error.
		row, err := s.Lookup(token)
		if err != nil {
			return err
		}
		return s.ValidateForPUT(row)
	}
	return nil
}

// ValidateForFinalize checks a row is ready to be finalized: kind matches,
// state == uploaded, not yet finalized, optional sha matches stored.
func (s *Store) ValidateForFinalize(row *PendingUpload, kind Kind, expectedSHA string) error {
	if row.Kind != string(kind) {
		return ErrKindMismatch
	}
	if row.State == string(StateFinalized) {
		return ErrAlreadyFinalized
	}
	if row.State != string(StateUploaded) {
		return ErrNotUploaded
	}
	if s.now().After(row.ExpiresAt) {
		return ErrExpired
	}
	if expectedSHA != "" && row.SHA256 != nil && *row.SHA256 != expectedSHA {
		return ErrSHA256Mismatch
	}
	return nil
}

// MarkFinalized transitions uploaded → finalized and removes the temp file.
// Returns the temp_path (already deleted) for callers that want to log it.
func (s *Store) MarkFinalized(token string) error {
	row, err := s.Lookup(token)
	if err != nil {
		return err
	}
	now := s.now()
	res := s.db.Model(&PendingUpload{}).
		Where("token = ? AND state = ?", token, string(StateUploaded)).
		Updates(map[string]interface{}{
			"state":        string(StateFinalized),
			"finalized_at": &now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		// Race: someone else finalized this token between Lookup and Update.
		return ErrAlreadyFinalized
	}
	if row.TempPath != nil && *row.TempPath != "" {
		_ = os.Remove(*row.TempPath)
	}
	return nil
}

// TempPathFor returns the canonical on-disk path for a given token.
// Server-controlled — clients never name the file.
func (s *Store) TempPathFor(token string) string {
	return filepath.Join(s.tempDir, token+".bin")
}

// Cleanup deletes pending/uploaded rows that have expired without being
// finalized, and removes their temp files. Idempotent — safe to call from a
// background ticker. Returns the number of rows removed.
func (s *Store) Cleanup() (int, error) {
	var rows []PendingUpload
	if err := s.db.Where("state <> ? AND expires_at < ?", string(StateFinalized), s.now()).
		Find(&rows).Error; err != nil {
		return 0, fmt.Errorf("query expired uploads: %w", err)
	}
	if len(rows) == 0 {
		return 0, nil
	}
	tokens := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.TempPath != nil && *r.TempPath != "" {
			_ = os.Remove(*r.TempPath)
		} else {
			_ = os.Remove(s.TempPathFor(r.Token))
		}
		tokens = append(tokens, r.Token)
	}
	if err := s.db.Where("token IN ?", tokens).Delete(&PendingUpload{}).Error; err != nil {
		return 0, fmt.Errorf("delete expired uploads: %w", err)
	}
	return len(rows), nil
}
