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

// A code run that wraps gets one background rectangle per line.
func TestCodeInlineBackgroundWrapped(t *testing.T) {
	got := codeFills(run(t, `<body><div style="width: 120px"><p>text <code>aaaa bbbb cccc dddd eeee ffff gggg</code></p></div></body>`))
	if len(got) < 2 {
		t.Fatalf("got %d background rects, want at least 2 for wrapped code", len(got))
	}
	ys := map[float64]bool{}
	for _, r := range got {
		ys[r.Y] = true
	}
	if len(ys) < 2 {
		t.Errorf("wrapped code backgrounds all share Y: %v", got)
	}
}

// The background hugs the code run's own font size, not the line box of
// neighbouring larger text.
func TestCodeInlineBackgroundRunHeight(t *testing.T) {
	plain := codeFills(run(t, "<body><p>a <code>xy</code> b</p></body>"))
	mixed := codeFills(run(t, `<body><p><span style="font-size: 32px">BIG</span> <code>xy</code></p></body>`))
	if len(plain) != 1 || len(mixed) != 1 {
		t.Fatalf("fills: plain %d, mixed %d, want 1 each", len(plain), len(mixed))
	}
	if d := mixed[0].H - plain[0].H; d > 1 || d < -1 {
		t.Errorf("code background height beside 32px text = %v, want about the plain height %v", mixed[0].H, plain[0].H)
	}
}

// A nested inline background wins for its own text; the outer span's
// background covers only the text outside the <code>.
func TestCodeInlineBackgroundNested(t *testing.T) {
	pages := run(t, `<body><p><span style="background-color: #dddddd">aa <code>xy</code> bb</span></p></body>`)
	if got := codeFills(pages); len(got) != 1 {
		t.Fatalf("got %d code background rects, want 1", len(got))
	}
	var outer int
	for _, p := range pages {
		for _, it := range p.Items {
			if r, ok := it.(RectItem); ok && r.Fill != nil && r.Fill.R == 0xdd/255.0 {
				outer++
			}
		}
	}
	if outer == 0 {
		t.Error("outer span background was not painted")
	}
}
