package layout

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

const (
	nbsp    = ' '
	objChar = '￼' // stands in for an inline image in a run
)

type segKind int

const (
	segText segKind = iota
	segBr           // forced line break
	segSoft         // end of paragraph only if one is open (block inside inline)
	segImg
)

// seg is a piece of inline content.
type seg struct {
	kind segKind
	text string // character data, or the src of an image
	st   *css.Style
}

type imgBox struct {
	path string
	w, h float64
}

type imgInfo struct {
	w, h int
	ok   bool
}

type cellr struct {
	r   rune
	st  *css.Style
	img *imgBox
}

type para struct {
	cells []cellr
	endSt *css.Style // style used for the height of an empty line
}

// collect appends the inline content of n to out.
func (l *layouter) collect(n *dom.Node, out *[]seg) {
	if n.IsText() {
		*out = append(*out, seg{kind: segText, text: n.Text, st: l.styles.Style(n)})
		return
	}
	st := l.styles.Style(n)
	if st.Display == "none" {
		return
	}
	switch n.Tag {
	case "br":
		*out = append(*out, seg{kind: segBr, st: st})
		return
	case "img":
		*out = append(*out, seg{kind: segImg, text: n.Attr["src"], st: st})
		return
	}
	block := isBlockLevel(st.Display)
	if block {
		*out = append(*out, seg{kind: segSoft, st: st})
	}
	for _, c := range n.Children {
		l.collect(c, out)
	}
	if block {
		*out = append(*out, seg{kind: segSoft, st: st})
	}
}

// loadImage reads (and caches) the dimensions of an image file.
func (l *layouter) loadImage(src string) (string, *imgInfo) {
	if src == "" || strings.Contains(src, "://") || strings.HasPrefix(src, "data:") {
		return "", &imgInfo{}
	}
	p := src
	if !filepath.IsAbs(p) {
		p = filepath.Join(l.opt.BaseDir, p)
	}
	if info, ok := l.imgs[p]; ok {
		return p, info
	}
	info := &imgInfo{}
	if f, err := os.Open(p); err == nil {
		if cfg, _, err := image.DecodeConfig(f); err == nil && cfg.Width > 0 && cfg.Height > 0 {
			info = &imgInfo{w: cfg.Width, h: cfg.Height, ok: true}
		}
		f.Close()
	}
	l.imgs[p] = info
	return p, info
}

// imageBox sizes an <img>; it returns nil if the image cannot be used.
func (l *layouter) imageBox(st *css.Style, src string, cw float64) *imgBox {
	path, info := l.loadImage(src)
	if !info.ok {
		return nil
	}
	w, h := float64(info.w)*0.75, float64(info.h)*0.75
	if st.Width != nil {
		nw := st.Width.Resolve(st.FontSize.Value, css.RootFontSize, cw)
		if nw > 0 {
			w, h = nw, nw*float64(info.h)/float64(info.w)
		}
	}
	return &imgBox{path: path, w: w, h: h}
}

func isSpace(r rune) bool { return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' }

func hasContent(cells []cellr) bool {
	for _, c := range cells {
		if c.r != ' ' {
			return true
		}
	}
	return false
}

// paragraphs normalizes whitespace and splits segs into paragraphs at <br>.
// A <br> always ends a paragraph (even an empty one); a trailing paragraph
// is kept only if it has visible content.
func (l *layouter) paragraphs(segs []seg, cw float64) []para {
	var out []para
	var cur []cellr
	var last *css.Style
	for _, s := range segs {
		last = s.st
		switch s.kind {
		case segText:
			for _, r := range s.text {
				if isSpace(r) {
					if len(cur) > 0 && cur[len(cur)-1].r != ' ' {
						cur = append(cur, cellr{r: ' ', st: s.st})
					}
					continue
				}
				cur = append(cur, cellr{r: r, st: s.st})
			}
		case segBr:
			out = append(out, para{cells: cur, endSt: s.st})
			cur = nil
		case segSoft:
			if hasContent(cur) {
				out = append(out, para{cells: cur, endSt: s.st})
			}
			cur = nil
		case segImg:
			if box := l.imageBox(s.st, s.text, cw); box != nil {
				cur = append(cur, cellr{r: objChar, st: s.st, img: box})
			}
		}
	}
	if hasContent(cur) {
		out = append(out, para{cells: cur, endSt: last})
	}
	return out
}

func (l *layouter) advance(c cellr) float64 {
	if c.img != nil {
		return c.img.w
	}
	r := c.r
	if r == nbsp {
		r = ' '
	}
	return l.face(c.st).Advance(r, c.st.FontSize.Value)
}

// inline lays out an inline run as line boxes inside the block style bst.
func (l *layouter) inline(segs []seg, bst *css.Style, x, y, cw float64) float64 {
	for _, p := range l.paragraphs(segs, cw) {
		rs := make([]rune, len(p.cells))
		adv := make([]float64, len(p.cells))
		for i, c := range p.cells {
			rs[i], adv[i] = c.r, l.advance(c)
		}
		spans := text.BreakSpans(rs, adv, cw)
		if len(spans) == 0 {
			spans = [][2]int{{0, 0}}
		}
		for _, sp := range spans {
			y = l.line(p.cells[sp[0]:sp[1]], adv[sp[0]:sp[1]], p.endSt, bst, x, y, cw)
		}
	}
	return y
}

// line emits one line box at y and returns the y below it.
func (l *layouter) line(cells []cellr, adv []float64, emptySt, bst *css.Style, x, y, cw float64) float64 {
	asc, desc := l.metrics(bst)
	if len(cells) == 0 && emptySt != nil {
		a, d := l.metrics(emptySt)
		asc, desc = max(asc, a), max(desc, d)
	}
	var width float64
	for i, c := range cells {
		width += adv[i]
		if c.img != nil {
			asc = max(asc, c.img.h)
			continue
		}
		a, d := l.metrics(c.st)
		asc, desc = max(asc, a), max(desc, d)
	}
	baseline := y + asc
	x0 := x
	switch bst.TextAlign {
	case "right":
		x0 = x + cw - width
	case "center":
		x0 = x + (cw-width)/2
	}
	x0 = max(x0, x)

	if m := l.marker; m != nil {
		l.marker = nil
		l.emit(prim{kind: pText, x: m.x - m.w, y: baseline, text: m.text, size: m.st.FontSize.Value,
			bold: m.st.Bold(), color: m.st.Color})
	}

	cx := x0
	for i := 0; i < len(cells); {
		c := cells[i]
		if c.img != nil {
			l.emit(prim{kind: pImage, x: cx, y: baseline - c.img.h, w: c.img.w, h: c.img.h, src: c.img.path})
			cx += adv[i]
			i++
			continue
		}
		var sb strings.Builder
		var w float64
		j := i
		for j < len(cells) && cells[j].img == nil && cells[j].st == c.st {
			r := cells[j].r
			if r == nbsp {
				r = ' '
			}
			sb.WriteRune(r)
			w += adv[j]
			j++
		}
		s := sb.String()
		ul := c.st.TextDecoration == "underline"
		if ul || strings.TrimSpace(s) != "" {
			l.emit(prim{kind: pText, x: cx, y: baseline, text: s, size: c.st.FontSize.Value,
				bold: c.st.Bold(), color: c.st.Color, underline: ul})
		}
		cx += w
		i = j
	}
	l.addUnit(y, y+asc+desc)
	return y + asc + desc
}

// wide approximates text.isWide: characters that may break on either side.
func wide(r rune) bool { return r >= 0x2E80 }

// inlineWidths returns the min-content (longest unbreakable word) and
// max-content (longest unwrapped paragraph) widths of an inline run.
func (l *layouter) inlineWidths(segs []seg, cw float64) (mn, mx float64) {
	for _, p := range l.paragraphs(segs, cw) {
		var total, word float64
		for _, c := range p.cells {
			a := l.advance(c)
			total += a
			switch {
			case c.r == ' ':
				mn, word = max(mn, word), 0
			case wide(c.r) && c.img == nil:
				mn, word = max(mn, word, a), 0
			default:
				word += a
			}
		}
		mn, mx = max(mn, word), max(mx, total)
	}
	return mn, mx
}

// contentWidths estimates the min- and max-content widths of n's content box.
func (l *layouter) contentWidths(n *dom.Node, st *css.Style, cw float64) (mn, mx float64) {
	sum := st.Display == "table-row"
	var run []*dom.Node
	add := func(a, b float64) {
		if sum {
			mn, mx = mn+a, mx+b
		} else {
			mn, mx = max(mn, a), max(mx, b)
		}
	}
	flush := func() {
		if len(run) == 0 {
			return
		}
		var segs []seg
		for _, c := range run {
			l.collect(c, &segs)
		}
		run = run[:0]
		add(l.inlineWidths(segs, cw))
	}
	for _, c := range n.Children {
		if c.IsText() {
			run = append(run, c)
			continue
		}
		cs := l.styles.Style(c)
		if cs.Display == "none" {
			continue
		}
		if !isBlockLevel(cs.Display) {
			run = append(run, c)
			continue
		}
		flush()
		a, b := l.contentWidths(c, cs, cw)
		bm := l.boxMetrics(cs, cw)
		if cs.Width != nil && cs.Width.Unit != css.UnitPercent {
			w := cs.Width.Resolve(cs.FontSize.Value, css.RootFontSize, cw)
			a, b = w, w
		}
		e := bm.hExtra() + bm.m[css.Left] + bm.m[css.Right]
		add(a+e, b+e)
	}
	flush()
	return mn, mx
}
