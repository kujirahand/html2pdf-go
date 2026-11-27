package text

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFontsRejectsFontWithoutJapaneseGlyphs(t *testing.T) {
	const src = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"
	data, err := os.ReadFile(src)
	if err != nil {
		t.Skipf("DejaVu Sans not available: %v", err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "DejaVuSans.ttf"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFonts([]string{dir}); err == nil {
		t.Fatal("LoadFonts accepted a font without Japanese glyphs, want error")
	}
}
