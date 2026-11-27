package layout

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

const (
	// maxFrameDepth limits frameset nesting, which also stops a frame that
	// refers back to its own document.
	maxFrameDepth = 8
	// maxTracks limits the number of rows or columns of a frameset.
	maxTracks = 100
)

// layoutFrameset lays out a frameset document as one page. opt is already
// in virtual (unscaled) units; scale is the factor Layout divided it by.
func layoutFrameset(fs *dom.Node, fonts *text.FontSet, opt Options, scale float64) (*Page, error) {
	if opt.depth >= maxFrameDepth {
		return nil, fmt.Errorf("layout: frames nested more than %d levels deep", maxFrameDepth)
	}
	page := &Page{Width: opt.PageWidth, Height: opt.PageHeight}
	areaW := opt.PageWidth - opt.MarginLeft - opt.MarginRight
	areaH := opt.PageHeight - opt.MarginTop - opt.MarginBottom
	cols := frameTracks(fs.Attr["cols"], areaW)
	rows := frameTracks(fs.Attr["rows"], areaH)
	frames := collectFrames(fs, nil)

	y := opt.MarginTop
	for r, rh := range rows {
		x := opt.MarginLeft
		for c, cw := range cols {
			if i := r*len(cols) + c; i < len(frames) && cw > 0 && rh > 0 {
				items, err := layoutFrame(frames[i], fonts, opt, scale, cw, rh)
				if err != nil {
					return nil, err
				}
				if items != nil {
					page.Items = append(page.Items, ClipItem{X: x, Y: y, W: cw, H: rh})
					for _, it := range items {
						page.Items = append(page.Items, translate(it, x, y))
					}
					page.Items = append(page.Items, ClipItem{End: true})
				}
			}
			x += cw
		}
		y += rh
	}
	return page, nil
}

// layoutFrame lays out the document a frame refers to in a w x h cell (in
// virtual units) and returns the items of its first page. It returns nil for
// an empty cell: no src, or a remote src.
func layoutFrame(frame *dom.Node, fonts *text.FontSet, opt Options, scale, w, h float64) ([]Item, error) {
	src := strings.TrimSpace(frame.Attr["src"])
	if src == "" || strings.Contains(src, "://") {
		return nil, nil
	}
	p := src
	if !filepath.IsAbs(p) {
		p = filepath.Join(opt.BaseDir, p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("layout: read frame: %w", err)
	}
	return layoutFrameDoc(string(data), filepath.Dir(p), fonts, opt, scale, w, h)
}

// layoutFrameDoc lays out the HTML source of a frame document, whose
// relative references resolve against dir, in a w x h box and returns the
// items of its first page.
func layoutFrameDoc(src, dir string, fonts *text.FontSet, opt Options, scale, w, h float64) ([]Item, error) {
	doc, err := dom.Parse(strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	sheets, err := css.ResolveSheets(doc, dir)
	if err != nil {
		return nil, err
	}
	pages, err := Layout(doc, css.ComputeStyles(doc, sheets), fonts, Options{
		// Layout divides the page size by scale again, giving a virtual
		// page of exactly the cell size.
		PageWidth:  w * scale,
		PageHeight: h * scale,
		BaseDir:    dir,
		Scale:      scale,
		depth:      opt.depth + 1,
	})
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return []Item{}, nil
	}
	return pages[0].Items, nil
}

// iframeItems lays out the document an <iframe> shows in a w x h content
// box (in virtual units) and returns the items of its first page. srcdoc
// takes precedence over src. It returns nil for an empty box: no src, or a
// remote src. Frames share the nesting limit of framesets.
func (l *layouter) iframeItems(n *dom.Node, w, h float64) ([]Item, error) {
	if l.opt.depth >= maxFrameDepth {
		return nil, fmt.Errorf("layout: frames nested more than %d levels deep", maxFrameDepth)
	}
	scale := l.opt.Scale
	if scale <= 0 {
		scale = 1
	}
	if srcdoc, ok := n.Attr["srcdoc"]; ok {
		return layoutFrameDoc(srcdoc, l.opt.BaseDir, l.fonts, l.opt, scale, w, h)
	}
	return layoutFrame(n, l.fonts, l.opt, scale, w, h)
}

// collectFrames appends the <frame> elements under n in document order,
// skipping <noframes>.
func collectFrames(n *dom.Node, out []*dom.Node) []*dom.Node {
	for _, c := range n.Children {
		switch c.Tag {
		case "frame":
			out = append(out, c)
		case "noframes":
		default:
			out = collectFrames(c, out)
		}
	}
	return out
}

// frameTracks resolves a rows or cols attribute into track sizes that fill
// total. Each comma-separated token is pixels (scaled by 0.75), a percentage
// of total, or a relative length such as "*" or "2*" that shares what the
// other tokens leave. A missing attribute means a single "100%" track.
func frameTracks(attr string, total float64) []float64 {
	if strings.TrimSpace(attr) == "" {
		attr = "100%"
	}
	toks := strings.Split(attr, ",")
	if len(toks) > maxTracks {
		toks = toks[:maxTracks]
	}
	sizes := make([]float64, len(toks))
	weights := make([]float64, len(toks))
	var used, wsum float64
	for i, t := range toks {
		t = strings.ToLower(strings.TrimSpace(t))
		switch {
		case strings.HasSuffix(t, "*"):
			wt := 1.0
			if v, ok := parseNumber(strings.TrimSuffix(t, "*")); ok && v > 0 {
				wt = v
			}
			weights[i] = wt
			wsum += wt
		case strings.HasSuffix(t, "%"):
			v, _ := parseNumber(strings.TrimSuffix(t, "%"))
			sizes[i] = total * v / 100
		default:
			v, _ := parseNumber(strings.TrimSuffix(t, "px"))
			sizes[i] = v * 0.75
		}
		sizes[i] = min(max(sizes[i], 0), total)
		used += sizes[i]
	}
	if wsum > 0 {
		rest := max(total-used, 0)
		for i, wt := range weights {
			if wt > 0 {
				sizes[i] = rest * wt / wsum
			}
		}
	}
	return sizes
}

// parseNumber parses a finite decimal number.
func parseNumber(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// translate moves an item by (dx, dy).
func translate(it Item, dx, dy float64) Item {
	switch it := it.(type) {
	case TextItem:
		it.X += dx
		it.Y += dy
		return it
	case RectItem:
		it.X += dx
		it.Y += dy
		return it
	case LineItem:
		it.X1 += dx
		it.X2 += dx
		it.Y1 += dy
		it.Y2 += dy
		return it
	case ImageItem:
		it.X += dx
		it.Y += dy
		return it
	case ClipItem:
		it.X += dx
		it.Y += dy
		return it
	}
	return it
}
