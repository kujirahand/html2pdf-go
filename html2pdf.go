// Package html2pdf converts HTML+CSS documents into PDF files. It is a pure-Go HTML-to-PDF rendering engine under development.
package html2pdf

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/layout"
	"github.com/kujirahand/html2pdf-go/internal/render"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

// Options configures a conversion.
type Options struct {
	// PageSize is a named page size such as "A4", "A5", or "Letter".
	PageSize string
	// PageWidth and PageHeight are in points; 0 means derive from PageSize.
	PageWidth  float64
	PageHeight float64
	// Landscape swaps the page width and height.
	Landscape bool
	// BaseDir is used to resolve relative paths such as images.
	BaseDir string
	// FontDirs lists extra directories to search for fonts.
	FontDirs []string
	// FontFile is the path of a font file used as the regular face instead
	// of the automatically selected one. Convert fails if it cannot be read.
	FontFile string
	// FontFileBold is the path of the bold face. If FontFile is set without
	// FontFileBold, bold text uses the regular face.
	FontFileBold string
	// Scale shrinks (<1) or enlarges (>1) the content, like wkhtmltopdf's
	// zoom or Chrome's printToPDF scale. 0 or less means 1.0.
	Scale float64
	// Margin is the page margin in points, applied to all four sides.
	// 0 means no margin.
	Margin float64
	// MarkdownCSS is extra CSS appended after DefaultMarkdownCSS. It is used
	// only by ConvertMarkdown and ConvertMarkdownFile; Convert ignores it.
	MarkdownCSS string
}

// pageSizes maps upper-case names to (width, height) in points.
var pageSizes = map[string][2]float64{
	"A4":     {595.28, 841.89},
	"A5":     {419.53, 595.28},
	"LETTER": {612, 792},
}

func (o *Options) pageSize() (w, h float64, err error) {
	w, h = o.PageWidth, o.PageHeight
	if w > 0 && h > 0 {
		return w, h, nil
	}
	name := strings.ToUpper(strings.TrimSpace(o.PageSize))
	if name == "" {
		name = "A4"
	}
	size, ok := pageSizes[name]
	if !ok {
		return 0, 0, fmt.Errorf("html2pdf: unknown page size %q", o.PageSize)
	}
	if w <= 0 {
		w = size[0]
	}
	if h <= 0 {
		h = size[1]
	}
	if o.Landscape {
		w, h = h, w
	}
	return w, h, nil
}

// Convert renders the HTML document src as a PDF and writes it to w.
func Convert(src []byte, w io.Writer, opts *Options) error {
	if opts == nil {
		opts = &Options{}
	}
	if math.IsNaN(opts.Scale) || math.IsInf(opts.Scale, 0) {
		return fmt.Errorf("html2pdf: invalid scale %v", opts.Scale)
	}
	pageW, pageH, err := opts.pageSize()
	if err != nil {
		return err
	}
	fonts, err := text.LoadFontSet(opts.FontDirs, opts.FontFile, opts.FontFileBold)
	if err != nil {
		return err
	}
	doc, err := dom.Parse(bytes.NewReader(src))
	if err != nil {
		return err
	}
	styles := css.ComputeStyles(doc, doc.StylesheetText)
	pages, err := layout.Layout(doc, styles, fonts, layout.Options{
		PageWidth:  pageW,
		PageHeight: pageH,
		BaseDir:    opts.BaseDir,
		Scale:      opts.Scale,

		MarginTop:    opts.Margin,
		MarginRight:  opts.Margin,
		MarginBottom: opts.Margin,
		MarginLeft:   opts.Margin,
	})
	if err != nil {
		return err
	}
	return render.Render(pages, fonts, opts.BaseDir, w)
}

// ConvertFile reads the HTML file at htmlPath and writes the PDF to pdfPath.
func ConvertFile(htmlPath, pdfPath string, opts *Options) error {
	src, err := os.ReadFile(htmlPath)
	if err != nil {
		return err
	}
	// Render into memory first so a failed conversion leaves no partial file.
	var buf bytes.Buffer
	if err := Convert(src, &buf, opts); err != nil {
		return err
	}
	return os.WriteFile(pdfPath, buf.Bytes(), 0o644)
}
