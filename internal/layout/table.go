package layout

import (
	"errors"
	"fmt"
	"sort"
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
	rowspan    int
	height     float64 // natural border-box height of the content
	need       float64 // height the rows must provide (height, or a CSS height minimum)
	minW, maxW float64 // outer (border-box) widths
	fixed      float64 // outer width, or -1
	pct        float64 // percentage of the table width, or -1
}

type trow struct {
	st    *css.Style
	cells []*tcell
	group int
	head  bool
}

func (l *layouter) collectRows(n *dom.Node, out *[]*trow, cw float64) {
	group := 0
	seenHead := false
	var collect func(*dom.Node, int, bool)
	collect = func(parent *dom.Node, groupID int, head bool) {
		for _, c := range parent.Children {
			if c.IsText() {
				continue
			}
			cs := l.styles.Style(c)
			switch cs.Display {
			case "table-row-group":
				group++
				isHead := c.Tag == "thead" && !seenHead
				if c.Tag == "thead" {
					seenHead = true
				}
				collect(c, group, isHead)
				group++
				groupID = group
			case "table-row":
				r := &trow{st: cs, group: groupID, head: head}
				for _, cc := range c.Children {
					if cc.IsText() {
						continue
					}
					ccs := l.styles.Style(cc)
					if ccs.Display != "table-cell" {
						continue
					}
					span, rowspan := 1, 1
					if v, err := strconv.Atoi(cc.Attr["colspan"]); err == nil && v > 1 {
						span = min(v, 1000)
					}
					if v, err := strconv.Atoi(cc.Attr["rowspan"]); err == nil && v > 1 {
						rowspan = min(v, 1000)
					} else if errors.Is(err, strconv.ErrRange) && v > 0 {
						rowspan = 1000
					}
					tc := &tcell{n: cc, st: ccs, span: span, rowspan: rowspan, fixed: -1, pct: -1}
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
				}
				*out = append(*out, r)
			default:
				if parent == n {
					group++
					groupID = group
				}
			}
		}
	}
	collect(n, 0, false)
}

// pageSim mirrors pagination while table units are being emitted.
type pageSim struct {
	cur           float64
	placed, force bool
}

func (s *pageSim) wouldBreak(u unit, avail float64) bool {
	return !u.forced && s.placed && (s.force || u.bottom-s.cur > avail+pageEps)
}
func (s *pageSim) feed(u unit, avail float64) bool {
	if u.forced {
		s.force = true
		return false
	}
	broke := s.wouldBreak(u, avail)
	if broke {
		s.cur = u.top
	}
	s.placed, s.force = true, false
	return broke
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
	busy := make([]int, 0)
	end := 0
	for i, r := range rows {
		if i == end {
			end = i + 1
			for end < len(rows) && rows[end].group == r.group {
				end++
			}
		}
		col := 0
		for _, c := range r.cells {
			c.rowspan = min(c.rowspan, end-i)
			for {
				if col+c.span > maxTableCols {
					l.err = fmt.Errorf("layout: table has too many columns (%d)", col+c.span)
					return 0
				}
				conflict := -1
				for j := col; j < col+c.span && j < len(busy); j++ {
					if busy[j] > i {
						conflict = j
						break
					}
				}
				if conflict < 0 {
					break
				}
				col = conflict + 1
			}
			c.col = col
			for len(busy) < col+c.span {
				busy = append(busy, 0)
			}
			for j := col; j < col+c.span; j++ {
				busy[j] = i + c.rowspan
			}
			col += c.span
			ncols = max(ncols, col)
		}
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
	yy := top + bm.b[css.Top] + bm.p[css.Top]
	// Measure all cells before distributing spanning-cell height.
	heights := make([]float64, len(rows))
	type spanning struct {
		row  int
		cell *tcell
	}
	var spans []spanning
	for i, r := range rows {
		for _, c := range r.cells {
			cellW := colX[c.col+c.span] - colX[c.col]
			innerW := max(cellW-c.bm.hExtra(), 0)
			var ch float64
			l.scratch(func() { ch = l.flow(c.n, c.st, 0, 0, innerW) })
			if l.err != nil {
				return 0
			}
			c.height = c.bm.b[css.Top] + c.bm.p[css.Top] + ch + c.bm.p[css.Bottom] + c.bm.b[css.Bottom]
			// A specified height is a border-box minimum for the cell (as in
			// browsers, where table-cell height acts as a minimum). Percentages
			// are ignored: a cell has no definite base for them. It raises the
			// rows only, so the content stays vertically centred as before;
			// taller content still wins.
			c.need = c.height
			if h := c.st.Height; h != nil && h.Unit != css.UnitPercent {
				c.need = max(c.need, h.Resolve(c.st.FontSize.Value, css.RootFontSize, 0))
			}
			if c.rowspan == 1 {
				heights[i] = max(heights[i], c.need)
			} else {
				spans = append(spans, spanning{i, c})
			}
		}
	}
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].row+spans[i].cell.rowspan < spans[j].row+spans[j].cell.rowspan })
	for _, sp := range spans {
		end := sp.row + sp.cell.rowspan
		var h float64
		for _, v := range heights[sp.row:end] {
			h += v
		}
		if h < sp.cell.need {
			heights[end-1] += sp.cell.need - h
		}
	}
	type block struct {
		start, end int
		height     float64
	}
	var blocks []block
	for i := 0; i < len(rows); {
		end := i + 1
		for j := i; j < end; j++ {
			for _, c := range rows[j].cells {
				end = max(end, j+c.rowspan)
			}
		}
		var h float64
		for _, v := range heights[i:end] {
			h += v
		}
		blocks = append(blocks, block{i, end, h})
		i = end
	}
	emitBlock := func(blk block, y float64) {
		pos := y
		for i := blk.start; i < blk.end; i++ {
			if rows[i].st.BackgroundColor != nil && heights[i] > 0 {
				l.emit(prim{kind: pFill, x: tx, y: pos, w: contentW, h: heights[i], color: *rows[i].st.BackgroundColor})
			}
			pos += heights[i]
		}
		pos = y
		for i := blk.start; i < blk.end; i++ {
			for _, c := range rows[i].cells {
				if l.err != nil {
					return
				}
				var h float64
				for _, v := range heights[i : i+c.rowspan] {
					h += v
				}
				cx := tx + colX[c.col]
				cellW := colX[c.col+c.span] - colX[c.col]
				innerW := max(cellW-c.bm.hExtra(), 0)
				if c.st.BackgroundColor != nil {
					l.emit(prim{kind: pFill, x: cx, y: pos, w: cellW, h: h, color: *c.st.BackgroundColor})
				}
				cy := pos + c.bm.b[css.Top] + c.bm.p[css.Top] + (h-c.height)/2
				l.noUnits++
				l.flow(c.n, c.st, cx+c.bm.b[css.Left]+c.bm.p[css.Left], cy, innerW)
				l.noUnits--
				l.drawBorders(c.st, c.bm, cx, pos, cellW, h, [4]float64{}, [4]float64{})
			}
			pos += heights[i]
		}
	}
	// Only the first thead is repeatable; preserve all groups in source order.
	headStart, headEnd := -1, -1
	for i, b := range blocks {
		if rows[b.start].head {
			headStart = i
			group := rows[b.start].group
			headEnd = i
			for headEnd < len(blocks) && rows[blocks[headEnd].start].group == group {
				headEnd++
			}
			break
		}
	}
	repeat := l.noUnits == 0 && !l.opt.NoRepeatThead && headStart >= 0
	var sim pageSim
	if repeat {
		for _, u := range l.units {
			sim.feed(u, l.avail())
		}
	}
	var headH float64
	if repeat {
		for _, b := range blocks[headStart:headEnd] {
			headH += b.height
		}
	}
	register := func(top, bottom float64) {
		if bottom > top {
			l.addUnit(top, bottom)
			if repeat {
				sim.feed(unit{top: top, bottom: bottom}, l.avail())
			}
		}
	}
	for i := 0; i < len(blocks); i++ {
		b := blocks[i]
		if repeat && i == headStart {
			start := yy
			for ; i < headEnd; i++ {
				emitBlock(blocks[i], yy)
				if l.err != nil {
					return 0
				}
				yy += blocks[i].height
			}
			if i < len(blocks) {
				emitBlock(blocks[i], yy)
				if l.err != nil {
					return 0
				}
				yy += blocks[i].height
			}
			register(start, yy)
			continue
		}
		start := yy
		if repeat && i >= headEnd && sim.wouldBreak(unit{top: yy, bottom: yy + b.height}, l.avail()) && headH+b.height <= l.avail() {
			for _, hb := range blocks[headStart:headEnd] {
				emitBlock(hb, yy)
				yy += hb.height
			}
		}
		emitBlock(b, yy)
		yy += b.height
		register(start, yy)
		if l.err != nil {
			return 0
		}
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
