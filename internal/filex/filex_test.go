package filex

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// setup returns a root directory holding in.txt, with a secret file in a
// sibling directory outside the root.
func setup(t *testing.T) (root, secret string) {
	t.Helper()
	parent := t.TempDir()
	root = filepath.Join(parent, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "in.txt"), []byte("inside"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret = filepath.Join(parent, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, secret
}

func TestReadUnderInside(t *testing.T) {
	root, _ := setup(t)
	data, err := ReadUnder(root, filepath.Join(root, "in.txt"), 100, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "inside" {
		t.Fatalf("got %q, want %q", data, "inside")
	}
}

func TestReadUnderEscapeRejected(t *testing.T) {
	root, secret := setup(t)
	for _, p := range []string{secret, filepath.Join(root, "..", "secret.txt")} {
		if _, err := ReadUnder(root, p, 100, false); err == nil {
			t.Errorf("ReadUnder(%q) outside the root was allowed", p)
		}
		if _, err := ReadUnder(root, p, 100, true); err != nil {
			t.Errorf("ReadUnder(%q, external): %v", p, err)
		}
	}
}

func TestReadUnderSymlinkEscapeRejected(t *testing.T) {
	root, secret := setup(t)
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(secret, link); err != nil {
		t.Skip(err)
	}
	if _, err := ReadUnder(root, link, 100, false); err == nil {
		t.Fatal("symlink escaping the root was read")
	}
}

func TestReadUnderNonRegular(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no /dev/null on Windows")
	}
	root, _ := setup(t)
	if _, err := ReadUnder(root, "/dev/null", 100, true); err == nil {
		t.Fatal("/dev/null was read as a regular file")
	}
}

func TestReadUnderSizeLimit(t *testing.T) {
	root, _ := setup(t)
	if _, err := ReadUnder(root, filepath.Join(root, "in.txt"), 3, false); err == nil {
		t.Fatal("file larger than the limit was read")
	}
}

func TestReadUnderNoRoot(t *testing.T) {
	root, _ := setup(t)
	p := filepath.Join(root, "in.txt")
	if _, err := ReadUnder("", p, 100, false); err == nil {
		t.Fatal("read with no root directory was allowed")
	}
	if _, err := ReadUnder("", p, 100, true); err != nil {
		t.Fatalf("external read with no root: %v", err)
	}
}

func TestOpenUnderReturnsFile(t *testing.T) {
	root, _ := setup(t)
	f, err := OpenUnder(root, filepath.Join(root, "in.txt"), false)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() != 6 {
		t.Fatalf("size = %d, want 6", fi.Size())
	}
}

func TestIsNetworkPath(t *testing.T) {
	for _, ref := range []string{"//host/share/x", `\\host\share\x`, `/\host\share\x`, `\/host/share/x`} {
		if !IsNetworkPath(ref) {
			t.Errorf("IsNetworkPath(%q) = false, want true", ref)
		}
	}
	for _, ref := range []string{"a/b.png", "/abs/path.png", "style.css", "../up.png"} {
		if IsNetworkPath(ref) {
			t.Errorf("IsNetworkPath(%q) = true, want false", ref)
		}
	}
}

// A nested document may reach files elsewhere in the root tree (including
// by absolute path), but never outside it.
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
	data, err := ReadUnder(root, filepath.Join(sub, "..", "shared.txt"), 100, false)
	if err != nil {
		t.Fatalf("in-root parent reference: %v", err)
	}
	if string(data) != "shared" {
		t.Fatalf("got %q, want %q", data, "shared")
	}
	if _, err := ReadUnder(root, shared, 100, false); err != nil {
		t.Fatalf("in-root absolute path: %v", err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	for _, p := range []string{outside, filepath.Join(root, "..", "outside.txt")} {
		if _, err := ReadUnder(root, p, 100, false); err == nil {
			t.Errorf("ReadUnder(%q) outside the root was allowed", p)
		}
	}
}
