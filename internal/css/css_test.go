package css

import (
	"math"
	"strings"
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/dom"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func find(n *dom.Node, tag string) *dom.Node {
	if n.Tag == tag {
		return n
	}
	for _, c := range n.Children {
		if r := find(c, tag); r != nil {
			return r
		}
	}
	return nil
}

func compute(t *testing.T, src string) (*dom.Document, *Computed) {
	t.Helper()
	doc, err := dom.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	return doc, ComputeStyles(doc, doc.StylesheetText)
}

func TestResolve(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"16px", 12}, {"2em", 24}, {"50%", 100}, {"25.4mm", 72}, {"2.54cm", 72},
		{"1in", 72}, {"3pt", 3}, {"1rem", 12}, {"0", 0},
	}
	for _, tt := range tests {
		l, ok := ParseLength(tt.in)
		if !ok {
			t.Fatalf("ParseLength(%q) failed", tt.in)
		}
		if got := l.Resolve(12, 12, 200); !near(got, tt.want) {
			t.Errorf("Resolve(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
	for _, bad := range []string{"2.em", "auto", "px", ""} {
		if _, ok := ParseLength(bad); ok {
			t.Errorf("ParseLength(%q) should fail", bad)
		}
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		in      string
		r, g, b float64
	}{
		{"navy", 0, 0, 128.0 / 255}, {"#fff", 1, 1, 1}, {"#000080", 0, 0, 128.0 / 255},
		{"rgb(255, 0, 0)", 1, 0, 0}, {"White", 1, 1, 1},
	}
	for _, tt := range tests {
		c, ok := ParseColor(tt.in)
		if !ok || !near(c.R, tt.r) || !near(c.G, tt.g) || !near(c.B, tt.b) {
			t.Errorf("ParseColor(%q) = %v, %v", tt.in, c, ok)
		}
	}
	if _, ok := ParseColor("nonsense"); ok {
		t.Error("ParseColor(nonsense) should fail")
	}
}

func TestCascade(t *testing.T) {
	doc, cs := compute(t, `<style>
p { color: red }
.a { color: blue }
div p { color: green }
p.a { color: navy }
* { margin: 0 }
.x { text-align: center }
</style>
<body><div><p class="a">t</p><p style="color: white">u</p><p>v</p></div></body>`)
	var ps []*dom.Node
	var walk func(n *dom.Node)
	walk = func(n *dom.Node) {
		if n.Tag == "p" {
			ps = append(ps, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(doc.Body)
	navy, _ := ParseColor("navy")
	white, _ := ParseColor("white")
	green, _ := ParseColor("green")
	if got := cs.Style(ps[0]).Color; got != navy {
		t.Errorf("p.a color = %v, want navy (class+element beats descendant)", got)
	}
	if got := cs.Style(ps[1]).Color; got != white {
		t.Errorf("inline style color = %v, want white", got)
	}
	if got := cs.Style(ps[2]).Color; got != green {
		t.Errorf("div p color = %v, want green (later rule wins on tie)", got)
	}
	// Author "* { margin: 0 }" overrides the UA "p { margin: 1em 0 }".
	if m := cs.Style(ps[0]).Margin[Top]; m.Value != 0 {
		t.Errorf("margin-top = %v, want 0", m)
	}
}

func TestInheritanceAndShorthand(t *testing.T) {
	doc, cs := compute(t, `<style>
body { font-size: 2em; text-align: center; padding: 1px 2px 3px }
div { border: 1px solid navy; border-bottom: 2px solid red; padding: 4px }
</style><body><div><span>x</span></div></body>`)
	body := cs.Style(doc.Body)
	if !near(body.FontSize.Value, 24) {
		t.Errorf("body font-size = %v pt, want 24", body.FontSize.Value)
	}
	if body.Padding[Right].Value != 2 || body.Padding[Bottom].Value != 3 || body.Padding[Left].Value != 2 {
		t.Errorf("padding expansion wrong: %v", body.Padding)
	}
	div := find(doc.Body, "div")
	span := find(doc.Body, "span")
	ds, ss := cs.Style(div), cs.Style(span)
	if ss.TextAlign != "center" || !near(ss.FontSize.Value, 24) {
		t.Errorf("span did not inherit: %+v", ss)
	}
	if ss.Padding[Top].Value != 0 || ss.BorderStyle[Top] != "none" {
		t.Errorf("non-inherited properties leaked into span")
	}
	navy, _ := ParseColor("navy")
	if ds.BorderStyle[Top] != "solid" || ds.BorderColor[Top] != navy || ds.BorderWidth[Top].Value != 1 {
		t.Errorf("border-top wrong: %v %v %v", ds.BorderStyle[Top], ds.BorderColor[Top], ds.BorderWidth[Top])
	}
	if ds.BorderWidth[Bottom].Value != 2 {
		t.Errorf("border-bottom width = %v, want 2", ds.BorderWidth[Bottom])
	}
}

func TestWidthHint(t *testing.T) {
	doc, cs := compute(t, `<style>.w { width: 30% }</style><body><table width="100%"><tr><td width="50%" class="w">x</td><td width="40"></td></tr></table></body>`)
	tds := find(doc.Body, "table").Children[0].Children[0].Children
	if w := cs.Style(find(doc.Body, "table")).Width; w == nil || w.Unit != UnitPercent || w.Value != 100 {
		t.Errorf("table width hint = %v", w)
	}
	if w := cs.Style(tds[0]).Width; w == nil || w.Value != 30 {
		t.Errorf("CSS should beat width attribute, got %v", w)
	}
	if w := cs.Style(tds[1]).Width; w == nil || w.Unit != UnitPX || w.Value != 40 {
		t.Errorf("bare numeric width hint = %v", w)
	}
}
