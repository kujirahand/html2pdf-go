package layout

import (
	"os"
	"strings"
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

func run(t *testing.T, src string) []*Page {
	t.Helper()
	return runOpts(t, src, Options{})
}

// runOpts lays out src on A4; only opt.Scale is taken from opt.
func runOpts(t *testing.T, src string, opt Options) []*Page {
	t.Helper()
	fonts, err := text.LoadFontSet(nil, "", "")
	if err != nil {
		t.Skipf("no system font available: %v", err)
	}
	doc, err := dom.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	styles := css.ComputeStyles(doc, doc.StylesheetText)
	pages, err := Layout(doc, styles, fonts, Options{PageWidth: 595.28, PageHeight: 841.89, BaseDir: "../../testdata", Scale: opt.Scale})
	if err != nil {
		t.Fatal(err)
	}
	return pages
}

func TestMinimalFixture(t *testing.T) {
	src, err := os.ReadFile("../../testdata/minimal.html")
	if err != nil {
		t.Fatal(err)
	}
	pages := run(t, string(src))
	if len(pages) < 1 {
		t.Fatalf("got %d pages, want >= 1", len(pages))
	}
	if len(pages[0].Items) == 0 {
		t.Error("first page has no items")
	}
}

func TestForcedBreak(t *testing.T) {
	pages := run(t, `<body><p>a</p><div style="page-break-after:always"></div><p>b</p></body>`)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	// No trailing blank page after a final forced break.
	pages = run(t, `<body><p>a</p><div style="page-break-after:always"></div></body>`)
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
}

func TestOverflowPaginates(t *testing.T) {
	pages := run(t, `<body>`+strings.Repeat("<p>line</p>", 200)+`</body>`)
	if len(pages) < 2 {
		t.Fatalf("got %d pages, want >= 2", len(pages))
	}
}

func TestInvoiceThreePages(t *testing.T) {
	src, err := os.ReadFile("../../testdata/invoice.html")
	if err != nil {
		t.Fatal(err)
	}
	// 実寸（Scale なし）で 3 ページに収まる。名前と印鑑を同じ行にしたことで
	// 差出人欄が低くなり、以前あった約 6pt の溢れは解消した。
	// Fits on 3 pages at real size; putting the name and the stamp on one
	// row made the sender box short enough to remove the old overflow.
	pages := run(t, string(src))
	if len(pages) != 3 {
		t.Errorf("invoice laid out on %d pages, want 3", len(pages))
	}
}

func TestInvoiceScaledPagesKeepRealSize(t *testing.T) {
	src, err := os.ReadFile("../../testdata/invoice.html")
	if err != nil {
		t.Fatal(err)
	}
	pages := runOpts(t, string(src), Options{Scale: 0.95})
	if len(pages) != 3 {
		t.Errorf("scaled invoice laid out on %d pages, want 3", len(pages))
	}
	for _, p := range pages {
		if p.Scale != 0.95 || p.Width < 595 || p.Width > 596 {
			t.Errorf("page = %vx%v scale %v, want real size with scale 0.95", p.Width, p.Height, p.Scale)
		}
	}
}

func TestInvoiceTableHeadersNotWrapped(t *testing.T) {
	src, err := os.ReadFile("../../testdata/invoice.html")
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, p := range run(t, string(src)) {
		for _, it := range p.Items {
			switch ti := it.(type) {
			case TextItem:
				found[ti.Text] = true
			case *TextItem:
				found[ti.Text] = true
			}
		}
	}
	for _, h := range []string{"数量", "単価", "金額", "備考"} {
		if !found[h] {
			t.Errorf("table header %q is not laid out as a single text run", h)
		}
	}
}

func TestSiblingMarginsCollapse(t *testing.T) {
	pages := run(t, `<body><div style="margin-bottom:40px">AAA</div><div style="margin-top:40px">BBB</div></body>`)
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	var ya, yb float64
	for _, it := range pages[0].Items {
		if tx, ok := it.(TextItem); ok {
			switch tx.Text {
			case "AAA":
				ya = tx.Y
			case "BBB":
				yb = tx.Y
			}
		}
	}
	if ya == 0 || yb == 0 {
		t.Fatalf("texts not found: AAA=%v BBB=%v", ya, yb)
	}
	// Line height 12pt + collapsed margin max(30pt, 30pt) = 42pt between
	// baselines. Without collapsing the gap would be 12 + 30 + 30 = 72pt.
	if d := yb - ya; d < 36 || d > 48 {
		t.Errorf("baseline distance = %.1f, want about 42 (collapsed margins)", d)
	}
}
