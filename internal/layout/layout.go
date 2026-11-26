package layout

import (
	"strconv"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

type primKind int

const (
	pNone primKind = iota
	pFill
	pHLine // horizontal line at y from x to x+w
	pVLine // vertical line at x from y to y+h
	pText
	pImage
)

// prim is a drawing primitive in flow coordinates.
type prim struct {
	kind       primKind
	x, y, w, h float64
	lw         float64
	color      css.Color
	text       string
	size       float64
	bold       bool
	underline  bool
	src        string
}

// unit is an atomic vertical extent that pagination never splits, or a
// forced page break marker.
type unit struct {
	top, bottom float64
	forced      bool
}

type marker struct {
	text string
	st   *css.Style
	x, w float64
}

type layouter struct {
	styles  *css.Computed
	fonts   *text.Fonts
	opt     Options
	prims   []prim
	units   []unit
	noUnits int // >0 while laying out table cell content
	marker  *marker
	imgs    map[string]*imgInfo
	// lastMB is the bottom margin of the previous block-level sibling in
	// the current flow, for sibling margin collapsing. flow() saves and
	// restores it so nested flows start with a clean state.
	lastMB float64
}

// boxm holds resolved margin, padding, and border widths (top, right, bottom, left).
type boxm struct{ m, p, b [4]float64 }

func (bm boxm) hExtra() float64 {
	return bm.p[css.Right] + bm.p[css.Left] + bm.b[css.Right] + bm.b[css.Left]
}

func (l *layouter) boxMetrics(st *css.Style, cw float64) boxm {
	fs := st.FontSize.Value
	var bm boxm
	for i := 0; i < 4; i++ {
		bm.m[i] = st.Margin[i].Resolve(fs, css.RootFontSize, cw)
		bm.p[i] = max(0, st.Padding[i].Resolve(fs, css.RootFontSize, cw))
		if st.BorderStyle[i] == "solid" {
			bm.b[i] = max(0, st.BorderWidth[i].Resolve(fs, css.RootFontSize, cw))
		}
	}
	return bm
}

func (l *layouter) emit(p prim) { l.prims = append(l.prims, p) }

func (l *layouter) reserve() int {
	l.prims = append(l.prims, prim{})
	return len(l.prims) - 1
}

func (l *layouter) addUnit(top, bottom float64) {
	if l.noUnits == 0 {
		l.units = append(l.units, unit{top: top, bottom: bottom})
	}
}

func (l *layouter) forceBreak() {
	if l.noUnits == 0 {
		l.units = append(l.units, unit{forced: true})
	}
}

// isWhitespaceRun reports whether a pending inline run contains nothing
// but whitespace text (no <br>, no <img>, no visible characters).
func isWhitespaceRun(run []*dom.Node) bool {
	var check func(n *dom.Node) bool
	check = func(n *dom.Node) bool {
		if n.IsText() {
			// Only ASCII whitespace counts; U+00A0 (&nbsp;) is visible content.
			return strings.Trim(n.Text, " \t\n\r\f\v") == ""
		}
		if n.Tag == "br" || n.Tag == "img" {
			return false
		}
		for _, c := range n.Children {
			if !check(c) {
				return false
			}
		}
		return true
	}
	for _, n := range run {
		if !check(n) {
			return false
		}
	}
	return true
}

// avail returns the usable height of one page.
func (l *layouter) avail() float64 {
	return l.opt.PageHeight - l.opt.MarginTop - l.opt.MarginBottom
}

// mergeUnits replaces the units added since index before with a single
// unit covering [top, bottom]. It reports false (and changes nothing)
// when a forced page break sits among them.
func (l *layouter) mergeUnits(before int, top, bottom float64) bool {
	if l.noUnits > 0 {
		return false
	}
	for i := before; i < len(l.units); i++ {
		if l.units[i].forced {
			return false
		}
	}
	if len(l.units) == before {
		l.units = append(l.units, unit{top: top, bottom: bottom})
		return true
	}
	l.units[before] = unit{top: top, bottom: bottom}
	l.units = l.units[:before+1]
	return true
}

// extendUnits stretches the units added since index before to cover a
// box's border and padding, so they stay on the same page as its content.
func (l *layouter) extendUnits(before int, top, bottom float64) {
	first, last := -1, -1
	for i := before; i < len(l.units); i++ {
		if !l.units[i].forced {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return
	}
	l.units[first].top = min(l.units[first].top, top)
	l.units[last].bottom = max(l.units[last].bottom, bottom)
}

// scratch runs f and discards everything it emitted; used for measuring.
func (l *layouter) scratch(f func()) {
	np, nu, mk := len(l.prims), len(l.units), l.marker
	f()
	l.prims, l.units, l.marker = l.prims[:np], l.units[:nu], mk
}

func (l *layouter) face(st *css.Style) *text.Face {
	if st.Bold() {
		return l.fonts.Bold
	}
	return l.fonts.Regular
}

// metrics returns the line-box ascent and descent contributed by st,
// including half-leading from its line-height.
func (l *layouter) metrics(st *css.Style) (asc, desc float64) {
	f, fs := l.face(st), st.FontSize.Value
	a, d := f.LineAscent(fs), f.LineDescent(fs)
	lh := a + d
	if st.LineHeightPt > 0 {
		lh = st.LineHeightPt
	} else if st.LineHeight > 0 {
		lh = st.LineHeight * fs
	}
	half := (lh - (a + d)) / 2
	return a + half, d + half
}

func isBlockLevel(display string) bool {
	switch display {
	case "block", "list-item", "table", "table-row", "table-cell", "table-row-group", "table-caption":
		return true
	}
	return false
}

func (l *layouter) drawBorders(st *css.Style, bm boxm, x, y, w, h float64) {
	b := bm.b
	if b[css.Top] > 0 {
		l.emit(prim{kind: pHLine, x: x, y: y + b[css.Top]/2, w: w, lw: b[css.Top], color: st.BorderColor[css.Top]})
	}
	if b[css.Bottom] > 0 {
		l.emit(prim{kind: pHLine, x: x, y: y + h - b[css.Bottom]/2, w: w, lw: b[css.Bottom], color: st.BorderColor[css.Bottom]})
	}
	if b[css.Left] > 0 {
		l.emit(prim{kind: pVLine, x: x + b[css.Left]/2, y: y, h: h, lw: b[css.Left], color: st.BorderColor[css.Left]})
	}
	if b[css.Right] > 0 {
		l.emit(prim{kind: pVLine, x: x + w - b[css.Right]/2, y: y, h: h, lw: b[css.Right], color: st.BorderColor[css.Right]})
	}
}

// block lays out element n as a block box inside a containing block whose
// content box has x-origin x and width cw, starting at flow position y.
// It returns the y position below the box including its bottom margin.
func (l *layouter) block(n *dom.Node, st *css.Style, x, y, cw float64) float64 {
	fs := st.FontSize.Value
	bm := l.boxMetrics(st, cw)
	var contentW float64
	if st.Width != nil {
		contentW = st.Width.Resolve(fs, css.RootFontSize, cw)
	} else {
		contentW = cw - bm.m[css.Left] - bm.m[css.Right] - bm.hExtra()
	}
	contentW = max(contentW, 0)
	boxX := x + bm.m[css.Left]
	boxW := contentW + bm.hExtra()
	// Sibling margin collapsing (CSS 2.1 §8.3.1, simplified): the gap
	// between two block siblings is max(prev bottom margin, this top
	// margin), not their sum. y already includes the previous margin, so
	// shift up by the overlap. Parent/child collapsing is not done.
	// Negative margins are not collapsed.
	overlap := 0.0
	if l.lastMB >= 0 && bm.m[css.Top] >= 0 {
		overlap = min(l.lastMB, bm.m[css.Top])
	}
	top := y + bm.m[css.Top] - overlap
	slot := l.reserve()
	before := len(l.units)

	cx := boxX + bm.b[css.Left] + bm.p[css.Left]
	if st.Display == "list-item" {
		l.marker = l.newMarker(n, st, cx)
	}
	end := l.flow(n, st, cx, top+bm.b[css.Top]+bm.p[css.Top], contentW)
	l.marker = nil
	bottom := end + bm.p[css.Bottom] + bm.b[css.Bottom]

	if st.BackgroundColor != nil && bottom > top {
		l.prims[slot] = prim{kind: pFill, x: boxX, y: top, w: boxW, h: bottom - top, color: *st.BackgroundColor}
	}
	l.drawBorders(st, bm, boxX, top, boxW, bottom-top)
	hasBorder := bm.b[css.Top]+bm.b[css.Right]+bm.b[css.Bottom]+bm.b[css.Left] > 0
	if len(l.units) == before && bottom > top {
		l.addUnit(top, bottom)
	} else if (hasBorder || st.BackgroundColor != nil) && bottom-top <= l.avail() && l.mergeUnits(before, top, bottom) {
		// A bordered or filled box that fits on one page is kept together:
		// splitting it at a line boundary would orphan its bottom border
		// and background on the next page.
	} else {
		l.extendUnits(before, top, bottom)
	}
	if st.PageBreakAfter {
		l.forceBreak()
	}
	l.lastMB = bm.m[css.Bottom]
	return bottom + bm.m[css.Bottom]
}

func (l *layouter) newMarker(n *dom.Node, st *css.Style, x float64) *marker {
	s := "• "
	if p := n.Parent; p != nil && p.Tag == "ol" {
		idx := 0
		for _, c := range p.Children {
			if c.IsText() {
				continue
			}
			if l.styles.Style(c).Display == "list-item" {
				idx++
			}
			if c == n {
				break
			}
		}
		s = strconv.Itoa(idx) + ". "
	}
	return &marker{text: s, st: st, x: x, w: l.face(st).Measure(s, st.FontSize.Value)}
}

// flow lays out the children of n inside a content box and returns the end y.
// Consecutive inline-level children form anonymous inline runs.
func (l *layouter) flow(n *dom.Node, st *css.Style, x, y, cw float64) float64 {
	savedMB := l.lastMB
	l.lastMB = 0
	defer func() { l.lastMB = savedMB }()
	var run []*dom.Node
	flush := func() {
		if len(run) == 0 {
			return
		}
		var segs []seg
		for _, c := range run {
			l.collect(c, &segs)
		}
		run = run[:0]
		y = l.inline(segs, st, x, y, cw)
		l.lastMB = 0 // inline content breaks margin adjacency
	}
	// flushRun flushes pending inline content before a block-level
	// sibling. A run of pure whitespace (the newlines between elements
	// in the source) is dropped instead: it must not break margin
	// adjacency, matching how browsers treat whitespace-only text
	// between blocks.
	flushRun := func() {
		if isWhitespaceRun(run) {
			run = run[:0]
			return
		}
		flush()
	}
	for _, c := range n.Children {
		if c.IsText() {
			run = append(run, c)
			continue
		}
		cs := l.styles.Style(c)
		switch {
		case cs.Display == "none":
		case cs.Display == "table":
			flushRun()
			y = l.table(c, cs, x, y, cw)
		case isBlockLevel(cs.Display):
			flushRun()
			y = l.block(c, cs, x, y, cw)
		default:
			run = append(run, c)
		}
	}
	flush()
	return y
}
