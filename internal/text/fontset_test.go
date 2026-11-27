package text

import (
	"os"
	"testing"
)

const notoSerifCJK = "/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc"

func TestFontSetFaceForNamedFamily(t *testing.T) {
	if _, err := os.Stat(notoSerifCJK); err != nil {
		t.Skipf("Noto Serif CJK not available: %v", err)
	}
	fs, err := LoadFontSet(nil, "", "")
	if err != nil {
		t.Skipf("no usable base font: %v", err)
	}
	got := fs.FaceFor([]string{"Noto Serif CJK JP", "serif"}, false)
	if got == nil || got == fs.Regular {
		t.Fatalf("FaceFor(Noto Serif CJK JP) = %p, want a face distinct from the base %p", got, fs.Regular)
	}
	if got.FamilyName() != "Noto Serif CJK JP" {
		t.Errorf("FamilyName = %q, want Noto Serif CJK JP", got.FamilyName())
	}
	if lower := fs.FaceFor([]string{"noto serif cjk jp"}, false); lower != got {
		t.Error("family lookup should be case-insensitive and cached")
	}
}

func TestFontSetFaceForFallsBackToBase(t *testing.T) {
	fs, err := LoadFontSet(nil, "", "")
	if err != nil {
		t.Skipf("no usable base font: %v", err)
	}
	if got := fs.FaceFor([]string{"serif", "sans-serif", "No Such Family 12345"}, false); got != fs.Regular {
		t.Error("unknown and generic families should give the base regular face")
	}
	if got := fs.FaceFor(nil, true); got != fs.Bold {
		t.Error("no families with bold should give the base bold face")
	}
}

func TestLoadFontSetMissingFile(t *testing.T) {
	if _, err := LoadFontSet(nil, "/nonexistent/font.ttf", ""); err == nil {
		t.Error("missing regular font file should fail")
	}
	if _, err := LoadFontSet(nil, "", "/nonexistent/bold.ttf"); err == nil {
		t.Error("missing bold font file should fail")
	}
}

func TestLoadFontSetRegularOnlyUsesItForBold(t *testing.T) {
	if _, err := os.Stat(notoSerifCJK); err != nil {
		t.Skipf("Noto Serif CJK not available: %v", err)
	}
	fs, err := LoadFontSet(nil, notoSerifCJK, "")
	if err != nil {
		t.Fatal(err)
	}
	if fs.Bold != fs.Regular {
		t.Error("with only a regular file, bold should reuse the regular face")
	}
}
