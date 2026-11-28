package css

import (
	"sort"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/warn"
)

// Computed maps document nodes to their computed styles.
type Computed struct {
	styles map[*dom.Node]*Style
}

// Style returns the computed style of n. Nodes outside the computed tree get
// the initial style.
func (c *Computed) Style(n *dom.Node) *Style {
	if c != nil {
		if s, ok := c.styles[n]; ok {
			return s
		}
	}
	s := defaultStyle()
	s.finish()
	return s
}

// uaSheet is the user-agent stylesheet.
const uaSheet = `
head, style, script, title, meta, link, noscript, template, object, embed, svg { display: none }
body, div, p, h1, h2, h3, h4, h5, h6, blockquote, pre, center, section, article, header, footer, address, page-break { display: block }
ul, ol { display: block }
iframe { display: block }
li { display: list-item }
table { display: table }
thead, tbody, tfoot { display: table-row-group }
tr { display: table-row }
td, th { display: table-cell }
caption { display: table-caption }

body { margin: 8px }
p, blockquote { margin: 1em 0 }
h1 { font-size: 2em; font-weight: bold; margin: 0.67em 0 }
h2 { font-size: 1.5em; font-weight: bold; margin: 0.83em 0 }
h3 { font-size: 1.17em; font-weight: bold; margin: 1em 0 }
h4 { font-weight: bold; margin: 1.33em 0 }
h5 { font-size: 0.83em; font-weight: bold; margin: 1.67em 0 }
h6 { font-size: 0.67em; font-weight: bold; margin: 2.33em 0 }
ul, ol { margin: 1em 0; padding-left: 40px }
ul ul, ul ol, ol ul, ol ol { margin: 0 }
b, strong { font-weight: bold }
u, ins { text-decoration: underline }
a { color: blue; text-decoration: underline }
center { text-align: center }
th { font-weight: bold; text-align: center }
td, th { padding: 1px }
page-break { page-break-after: always }
code, kbd, samp, tt { font-family: monospace; background-color: #f0f0f0 }
pre { font-family: monospace }
`

var uaRules = parseSheet(uaSheet)

// matched is a declaration block from a matching selector.
type matched struct {
	a, b, c, order int
	decls          []decl
}

// matchRules returns the declaration blocks of rules matching n, sorted by
// specificity and then source order.
func matchRules(rules []rule, n *dom.Node) []matched {
	var out []matched
	for i, r := range rules {
		for _, sel := range r.sels {
			if sel.matches(n) {
				a, b, c := sel.specificity()
				out = append(out, matched{a, b, c, i, r.decls})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		x, y := out[i], out[j]
		if x.a != y.a {
			return x.a < y.a
		}
		if x.b != y.b {
			return x.b < y.b
		}
		if x.c != y.c {
			return x.c < y.c
		}
		return x.order < y.order
	})
	return out
}

// ComputeStyles runs the cascade over doc.Body and its descendants. The
// cascade order, lowest to highest, is: HTML width and height attribute hints, user-agent
// stylesheet, author stylesheets (by specificity, then source order), and the
// inline style attribute.
func ComputeStyles(doc *dom.Document, stylesheetText []string) *Computed {
	return ComputeStylesW(doc, stylesheetText, nil)
}

func ComputeStylesW(doc *dom.Document, stylesheetText []string, w *warn.Collector) *Computed {
	c := &Computed{styles: map[*dom.Node]*Style{}}
	var author []rule
	for _, t := range stylesheetText {
		author = append(author, parseSheetW(t, w)...)
	}
	root := defaultStyle()
	root.finish()
	var walk func(n *dom.Node, parent *Style)
	walk = func(n *dom.Node, parent *Style) {
		s := inheritFrom(parent)
		if n.IsText() {
			s.finish()
			c.styles[n] = s
			return
		}
		checkAttrs(n, w)
		pfs := parent.FontSize.Value
		if v, ok := n.Attr["width"]; ok {
			if l, ok := ParseLength(v); ok && l.Value >= 0 {
				if l.Unit == UnitNone {
					l.Unit = UnitPX
				}
				s.Width = &l
			}
		}
		if v, ok := n.Attr["height"]; ok {
			// A percentage has no definite parent height to refer to, so
			// only plain pixel numbers are hints.
			if l, ok := ParseLength(v); ok && l.Value >= 0 && l.Unit == UnitNone {
				l.Unit = UnitPX
				s.Height = &l
			}
		}
		for _, rules := range [][]rule{uaRules, author} {
			for _, m := range matchRules(rules, n) {
				for _, d := range m.decls {
					s.apply(d.prop, d.val, pfs)
				}
			}
		}
		if inline, ok := n.Attr["style"]; ok {
			for _, d := range parseDecls(strings.TrimSpace(inline)) {
				checkDecl(d, w)
				s.apply(d.prop, d.val, pfs)
			}
		}
		s.finish()
		c.styles[n] = s
		for _, ch := range n.Children {
			walk(ch, s)
		}
	}
	if doc != nil && doc.Body != nil {
		walk(doc.Body, root)
	}
	return c
}
