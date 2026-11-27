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

func TestLoadFontsEarlierDirWinsOverNamePriority(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "fonts", "ipaexg.ttf"))
	if err != nil {
		t.Skipf("IPAex font not available: %v", err)
	}
	dir1, dir2 := t.TempDir(), t.TempDir()
	want := filepath.Join(dir1, "Custom.ttf")
	if err := os.WriteFile(want, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "NotoSansCJK-Regular.ttc"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	fonts, err := LoadFonts([]string{dir1, dir2})
	if err != nil {
		t.Fatal(err)
	}
	if got := fonts.Regular.path; got != want {
		t.Errorf("Regular loaded from %q, want %q", got, want)
	}
}
