package uploads

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
)

// Handler exposes PUT /api/uploads/:token as a plain net/http.Handler. We
// deliberately bypass Fiber's body-buffering pipeline here so multi-hundred-MB
// theme/extension uploads stream straight to disk without ever sitting in
// memory in a single allocation. The token is the auth — there is no session,
// no cookie, no bearer header.
type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

func writeJSONErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ServeHTTP handles PUT /api/uploads/<token>. Returns 200 + {size, sha256} on
// success.
//
// Error mapping:
//
//	404 — unknown token
//	405 — non-PUT method
//	410 — expired
//	409 — already uploaded / finalized (single-use)
//	413 — body exceeded the row's max_bytes
//	500 — disk / DB failures
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		writeJSONErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	token := tokenFromPath(r.URL.Path)
	if token == "" {
		writeJSONErr(w, http.StatusNotFound, "unknown token")
		return
	}
	row, err := h.store.Lookup(token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeJSONErr(w, http.StatusNotFound, "unknown token")
			return
		}
		writeJSONErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.store.ValidateForPUT(row); err != nil {
		switch {
		case errors.Is(err, ErrExpired):
			writeJSONErr(w, http.StatusGone, "token expired")
		case errors.Is(err, ErrAlreadyUploaded), errors.Is(err, ErrAlreadyFinalized):
			writeJSONErr(w, http.StatusConflict, err.Error())
		default:
			writeJSONErr(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	tempPath := h.store.TempPathFor(token)
	// O_EXCL guards against two concurrent PUTs with the same token writing to
	// the same file — only one Open succeeds.
	f, openErr := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if openErr != nil {
		if os.IsExist(openErr) {
			writeJSONErr(w, http.StatusConflict, "upload in progress")
			return
		}
		writeJSONErr(w, http.StatusInternalServerError, openErr.Error())
		return
	}
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(tempPath)
	}

	hasher := sha256.New()
	limited := io.LimitReader(r.Body, row.MaxBytes+1)
	written, copyErr := io.Copy(io.MultiWriter(f, hasher), limited)
	if copyErr != nil {
		cleanup()
		writeJSONErr(w, http.StatusInternalServerError, copyErr.Error())
		return
	}
	if written > row.MaxBytes {
		cleanup()
		writeJSONErr(w, http.StatusRequestEntityTooLarge, "body exceeds max_bytes")
		return
	}
	if err := f.Sync(); err != nil {
		cleanup()
		writeJSONErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tempPath)
		writeJSONErr(w, http.StatusInternalServerError, err.Error())
		return
	}

	sum := hex.EncodeToString(hasher.Sum(nil))
	if err := h.store.MarkUploaded(token, written, sum, tempPath); err != nil {
		_ = os.Remove(tempPath)
		switch {
		case errors.Is(err, ErrExpired):
			writeJSONErr(w, http.StatusGone, "token expired")
		case errors.Is(err, ErrAlreadyUploaded), errors.Is(err, ErrAlreadyFinalized):
			writeJSONErr(w, http.StatusConflict, err.Error())
		default:
			writeJSONErr(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"size":   written,
		"sha256": sum,
	})
}

// tokenFromPath extracts the trailing path segment after /api/uploads/.
// Works regardless of whether the route was mounted via Fiber's router or as
// a raw http.Handler at /api/uploads/.
func tokenFromPath(p string) string {
	const prefix = "/api/uploads/"
	idx := strings.Index(p, prefix)
	if idx < 0 {
		return ""
	}
	tail := p[idx+len(prefix):]
	// Defend against trailing slashes and accidental subpaths.
	if i := strings.IndexByte(tail, '/'); i >= 0 {
		tail = tail[:i]
	}
	return tail
}
