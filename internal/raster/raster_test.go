package raster

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/layout"
	"github.com/kujirahand/html2pdf-go/internal/text"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	pageW = 200.0
	pageH = 100.0
)

// testFonts loads Go Regular so results do not depend on system fonts.
func testFonts(t *testing.T) *text.FontSet {
	t.Helper()
	path := filepath.Join(t.TempDir(), "regular.ttf")
	if err := os.WriteFile(path, goregular.TTF, 0o600); err != nil {
		t.Fatal(err)
	}
	fonts, err := text.LoadFontSet([]string{filepath.Dir(path)}, path, "")
	if err != nil {
		t.Fatal(err)
	}
	return fonts
}

func layoutPages(t *testing.T, fonts *text.FontSet, src string) []*layout.Page {
	t.Helper()
	doc, err := dom.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	styles := css.ComputeStyles(doc, doc.StylesheetText)
	pages, err := layout.Layout(doc, styles, fonts, layout.Options{PageWidth: pageW, PageHeight: pageH, BaseDir: "../../testdata"})
	if err != nil {
		t.Fatal(err)
	}
	return pages
}

func TestRenderSize(t *testing.T) {
	fonts := testFonts(t)
	pages := layoutPages(t, fonts, "<p>hello</p>")
	for _, c := range []struct {
		scale float64
		w, h  int
	}{{0, 400, 200}, {1, 200, 100}, {1.5, 300, 150}} {
		imgs, err := Render(pages, fonts, "", false, c.scale)
		if err != nil {
			t.Fatal(err)
		}
		if len(imgs) != 1 {
			t.Fatalf("scale %v: %d images, want 1", c.scale, len(imgs))
		}
		if b := imgs[0].Bounds(); b.Dx() != c.w || b.Dy() != c.h {
			t.Errorf("scale %v: size %dx%d, want %dx%d", c.scale, b.Dx(), b.Dy(), c.w, c.h)
		}
	}
}

func TestRenderRect(t *testing.T) {
	fonts := testFonts(t)
	page := &layout.Page{Width: pageW, Height: pageH, Items: []layout.Item{
		layout.RectItem{X: 20, Y: 20, W: 100, H: 50, Fill: &css.Color{R: 1}},
	}}
	imgs, err := Render([]*layout.Page{page}, fonts, "", false, 2)
	if err != nil {
		t.Fatal(err)
	}
	img := imgs[0]
	if got, want := img.NRGBAAt(140, 90), (color.NRGBA{R: 255, A: 255}); got != want {
		t.Errorf("center pixel = %v, want %v", got, want)
	}
	if got, want := img.NRGBAAt(10, 10), (color.NRGBA{R: 255, G: 255, B: 255, A: 255}); got != want {
		t.Errorf("outside pixel = %v, want white", got)
	}
}

func TestRenderRoundedStrokeAndLines(t *testing.T) {
	fonts := testFonts(t)
	blue := css.Color{B: 1}
	page := &layout.Page{Width: pageW, Height: pageH, Items: []layout.Item{
		layout.RectItem{X: 10, Y: 10, W: 80, H: 60, Stroke: &blue, LineWidth: 4,
			RadiusX: [4]float64{10, 10, 10, 10}, RadiusY: [4]float64{10, 10, 10, 10}},
		layout.LineItem{X1: 100, Y1: 20, X2: 190, Y2: 20, LineWidth: 4, Color: css.Color{R: 1}, Style: "dashed"},
		layout.LineItem{X1: 100, Y1: 50, X2: 190, Y2: 50, LineWidth: 4, Color: css.Color{R: 1}, Style: "dotted"},
	}}
	imgs, err := Render([]*layout.Page{page}, fonts, "", false, 2)
	if err != nil {
		t.Fatal(err)
	}
	img := imgs[0]
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	// Left edge of the stroked rectangle is blue; its interior stays white.
	if got := img.NRGBAAt(20, 80); got != (color.NRGBA{B: 255, A: 255}) {
		t.Errorf("stroke pixel = %v, want blue", got)
	}
	if got := img.NRGBAAt(90, 80); got != white {
		t.Errorf("interior pixel = %v, want white", got)
	}
	// First dash covers x 100..112 pt, the gap follows.
	if got := img.NRGBAAt(210, 40); got != (color.NRGBA{R: 255, A: 255}) {
		t.Errorf("dash pixel = %v, want red", got)
	}
	if got := img.NRGBAAt(236, 40); got != white {
		t.Errorf("gap pixel = %v, want white", got)
	}
}

func TestRenderText(t *testing.T) {
	fonts := testFonts(t)
	pages := layoutPages(t, fonts, "<p style='font-size:24pt'>Hello</p>")
	imgs, err := Render(pages, fonts, "", false, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !hasInk(imgs[0], imgs[0].Bounds()) {
		t.Error("text page is blank")
	}
}

func hasInk(img *image.NRGBA, r image.Rectangle) bool {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if c := img.NRGBAAt(x, y); c.R != 255 || c.G != 255 || c.B != 255 {
				return true
			}
		}
	}
	return false
}

func TestRenderTwoPages(t *testing.T) {
	fonts := testFonts(t)
	pages := layoutPages(t, fonts, "<p>one</p><page-break><p>two</p>")
	imgs, err := Render(pages, fonts, "", false, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 2 {
		t.Fatalf("got %d images, want 2", len(imgs))
	}
}

func TestRenderClip(t *testing.T) {
	fonts := testFonts(t)
	page := &layout.Page{Width: pageW, Height: pageH, Items: []layout.Item{
		layout.ClipItem{X: 0, Y: 0, W: 50, H: 100},
		layout.RectItem{X: 0, Y: 0, W: 200, H: 100, Fill: &css.Color{G: 1}},
		layout.ClipItem{End: true},
	}}
	imgs, err := Render([]*layout.Page{page}, fonts, "", false, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := imgs[0].NRGBAAt(25, 50); got != (color.NRGBA{G: 255, A: 255}) {
		t.Errorf("inside clip = %v, want green", got)
	}
	if got := imgs[0].NRGBAAt(150, 50); got != (color.NRGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Errorf("outside clip = %v, want white", got)
	}
}

func TestRenderImageSkippedWhenMissing(t *testing.T) {
	fonts := testFonts(t)
	page := &layout.Page{Width: pageW, Height: pageH, Items: []layout.Item{
		layout.ImageItem{Src: "missing.png", Dir: t.TempDir(), W: 10, H: 10},
	}}
	if _, err := Render([]*layout.Page{page}, fonts, t.TempDir(), false, 1); err != nil {
		t.Fatalf("a missing image must be skipped, got %v", err)
	}
}

func TestRenderRejectsHugeScale(t *testing.T) {
	fonts := testFonts(t)
	page := &layout.Page{Width: pageW, Height: pageH}
	if _, err := Render([]*layout.Page{page}, fonts, "", false, 1e6); err == nil {
		t.Error("expected an error for an oversized bitmap")
	}
}

// Clip layers are full-page bitmaps; their total size is budgeted so
// deeply nested frames on a huge page cannot multiply memory without
// limit.
func TestRenderClipBudget(t *testing.T) {
	saved := maxClipPixels
	defer func() { maxClipPixels = saved }()
	maxClipPixels = 10 // smaller than one page layer
	fonts := testFonts(t)
	page := &layout.Page{Width: pageW, Height: pageH, Items: []layout.Item{
		layout.ClipItem{X: 0, Y: 0, W: 50, H: 100},
		layout.RectItem{X: 0, Y: 0, W: 200, H: 100, Fill: &css.Color{G: 1}},
		layout.ClipItem{End: true},
	}}
	if _, err := Render([]*layout.Page{page}, fonts, "", false, 1); err == nil {
		t.Fatal("clip over the layer budget was accepted")
	}
}
