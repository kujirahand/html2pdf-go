package pdf

import (
	"bytes"
	"fmt"
	"strings"
)

// Page is a single page; drawing methods append to its content stream.
type Page struct {
	w             *Writer
	width, height float64
	scale         float64 // multiplier for drawing coordinates; <=0 means 1
	buf           bytes.Buffer
	chunks        []*fontChunk // font chunks used, in first-use order
	chunkSet      map[*fontChunk]bool
	images        []*Image
	imageSet      map[*Image]bool
	objID         int
	contentID     int
	annots        []annotation
}

// AddPage appends a page of the given size in points.
func (w *Writer) AddPage(width, height float64) *Page {
	p := &Page{
		w:        w,
		width:    width,
		height:   height,
		chunkSet: make(map[*fontChunk]bool),
		imageSet: make(map[*Image]bool),
	}
	w.pages = append(w.pages, p)
	return p
}

// SetScale makes the drawing methods multiply coordinates, sizes, line widths,
// and font sizes by s. Values <= 0 or 1 leave them unchanged.
func (p *Page) SetScale(s float64) { p.scale = s }

// sc applies the page scale to v.
func (p *Page) sc(v float64) float64 {
	if p.scale <= 0 || p.scale == 1 {
		return v
	}
	return v * p.scale
}

// fy converts a top-left-origin y to PDF's bottom-left-origin y.
func (p *Page) fy(y float64) float64 { return p.height - y }

// SetFillColor sets the fill color; components are 0..1.
func (p *Page) SetFillColor(r, g, b float64) {
	fmt.Fprintf(&p.buf, "%s %s %s rg\n", num(r), num(g), num(b))
}

// SetStrokeColor sets the stroke color; components are 0..1.
func (p *Page) SetStrokeColor(r, g, b float64) {
	fmt.Fprintf(&p.buf, "%s %s %s RG\n", num(r), num(g), num(b))
}

func (p *Page) rect(x, y, w, h float64, op string) {
	x, y, w, h = p.sc(x), p.sc(y), p.sc(w), p.sc(h)
	fmt.Fprintf(&p.buf, "%s %s %s %s re %s\n", num(x), num(p.fy(y+h)), num(w), num(h), op)
}

// FillRect fills a rectangle whose top-left corner is (x, y).
func (p *Page) FillRect(x, y, w, h float64) { p.rect(x, y, w, h, "f") }

// StrokeRect strokes a rectangle outline.
func (p *Page) StrokeRect(x, y, w, h, lineWidth float64) {
	fmt.Fprintf(&p.buf, "%s w\n", num(p.sc(lineWidth)))
	p.rect(x, y, w, h, "S")
}

// FillStrokeRect fills and strokes a rectangle.
func (p *Page) FillStrokeRect(x, y, w, h, lineWidth float64) {
	fmt.Fprintf(&p.buf, "%s w\n", num(p.sc(lineWidth)))
	p.rect(x, y, w, h, "B")
}

// ClipRect saves the graphics state and restricts drawing to a rectangle whose
// top-left corner is (x, y). Call Restore to end the clip.
func (p *Page) ClipRect(x, y, w, h float64) {
	x, y, w, h = p.sc(x), p.sc(y), p.sc(w), p.sc(h)
	fmt.Fprintf(&p.buf, "q\n%s %s %s %s re W n\n", num(x), num(p.fy(y+h)), num(w), num(h))
}

// Restore restores the graphics state saved by ClipRect.
func (p *Page) Restore() { p.buf.WriteString("Q\n") }

// kappa is the cubic Bezier control-point factor that approximates a
// quarter circle.
const kappa = 0.5522847498

// roundedRect appends a closed rounded-rectangle path followed by op. Radii
// are ordered top-left, top-right, bottom-right, bottom-left and are clamped
// to half the width and height; a zero radius gives a square corner.
func (p *Page) roundedRect(x, y, w, h float64, rx, ry [4]float64, op string) {
	x, y, w, h = p.sc(x), p.sc(y), p.sc(w), p.sc(h)
	for i := 0; i < 4; i++ {
		rx[i] = min(max(p.sc(rx[i]), 0), w/2)
		ry[i] = min(max(p.sc(ry[i]), 0), h/2)
	}
	pt := func(px, py float64) string { return num(px) + " " + num(p.fy(py)) }
	curve := func(x1, y1, x2, y2, x3, y3 float64) {
		fmt.Fprintf(&p.buf, "%s %s %s c\n", pt(x1, y1), pt(x2, y2), pt(x3, y3))
	}
	r, b := x+w, y+h
	// Start after the top-left corner and walk clockwise (in top-left-origin
	// coordinates).
	fmt.Fprintf(&p.buf, "%s m\n", pt(x+rx[0], y))
	fmt.Fprintf(&p.buf, "%s l\n", pt(r-rx[1], y))
	if rx[1] > 0 && ry[1] > 0 {
		curve(r-rx[1]+rx[1]*kappa, y, r, y+ry[1]-ry[1]*kappa, r, y+ry[1])
	}
	fmt.Fprintf(&p.buf, "%s l\n", pt(r, b-ry[2]))
	if rx[2] > 0 && ry[2] > 0 {
		curve(r, b-ry[2]+ry[2]*kappa, r-rx[2]+rx[2]*kappa, b, r-rx[2], b)
	}
	fmt.Fprintf(&p.buf, "%s l\n", pt(x+rx[3], b))
	if rx[3] > 0 && ry[3] > 0 {
		curve(x+rx[3]-rx[3]*kappa, b, x, b-ry[3]+ry[3]*kappa, x, b-ry[3])
	}
	fmt.Fprintf(&p.buf, "%s l\n", pt(x, y+ry[0]))
	if rx[0] > 0 && ry[0] > 0 {
		curve(x, y+ry[0]-ry[0]*kappa, x+rx[0]-rx[0]*kappa, y, x+rx[0], y)
	}
	fmt.Fprintf(&p.buf, "h %s\n", op)
}

// FillRoundedRect fills a rectangle with elliptical corners; rx and ry are
// the corner radii (top-left, top-right, bottom-right, bottom-left).
func (p *Page) FillRoundedRect(x, y, w, h float64, rx, ry [4]float64) {
	p.roundedRect(x, y, w, h, rx, ry, "f")
}

// StrokeRoundedRect strokes a rounded rectangle outline.
func (p *Page) StrokeRoundedRect(x, y, w, h, lineWidth float64, rx, ry [4]float64) {
	fmt.Fprintf(&p.buf, "%s w\n", num(p.sc(lineWidth)))
	p.roundedRect(x, y, w, h, rx, ry, "S")
}

// FillStrokeRoundedRect fills and strokes a rounded rectangle.
func (p *Page) FillStrokeRoundedRect(x, y, w, h, lineWidth float64, rx, ry [4]float64) {
	fmt.Fprintf(&p.buf, "%s w\n", num(p.sc(lineWidth)))
	p.roundedRect(x, y, w, h, rx, ry, "B")
}

// DrawLine strokes a line segment.
func (p *Page) DrawLine(x1, y1, x2, y2, lineWidth float64) {
	x1, y1, x2, y2, lineWidth = p.sc(x1), p.sc(y1), p.sc(x2), p.sc(y2), p.sc(lineWidth)
	fmt.Fprintf(&p.buf, "%s w\n%s %s m %s %s l S\n",
		num(lineWidth), num(x1), num(p.fy(y1)), num(x2), num(p.fy(y2)))
}

// DrawStyledLine strokes a line segment in the given style: "dashed" (dashes
// of 3 line widths with 2 line width gaps), "dotted" (round dots spaced 2 line
// widths apart), or anything else for a solid line. The dash and cap settings
// are scoped to the line with q/Q.
func (p *Page) DrawStyledLine(x1, y1, x2, y2, lineWidth float64, style string) {
	lw := p.sc(lineWidth)
	var setup string
	switch style {
	case "dashed":
		setup = fmt.Sprintf("[%s %s] 0 d\n", num(3*lw), num(2*lw))
	case "dotted":
		setup = fmt.Sprintf("1 J\n[0 %s] 0 d\n", num(2*lw))
	default:
		p.DrawLine(x1, y1, x2, y2, lineWidth)
		return
	}
	p.buf.WriteString("q\n" + setup)
	p.DrawLine(x1, y1, x2, y2, lineWidth)
	p.buf.WriteString("Q\n")
}

// ShowText draws s with its baseline origin at (x, y) in the current fill color.
func (p *Page) ShowText(f *Font, size, x, y float64, s string) {
	size, x, y = p.sc(size), p.sc(x), p.sc(y)
	type run struct {
		chunk *fontChunk
		codes []byte
	}
	var runs []run
	for _, r := range s {
		if r < 0x20 {
			continue
		}
		c, code := f.code(r)
		if len(runs) == 0 || runs[len(runs)-1].chunk != c {
			runs = append(runs, run{chunk: c})
		}
		last := &runs[len(runs)-1]
		last.codes = append(last.codes, code)
	}
	if len(runs) == 0 {
		return
	}
	fmt.Fprintf(&p.buf, "BT\n1 0 0 1 %s %s Tm\n", num(x), num(p.fy(y)))
	for _, r := range runs {
		if !p.chunkSet[r.chunk] {
			p.chunkSet[r.chunk] = true
			p.chunks = append(p.chunks, r.chunk)
		}
		// The text position advances after each Tj, so chunk switches stay aligned.
		fmt.Fprintf(&p.buf, "/%s %s Tf\n<%X> Tj\n", r.chunk.name, num(size), r.codes)
	}
	p.buf.WriteString("ET\n")
}

// DrawImage draws img into the rectangle with top-left corner (x, y).
func (p *Page) DrawImage(img *Image, x, y, w, h float64) {
	if !p.imageSet[img] {
		p.imageSet[img] = true
		p.images = append(p.images, img)
	}
	x, y, w, h = p.sc(x), p.sc(y), p.sc(w), p.sc(h)
	fmt.Fprintf(&p.buf, "q %s 0 0 %s %s %s cm /%s Do Q\n",
		num(w), num(h), num(x), num(p.fy(y+h)), img.name())
}

func (w *Writer) writePage(p *Page) {
	w.stream(p.contentID, "", p.buf.Bytes(), true)
	var res strings.Builder
	res.WriteString("<<")
	if len(p.chunks) > 0 {
		res.WriteString(" /Font <<")
		for _, c := range p.chunks {
			fmt.Fprintf(&res, " /%s %d 0 R", c.name, c.objID)
		}
		res.WriteString(" >>")
	}
	if len(p.images) > 0 {
		res.WriteString(" /XObject <<")
		for _, im := range p.images {
			fmt.Fprintf(&res, " /%s %d 0 R", im.name(), im.objID)
		}
		res.WriteString(" >>")
	}
	res.WriteString(" >>")
	extra := ""
	if len(p.annots) > 0 {
		var b strings.Builder
		b.WriteString(" /Annots [")
		for _, a := range p.annots {
			fmt.Fprintf(&b, " %d 0 R", a.id)
		}
		b.WriteString(" ]")
		extra = b.String()
	}
	w.obj(p.objID, fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %s %s] /Resources %s /Contents %d 0 R%s >>",
		w.pagesID, num(p.width), num(p.height), res.String(), p.contentID, extra))
}
