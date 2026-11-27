package layout

import (
	"strings"
	"testing"
)

func pageTexts(pg *Page) string {
	var sb strings.Builder
	for _, it := range pg.Items {
		if ti, ok := it.(TextItem); ok {
			sb.WriteString(ti.Text)
		}
	}
	return sb.String()
}

func TestPageBreakTag(t *testing.T) {
	for _, src := range []string{
		"<body><p>ONE</p><page-break></page-break><p>TWO</p></body>",
		"<body><p>ONE</p><page-break><p>TWO</p></body>",
	} {
		pages := run(t, src)
		if len(pages) != 2 {
			t.Fatalf("%s: got %d pages, want 2", src, len(pages))
		}
		if got := pageTexts(pages[0]); !strings.Contains(got, "ONE") || strings.Contains(got, "TWO") {
			t.Errorf("%s: page 1 text = %q", src, got)
		}
		if got := pageTexts(pages[1]); !strings.Contains(got, "TWO") || strings.Contains(got, "ONE") {
			t.Errorf("%s: page 2 text = %q", src, got)
		}
	}
}

func TestPageBreakTagNoBlankPages(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want int
	}{
		{"leading", "<body><page-break><p>ONE</p></body>", 1},
		{"trailing", "<body><p>ONE</p><page-break></body>", 1},
		{"consecutive", "<body><p>ONE</p><page-break><page-break><p>TWO</p></body>", 2},
	}
	for _, c := range cases {
		if pages := run(t, c.src); len(pages) != c.want {
			t.Errorf("%s: got %d pages, want %d", c.name, len(pages), c.want)
		}
	}
}
