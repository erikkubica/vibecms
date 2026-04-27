package cms

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeBinary(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fakebin")
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestHashFile_Stable(t *testing.T) {
	path := writeBinary(t, "hello world")
	a, err := hashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := hashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("hash unstable: %s vs %s", a, b)
	}
	// Known SHA-256 of "hello world".
	const wanted = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if a != wanted {
		t.Fatalf("got %s want %s", a, wanted)
	}
}

func TestHashFile_DiffersOnTamper(t *testing.T) {
	a := writeBinary(t, "version-1")
	b := writeBinary(t, "version-2")
	hashA, _ := hashFile(a)
	hashB, _ := hashFile(b)
	if hashA == hashB {
		t.Fatal("different bytes produced same hash")
	}
}

func TestPluginPinSettingKey_BasenameOnly(t *testing.T) {
	// Pinning must survive an extension being moved to a different
	// install path — the key uses only the basename.
	got := pluginPinSettingKey("media-manager", "bin/media-manager-binary")
	want := "plugin.media-manager.media-manager-binary.sha256"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	gotBare := pluginPinSettingKey("media-manager", "media-manager-binary")
	if gotBare != want {
		t.Fatalf("basename-only path gave %q, want %q", gotBare, want)
	}
}

func TestVerifyPluginBinary_NilDB_Skips(t *testing.T) {
	// Tests/bootstrap path: nil DB → skip. The function must not
	// crash and must not block plugin loading.
	path := writeBinary(t, "anything")
	if err := verifyPluginBinary(nil, "x", "x", path); err != nil {
		t.Fatalf("nil db should be permissive, got %v", err)
	}
}

// Missing-file detection is the caller's job (StartPlugins does an
// os.Stat first). The verifier's contract starts at "if the file is
// here, hash it and compare to the pin", so we don't test missing
// file behavior — exercising that would just lock in incidental
// behavior of the early-return-on-nil-db short circuit.

// HashPluginBinary is the exported entry point used by the
// `vibecms verify-plugin` CLI helper. Make sure it returns the same
// value as the internal helper so the printed digest matches what
// the kernel will compute at load time.
func TestHashPluginBinary_MatchesInternal(t *testing.T) {
	path := writeBinary(t, "abc")
	internal, err := hashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := HashPluginBinary(path)
	if err != nil {
		t.Fatal(err)
	}
	if internal != exported {
		t.Fatalf("HashPluginBinary diverged from internal: %s vs %s", internal, exported)
	}
}

// keep imports honest when removing or changing tests
var _ = errors.Is
