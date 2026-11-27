package text

import (
	"strings"
	"testing"
)

func testFace(t *testing.T) *Face {
	t.Helper()
	fonts, err := LoadFonts(nil)
	if err != nil {
		t.Skipf("no system font available: %v", err)
	}
	return fonts.Regular
}

func TestMeasure(t *testing.T) {
	f := testFace(t)
	w10 := f.Measure("abc", 10)
	if w10 <= 0 {
		t.Fatalf("Measure(abc, 10) = %v, want > 0", w10)
	}
	w20 := f.Measure("abc", 20)
	if d := w20 - 2*w10; d > 1e-6 || d < -1e-6 {
		t.Errorf("Measure should scale linearly: 10pt=%v 20pt=%v", w10, w20)
	}
	if f.Ascent(10) <= 0 || f.Descent(10) <= 0 {
		t.Errorf("Ascent/Descent should be positive: %v %v", f.Ascent(10), f.Descent(10))
	}
}

func TestLineMetricsTypo(t *testing.T) {
	f := testFace(t)
	if f.LineAscent(100) <= 0 || f.LineDescent(100) < 0 {
		t.Fatalf("invalid line metrics: %v %v", f.LineAscent(100), f.LineDescent(100))
	}
	if f.lineAsc == f.ascent && f.lineDsc == f.descent {
		t.Skip("font does not use OS/2 typo metrics")
	}
	line, full := f.LineAscent(100)+f.LineDescent(100), f.Ascent(100)+f.Descent(100)
	if line >= full {
		t.Errorf("typo line height %v should be smaller than Ascent+Descent %v", line, full)
	}
}

func TestBreakLinesBasic(t *testing.T) {
	f := testFace(t)
	if got := BreakLines(f, "", 10, 100); len(got) != 0 {
		t.Errorf("empty string: got %q, want no lines", got)
	}
	if got := BreakLines(f, "abc def", 10, 0); len(got) != 1 || got[0] != "abc def" {
		t.Errorf("maxWidth<=0: got %q", got)
	}
	width := f.Measure("hello world", 10) + 0.01
	got := BreakLines(f, "hello world foo", 10, width)
	if len(got) != 2 || got[0] != "hello world" || got[1] != "foo" {
		t.Errorf("word wrap: got %q", got)
	}
}

func TestBreakLinesKinsoku(t *testing.T) {
	f := testFace(t)
	w := f.Advance('あ', 10)
	if w <= 0 {
		t.Skip("font has no usable width for 'あ'")
	}
	s := "あいう、えお「かき」くけこ"
	got := BreakLines(f, s, 10, 3*w+0.01)
	if strings.Join(got, "") != s {
		t.Fatalf("lines do not reassemble: %q", got)
	}
	if got[0] != "あいう、" {
		t.Errorf("first line = %q, want the comma to hang on it", got[0])
	}
	for i, l := range got {
		rs := []rune(l)
		if i > 0 && noStart(rs[0]) {
			t.Errorf("line %d %q starts with a prohibited character", i, l)
		}
		if i < len(got)-1 && noEnd(rs[len(rs)-1]) {
			t.Errorf("line %d %q ends with a prohibited character", i, l)
		}
	}
}

func TestOutlineYUp(t *testing.T) {
	f := testFace(t)
	if f.GlyphID('H') == 0 {
		t.Skip("font has no 'H' glyph")
	}
	segs := f.Outline('H')
	if len(segs) == 0 {
		t.Fatal("Outline('H') is empty")
	}
	minY, maxY := 1e9, -1e9
	for _, s := range segs {
		for _, p := range s.Points {
			minY = min(minY, p.Y)
			maxY = max(maxY, p.Y)
		}
	}
	if maxY <= 0 || minY < -0.05*f.UnitsPerEm() {
		t.Errorf("'H' should sit above the baseline (Y up): minY=%v maxY=%v", minY, maxY)
	}
}
