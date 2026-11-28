// Package raster draws laid-out pages into bitmaps. It is the PNG
// counterpart of package render, which draws the same items through the pdf
// package.
package raster

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"path/filepath"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/datauri"
	"github.com/kujirahand/html2pdf-go/internal/filex"
	"github.com/kujirahand/html2pdf-go/internal/layout"
	"github.com/kujirahand/html2pdf-go/internal/text"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

// Image dimension limits; larger images are skipped. Keep in sync with
// internal/layout and internal/render.
const (
	maxImageDim    = 8192
	maxImagePixels = 33000000
)

// maxImageFileSize caps the size of an image file.
const maxImageFileSize = 64 << 20

// maxTotalImagePixels bounds the decoded pixels of all images in one
// conversion: each image is retained until the last page is drawn.
const maxTotalImagePixels = 100_000_000

// Bitmap limits: a page may be at most maxPageSide pixels on a side and
// maxPagePixels in area, and all pages together maxTotalPixels, so an
// oversized scale fails instead of exhausting memory.
const (
	maxPageSide    = 32768
	maxPagePixels  = 50_000_000
	maxTotalPixels = 200_000_000
)

// maxDashes bounds the dashes or dots of one line; a longer pattern is
// drawn solid.
const maxDashes = 50000

// defaultScale is used when the scale argument is not positive.
const defaultScale = 2

type point struct{ x, y float64 }

type faceKey struct {
	face *text.Face
	size float64
}

// clipLayer remembers the target below a clip and the clip rectangle in
// pixels.
type clipLayer struct {
	below *image.NRGBA
	rect  image.Rectangle
}

type renderer struct {
	fonts *text.FontSet
	// rootDir confines image files to the outermost document's directory
	// tree; see filex.ReadUnder.
	rootDir string
	// allowExternal lets image files lie outside rootDir.
	allowExternal bool
	// scale is the number of pixels per point.
	scale float64
	// pageScale is the layout page's own scale; k = scale*pageScale
	// converts item coordinates to pixels.
	pageScale float64
	k         float64

	// dst is the current drawing target: the page image, or the topmost
	// clip layer.
	dst   *image.NRGBA
	stack []clipLayer

	ras   vector.Rasterizer
	faces map[faceKey]font.Face

	totalPixels int
	images      map[string]image.Image
	// failed caches image load errors by the same key as images.
	failed map[string]error
}

// Render draws pages into bitmaps, one per page, sized Width x Height points
// times scale pixels (scale <= 0 means 2, i.e. 144 dpi). Image files are
// resolved against the directory each ImageItem names and confined to
// rootDir unless allowExternal is set, like package render.
func Render(pages []*layout.Page, fonts *text.FontSet, rootDir string, allowExternal bool, scale float64) ([]*image.NRGBA, error) {
	if scale <= 0 {
		scale = defaultScale
	}
	if math.IsNaN(scale) || math.IsInf(scale, 0) {
		return nil, fmt.Errorf("raster: invalid scale %v", scale)
	}
	r := &renderer{
		fonts:         fonts,
		rootDir:       rootDir,
		allowExternal: allowExternal,
		scale:         scale,
		faces:         map[faceKey]font.Face{},
		images:        map[string]image.Image{},
		failed:        map[string]error{},
	}
	defer func() {
		for _, f := range r.faces {
			if f != nil {
				f.Close()
			}
		}
	}()
	out := make([]*image.NRGBA, 0, len(pages))
	total := 0
	for _, p := range pages {
		w, h := math.Round(p.Width*scale), math.Round(p.Height*scale)
		if !(w >= 1 && h >= 1) || w > maxPageSide || h > maxPageSide || w*h > maxPagePixels {
			return nil, fmt.Errorf("raster: page of %vx%v points at scale %v is out of range", p.Width, p.Height, scale)
		}
		total += int(w) * int(h)
		if total > maxTotalPixels {
			return nil, errors.New("raster: the pages exceed the total bitmap budget")
		}
		page := image.NewNRGBA(image.Rect(0, 0, int(w), int(h)))
		draw.Draw(page, page.Bounds(), image.White, image.Point{}, draw.Src)

		r.pageScale = p.Scale
		if r.pageScale <= 0 {
			r.pageScale = 1
		}
		r.k = scale * r.pageScale
		r.dst = page
		r.stack = r.stack[:0]
		for _, it := range p.Items {
			r.item(it)
		}
		// Close clips the page left open.
		for len(r.stack) > 0 {
			r.popClip()
		}
		out = append(out, page)
	}
	return out, nil
}

func (r *renderer) item(it layout.Item) {
	switch it := it.(type) {
	case layout.TextItem:
		r.text(it)
	case layout.RectItem:
		r.rect(it)
	case layout.LineItem:
		r.line(it)
	case layout.ImageItem:
		r.image(it)
	case layout.ClipItem:
		if it.End {
			r.popClip()
		} else {
			r.pushClip(it)
		}
	}
}

// toColor converts a CSS color (components 0..1) to an opaque pixel color.
func toColor(c css.Color) color.NRGBA {
	u := func(v float64) uint8 {
		if math.IsNaN(v) {
			return 0
		}
		return uint8(math.Round(min(max(v, 0), 1) * 255))
	}
	return color.NRGBA{R: u(c.R), G: u(c.G), B: u(c.B), A: 255}
}

func (r *renderer) pushClip(it layout.ClipItem) {
	x0, y0 := sane(it.X*r.k), sane(it.Y*r.k)
	x1, y1 := sane((it.X+it.W)*r.k), sane((it.Y+it.H)*r.k)
	rect := image.Rect(int(math.Floor(x0)), int(math.Floor(y0)), int(math.Ceil(x1)), int(math.Ceil(y1))).
		Intersect(r.dst.Bounds())
	r.stack = append(r.stack, clipLayer{below: r.dst, rect: rect})
	r.dst = image.NewNRGBA(r.dst.Bounds())
}

func (r *renderer) popClip() {
	if len(r.stack) == 0 {
		return
	}
	top := r.stack[len(r.stack)-1]
	r.stack = r.stack[:len(r.stack)-1]
	if !top.rect.Empty() {
		draw.Draw(top.below, top.rect, r.dst, top.rect.Min, draw.Over)
	}
	r.dst = top.below
}

// rasterFace returns the cached font.Face for face at size points (before
// the render scale), or nil if it cannot be created.
func (r *renderer) rasterFace(face *text.Face, size float64) font.Face {
	key := faceKey{face, size}
	f, ok := r.faces[key]
	if !ok {
		var err error
		if f, err = face.RasterFace(size, r.scale); err != nil {
			f = nil
		}
		r.faces[key] = f
	}
	return f
}

func (r *renderer) text(it layout.TextItem) {
	face := it.Face
	if face == nil && r.fonts != nil {
		face = r.fonts.Regular
		if it.Bold {
			face = r.fonts.Bold
		}
	}
	if face == nil || !(it.Size > 0) {
		return
	}
	col := toColor(it.Color)
	if rf := r.rasterFace(face, it.Size*r.pageScale); rf != nil {
		// Draw rune by rune and advance by the layout's own advances, so
		// the glyphs sit exactly where layout measured them.
		d := font.Drawer{Dst: r.dst, Src: image.NewUniform(col), Face: rf}
		x, y := sane(it.X*r.k), sane(it.Y*r.k)
		for _, ch := range it.Text {
			if ch < 0x20 {
				continue
			}
			d.Dot = fixed.Point26_6{X: fixed.Int26_6(math.Round(x * 64)), Y: fixed.Int26_6(math.Round(y * 64))}
			d.DrawString(string(ch))
			x += face.Advance(ch, it.Size) * r.k
		}
	}
	if it.Underline {
		width := face.Measure(it.Text, it.Size)
		thickness := max(it.Size/18, 0.5)
		y := it.Y + it.Size*0.12
		r.line(layout.LineItem{X1: it.X, Y1: y, X2: it.X + width, Y2: y, LineWidth: thickness, Color: it.Color})
	}
}

// sane maps NaN to 0 and clamps v so it converts to int safely.
func sane(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return min(max(v, -1e7), 1e7)
}

// fill fills the union of the polygons (pixel coordinates) in col. Polygons
// wound opposite to the others cut holes.
func (r *renderer) fill(col color.NRGBA, polys ...[]point) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, poly := range polys {
		for i := range poly {
			p := point{sane(poly[i].x), sane(poly[i].y)}
			poly[i] = p
			minX, maxX = min(minX, p.x), max(maxX, p.x)
			minY, maxY = min(minY, p.y), max(maxY, p.y)
		}
	}
	box := image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX)), int(math.Ceil(maxY))).
		Intersect(r.dst.Bounds())
	if box.Empty() {
		return
	}
	r.ras.Reset(box.Dx(), box.Dy())
	for _, poly := range polys {
		for i, p := range poly {
			x, y := float32(p.x-float64(box.Min.X)), float32(p.y-float64(box.Min.Y))
			if i == 0 {
				r.ras.MoveTo(x, y)
			} else {
				r.ras.LineTo(x, y)
			}
		}
		if len(poly) > 0 {
			r.ras.ClosePath()
		}
	}
	r.ras.Draw(r.dst, box, image.NewUniform(col), image.Point{})
}

// roundedPoly returns the outline of a rectangle with elliptical corners as
// a clockwise polygon (y down). Radii are ordered top-left, top-right,
// bottom-right, bottom-left; a corner whose radius is zero in either
// direction is square.
func roundedPoly(x, y, w, h float64, rx, ry [4]float64) []point {
	r, b := x+w, y+h
	centers := [4]point{{x + rx[0], y + ry[0]}, {r - rx[1], y + ry[1]}, {r - rx[2], b - ry[2]}, {x + rx[3], b - ry[3]}}
	corners := [4]point{{x, y}, {r, y}, {r, b}, {x, b}}
	var poly []point
	for i := 0; i < 4; i++ {
		if !(rx[i] > 0 && ry[i] > 0) {
			poly = append(poly, corners[i])
			continue
		}
		steps := min(4+int(math.Ceil(0.8*max(rx[i], ry[i]))), 64)
		for s := 0; s <= steps; s++ {
			a := (float64(i+2) + float64(s)/float64(steps)) * math.Pi / 2
			poly = append(poly, point{centers[i].x + rx[i]*math.Cos(a), centers[i].y + ry[i]*math.Sin(a)})
		}
	}
	return poly
}

func reversed(poly []point) []point {
	out := make([]point, len(poly))
	for i, p := range poly {
		out[len(poly)-1-i] = p
	}
	return out
}

func (r *renderer) rect(it layout.RectItem) {
	x, y, w, h := sane(it.X*r.k), sane(it.Y*r.k), sane(it.W*r.k), sane(it.H*r.k)
	var rx, ry [4]float64
	if it.RadiusX != [4]float64{} && it.RadiusY != [4]float64{} {
		for i := range rx {
			rx[i] = min(max(sane(it.RadiusX[i]*r.k), 0), w/2)
			ry[i] = min(max(sane(it.RadiusY[i]*r.k), 0), h/2)
			if !(rx[i] > 0 && ry[i] > 0) {
				rx[i], ry[i] = 0, 0
			}
		}
	}
	if it.Fill != nil && w > 0 && h > 0 {
		r.fill(toColor(*it.Fill), roundedPoly(x, y, w, h, rx, ry))
	}
	if it.Stroke == nil {
		return
	}
	lw := sane(it.LineWidth * r.k)
	if lw <= 0 {
		lw = 1 // a hairline is one device pixel wide
	}
	// The stroke is centered on the outline: a ring between the outline
	// grown by half the width and the outline shrunk by it.
	d := lw / 2
	var orx, ory, irx, iry [4]float64
	for i := range rx {
		if rx[i] > 0 {
			orx[i], ory[i] = rx[i]+d, ry[i]+d
		}
		if rx[i] > d && ry[i] > d {
			irx[i], iry[i] = rx[i]-d, ry[i]-d
		}
	}
	outer := roundedPoly(x-d, y-d, w+lw, h+lw, orx, ory)
	if w-lw > 0 && h-lw > 0 {
		inner := reversed(roundedPoly(x+d, y+d, w-lw, h-lw, irx, iry))
		r.fill(toColor(*it.Stroke), outer, inner)
	} else {
		r.fill(toColor(*it.Stroke), outer)
	}
}

// line strokes a straight line like pdf.Page.DrawStyledLine: dashes are 3
// line widths long with 2 line width gaps, dots are round and spaced 2 line
// widths apart.
func (r *renderer) line(it layout.LineItem) {
	x1, y1, x2, y2 := sane(it.X1*r.k), sane(it.Y1*r.k), sane(it.X2*r.k), sane(it.Y2*r.k)
	lw := sane(it.LineWidth * r.k)
	if lw <= 0 {
		lw = 1
	}
	length := math.Hypot(x2-x1, y2-y1)
	if length == 0 {
		return
	}
	ux, uy := (x2-x1)/length, (y2-y1)/length
	hw := lw / 2
	// span returns the butt-capped quad covering [a, b] along the line.
	span := func(a, b float64) []point {
		ax, ay := x1+ux*a, y1+uy*a
		bx, by := x1+ux*b, y1+uy*b
		return []point{
			{ax - uy*hw, ay + ux*hw}, {bx - uy*hw, by + ux*hw},
			{bx + uy*hw, by - ux*hw}, {ax + uy*hw, ay - ux*hw},
		}
	}
	var polys [][]point
	switch {
	case it.Style == "dashed" && length/(5*lw) < maxDashes:
		for t := 0.0; t < length; t += 5 * lw {
			polys = append(polys, span(t, min(t+3*lw, length)))
		}
	case it.Style == "dotted" && length/(2*lw) < maxDashes:
		var round [4]float64
		for i := range round {
			round[i] = hw
		}
		for t := 0.0; t <= length; t += 2 * lw {
			polys = append(polys, roundedPoly(x1+ux*t-hw, y1+uy*t-hw, lw, lw, round, round))
		}
	default:
		polys = [][]point{span(0, length)}
	}
	r.fill(toColor(it.Color), polys...)
}

func (r *renderer) image(it layout.ImageItem) {
	// An unusable image is skipped, matching layout and render, rather
	// than failing the whole conversion.
	img, err := r.loadImage(it.Src, it.Dir)
	if err != nil {
		return
	}
	x0, y0 := sane(it.X*r.k), sane(it.Y*r.k)
	x1, y1 := sane((it.X+it.W)*r.k), sane((it.Y+it.H)*r.k)
	dr := image.Rect(int(math.Round(x0)), int(math.Round(y0)), int(math.Round(x1)), int(math.Round(y1)))
	if dr.Empty() {
		return
	}
	draw.ApproxBiLinear.Scale(r.dst, dr, img, img.Bounds(), draw.Over, nil)
}

// loadImage decodes src (a file path resolved against dir, or a data: URI).
// Both successes and failures are cached: each distinct image is loaded at
// most once per conversion.
func (r *renderer) loadImage(src, dir string) (image.Image, error) {
	// A data URI is the same image wherever it appears; a file depends on
	// the directory.
	key := src
	if len(src) < 5 || !strings.EqualFold(src[:5], "data:") {
		key = dir + "\x00" + src
	}
	if img, ok := r.images[key]; ok {
		return img, nil
	}
	if err, ok := r.failed[key]; ok {
		return nil, err
	}
	img, err := r.decodeImage(src, dir)
	if err != nil {
		r.failed[key] = err
		return nil, err
	}
	r.images[key] = img
	return img, nil
}

func (r *renderer) decodeImage(src, dir string) (image.Image, error) {
	data, isData, err := datauri.Decode(src)
	label := src
	if isData {
		label = "data URI"
	} else {
		p := src
		if !filepath.IsAbs(p) {
			p = filepath.Join(dir, p)
		}
		data, err = filex.ReadUnder(r.rootDir, p, maxImageFileSize, r.allowExternal)
	}
	if err != nil {
		return nil, fmt.Errorf("raster: read image: %w", err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("raster: decode image %s: %w", label, err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImageDim || cfg.Height > maxImageDim ||
		cfg.Width*cfg.Height > maxImagePixels {
		return nil, fmt.Errorf("raster: image %s too large (%dx%d)", label, cfg.Width, cfg.Height)
	}
	if r.totalPixels+cfg.Width*cfg.Height > maxTotalImagePixels {
		return nil, fmt.Errorf("raster: image %s exceeds the total image budget of %d pixels", label, maxTotalImagePixels)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("raster: decode image %s: %w", label, err)
	}
	r.totalPixels += cfg.Width * cfg.Height
	return img, nil
}
