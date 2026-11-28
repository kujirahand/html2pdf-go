package html2pdf

import (
	"bytes"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/layout"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

func TestPageBands(t *testing.T) {
	skipWithoutFonts(t)
	fonts, err := text.LoadFontSet(nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	old := now
	now = func() time.Time { return time.Date(2024, 2, 3, 12, 0, 0, 0, time.Local) }
	defer func() { now = old }()
	pages := []*layout.Page{{Width: 300, Height: 400, Scale: 2}, {Width: 300, Height: 400, Scale: 2}}
	opts := &Options{Header: "{page}/{pages} {title} {date} {file}\n{foo}", Footer: "footer\ttext", FileName: "input.html"}
	addPageBands(pages, fonts, opts, "Example")
	for i, p := range pages {
		h := p.Items[len(p.Items)-2].(layout.TextItem)
		f := p.Items[len(p.Items)-1].(layout.TextItem)
		expected := []string{"1/2 Example 2024-02-03 input.html {foo}", "2/2 Example 2024-02-03 input.html {foo}"}[i]
		if h.Text != expected || f.Text != "footer text" {
			t.Fatalf("%q / %q", h.Text, f.Text)
		}
		if h.Size*2 != 9 || h.Y*2 >= 24 || f.Y*2 <= 376 {
			t.Fatalf("incorrect scaled geometry: %+v %+v", h, f)
		}
	}
	p := &layout.Page{Width: 50, Height: 100, Scale: 1}
	addPageBands([]*layout.Page{p}, fonts, &Options{Header: strings.Repeat("long ", 40)}, "")
	item := p.Items[len(p.Items)-1].(layout.TextItem)
	if !strings.HasSuffix(item.Text, "…") || fonts.Regular.Measure(item.Text, 9) > 50 {
		t.Fatalf("bad truncation %q", item.Text)
	}
}

func TestBandMarginsAndPagination(t *testing.T) {
	skipWithoutFonts(t)
	fonts, _ := text.LoadFontSet(nil, "", "")
	doc, _ := dom.Parse(strings.NewReader(`<style>body { margin: 0 } p { margin: 0 }</style><p>first</p><page-break><p>second</p><page-break>`))
	styles := css.ComputeStyles(doc, nil)
	layoutPages := func(m float64, enabled bool) []*layout.Page {
		band := ""
		if enabled {
			band = "x"
		}
		pages, err := layout.Layout(doc, styles, fonts, layout.Options{PageWidth: 200, PageHeight: 100, MarginTop: bandMargin(m, band), MarginBottom: bandMargin(m, band)})
		if err != nil {
			t.Fatal(err)
		}
		return pages
	}
	pages := layoutPages(0, true)
	if len(pages) != 2 {
		t.Fatalf("pages=%d", len(pages))
	}
	for _, p := range pages {
		for _, v := range p.Items {
			if x, ok := v.(layout.TextItem); ok && (x.Y-fonts.Regular.LineAscent(x.Size) < 24-0.1 || x.Y+fonts.Regular.LineDescent(x.Size) > 76+0.1) {
				t.Fatalf("body in band: %+v", x)
			}
		}
	}
	a, b := layoutPages(30, false), layoutPages(30, true)
	if a[0].Items[0] != b[0].Items[0] {
		t.Fatal("body moved with sufficient margin")
	}
	addPageBands(pages, fonts, &Options{Header: "{page}/{pages}"}, "")
	for i, p := range pages {
		h := p.Items[len(p.Items)-1].(layout.TextItem)
		if h.Text != []string{"1/2", "2/2"}[i] {
			t.Fatal(h.Text)
		}
	}
}

func TestBandsPNGAndEmptyCompatibility(t *testing.T) {
	skipWithoutFonts(t)
	src := []byte(`<p>body</p>`)
	var a, b bytes.Buffer
	if err := Convert(src, &a, nil); err != nil {
		t.Fatal(err)
	}
	if err := Convert(src, &b, &Options{Header: "", Footer: ""}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("empty bands changed output")
	}
	images, err := ConvertPNG(src, &Options{Header: "header", Footer: "footer", Scale: 2}, 1)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(images[0]))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range [][2]int{{0, 24}, {img.Bounds().Dy() - 24, img.Bounds().Dy()}} {
		found := false
		for y := r[0]; y < r[1]; y++ {
			for x := 0; x < img.Bounds().Dx(); x++ {
				red, green, blue, _ := img.At(x, y).RGBA()
				if math.Min(float64(red), math.Min(float64(green), float64(blue))) < 0xff00 {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Fatalf("no pixels in %v", r)
		}
	}
}

func TestMarkdownConversionTitle(t *testing.T) {
	src := []byte("# Hello *world* & more\n")
	raw, err := MarkdownToHTML(src)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("<title>")) {
		t.Fatal("MarkdownToHTML changed")
	}
	converted, err := markdownConversionDocument(src, "")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := dom.Parse(bytes.NewReader(converted))
	if err != nil || doc.Title != "Hello world & more" {
		t.Fatalf("title %q: %v", doc.Title, err)
	}
}

func TestFileNamesForFileConversions(t *testing.T) {
	skipWithoutFonts(t)
	dir := t.TempDir()
	for _, tc := range []struct {
		name, src string
		markdown  bool
	}{{"document.html", "<p>body</p>", false}, {"document.md", "# body", true}} {
		input := filepath.Join(dir, tc.name)
		output := filepath.Join(dir, tc.name+".pdf")
		if err := os.WriteFile(input, []byte(tc.src), 0600); err != nil {
			t.Fatal(err)
		}
		for _, custom := range []string{"", "override"} {
			opts := &Options{Header: "{file}", FileName: custom, BaseDir: dir}
			var err error
			if tc.markdown {
				err = ConvertMarkdownFile(input, output, opts)
			} else {
				err = ConvertFile(input, output, opts)
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(output)
			if err != nil {
				t.Fatal(err)
			}
			expected := custom
			if expected == "" {
				expected = tc.name
			}
			var buf bytes.Buffer
			compare := *opts
			compare.FileName = expected
			if tc.markdown {
				err = ConvertMarkdown([]byte(tc.src), &buf, &compare)
			} else {
				err = Convert([]byte(tc.src), &buf, &compare)
			}
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, buf.Bytes()) || opts.FileName != custom {
				t.Fatalf("filename or options mismatch for %s", tc.name)
			}
		}
	}
}

func TestFramesetWithBands(t *testing.T) {
	skipWithoutFonts(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "frame.html"), []byte("<p>frame content</p>"), 0600); err != nil {
		t.Fatal(err)
	}
	src := []byte(`<frameset><frame src="frame.html"></frameset>`)
	var pdf bytes.Buffer
	opts := &Options{BaseDir: dir, Header: "header", Footer: "footer", Scale: 0.7}
	if err := Convert(src, &pdf, opts); err != nil {
		t.Fatal(err)
	}
	if pages, err := ConvertPNG(src, opts, 1); err != nil || len(pages) != 1 {
		t.Fatalf("frameset PNG: %d pages, %v", len(pages), err)
	}
}
