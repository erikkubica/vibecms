package uploads

import (
	"crypto/rand"
	"encoding/hex"
)

// newToken returns a 64-character hex string (~256 bits of entropy).
// Tokens are opaque; clients only echo them back in the URL and finalize call.
func newToken() (string, error) {
	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
