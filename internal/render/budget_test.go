package render

import (
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/pdf"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
}

func newTestRenderer(rootDir string, allowExternal bool) *renderer {
	return &renderer{
		w:             pdf.NewWriter(io.Discard),
		rootDir:       rootDir,
		allowExternal: allowExternal,
		pdfFont:       map[*text.Face]*pdf.Font{},
		images:        map[string]*pdf.Image{},
	}
}

// Each image counts once towards the total budget; the one that would
// exceed it is refused, while an image already added stays usable.
func TestTotalImageBudget(t *testing.T) {
	saved := maxTotalImagePixels
	maxTotalImagePixels = 150
	defer func() { maxTotalImagePixels = saved }()

	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "a.png"), 10, 10)
	writePNG(t, filepath.Join(dir, "b.png"), 10, 10)
	r := newTestRenderer(dir, false)
	if _, err := r.image("a.png", dir); err != nil {
		t.Fatalf("first image: %v", err)
	}
	if _, err := r.image("a.png", dir); err != nil {
		t.Fatalf("cached image counted again: %v", err)
	}
	if r.totalPixels != 100 {
		t.Fatalf("totalPixels = %d, want 100", r.totalPixels)
	}
	if _, err := r.image("b.png", dir); err == nil {
		t.Fatal("image over the total budget was accepted")
	}
	if r.totalPixels != 100 {
		t.Fatalf("refused image was counted: totalPixels = %d", r.totalPixels)
	}
}

func TestImageConfinement(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "doc")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.png")
	writePNG(t, outside, 4, 4)
	for _, src := range []string{outside, "../outside.png"} {
		if _, err := newTestRenderer(dir, false).image(src, dir); err == nil {
			t.Errorf("image %q outside the directory was read by default", src)
		}
		if _, err := newTestRenderer(dir, true).image(src, dir); err != nil {
			t.Errorf("image %q with allowExternal: %v", src, err)
		}
	}
}

// A failed image is loaded at most once: after the first failure, raising
// the budget must not make a retry succeed, because no retry happens.
func TestFailedImageIsCached(t *testing.T) {
	saved := maxTotalImagePixels
	defer func() { maxTotalImagePixels = saved }()

	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "a.png"), 10, 10)
	r := newTestRenderer(dir, false)

	maxTotalImagePixels = 50 // too small for the 100-pixel image
	if _, err := r.image("a.png", dir); err == nil {
		t.Fatal("over-budget image was accepted")
	}
	maxTotalImagePixels = 1000 // would now fit, if it were loaded again
	if _, err := r.image("a.png", dir); err == nil {
		t.Fatal("failed image was retried instead of served from the failure cache")
	}
	if r.totalPixels != 0 {
		t.Fatalf("totalPixels = %d, want 0", r.totalPixels)
	}
}
