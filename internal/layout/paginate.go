package layout

import "sort"

const pageEps = 0.01

// paginate chooses page start positions in flow space and maps every
// primitive to its page.
func (l *layouter) paginate() []*Page {
	o := l.opt
	avail := o.PageHeight - o.MarginTop - o.MarginBottom
	starts := []float64{0}
	placed, force := false, false
	for _, u := range l.units {
		if u.forced {
			force = true
			continue
		}
		cur := starts[len(starts)-1]
		if placed && (force || u.bottom-cur > avail+pageEps) {
			starts = append(starts, u.top)
		}
		placed, force = true, false
	}

	pages := make([]*Page, len(starts))
	for i := range pages {
		pages[i] = &Page{Width: o.PageWidth, Height: o.PageHeight}
	}
	pageOf := func(y float64) int {
		k := sort.Search(len(starts), func(i int) bool { return starts[i] > y+pageEps }) - 1
		return max(k, 0)
	}
	// span clips the vertical extent [y, y+h] to page k.
	span := func(k int, y, h float64) (top, bot float64) {
		top, bot = max(y, starts[k]), y+h
		if k+1 < len(starts) {
			bot = min(bot, starts[k+1])
		}
		return top, bot
	}
	off := func(k int) float64 { return o.MarginTop - starts[k] }

	for _, p := range l.prims {
		switch p.kind {
		case pFill:
			for k := pageOf(p.y); k <= pageOf(p.y+p.h-pageEps); k++ {
				if top, bot := span(k, p.y, p.h); bot-top > pageEps {
					c := p.color
					pages[k].Items = append(pages[k].Items, RectItem{X: p.x, Y: top + off(k), W: p.w, H: bot - top, Fill: &c, RadiusX: p.rx, RadiusY: p.ry})
				}
			}
		case pRoundStroke:
			for k := pageOf(p.y); k <= pageOf(p.y+p.h-pageEps); k++ {
				if top, bot := span(k, p.y, p.h); bot-top > pageEps {
					c := p.color
					pages[k].Items = append(pages[k].Items, RectItem{X: p.x, Y: top + off(k), W: p.w, H: bot - top, Stroke: &c, LineWidth: p.lw, RadiusX: p.rx, RadiusY: p.ry})
				}
			}
		case pHLine:
			k := pageOf(p.y)
			y := p.y + off(k)
			pages[k].Items = append(pages[k].Items, LineItem{X1: p.x, Y1: y, X2: p.x + p.w, Y2: y, LineWidth: p.lw, Color: p.color, Style: p.style})
		case pVLine:
			for k := pageOf(p.y); k <= pageOf(p.y+p.h-pageEps); k++ {
				if top, bot := span(k, p.y, p.h); bot-top > pageEps {
					pages[k].Items = append(pages[k].Items, LineItem{X1: p.x, Y1: top + off(k), X2: p.x, Y2: bot + off(k), LineWidth: p.lw, Color: p.color, Style: p.style})
				}
			}
		case pText:
			k := pageOf(p.y)
			pages[k].Items = append(pages[k].Items, TextItem{Text: p.text, X: p.x, Y: p.y + off(k), Size: p.size,
				Bold: p.bold, Face: p.face, Color: p.color, Underline: p.underline})
		case pImage:
			k := pageOf(p.y)
			pages[k].Items = append(pages[k].Items, ImageItem{Src: p.src, X: p.x, Y: p.y + off(k), W: p.w, H: p.h})
		}
	}
	// Drop pages that carry no content of their own. An empty page-break
	// element can start a page and then force the following content onto
	// yet another page, and a block whose border box crosses a page
	// boundary by a fraction of a point leaves stray border fragments on
	// the next page. Both would otherwise surface as phantom blank pages.
	// (A deliberately blank page cannot be expressed yet.)
	hasContent := func(pg *Page) bool {
		for _, it := range pg.Items {
			switch it.(type) {
			case TextItem, ImageItem:
				return true
			}
		}
		return false
	}
	kept := pages[:0]
	for _, pg := range pages {
		if hasContent(pg) {
			kept = append(kept, pg)
		}
	}
	if len(kept) == 0 {
		kept = pages[:1]
	}
	return kept
}
