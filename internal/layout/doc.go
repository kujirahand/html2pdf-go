// Package layout builds the box tree, performs block and table layout, and paginates the result.
//
// Layout runs in a single continuous "flow" coordinate space (as if the page
// were infinitely tall) and records drawing primitives plus atomic units
// (lines, table rows). Pagination then picks page starts between units and
// maps each primitive to a page.
package layout

import (
	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

// Options configures page geometry. All values are in points.
type Options struct {
	PageWidth, PageHeight                            float64
	MarginTop, MarginRight, MarginBottom, MarginLeft float64
	// BaseDir resolves relative image paths.
	BaseDir string
	// RootDir confines every file a document reads during one conversion.
	// It stays the outermost document's BaseDir while nested frame
	// documents resolve their references against their own directory.
	// Empty means BaseDir; Layout fills it in.
	RootDir string
	// AllowExternalFiles lets images and frame documents lie outside
	// RootDir; see filex.ReadUnder.
	AllowExternalFiles bool
	// Scale shrinks (<1) or enlarges (>1) the content; 0 or less means 1.
	// Layout runs on a virtual page of the real size divided by Scale.
	Scale float64

	// depth is the frame nesting level; 0 for the outermost document.
	depth int
	// frames counts the frame documents laid out so far, shared by all
	// nested layouts of one conversion. Layout creates it when nil.
	frames *int
	// work counts layout work units, shared by all nested layouts of one
	// conversion like frames, so frame documents cannot each spend a full
	// budget. Layout creates it when nil.
	work *int
}

// Page is one laid-out page.
type Page struct {
	Width, Height float64
	// Scale is the factor the renderer applies to all Items coordinates.
	Scale float64
	Items []Item
}

// Item is a drawing command.
type Item interface{ isItem() }

// TextItem draws Text with its baseline starting at (X, Y).
type TextItem struct {
	Text      string
	X, Y      float64
	Size      float64
	Bold      bool
	Face      *text.Face // resolved font; nil means the base face chosen by Bold
	Color     css.Color
	Underline bool
}

// RectItem draws a rectangle; Fill and Stroke are optional.
type RectItem struct {
	X, Y, W, H float64
	Fill       *css.Color
	Stroke     *css.Color
	LineWidth  float64
	// RadiusX and RadiusY are the corner radii (top-left, top-right,
	// bottom-right, bottom-left); all zero means a square rectangle.
	RadiusX, RadiusY [4]float64
}

// LineItem draws a straight line. Style is "" or "solid" for a solid line,
// or "dashed" or "dotted".
type LineItem struct {
	X1, Y1, X2, Y2, LineWidth float64
	Color                     css.Color
	Style                     string
}

// ImageItem draws the image at Src, a file path resolved against Dir or a
// data: URI.
type ImageItem struct {
	Src, Dir   string
	X, Y, W, H float64
}

// ClipItem starts a clip to the rectangle (X, Y, W, H), or with End set ends
// the innermost one. Frameset pages and pages with an iframe contain them,
// always in pairs.
type ClipItem struct {
	X, Y, W, H float64
	End        bool
}

func (ClipItem) isItem()  {}
func (TextItem) isItem()  {}
func (RectItem) isItem()  {}
func (LineItem) isItem()  {}
func (ImageItem) isItem() {}

// Layout lays out doc using the computed styles and returns the pages.
func Layout(doc *dom.Document, styles *css.Computed, fonts *text.FontSet, opt Options) ([]*Page, error) {
	scale := opt.Scale
	if scale <= 0 {
		scale = 1
	}
	if opt.frames == nil {
		opt.frames = new(int)
	}
	if opt.work == nil {
		opt.work = new(int)
	}
	if opt.RootDir == "" {
		opt.RootDir = opt.BaseDir
	}
	if scale != 1 {
		// Lay out on a virtual page so that scaling it back gives the real page.
		opt.PageWidth /= scale
		opt.PageHeight /= scale
		opt.MarginTop /= scale
		opt.MarginRight /= scale
		opt.MarginBottom /= scale
		opt.MarginLeft /= scale
	}
	var pages []*Page
	if doc.Frameset != nil {
		page, err := layoutFrameset(doc.Frameset, fonts, opt, scale)
		if err != nil {
			return nil, err
		}
		pages = []*Page{page}
	} else {
		l := &layouter{styles: styles, fonts: fonts, opt: opt, imgs: map[string]*imgInfo{}}
		cw := opt.PageWidth - opt.MarginLeft - opt.MarginRight
		l.block(doc.Body, styles.Style(doc.Body), opt.MarginLeft, 0, cw)
		if l.err != nil {
			return nil, l.err
		}
		pages = l.paginate()
	}
	for _, p := range pages {
		p.Width *= scale
		p.Height *= scale
		p.Scale = scale
	}
	return pages, nil
}
