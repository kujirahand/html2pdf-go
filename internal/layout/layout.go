package layout

import (
	"fmt"
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
	pRoundStroke // rounded rectangle outline; x, y, w, h is the path, lw its width
	pFrame       // iframe document: items relative to x, y, clipped to w x h
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
	face       *text.Face
	underline  bool
	src        string
	dir        string // directory a pImage src resolves against
	// rx and ry are corner radii for pFill and pRoundStroke (top-left,
	// top-right, bottom-right, bottom-left); all zero means square.
	rx, ry [4]float64
	// style is the line style for pHLine and pVLine: "" or "solid",
	// "dashed" or "dotted".
	style string
	// items is the first page of the frame document for pFrame.
	items []Item
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
	fonts   *text.FontSet
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
	// err is the first error from an iframe's document or a tripped limit.
	err error
}

// maxLayoutWork bounds the work one document may cause. Nested tables lay
// out their cells repeatedly at every level, so cost grows exponentially
// with depth; this turns a hostile document into an error instead of a
// hang. A variable so tests can lower it.
var maxLayoutWork = 3_000_000

// step counts one unit of layout work and records an error once the budget
// is spent. The counter is shared by every nested layout of one conversion
// (Options.work). Callers check l.err to unwind.
func (l *layouter) step() {
	if l.err != nil {
		return
	}
	if l.opt.work == nil {
		l.opt.work = new(int)
	}
	*l.opt.work += 1
	if *l.opt.work > maxLayoutWork {
		l.err = fmt.Errorf("layout: document is too complex (work limit %d exceeded)", maxLayoutWork)
	}
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
		if s := st.BorderStyle[i]; s != "none" && s != "hidden" && s != "" {
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

// face returns the face for st's font-family list and weight.
func (l *layouter) face(st *css.Style) *text.Face {
	return l.fonts.FaceFor(st.FontFamily, st.Bold())
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

// boxRadii resolves the border radii of a w x h border box. X percentages
// refer to the box width and Y percentages to the box height. Following CSS
// (§5.5), if the radii on any side add up to more than that side's length,
// all radii are scaled down by the same factor.
func boxRadii(st *css.Style, w, h float64) (rx, ry [4]float64) {
	fs := st.FontSize.Value
	for i := 0; i < 4; i++ {
		rx[i] = max(0, st.BorderRadiusX[i].Resolve(fs, css.RootFontSize, w))
		ry[i] = max(0, st.BorderRadiusY[i].Resolve(fs, css.RootFontSize, h))
		if rx[i] == 0 || ry[i] == 0 {
			// A corner with a zero radius on either axis is square.
			rx[i], ry[i] = 0, 0
		}
	}
	f := 1.0
	fit := func(sum, length float64) {
		if sum > length && sum > 0 {
			f = min(f, length/sum)
		}
	}
	fit(rx[css.TopLeft]+rx[css.TopRight], w)
	fit(rx[css.BottomLeft]+rx[css.BottomRight], w)
	fit(ry[css.TopLeft]+ry[css.BottomLeft], h)
	fit(ry[css.TopRight]+ry[css.BottomRight], h)
	if f < 1 {
		for i := 0; i < 4; i++ {
			rx[i] *= f
			ry[i] *= f
		}
	}
	return rx, ry
}

// isRounded reports whether any corner has a non-zero radius.
func isRounded(rx, ry [4]float64) bool {
	for i := 0; i < 4; i++ {
		if rx[i] > 0 && ry[i] > 0 {
			return true
		}
	}
	return false
}

// drawBorders draws the borders of a box. With rounded corners and a uniform
// border (same width and color on every side) it emits one rounded stroke;
// borders that differ per side fall back to four straight lines, which ignore
// the radii. The rounded stroke is solid only, so a box with any dashed,
// dotted or double side also falls back to straight lines.
func (l *layouter) drawBorders(st *css.Style, bm boxm, x, y, w, h float64, rx, ry [4]float64) {
	b := bm.b
	allSolid := true
	for _, s := range st.BorderStyle {
		if s != "solid" {
			allSolid = false
		}
	}
	if allSolid && isRounded(rx, ry) && b[0] > 0 && b[0] == b[1] && b[0] == b[2] && b[0] == b[3] &&
		st.BorderColor[0] == st.BorderColor[1] && st.BorderColor[0] == st.BorderColor[2] && st.BorderColor[0] == st.BorderColor[3] {
		// Stroke along the middle of the border: inset the box by half the
		// width and shrink the radii to match.
		half := b[0] / 2
		p := prim{kind: pRoundStroke, x: x + half, y: y + half, w: max(w-b[0], 0), h: max(h-b[0], 0), lw: b[0], color: st.BorderColor[0]}
		for i := 0; i < 4; i++ {
			p.rx[i] = max(rx[i]-half, 0)
			p.ry[i] = max(ry[i]-half, 0)
		}
		l.emit(p)
		return
	}
	if b[css.Top] > 0 {
		l.emitBorderLine(pHLine, x, y+b[css.Top]/2, w, b[css.Top], st.BorderColor[css.Top], st.BorderStyle[css.Top])
	}
	if b[css.Bottom] > 0 {
		l.emitBorderLine(pHLine, x, y+h-b[css.Bottom]/2, w, b[css.Bottom], st.BorderColor[css.Bottom], st.BorderStyle[css.Bottom])
	}
	if b[css.Left] > 0 {
		l.emitBorderLine(pVLine, x+b[css.Left]/2, y, h, b[css.Left], st.BorderColor[css.Left], st.BorderStyle[css.Left])
	}
	if b[css.Right] > 0 {
		l.emitBorderLine(pVLine, x+w-b[css.Right]/2, y, h, b[css.Right], st.BorderColor[css.Right], st.BorderStyle[css.Right])
	}
}

// emitBorderLine emits one border side centered on (x, y) with length n
// along its axis (width for pHLine, height for pVLine). A double border of
// width lw becomes two solid lines of lw/3, offset by lw/3 either side of the
// center, so the PDF layer only needs to know about dashes and dots.
func (l *layouter) emitBorderLine(kind primKind, x, y, n, lw float64, color css.Color, style string) {
	mk := func(x, y, lw float64, style string) prim {
		p := prim{kind: kind, x: x, y: y, lw: lw, color: color, style: style}
		if kind == pHLine {
			p.w = n
		} else {
			p.h = n
		}
		return p
	}
	if style != "double" {
		l.emit(mk(x, y, lw, style))
		return
	}
	third := lw / 3
	if kind == pHLine {
		l.emit(mk(x, y-third, third, ""))
		l.emit(mk(x, y+third, third, ""))
	} else {
		l.emit(mk(x-third, y, third, ""))
		l.emit(mk(x+third, y, third, ""))
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

	rx, ry := boxRadii(st, boxW, bottom-top)
	if st.BackgroundColor != nil && bottom > top {
		l.prims[slot] = prim{kind: pFill, x: boxX, y: top, w: boxW, h: bottom - top, color: *st.BackgroundColor, rx: rx, ry: ry}
	}
	l.drawBorders(st, bm, boxX, top, boxW, bottom-top, rx, ry)
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

// iframeSize returns the content-box size of an <iframe>: the CSS (or
// width/height attribute) size, or the HTML default of 300x150 px. cw is the
// width a percentage width refers to.
func iframeSize(st *css.Style, cw float64) (w, h float64) {
	fs := st.FontSize.Value
	w, h = 300*0.75, 150*0.75
	if st.Width != nil {
		w = st.Width.Resolve(fs, css.RootFontSize, cw)
	}
	if st.Height != nil {
		h = st.Height.Resolve(fs, css.RootFontSize, 0)
	}
	return max(w, 0), max(h, 0)
}

// iframe lays out element n as a block-level embedded box. Its content box is
// the CSS or attribute size, or the HTML default of 300x150 px, and the frame
// document is placed in it. Margin, padding, border and background work as
// for a block; the box is one unbreakable unit.
func (l *layouter) iframe(n *dom.Node, st *css.Style, x, y, cw float64) float64 {
	bm := l.boxMetrics(st, cw)
	contentW, contentH := iframeSize(st, cw)
	boxX := x + bm.m[css.Left]
	boxW := contentW + bm.hExtra()
	boxH := contentH + bm.p[css.Top] + bm.p[css.Bottom] + bm.b[css.Top] + bm.b[css.Bottom]
	// Sibling margin collapsing, as in block.
	overlap := 0.0
	if l.lastMB >= 0 && bm.m[css.Top] >= 0 {
		overlap = min(l.lastMB, bm.m[css.Top])
	}
	top := y + bm.m[css.Top] - overlap
	bottom := top + boxH

	// Layout has no error path through block and flow, so the first frame
	// error is kept for Layout to return; later frames are skipped.
	var items []Item
	if contentW > 0 && contentH > 0 && l.err == nil {
		var err error
		if items, err = l.iframeItems(n, contentW, contentH); err != nil {
			l.err = err
		}
	}
	rx, ry := boxRadii(st, boxW, boxH)
	if st.BackgroundColor != nil && bottom > top {
		l.emit(prim{kind: pFill, x: boxX, y: top, w: boxW, h: boxH, color: *st.BackgroundColor, rx: rx, ry: ry})
	}
	if items != nil {
		l.emit(prim{kind: pFrame, x: boxX + bm.b[css.Left] + bm.p[css.Left], y: top + bm.b[css.Top] + bm.p[css.Top],
			w: contentW, h: contentH, items: items})
	}
	l.drawBorders(st, bm, boxX, top, boxW, boxH, rx, ry)
	if bottom > top {
		l.addUnit(top, bottom)
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
	l.step()
	if l.err != nil {
		return 0
	}
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
		case c.Tag == "iframe" && cs.Display == "block":
			flushRun()
			y = l.iframe(c, cs, x, y, cw)
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
