package layout

import (
	"fmt"
	"strconv"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
)

// maxTableCols limits the columns of one table (colspans included), which
// size several per-column arrays.
const maxTableCols = 10000

type tcell struct {
	n          *dom.Node
	st         *css.Style
	bm         boxm
	col, span  int
	minW, maxW float64 // outer (border-box) widths
	fixed      float64 // outer width, or -1
	pct        float64 // percentage of the table width, or -1
}

type trow struct {
	n     *dom.Node
	st    *css.Style
	cells []*tcell
}

func (l *layouter) collectRows(n *dom.Node, out *[]*trow, cw float64) {
	for _, c := range n.Children {
		if c.IsText() {
			continue
		}
		cs := l.styles.Style(c)
		switch cs.Display {
		case "table-row-group":
			l.collectRows(c, out, cw)
		case "table-row":
			r := &trow{n: c, st: cs}
			col := 0
			for _, cc := range c.Children {
				if cc.IsText() {
					continue
				}
				ccs := l.styles.Style(cc)
				if ccs.Display != "table-cell" {
					continue
				}
				span := 1
				if v, err := strconv.Atoi(cc.Attr["colspan"]); err == nil && v > 1 {
					span = min(v, 1000)
				}
				tc := &tcell{n: cc, st: ccs, col: col, span: span, fixed: -1, pct: -1}
				tc.bm = l.boxMetrics(ccs, cw)
				extra := tc.bm.hExtra()
				mn, mx := l.contentWidths(cc, ccs, cw)
				tc.minW, tc.maxW = mn+extra, mx+extra
				if w := ccs.Width; w != nil {
					if w.Unit == css.UnitPercent {
						tc.pct = w.Value
					} else {
						tc.fixed = w.Resolve(ccs.FontSize.Value, css.RootFontSize, cw) + extra
					}
				}
				r.cells = append(r.cells, tc)
				col += span
			}
			*out = append(*out, r)
		}
	}
}

// table lays out a table box. Widths follow a simple algorithm: explicit
// cell widths first, then the remaining columns get their max-content width
// (surplus shared equally), shrinking towards min-content when space is short.
func (l *layouter) table(n *dom.Node, st *css.Style, x, y, cw float64) float64 {
	if l.err != nil {
		return 0
	}
	fs := st.FontSize.Value
	bm := l.boxMetrics(st, cw)
	var rows []*trow
	l.collectRows(n, &rows, cw)

	ncols := 0
	for _, r := range rows {
		if len(r.cells) > 0 {
			last := r.cells[len(r.cells)-1]
			ncols = max(ncols, last.col+last.span)
		}
	}
	if ncols > maxTableCols {
		l.err = fmt.Errorf("layout: table has too many columns (%d)", ncols)
		return 0
	}
	// Sibling margin collapsing, as in block().
	overlap := 0.0
	if l.lastMB >= 0 && bm.m[css.Top] >= 0 {
		overlap = min(l.lastMB, bm.m[css.Top])
	}
	top := y + bm.m[css.Top] - overlap
	if ncols == 0 {
		l.lastMB = bm.m[css.Bottom]
		return top + bm.b[css.Top] + bm.p[css.Top] + bm.p[css.Bottom] + bm.b[css.Bottom] + bm.m[css.Bottom]
	}

	fixed := make([]float64, ncols)
	pct := make([]float64, ncols)
	minW := make([]float64, ncols)
	maxW := make([]float64, ncols)
	for i := range fixed {
		fixed[i], pct[i] = -1, -1
	}
	for _, r := range rows {
		for _, c := range r.cells {
			if c.span != 1 {
				continue
			}
			fixed[c.col] = max(fixed[c.col], c.fixed)
			pct[c.col] = max(pct[c.col], c.pct)
			minW[c.col] = max(minW[c.col], c.minW)
			maxW[c.col] = max(maxW[c.col], c.maxW)
		}
	}
	// Spanning cells only widen their columns when those are too narrow.
	for _, r := range rows {
		for _, c := range r.cells {
			if c.span == 1 {
				continue
			}
			var sMin, sMax float64
			for i := c.col; i < c.col+c.span; i++ {
				sMin += minW[i]
				sMax += maxW[i]
			}
			for i := c.col; i < c.col+c.span; i++ {
				if sMin < c.minW {
					minW[i] += (c.minW - sMin) / float64(c.span)
				}
				if sMax < c.maxW {
					maxW[i] += (c.maxW - sMax) / float64(c.span)
				}
			}
		}
	}

	extras := bm.hExtra()
	avail := cw - bm.m[css.Left] - bm.m[css.Right]
	autoWidth := st.Width == nil
	var tableW float64
	pref := make([]float64, ncols)
	var sumPref float64
	if autoWidth {
		for i := range pref {
			switch {
			case fixed[i] >= 0:
				pref[i] = fixed[i]
			case pct[i] >= 0:
				pref[i] = max(maxW[i], minW[i])
			default:
				pref[i] = maxW[i]
			}
			sumPref += pref[i]
		}
		tableW = min(avail, sumPref+extras)
	} else {
		tableW = max(st.Width.Resolve(fs, css.RootFontSize, cw), extras)
	}
	contentW := tableW - extras

	widths := make([]float64, ncols)
	if autoWidth && sumPref <= contentW+1e-6 {
		copy(widths, pref)
	} else {
		done := make([]bool, ncols)
		rem := contentW
		nAuto := 0
		for i := range widths {
			switch {
			case fixed[i] >= 0:
				widths[i], done[i] = fixed[i], true
			case pct[i] >= 0:
				widths[i], done[i] = contentW*pct[i]/100, true
			default:
				nAuto++
				continue
			}
			rem -= widths[i]
		}
		if nAuto > 0 {
			var sumMin, sumMax float64
			for i := range widths {
				if !done[i] {
					sumMin += minW[i]
					sumMax += max(maxW[i], minW[i])
				}
			}
			rem = max(rem, 0)
			switch {
			case rem >= sumMax:
				// Every auto column gets its max-content width; the surplus is shared equally.
				extra := (rem - sumMax) / float64(nAuto)
				for i := range widths {
					if !done[i] {
						widths[i] = max(maxW[i], minW[i]) + extra
					}
				}
			case rem > sumMin:
				// Shrink from max-content towards min-content proportionally.
				t := (rem - sumMin) / (sumMax - sumMin)
				for i := range widths {
					if !done[i] {
						widths[i] = minW[i] + (max(maxW[i], minW[i])-minW[i])*t
					}
				}
			default:
				for i := range widths {
					if !done[i] {
						widths[i] = minW[i]
					}
				}
			}
			nAuto = 0
		}
		if nAuto == 0 && !autoWidth {
			// All columns are fixed: scale them to fill the table.
			var sum float64
			for _, w := range widths {
				sum += w
			}
			if sum > 0 && sum < contentW-1e-6 {
				for i := range widths {
					widths[i] *= contentW / sum
				}
			}
		}
	}
	colX := make([]float64, ncols+1)

	left := x + bm.m[css.Left]
	if st.TextAlign == "center" && tableW+bm.m[css.Left]+bm.m[css.Right] < cw {
		left = x + (cw-tableW)/2
	}
	tx := left + bm.b[css.Left] + bm.p[css.Left]
	for i, w := range widths {
		colX[i+1] = colX[i] + w
	}

	slot := l.reserve()
	before := len(l.units)
	if id := n.Attr["id"]; id != "" {
		l.emit(prim{kind: pTarget, x: left, y: top, w: top, id: id})
	}
	yy := top + bm.b[css.Top] + bm.p[css.Top]
	for _, r := range rows {
		rowTop := yy
		if id := r.n.Attr["id"]; id != "" {
			l.emit(prim{kind: pTarget, x: tx, y: rowTop, w: rowTop, id: id})
		}
		heights := make([]float64, len(r.cells))
		var rowH float64
		for i, c := range r.cells {
			cellW := colX[c.col+c.span] - colX[c.col]
			innerW := max(cellW-c.bm.hExtra(), 0)
			var ch float64
			l.scratch(func() { ch = l.flow(c.n, c.st, 0, 0, innerW) })
			heights[i] = c.bm.b[css.Top] + c.bm.p[css.Top] + ch + c.bm.p[css.Bottom] + c.bm.b[css.Bottom]
			rowH = max(rowH, heights[i])
		}
		rowSlot := l.reserve()
		for i, c := range r.cells {
			cx := tx + colX[c.col]
			cellW := colX[c.col+c.span] - colX[c.col]
			innerW := max(cellW-c.bm.hExtra(), 0)
			if id := c.n.Attr["id"]; id != "" {
				l.emit(prim{kind: pTarget, x: cx, y: rowTop, w: rowTop, id: id})
			}
			if c.st.BackgroundColor != nil {
				l.emit(prim{kind: pFill, x: cx, y: rowTop, w: cellW, h: rowH, color: *c.st.BackgroundColor})
			}
			// Cell content is vertically centered, as in HTML's default.
			cy := rowTop + c.bm.b[css.Top] + c.bm.p[css.Top] + (rowH-heights[i])/2
			l.noUnits++
			l.flow(c.n, c.st, cx+c.bm.b[css.Left]+c.bm.p[css.Left], cy, innerW)
			l.noUnits--
			l.drawBorders(c.st, c.bm, cx, rowTop, cellW, rowH, [4]float64{}, [4]float64{}) // cell radii are not supported
		}
		if r.st.BackgroundColor != nil && rowH > 0 {
			l.prims[rowSlot] = prim{kind: pFill, x: tx, y: rowTop, w: contentW, h: rowH, color: *r.st.BackgroundColor}
		}
		if rowH > 0 {
			l.addUnit(rowTop, rowTop+rowH)
		}
		yy += rowH
	}
	bottom := yy + bm.p[css.Bottom] + bm.b[css.Bottom]
	rx, ry := boxRadii(st, tableW, bottom-top)
	if st.BackgroundColor != nil {
		l.prims[slot] = prim{kind: pFill, x: left, y: top, w: tableW, h: bottom - top, color: *st.BackgroundColor, rx: rx, ry: ry}
	}
	l.drawBorders(st, bm, left, top, tableW, bottom-top, rx, ry)
	l.extendUnits(before, top, bottom)
	if st.PageBreakAfter {
		l.forceBreak()
	}
	l.lastMB = bm.m[css.Bottom]
	return bottom + bm.m[css.Bottom]
}
