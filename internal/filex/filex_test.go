package filex

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setup returns a base directory holding in.txt next to a secret file
// outside it.
func setup(t *testing.T) (base, secret string) {
	t.Helper()
	root := t.TempDir()
	base = filepath.Join(root, "doc")
	if err := os.Mkdir(base, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "in.txt"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret = filepath.Join(root, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	return base, secret
}

func TestReadInside(t *testing.T) {
	base, _ := setup(t)
	for _, allow := range []bool{false, true} {
		data, err := Read(base, "in.txt", 100, allow)
		if err != nil || string(data) != "inside" {
			t.Errorf("allow=%v: got %q, %v", allow, data, err)
		}
	}
}

func TestReadEscapeRejected(t *testing.T) {
	base, _ := setup(t)
	if _, err := Read(base, "../secret.txt", 100, false); err == nil {
		t.Error("../ escape was allowed")
	}
	if data, err := Read(base, "../secret.txt", 100, true); err != nil || string(data) != "secret" {
		t.Errorf("external ../: got %q, %v", data, err)
	}
}

func TestReadSymlinkEscapeRejected(t *testing.T) {
	base, secret := setup(t)
	if err := os.Symlink(secret, filepath.Join(base, "link.txt")); err != nil {
		t.Skip("symlinks unavailable:", err)
	}
	if _, err := Read(base, "link.txt", 100, false); err == nil {
		t.Error("symlink escape was allowed")
	}
}

func TestReadAbsolute(t *testing.T) {
	base, secret := setup(t)
	if _, err := Read(base, secret, 100, false); err == nil {
		t.Error("absolute path was allowed while confined")
	}
	if data, err := Read(base, secret, 100, true); err != nil || string(data) != "secret" {
		t.Errorf("external absolute: got %q, %v", data, err)
	}
}

func TestReadNonRegular(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /dev/null")
	}
	if _, err := Read("/", "dev/null", 100, false); err == nil {
		t.Error("device file was read while confined")
	}
	if _, err := Read("", "/dev/null", 100, true); err == nil {
		t.Error("device file was read in external mode")
	}
	if _, err := Open("", "/dev/zero", true); err == nil {
		t.Error("Open accepted /dev/zero")
	}
}

func TestReadSizeLimit(t *testing.T) {
	base, _ := setup(t)
	if err := os.WriteFile(filepath.Join(base, "big.txt"), []byte(strings.Repeat("x", 11)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(base, "big.txt", 10, false); err == nil {
		t.Error("file over the limit was read")
	}
	if data, err := Read(base, "big.txt", 11, false); err != nil || len(data) != 11 {
		t.Errorf("file at the limit: %d bytes, %v", len(data), err)
	}
}

func TestReadNetworkPathRejected(t *testing.T) {
	base, _ := setup(t)
	for _, ref := range []string{"//host/share/x", `\\host\share\x`} {
		for _, allow := range []bool{false, true} {
			if _, err := Read(base, ref, 100, allow); err == nil {
				t.Errorf("%q allow=%v was accepted", ref, allow)
			}
		}
	}
}

func TestOpenReturnsFile(t *testing.T) {
	base, _ := setup(t)
	f, err := Open(base, "in.txt", false)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

// ReadUnder confines an already-resolved path to the root: a nested
// document may reach files elsewhere in the root tree (including by
// absolute path), but never outside it.
func TestReadUnderRootConfinement(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	shared := filepath.Join(root, "shared.txt")
	if err := os.WriteFile(shared, []byte("shared"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Resolved against the nested document's directory, escaping into the
	// root is fine.
	data, err := ReadUnder(root, filepath.Join(sub, "..", "shared.txt"), 100, false)
	if err != nil {
		t.Fatalf("in-root parent reference: %v", err)
	}
	if string(data) != "shared" {
		t.Fatalf("got %q, want %q", data, "shared")
	}
	// An absolute path inside the root is fine too.
	if _, err := ReadUnder(root, shared, 100, false); err != nil {
		t.Fatalf("in-root absolute path: %v", err)
	}
	// Anything outside the root is refused, however it is written.
	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	for _, p := range []string{outside, filepath.Join(root, "..", "outside.txt")} {
		if _, err := ReadUnder(root, p, 100, false); err == nil {
			t.Errorf("ReadUnder(%q) outside the root was allowed", p)
		}
		if _, err := ReadUnder(root, p, 100, true); err != nil && p == outside {
			// allowExternal reads the path as given; the file does not
			// exist in this test, so only the confinement error matters.
			var pe *os.PathError
			if !errors.As(err, &pe) {
				t.Errorf("ReadUnder(%q, external): got %v, want a plain read error", p, err)
			}
		}
	}
}
