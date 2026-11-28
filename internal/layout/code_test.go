package layout

import (
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/text"
)

// textItems returns every TextItem on the pages.
func textItems(pages []*Page) []TextItem {
	var out []TextItem
	for _, p := range pages {
		for _, it := range p.Items {
			if t, ok := it.(TextItem); ok {
				out = append(out, t)
			}
		}
	}
	return out
}

// Runes the monospace face lacks are drawn with the default face instead of
// tofu, and ASCII keeps the monospace face.
func TestCodeCJKFallback(t *testing.T) {
	fonts, err := text.LoadFontSet(nil, "", "")
	if err != nil {
		t.Skipf("no system font available: %v", err)
	}
	mono := fonts.FaceFor([]string{"monospace"}, false)
	const jp = "こんにちは"
	if fonts.Regular.GlyphID('こ') == 0 || mono.GlyphID('こ') != 0 {
		t.Skip("needs a default face with Japanese glyphs and a monospace face without")
	}
	items := textItems(layoutWithFonts(t, `<body><pre>x := "`+jp+`"</pre></body>`, Options{}, fonts))
	var sawJP, sawASCII bool
	for _, it := range items {
		for _, r := range it.Text {
			switch {
			case r >= 0x3000:
				sawJP = true
				if it.Face == nil || it.Face.GlyphID(r) == 0 {
					t.Errorf("rune %q drawn with a face lacking its glyph (item %q)", r, it.Text)
				}
			case r == 'x':
				sawASCII = true
				if it.Face != mono {
					t.Errorf("ASCII item %q does not use the monospace face", it.Text)
				}
			}
		}
	}
	if !sawJP || !sawASCII {
		t.Errorf("items %v: sawJP=%v sawASCII=%v", items, sawJP, sawASCII)
	}
}

// Code the monospace face fully covers stays a single item with that face.
func TestCodeASCIISingleFace(t *testing.T) {
	fonts, err := text.LoadFontSet(nil, "", "")
	if err != nil {
		t.Skipf("no system font available: %v", err)
	}
	mono := fonts.FaceFor([]string{"monospace"}, false)
	items := textItems(layoutWithFonts(t, `<body><pre>x := 1</pre></body>`, Options{}, fonts))
	if len(items) != 1 || items[0].Face != mono {
		t.Errorf("got %d items %v, want one item with the monospace face", len(items), items)
	}
}

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
