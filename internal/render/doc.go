// Package render draws laid-out pages through the pdf package.
package render

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/datauri"
	"github.com/kujirahand/html2pdf-go/internal/filex"
	"github.com/kujirahand/html2pdf-go/internal/layout"
	"github.com/kujirahand/html2pdf-go/internal/pdf"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

// Image dimension limits; larger images are skipped. Keep in sync with
// internal/layout.
const (
	maxImageDim    = 8192
	maxImagePixels = 33000000
)

// maxImageFileSize caps the size of an image file.
const maxImageFileSize = 64 << 20

// maxTotalImagePixels bounds the decoded pixels of all images in one PDF:
// each image is retained until the PDF is written. A variable so tests can
// lower it.
var maxTotalImagePixels = 100_000_000

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

type renderer struct {
	w     *pdf.Writer
	fonts *text.FontSet
	// rootDir confines image files to the outermost document's directory
	// tree; see filex.ReadUnder.
	rootDir string
	// allowExternal lets image files lie outside rootDir.
	allowExternal bool
	// totalPixels is the pixel count of the images added so far.
	totalPixels int
	pdfFont     map[*text.Face]*pdf.Font
	images      map[string]*pdf.Image
	// failed caches image load errors by the same key as images, so a
	// document referencing one unusable image many times cannot make the
	// renderer read and decode it again for every occurrence.
	failed map[string]error
}

// Render writes pages to w as a PDF document. Image files are resolved
// against the directory each ImageItem names (the directory of the document
// that referenced them) and confined to rootDir, the outermost document's
// directory, unless allowExternal is set.
func Render(pages []*layout.Page, fonts *text.FontSet, rootDir string, allowExternal bool, w io.Writer) error {
	r := &renderer{
		w:             pdf.NewWriter(w),
		fonts:         fonts,
		rootDir:       rootDir,
		allowExternal: allowExternal,
		pdfFont:       map[*text.Face]*pdf.Font{},
		images:        map[string]*pdf.Image{},
	}
	for _, p := range pages {
		pg := r.w.AddPage(p.Width, p.Height)
		pg.SetScale(p.Scale)
		for _, it := range p.Items {
			if err := r.item(pg, it); err != nil {
				return err
			}
		}
	}
	return r.w.Close()
}

// font returns the PDF font for face, creating it on first use.
func (r *renderer) font(face *text.Face) *pdf.Font {
	f, ok := r.pdfFont[face]
	if !ok {
		f = r.w.NewFont(face)
		r.pdfFont[face] = f
	}
	return f
}

func (r *renderer) item(pg *pdf.Page, it layout.Item) error {
	switch it := it.(type) {
	case layout.TextItem:
		face := it.Face
		if face == nil {
			face = r.fonts.BaseFace(it.Bold, it.Italic)
		}
		pg.SetFillColor(it.Color.R, it.Color.G, it.Color.B)
		pg.ShowText(r.font(face), it.Size, it.X, it.Y, it.Text)
		if it.Underline {
			width := face.Measure(it.Text, it.Size)
			thickness := max(it.Size/18, 0.5)
			y := it.Y + it.Size*0.12
			pg.SetStrokeColor(it.Color.R, it.Color.G, it.Color.B)
			pg.DrawLine(it.X, y, it.X+width, y, thickness)
		}
	case layout.RectItem:
		if it.Fill != nil {
			pg.SetFillColor(it.Fill.R, it.Fill.G, it.Fill.B)
		}
		if it.Stroke != nil {
			pg.SetStrokeColor(it.Stroke.R, it.Stroke.G, it.Stroke.B)
		}
		rounded := it.RadiusX != [4]float64{} && it.RadiusY != [4]float64{}
		switch {
		case rounded && it.Fill != nil && it.Stroke != nil:
			pg.FillStrokeRoundedRect(it.X, it.Y, it.W, it.H, it.LineWidth, it.RadiusX, it.RadiusY)
		case rounded && it.Fill != nil:
			pg.FillRoundedRect(it.X, it.Y, it.W, it.H, it.RadiusX, it.RadiusY)
		case rounded && it.Stroke != nil:
			pg.StrokeRoundedRect(it.X, it.Y, it.W, it.H, it.LineWidth, it.RadiusX, it.RadiusY)
		case it.Fill != nil && it.Stroke != nil:
			pg.FillStrokeRect(it.X, it.Y, it.W, it.H, it.LineWidth)
		case it.Fill != nil:
			pg.FillRect(it.X, it.Y, it.W, it.H)
		case it.Stroke != nil:
			pg.StrokeRect(it.X, it.Y, it.W, it.H, it.LineWidth)
		}
	case layout.ClipItem:
		if it.End {
			pg.Restore()
		} else {
			pg.ClipRect(it.X, it.Y, it.W, it.H)
		}
	case layout.LineItem:
		pg.SetStrokeColor(it.Color.R, it.Color.G, it.Color.B)
		pg.DrawStyledLine(it.X1, it.Y1, it.X2, it.Y2, it.LineWidth, it.Style)
	case layout.ImageItem:
		// An unusable image is skipped, matching layout's handling of
		// unreadable images, rather than failing the whole render.
		if img, err := r.image(it.Src, it.Dir); err == nil {
			pg.DrawImage(img, it.X, it.Y, it.W, it.H)
		}
	}
	return nil
}

// image loads src (a file path resolved against dir, or a data: URI) and
// registers it with the PDF writer. Both successes and failures are
// cached: each distinct image is loaded at most once per document.
func (r *renderer) image(src, dir string) (*pdf.Image, error) {
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
	img, err := r.loadImage(src, dir, key)
	if err != nil {
		if r.failed == nil {
			r.failed = map[string]error{}
		}
		r.failed[key] = err
		return nil, err
	}
	return img, nil
}

func (r *renderer) loadImage(src, dir, key string) (*pdf.Image, error) {
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
		return nil, fmt.Errorf("render: read image: %w", err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("render: decode image %s: %w", label, err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImageDim || cfg.Height > maxImageDim ||
		cfg.Width*cfg.Height > maxImagePixels {
		return nil, fmt.Errorf("render: image %s too large (%dx%d)", label, cfg.Width, cfg.Height)
	}
	if r.totalPixels+cfg.Width*cfg.Height > maxTotalImagePixels {
		return nil, fmt.Errorf("render: image %s exceeds the total image budget of %d pixels", label, maxTotalImagePixels)
	}
	if !bytes.HasPrefix(data, pngSignature) {
		decoded, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("render: decode image %s: %w", label, err)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, decoded); err != nil {
			return nil, fmt.Errorf("render: re-encode image %s: %w", label, err)
		}
		data = buf.Bytes()
	}
	img, err := r.w.AddPNGImage(data)
	if err != nil {
		return nil, err
	}
	r.totalPixels += cfg.Width * cfg.Height
	r.images[key] = img
	return img, nil
}
