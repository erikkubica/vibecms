package uploads

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"squilla/internal/testutil"
)

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	db := testutil.NewSQLiteDB(t)
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "pending")
	store, err := NewStore(db, dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store, dir
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func issue(t *testing.T, store *Store, kind Kind, max int64) *PendingUpload {
	t.Helper()
	row, err := store.Issue(IssueOptions{
		Kind:     kind,
		UserID:   1,
		Filename: "x.bin",
		MaxBytes: max,
		TTL:      time.Minute,
	})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	return row
}

func TestIssueValidation(t *testing.T) {
	store, _ := newStore(t)

	if _, err := store.Issue(IssueOptions{Kind: "bogus", UserID: 1, MaxBytes: 100}); !errors.Is(err, ErrInvalidKind) {
		t.Fatalf("expected ErrInvalidKind, got %v", err)
	}
	if _, err := store.Issue(IssueOptions{Kind: KindMedia, UserID: 1, MaxBytes: 0}); err == nil {
		t.Fatal("expected error for non-positive max_bytes")
	}
	row := issue(t, store, KindMedia, 100)
	if len(row.Token) != 64 {
		t.Fatalf("expected 64-char token, got %d", len(row.Token))
	}
	if row.State != string(StatePending) {
		t.Fatalf("expected pending state, got %s", row.State)
	}
}

func putBytes(t *testing.T, srv *httptest.Server, token string, body []byte) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, srv.URL+"/api/uploads/"+token, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("new req: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	return resp
}

func decodeJSON(t *testing.T, r io.Reader) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(r).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestPUTHappyPath(t *testing.T) {
	store, _ := newStore(t)
	srv := httptest.NewServer(NewHandler(store))
	t.Cleanup(srv.Close)

	row := issue(t, store, KindMedia, 1024)
	body := []byte("hello world")
	resp := putBytes(t, srv, row.Token, body)
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	out := decodeJSON(t, resp.Body)
	if int64(out["size"].(float64)) != int64(len(body)) {
		t.Fatalf("size mismatch: %v", out["size"])
	}
	if out["sha256"].(string) != sha256Hex(body) {
		t.Fatalf("sha256 mismatch")
	}

	got, err := store.Lookup(row.Token)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got.State != string(StateUploaded) {
		t.Fatalf("expected uploaded state, got %s", got.State)
	}
	if got.SizeBytes == nil || *got.SizeBytes != int64(len(body)) {
		t.Fatalf("size_bytes wrong: %v", got.SizeBytes)
	}

	// Temp file exists with the right contents.
	data, err := os.ReadFile(*got.TempPath)
	if err != nil {
		t.Fatalf("read temp: %v", err)
	}
	if !bytes.Equal(data, body) {
		t.Fatalf("temp file body mismatch")
	}
}

func TestPUTUnknownToken(t *testing.T) {
	store, _ := newStore(t)
	srv := httptest.NewServer(NewHandler(store))
	t.Cleanup(srv.Close)

	resp := putBytes(t, srv, strings.Repeat("z", 64), []byte("x"))
	defer resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestPUTExpired(t *testing.T) {
	store, _ := newStore(t)
	// Force "now" forward so the issued row is already expired.
	base := time.Now()
	store.SetClock(func() time.Time { return base })
	row := issue(t, store, KindMedia, 100)
	store.SetClock(func() time.Time { return base.Add(time.Hour) })

	srv := httptest.NewServer(NewHandler(store))
	t.Cleanup(srv.Close)

	resp := putBytes(t, srv, row.Token, []byte("hi"))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusGone {
		t.Fatalf("expected 410, got %d", resp.StatusCode)
	}
}

func TestPUTOversize(t *testing.T) {
	store, dir := newStore(t)
	srv := httptest.NewServer(NewHandler(store))
	t.Cleanup(srv.Close)

	row := issue(t, store, KindMedia, 4)
	resp := putBytes(t, srv, row.Token, []byte("too big"))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", resp.StatusCode)
	}
	// Temp file must be gone.
	if _, err := os.Stat(filepath.Join(dir, row.Token+".bin")); !os.IsNotExist(err) {
		t.Fatalf("expected temp file removed, stat err=%v", err)
	}
	// Row stays pending.
	got, _ := store.Lookup(row.Token)
	if got.State != string(StatePending) {
		t.Fatalf("expected state still pending, got %s", got.State)
	}
}

func TestPUTDoubleUpload(t *testing.T) {
	store, _ := newStore(t)
	srv := httptest.NewServer(NewHandler(store))
	t.Cleanup(srv.Close)

	row := issue(t, store, KindMedia, 100)
	resp1 := putBytes(t, srv, row.Token, []byte("first"))
	resp1.Body.Close()
	if resp1.StatusCode != 200 {
		t.Fatalf("expected first PUT 200, got %d", resp1.StatusCode)
	}
	resp2 := putBytes(t, srv, row.Token, []byte("second"))
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("expected second PUT 409, got %d", resp2.StatusCode)
	}
}

func TestFinalizeKindMismatch(t *testing.T) {
	store, _ := newStore(t)
	row := issue(t, store, KindMedia, 100)
	if err := store.MarkUploaded(row.Token, 5, sha256Hex([]byte("hello")), store.TempPathFor(row.Token)); err != nil {
		t.Fatalf("MarkUploaded: %v", err)
	}
	got, _ := store.Lookup(row.Token)
	if err := store.ValidateForFinalize(got, KindTheme, ""); !errors.Is(err, ErrKindMismatch) {
		t.Fatalf("expected ErrKindMismatch, got %v", err)
	}
}

func TestFinalizeSHA256Mismatch(t *testing.T) {
	store, _ := newStore(t)
	row := issue(t, store, KindMedia, 100)
	body := []byte("hello")
	if err := store.MarkUploaded(row.Token, int64(len(body)), sha256Hex(body), store.TempPathFor(row.Token)); err != nil {
		t.Fatalf("MarkUploaded: %v", err)
	}
	got, _ := store.Lookup(row.Token)
	if err := store.ValidateForFinalize(got, KindMedia, "deadbeef"); !errors.Is(err, ErrSHA256Mismatch) {
		t.Fatalf("expected ErrSHA256Mismatch, got %v", err)
	}
	if err := store.ValidateForFinalize(got, KindMedia, sha256Hex(body)); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestDoubleFinalize(t *testing.T) {
	store, dir := newStore(t)
	row := issue(t, store, KindMedia, 100)
	body := []byte("hello")
	tempPath := filepath.Join(dir, row.Token+".bin")
	if err := os.WriteFile(tempPath, body, 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := store.MarkUploaded(row.Token, int64(len(body)), sha256Hex(body), tempPath); err != nil {
		t.Fatalf("MarkUploaded: %v", err)
	}
	if err := store.MarkFinalized(row.Token); err != nil {
		t.Fatalf("first MarkFinalized: %v", err)
	}
	if err := store.MarkFinalized(row.Token); !errors.Is(err, ErrAlreadyFinalized) {
		t.Fatalf("expected ErrAlreadyFinalized, got %v", err)
	}
	// Temp file should be gone after finalize.
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("expected temp file removed, stat err=%v", err)
	}
}

func TestCleanupRemovesExpired(t *testing.T) {
	store, dir := newStore(t)
	base := time.Now()
	store.SetClock(func() time.Time { return base })

	row := issue(t, store, KindMedia, 100)
	tempPath := filepath.Join(dir, row.Token+".bin")
	if err := os.WriteFile(tempPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := store.MarkUploaded(row.Token, 1, sha256Hex([]byte("x")), tempPath); err != nil {
		t.Fatalf("MarkUploaded: %v", err)
	}
	// Jump past expiry without finalizing.
	store.SetClock(func() time.Time { return base.Add(time.Hour) })
	n, err := store.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 row swept, got %d", n)
	}
	if _, err := store.Lookup(row.Token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after cleanup, got %v", err)
	}
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Fatalf("expected temp file removed, stat err=%v", err)
	}
}

func TestCleanupKeepsFinalized(t *testing.T) {
	store, dir := newStore(t)
	base := time.Now()
	store.SetClock(func() time.Time { return base })

	row := issue(t, store, KindMedia, 100)
	tempPath := filepath.Join(dir, row.Token+".bin")
	if err := os.WriteFile(tempPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := store.MarkUploaded(row.Token, 1, sha256Hex([]byte("x")), tempPath); err != nil {
		t.Fatalf("MarkUploaded: %v", err)
	}
	if err := store.MarkFinalized(row.Token); err != nil {
		t.Fatalf("MarkFinalized: %v", err)
	}
	store.SetClock(func() time.Time { return base.Add(time.Hour) })
	n, err := store.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 rows swept, got %d", n)
	}
	if _, err := store.Lookup(row.Token); err != nil {
		t.Fatalf("finalized row should survive cleanup, got %v", err)
	}
}

func TestConcurrentPUTs(t *testing.T) {
	store, _ := newStore(t)
	srv := httptest.NewServer(NewHandler(store))
	t.Cleanup(srv.Close)

	row := issue(t, store, KindMedia, 1024*1024)
	const N = 8
	body := bytes.Repeat([]byte("a"), 1024)

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		ok       int
		conflict int
	)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := putBytes(t, srv, row.Token, body)
			defer resp.Body.Close()
			mu.Lock()
			defer mu.Unlock()
			switch resp.StatusCode {
			case 200:
				ok++
			case http.StatusConflict:
				conflict++
			}
		}()
	}
	wg.Wait()
	if ok != 1 {
		t.Fatalf("expected exactly one 200, got %d (conflicts=%d)", ok, conflict)
	}
}

func TestTokenFromPath(t *testing.T) {
	cases := map[string]string{
		"/api/uploads/abc":          "abc",
		"/api/uploads/abc/":         "abc",
		"/api/uploads/abc/x":        "abc",
		"/something/api/uploads/xy": "xy",
		"/no/match":                 "",
	}
	for in, want := range cases {
		if got := tokenFromPath(in); got != want {
			t.Errorf("tokenFromPath(%q)=%q, want %q", in, got, want)
		}
	}
}
