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
	"os"
	"path/filepath"

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

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

type renderer struct {
	w       *pdf.Writer
	fonts   *text.FontSet
	baseDir string
	pdfFont map[*text.Face]*pdf.Font
	images  map[string]*pdf.Image
}

// Render writes pages to w as a PDF document.
func Render(pages []*layout.Page, fonts *text.FontSet, baseDir string, w io.Writer) error {
	r := &renderer{
		w:       pdf.NewWriter(w),
		fonts:   fonts,
		baseDir: baseDir,
		pdfFont: map[*text.Face]*pdf.Font{},
		images:  map[string]*pdf.Image{},
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
			face = r.fonts.Regular
			if it.Bold {
				face = r.fonts.Bold
			}
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
		if img, err := r.image(it.Src); err == nil {
			pg.DrawImage(img, it.X, it.Y, it.W, it.H)
		}
	}
	return nil
}

// image loads src (a file path) and registers it with the PDF writer.
func (r *renderer) image(src string) (*pdf.Image, error) {
	if img, ok := r.images[src]; ok {
		return img, nil
	}
	data, err := os.ReadFile(src)
	if err != nil && !filepath.IsAbs(src) {
		data, err = os.ReadFile(filepath.Join(r.baseDir, src))
	}
	if err != nil {
		return nil, fmt.Errorf("render: read image: %w", err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("render: decode image %s: %w", src, err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxImageDim || cfg.Height > maxImageDim ||
		cfg.Width*cfg.Height > maxImagePixels {
		return nil, fmt.Errorf("render: image %s too large (%dx%d)", src, cfg.Width, cfg.Height)
	}
	if !bytes.HasPrefix(data, pngSignature) {
		decoded, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("render: decode image %s: %w", src, err)
		}
		var buf bytes.Buffer
		if err := png.Encode(&buf, decoded); err != nil {
			return nil, fmt.Errorf("render: re-encode image %s: %w", src, err)
		}
		data = buf.Bytes()
	}
	img, err := r.w.AddPNGImage(data)
	if err != nil {
		return nil, err
	}
	r.images[src] = img
	return img, nil
}
