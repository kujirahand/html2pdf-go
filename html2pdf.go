// Package html2pdf converts HTML+CSS documents into PDF files. It is a pure-Go HTML-to-PDF rendering engine under development.
package html2pdf

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/layout"
	"github.com/kujirahand/html2pdf-go/internal/raster"
	"github.com/kujirahand/html2pdf-go/internal/render"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

// FontUse describes a font face resolved during conversion.
type FontUse struct {
	// Family is the font-family name that matched, normalized for matching
	// (lower-cased), or "" for the default face used when no listed family
	// was found.
	Family string
	// Bold reports whether bold was requested. If there is no bold face,
	// Path is the regular file and Bold is still true.
	Bold bool
	// Path is the font file the face was loaded from.
	Path string
}

// Options configures a conversion.
type Options struct {
	// PageSize is a named page size such as "A4", "A5", or "Letter".
	PageSize string
	// PageWidth and PageHeight are in points; 0 means derive from PageSize.
	PageWidth  float64
	PageHeight float64
	// Landscape swaps the page width and height.
	Landscape bool
	// BaseDir is used to resolve relative paths such as images, and
	// confines the files a document can read to that directory's tree.
	// If it is empty, Convert lets documents read no local files at all;
	// ConvertFile and ConvertMarkdownFile default it to the converted
	// file's directory.
	BaseDir string
	// AllowExternalFiles permits documents to reference files outside
	// BaseDir, by absolute path or with "..". By default (false), images,
	// stylesheets and frame documents can only be read from inside BaseDir.
	// Only enable it for documents you trust.
	AllowExternalFiles bool
	// FontDirs lists the directories to search for fonts. Only these are
	// searched; if empty, the operating system's standard font directories
	// are used.
	FontDirs []string
	// OnFontUse, if non-nil, is called once for each distinct font face
	// that is resolved during conversion.
	OnFontUse func(FontUse)
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
	if opts.OnFontUse != nil {
		fonts.OnUse = func(family string, bold bool, path string) {
			opts.OnFontUse(FontUse{Family: family, Bold: bold, Path: path})
		}
	}
	doc, err := dom.Parse(bytes.NewReader(src))
	if err != nil {
		return err
	}
	sheets, err := css.ResolveSheets(doc, opts.BaseDir, opts.BaseDir, opts.AllowExternalFiles)
	if err != nil {
		return err
	}
	styles := css.ComputeStyles(doc, sheets)
	pages, err := layout.Layout(doc, styles, fonts, layout.Options{
		PageWidth:  pageW,
		PageHeight: pageH,
		BaseDir:    opts.BaseDir,
		RootDir:    opts.BaseDir,
		Scale:      opts.Scale,

		AllowExternalFiles: opts.AllowExternalFiles,

		MarginTop:    opts.Margin,
		MarginRight:  opts.Margin,
		MarginBottom: opts.Margin,
		MarginLeft:   opts.Margin,
	})
	if err != nil {
		return err
	}
	return render.Render(pages, fonts, opts.BaseDir, opts.AllowExternalFiles, w)
}

// ConvertFile reads the HTML file at htmlPath and writes the PDF to pdfPath.
func ConvertFile(htmlPath, pdfPath string, opts *Options) error {
	src, err := os.ReadFile(htmlPath)
	if err != nil {
		return err
	}
	// Work on a copy so the caller's Options is never modified.
	var o Options
	if opts != nil {
		o = *opts
	}
	if o.BaseDir == "" {
		o.BaseDir = filepath.Dir(htmlPath)
	}
	// Render into memory first so a failed conversion leaves no partial file.
	var buf bytes.Buffer
	if err := Convert(src, &buf, &o); err != nil {
		return err
	}
	return os.WriteFile(pdfPath, buf.Bytes(), 0o644)
}

// ConvertPNG converts the HTML document src into one PNG image per page,
// each returned as PNG-encoded bytes. scale is the rasterization scale
// relative to 72-dpi points (2 = 144 dpi); a value <= 0 means 2. File
// access follows the same rules as Convert (see Options.BaseDir and
// Options.AllowExternalFiles).
func ConvertPNG(src []byte, opts *Options, scale float64) ([][]byte, error) {
	if opts == nil {
		opts = &Options{}
	}
	if math.IsNaN(opts.Scale) || math.IsInf(opts.Scale, 0) {
		return nil, fmt.Errorf("html2pdf: invalid scale %v", opts.Scale)
	}
	if math.IsNaN(scale) || math.IsInf(scale, 0) {
		return nil, fmt.Errorf("html2pdf: invalid PNG scale %v", scale)
	}
	pageW, pageH, err := opts.pageSize()
	if err != nil {
		return nil, err
	}
	fonts, err := text.LoadFontSet(opts.FontDirs, opts.FontFile, opts.FontFileBold)
	if err != nil {
		return nil, err
	}
	if opts.OnFontUse != nil {
		fonts.OnUse = func(family string, bold bool, path string) {
			opts.OnFontUse(FontUse{Family: family, Bold: bold, Path: path})
		}
	}
	doc, err := dom.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	sheets, err := css.ResolveSheets(doc, opts.BaseDir, opts.BaseDir, opts.AllowExternalFiles)
	if err != nil {
		return nil, err
	}
	styles := css.ComputeStyles(doc, sheets)
	pages, err := layout.Layout(doc, styles, fonts, layout.Options{
		PageWidth:  pageW,
		PageHeight: pageH,
		BaseDir:    opts.BaseDir,
		RootDir:    opts.BaseDir,
		Scale:      opts.Scale,

		AllowExternalFiles: opts.AllowExternalFiles,

		MarginTop:    opts.Margin,
		MarginRight:  opts.Margin,
		MarginBottom: opts.Margin,
		MarginLeft:   opts.Margin,
	})
	if err != nil {
		return nil, err
	}
	var out [][]byte
	err = raster.RenderEach(pages, fonts, opts.BaseDir, opts.AllowExternalFiles, scale, func(img *image.NRGBA) error {
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return err
		}
		out = append(out, buf.Bytes())
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
