package layout

import (
	"math"
	"testing"
)

func lineItems(pg *Page) []LineItem {
	var out []LineItem
	for _, it := range pg.Items {
		if l, ok := it.(LineItem); ok {
			out = append(out, l)
		}
	}
	return out
}

func TestDashedBorderStyle(t *testing.T) {
	pages := run(t, `<div style="border: 2px dashed black">hello</div>`)
	lines := lineItems(pages[0])
	if len(lines) != 4 {
		t.Fatalf("got %d lines, want 4", len(lines))
	}
	for _, l := range lines {
		if l.Style != "dashed" {
			t.Errorf("line style = %q, want dashed", l.Style)
		}
	}
}

func TestDoubleBorderSplitsIntoTwoSolidLines(t *testing.T) {
	pages := run(t, `<div style="border-top: 6px double black">hello</div>`)
	lines := lineItems(pages[0])
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	// 6px = 4.5pt, so each line is 1.5pt wide, offset 1.5pt from the center.
	w := 4.5 / 3
	for _, l := range lines {
		if l.Style != "" && l.Style != "solid" {
			t.Errorf("line style = %q, want solid", l.Style)
		}
		if math.Abs(l.LineWidth-w) > 1e-6 {
			t.Errorf("line width = %v, want %v", l.LineWidth, w)
		}
	}
	if d := math.Abs(lines[0].Y1 - lines[1].Y1); math.Abs(d-2*w) > 1e-6 {
		t.Errorf("lines are %v apart, want %v", d, 2*w)
	}
}

func TestRoundedDashedFallsBackToLines(t *testing.T) {
	pages := run(t, `<div style="border: 2px dashed black; border-radius: 8px">hello</div>`)
	if _, stroke := rectItems(pages[0]); stroke != nil {
		t.Errorf("dashed border should not use the rounded stroke")
	}
	if len(lineItems(pages[0])) != 4 {
		t.Errorf("want 4 straight dashed lines")
	}
}
