package layout

import "testing"

// codeFills returns the RectItems filled with the UA <code> background.
func codeFills(pages []*Page) []RectItem {
	var out []RectItem
	for _, p := range pages {
		for _, it := range p.Items {
			if r, ok := it.(RectItem); ok && r.Fill != nil && r.Fill.R == 0xf0/255.0 && r.Fill.G == 0xf0/255.0 && r.Fill.B == 0xf0/255.0 {
				out = append(out, r)
			}
		}
	}
	return out
}

func TestCodeInlineBackground(t *testing.T) {
	got := codeFills(run(t, "<body><p>a <code>xy</code> b</p></body>"))
	if len(got) != 1 {
		t.Fatalf("got %d background rects, want 1", len(got))
	}
	if w := got[0].W; w <= 0 || w >= 595.28-16 {
		t.Errorf("background width = %v, want > 0 and less than the line width", w)
	}
	if got := codeFills(run(t, "<body><p>a xy b</p></body>")); len(got) != 0 {
		t.Errorf("document without <code> has %d background rects, want 0", len(got))
	}
}
