package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

func TestFrameTracks(t *testing.T) {
	tests := []struct {
		attr string
		want []float64
	}{
		{"", []float64{400}},
		{"50%,50%", []float64{200, 200}},
		{"100px,*", []float64{75, 325}},
		{"*,2*", []float64{400.0 / 3, 800.0 / 3}},
		{"25%,*,100px", []float64{100, 225, 75}},
	}
	for _, tt := range tests {
		got := frameTracks(tt.attr, 400)
		if len(got) != len(tt.want) {
			t.Errorf("frameTracks(%q) = %v, want %v", tt.attr, got, tt.want)
			continue
		}
		for i := range got {
			if !near(got[i], tt.want[i]) {
				t.Errorf("frameTracks(%q) = %v, want %v", tt.attr, got, tt.want)
				break
			}
		}
	}
}

func near(a, b float64) bool { return a-b < 1e-6 && b-a < 1e-6 }

func layoutFrameFiles(t *testing.T, files map[string]string, main string) []*Page {
	t.Helper()
	fonts, err := text.LoadFontSet(nil, "", "")
	if err != nil {
		t.Skipf("no system font available: %v", err)
	}
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	doc, err := dom.Parse(strings.NewReader(files[main]))
	if err != nil {
		t.Fatal(err)
	}
	sheets, err := css.ResolveSheets(doc, dir)
	if err != nil {
		t.Fatal(err)
	}
	pages, err := Layout(doc, css.ComputeStyles(doc, sheets), fonts, Options{PageWidth: 400, PageHeight: 300, BaseDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	return pages
}

func TestFrameset(t *testing.T) {
	pages := layoutFrameFiles(t, map[string]string{
		"index.html": `<frameset cols="50%,50%"><frame src="a.html"><frame src="b.html"></frameset>`,
		"a.html":     `<body><p>LEFTFRAME</p></body>`,
		"b.html":     `<body><p>RIGHTFRAME</p></body>`,
	}, "index.html")
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	var left, right *TextItem
	clips, ends := 0, 0
	for _, it := range pages[0].Items {
		switch it := it.(type) {
		case TextItem:
			if strings.Contains(it.Text, "LEFTFRAME") {
				cp := it
				left = &cp
			}
			if strings.Contains(it.Text, "RIGHTFRAME") {
				cp := it
				right = &cp
			}
		case ClipItem:
			if it.End {
				ends++
			} else {
				clips++
			}
		}
	}
	if left == nil || right == nil {
		t.Fatalf("frame texts missing: left=%v right=%v", left, right)
	}
	if left.X >= 200 || right.X < 200 {
		t.Errorf("frame texts misplaced: left.X=%v right.X=%v", left.X, right.X)
	}
	if clips != 2 || ends != 2 {
		t.Errorf("clips=%d ends=%d, want 2 and 2", clips, ends)
	}
}

func TestFramesetMissingFrame(t *testing.T) {
	fonts, err := text.LoadFontSet(nil, "", "")
	if err != nil {
		t.Skipf("no system font available: %v", err)
	}
	doc, _ := dom.Parse(strings.NewReader(`<frameset><frame src="nope.html"></frameset>`))
	_, err = Layout(doc, css.ComputeStyles(doc, nil), fonts, Options{PageWidth: 400, PageHeight: 300, BaseDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for unreadable frame")
	}
}
