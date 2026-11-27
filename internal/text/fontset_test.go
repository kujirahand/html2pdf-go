package text

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func TestFaceForMonospace(t *testing.T) {
	fs, err := LoadFontSet(nil, "", "")
	if err != nil {
		t.Skipf("no usable base font: %v", err)
	}
	// When the font index contains any "mono" family, monospace must
	// resolve to one of them; only skip the strict check when the system
	// genuinely has no monospace font.
	fs.once.Do(fs.buildIndex)
	hasMono := false
	for k := range fs.index {
		if strings.Contains(k, "mono") {
			hasMono = true
		}
	}
	if got := fs.FaceFor([]string{"monospace"}, false); hasMono {
		if got == fs.Regular {
			t.Error("monospace fell back to the base face although mono families are installed")
		} else if name := strings.ToLower(got.FamilyName()); !strings.Contains(name, "mono") && !containsString(preferredMonoNames(runtime.GOOS, detectMonoLocaleClass()), name) {
			// A browser-default face such as Courier need not have
			// "mono" in its name; anything else is unexpected.
			t.Errorf("monospace resolved to %q, want a Mono family or a browser default", got.FamilyName())
		}
	}
	if got := fs.FaceFor([]string{"serif"}, false); got != fs.Regular {
		t.Error("serif should still give the base regular face")
	}
}

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

func TestIndexPrefersUprightRegular(t *testing.T) {
	fs, err := LoadFontSet(nil, "", "")
	if err != nil {
		t.Skipf("no system fonts: %v", err)
	}
	fs.once.Do(fs.buildIndex)
	checked := 0
	for _, key := range []string{"dejavu sans mono", "liberation mono", "noto sans mono"} {
		fam := fs.index[key]
		if fam == nil || fam.regular == "" {
			continue
		}
		f, err := loadFace(fam.regular)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(f.subfamily, "oblique") || strings.Contains(f.subfamily, "italic") {
			t.Errorf("%s: regular slot picked the slanted face %s (subfamily %q)", key, fam.regular, f.subfamily)
		}
		if strings.Contains(f.subfamily, "black") {
			t.Errorf("%s: regular slot picked a Black face %s", key, fam.regular)
		}
		checked++
	}
	if checked == 0 {
		t.Skip("no known monospace family installed")
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
