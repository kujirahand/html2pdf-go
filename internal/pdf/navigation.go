package pdf

import (
	"fmt"
	"net/url"
	"strings"
)

type destination struct {
	page *Page
	x, y float64
}
type annotation struct {
	x, y, w, h float64
	uri        string
	dest       *destination
	id         int
}
type outline struct {
	title    string
	level    int
	dest     destination
	id       int
	parent   *outline
	children []*outline
}

// AddLink records a URI or an internal destination. Coordinates are top-left based.
func (p *Page) AddLink(x, y, w, h float64, uri string, target *Page, tx, ty float64) {
	a := annotation{x: x, y: y, w: w, h: h, uri: uri}
	if target != nil {
		a.dest = &destination{target, tx, ty}
	}
	p.annots = append(p.annots, a)
}

// AddBookmark adds a heading in document order.
func (w *Writer) AddBookmark(title string, level int, page *Page, x, y float64) {
	if title == "" || page == nil {
		return
	}
	o := &outline{title: title, level: level, dest: destination{page, x, y}}
	for len(w.outlineStack) > 0 && w.outlineStack[len(w.outlineStack)-1].level >= level {
		w.outlineStack = w.outlineStack[:len(w.outlineStack)-1]
	}
	if len(w.outlineStack) == 0 {
		w.outlines = append(w.outlines, o)
	} else {
		o.parent = w.outlineStack[len(w.outlineStack)-1]
		o.parent.children = append(o.parent.children, o)
	}
	w.outlineStack = append(w.outlineStack, o)
}

func (d destination) pdf() string {
	return fmt.Sprintf("[%d 0 R /XYZ %s %s null]", d.page.objID, num(d.page.sc(d.x)), num(d.page.fy(d.page.sc(d.y))))
}

func (w *Writer) allocNavigation() int {
	for _, p := range w.pages {
		for i := range p.annots {
			p.annots[i].id = w.alloc()
		}
	}
	if len(w.outlines) == 0 {
		return 0
	}
	root := w.alloc()
	var alloc func([]*outline)
	alloc = func(os []*outline) {
		for _, o := range os {
			o.id = w.alloc()
			alloc(o.children)
		}
	}
	alloc(w.outlines)
	return root
}

func (w *Writer) writeNavigation(root int) {
	for _, p := range w.pages {
		for _, a := range p.annots {
			x, y, ww, h := p.sc(a.x), p.sc(a.y), p.sc(a.w), p.sc(a.h)
			action := ""
			if a.dest != nil {
				action = " /Dest " + a.dest.pdf()
			} else {
				action = " /A << /S /URI /URI " + pdfString(a.uri) + " >>"
			}
			w.obj(a.id, fmt.Sprintf("<< /Type /Annot /Subtype /Link /Rect [%s %s %s %s] /Border [0 0 0]%s >>", num(x), num(p.fy(y+h)), num(x+ww), num(p.fy(y)), action))
		}
	}
	if root == 0 {
		return
	}
	var count func([]*outline) int
	count = func(os []*outline) int {
		n := 0
		for _, o := range os {
			n += 1 + count(o.children)
		}
		return n
	}
	w.obj(root, fmt.Sprintf("<< /Type /Outlines /First %d 0 R /Last %d 0 R /Count %d >>", w.outlines[0].id, w.outlines[len(w.outlines)-1].id, count(w.outlines)))
	var write func([]*outline, int)
	write = func(os []*outline, parent int) {
		for i, o := range os {
			body := fmt.Sprintf("<< /Title %s /Parent %d 0 R /Dest %s", pdfString(o.title), parent, o.dest.pdf())
			if i > 0 {
				body += fmt.Sprintf(" /Prev %d 0 R", os[i-1].id)
			}
			if i+1 < len(os) {
				body += fmt.Sprintf(" /Next %d 0 R", os[i+1].id)
			}
			if len(o.children) > 0 {
				body += fmt.Sprintf(" /First %d 0 R /Last %d 0 R /Count %d", o.children[0].id, o.children[len(o.children)-1].id, count(o.children))
			}
			w.obj(o.id, body+" >>")
			write(o.children, o.id)
		}
	}
	write(w.outlines, root)
}

// SafeURI accepts only external link schemes and encodes unsafe bytes.
func SafeURI(s string) string {
	s = strings.TrimSpace(s)
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host == "" {
			return ""
		}
	case "mailto":
		if u.Opaque == "" && u.Path == "" {
			return ""
		}
	default:
		return ""
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x80 || c == ' ' {
			fmt.Fprintf(&b, "%%%02X", c)
		} else {
			b.WriteByte(c)
		}
	}
	return b.String()
}
