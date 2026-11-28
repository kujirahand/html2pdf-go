package text

import (
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestItalicFamilyResolution(t *testing.T) {
	dir := t.TempDir()
	for name, data := range map[string][]byte{"regular.ttf": goregular.TTF, "bold.ttf": gobold.TTF, "italic.ttf": goitalic.TTF, "bolditalic.ttf": gobolditalic.TTF} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	fs, err := LoadFontSet([]string{dir}, filepath.Join(dir, "regular.ttf"), "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if fs.Italic != nil || fs.BoldItalic != nil || fs.Bold != fs.Regular {
		t.Fatal("explicit regular must not auto-select styles")
	}
	for _, tc := range []struct {
		bold, italic bool
		name         string
	}{{false, false, "regular.ttf"}, {true, false, "bold.ttf"}, {false, true, "italic.ttf"}, {true, true, "bolditalic.ttf"}} {
		f := fs.FaceFor([]string{"Go"}, tc.bold, tc.italic)
		if f == nil || filepath.Base(f.path) != tc.name {
			t.Errorf("%+v: got %v", tc, f)
		}
	}
	fam := &family{regular: "r", bold: "b", italic: "i"}
	if got := fam.path(true, true); got != "b" {
		t.Errorf("bold italic fallback: %s", got)
	}
	fam.bold = ""
	if got := fam.path(true, true); got != "r" {
		t.Errorf("bold italic regular fallback: %s", got)
	}
	fam.italic = ""
	if got := fam.path(false, true); got != "r" {
		t.Errorf("italic fallback: %s", got)
	}
	for _, bad := range []int{0, 1} {
		a, b := "", ""
		if bad == 0 {
			a = filepath.Join(dir, "missing")
		} else {
			b = filepath.Join(dir, "missing")
		}
		if _, err := LoadFontSet(nil, filepath.Join(dir, "regular.ttf"), "", a, b); err == nil {
			t.Error("missing font accepted")
		}
	}
}

// TestBaseItalicAutoResolution checks that, without an explicit font, the
// base italic slots come from the base regular family (resolved lazily) and
// that an explicit italic file wins over them.
func TestBaseItalicAutoResolution(t *testing.T) {
	dir := t.TempDir()
	for name, data := range map[string][]byte{"regular.ttf": goregular.TTF, "bold.ttf": gobold.TTF, "italic.ttf": goitalic.TTF, "bolditalic.ttf": gobolditalic.TTF} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	// LoadFonts wants a font with Japanese glyphs, which the Go fonts are
	// not, so build the FontSet the way LoadFontSet does for the auto case.
	reg, err := loadFace(filepath.Join(dir, "regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	bold, err := loadFace(filepath.Join(dir, "bold.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	fs := &FontSet{dirs: []string{dir}, faces: map[string]*Face{}, Regular: reg, Bold: bold,
		autoItalicFamily: strings.ToLower(reg.family)}
	if fs.Italic != nil || fs.BoldItalic != nil {
		t.Fatal("italic slots must be resolved lazily")
	}
	for _, tc := range []struct {
		bold, italic bool
		name         string
	}{{false, false, "regular.ttf"}, {true, false, "bold.ttf"}, {false, true, "italic.ttf"}, {true, true, "bolditalic.ttf"}} {
		if f := fs.BaseFace(tc.bold, tc.italic); f == nil || filepath.Base(f.path) != tc.name {
			t.Errorf("%+v: got %v", tc, f)
		}
	}
	fs = &FontSet{dirs: []string{dir}, faces: map[string]*Face{}, Regular: reg, Bold: bold,
		autoItalicFamily: strings.ToLower(reg.family), explicitItalic: true}
	if fs.Italic, err = loadFace(filepath.Join(dir, "bolditalic.ttf")); err != nil {
		t.Fatal(err)
	}
	if f := fs.BaseFace(false, true); f == nil || filepath.Base(f.path) != "bolditalic.ttf" {
		t.Errorf("explicit italic file must win, got %v", f)
	}
	if f := fs.BaseFace(true, true); f == nil || filepath.Base(f.path) != "bolditalic.ttf" {
		t.Errorf("bold italic auto slot, got %v", f)
	}
}
