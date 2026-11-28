package layout

import (
	"math"
	"testing"
)

// cellFills returns the heights of all filled RectItems.
func cellFills(t *testing.T, src string) []float64 {
	t.Helper()
	var hs []float64
	for _, p := range run(t, src) {
		for _, it := range p.Items {
			if r, ok := it.(RectItem); ok && r.Fill != nil {
				hs = append(hs, r.H)
			}
		}
	}
	return hs
}

func TestTableCellHeightEmptyCells(t *testing.T) {
	hs := cellFills(t, `<table style="border-collapse:collapse"><tr>
<td style="width:30px;height:40px;background-color:#000"></td>
<td style="width:10px"></td>
<td style="width:20px;height:40px;background-color:#000"></td>
</tr></table>`)
	if len(hs) != 2 {
		t.Fatalf("got %d fills, want 2", len(hs))
	}
	for _, h := range hs {
		if math.Abs(h-30) > 0.01 {
			t.Errorf("fill height = %v, want 30pt (40px)", h)
		}
	}
}

func TestTableCellHeightIsMinimum(t *testing.T) {
	small := cellFills(t, `<table><tr><td style="height:1px;background-color:#000">A<br>B<br>C</td></tr></table>`)
	none := cellFills(t, `<table><tr><td style="background-color:#000">A<br>B<br>C</td></tr></table>`)
	if len(small) != 1 || len(none) != 1 || math.Abs(small[0]-none[0]) > 0.01 {
		t.Fatalf("small = %v, none = %v; content height must win", small, none)
	}
}

func TestTableCellHeightBorderBox(t *testing.T) {
	hs := cellFills(t, `<table><tr><td style="height:40px;padding:4px;border:1px solid #000;background-color:#000"></td></tr></table>`)
	if len(hs) != 1 || math.Abs(hs[0]-30) > 0.01 {
		t.Fatalf("fill heights = %v, want [30] (height is a border-box minimum)", hs)
	}
}
