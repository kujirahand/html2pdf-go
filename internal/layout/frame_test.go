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

// iframeClips returns the clip rectangles (not the End markers) of a page.
func iframeClips(p *Page) []ClipItem {
	var out []ClipItem
	for _, it := range p.Items {
		if c, ok := it.(ClipItem); ok && !c.End {
			out = append(out, c)
		}
	}
	return out
}

func findText(p *Page, s string) *TextItem {
	for _, it := range p.Items {
		if ti, ok := it.(TextItem); ok && strings.Contains(ti.Text, s) {
			return &ti
		}
	}
	return nil
}

func TestIframeSrc(t *testing.T) {
	pages := layoutFrameFiles(t, map[string]string{
		"index.html": `<body><iframe src="a.html" width="200" height="100"></iframe></body>`,
		"a.html":     `<body><p>INSIDEFRAME</p></body>`,
	}, "index.html")
	clips := iframeClips(pages[0])
	// The body margin is 8px = 6pt; width and height are px * 0.75.
	if len(clips) != 1 || clips[0] != (ClipItem{X: 6, Y: 6, W: 150, H: 75}) {
		t.Fatalf("clips = %+v, want one at (6,6) 150x75", clips)
	}
	ti := findText(pages[0], "INSIDEFRAME")
	if ti == nil {
		t.Fatal("frame text missing")
	}
	if ti.X < 6 || ti.X > 156 || ti.Y < 6 || ti.Y > 81 {
		t.Errorf("frame text at (%v,%v), outside the box", ti.X, ti.Y)
	}
}

func TestIframeSizeAndBox(t *testing.T) {
	tests := []struct {
		name, html string
		want       ClipItem
	}{
		{"default", `<iframe srcdoc="x"></iframe>`, ClipItem{X: 6, Y: 6, W: 225, H: 112.5}},
		{"css over attr", `<iframe srcdoc="x" width="100" height="100" style="width:80px;height:40px"></iframe>`,
			ClipItem{X: 6, Y: 6, W: 60, H: 30}},
		{"percent width", `<iframe srcdoc="x" width="50%"></iframe>`, ClipItem{X: 6, Y: 6, W: 194, H: 112.5}},
		{"percent height is default", `<iframe srcdoc="x" height="50%"></iframe>`, ClipItem{X: 6, Y: 6, W: 225, H: 112.5}},
		// margin 10px = 7.5pt, border 2px = 1.5pt, padding 4px = 3pt.
		{"padding and border", `<iframe srcdoc="x" style="width:100px;height:50px;padding:4px;border:2px solid black;margin:10px"></iframe>`,
			ClipItem{X: 6 + 7.5 + 1.5 + 3, Y: 6 + 7.5 + 1.5 + 3, W: 75, H: 37.5}},
	}
	for _, tt := range tests {
		pages := layoutFrameFiles(t, map[string]string{"index.html": `<body>` + tt.html + `</body>`}, "index.html")
		clips := iframeClips(pages[0])
		if len(clips) != 1 || !near(clips[0].X, tt.want.X) || !near(clips[0].Y, tt.want.Y) ||
			!near(clips[0].W, tt.want.W) || !near(clips[0].H, tt.want.H) {
			t.Errorf("%s: clips = %+v, want %+v", tt.name, clips, tt.want)
		}
	}
}

func TestIframeSrcdoc(t *testing.T) {
	// srcdoc wins over src, so the missing file is never read.
	pages := layoutFrameFiles(t, map[string]string{
		"index.html": `<body><iframe src="nope.html" srcdoc="<p>FROMSRCDOC</p>"></iframe></body>`,
	}, "index.html")
	if findText(pages[0], "FROMSRCDOC") == nil {
		t.Error("srcdoc text missing")
	}
	if len(iframeClips(pages[0])) != 1 {
		t.Error("srcdoc iframe not clipped")
	}
}

func TestIframeEmpty(t *testing.T) {
	// No src and a remote src draw nothing, but do not fail.
	for _, h := range []string{`<iframe></iframe>`, `<iframe src="http://example.com/"></iframe>`} {
		pages := layoutFrameFiles(t, map[string]string{"index.html": `<body>` + h + `</body>`}, "index.html")
		if n := len(iframeClips(pages[0])); n != 0 {
			t.Errorf("%s: %d clips, want 0", h, n)
		}
	}
}

func TestIframeInlineNotDrawn(t *testing.T) {
	pages := layoutFrameFiles(t, map[string]string{
		"index.html": `<body><p>before <span><iframe srcdoc="<p>HIDDENFRAME</p>">FALLBACK</iframe></span></p></body>`,
	}, "index.html")
	if n := len(iframeClips(pages[0])); n != 0 || findText(pages[0], "HIDDENFRAME") != nil || findText(pages[0], "FALLBACK") != nil {
		t.Errorf("inline iframe was drawn: %d clips", n)
	}
}

func TestIframeMissingSrc(t *testing.T) {
	fonts, err := text.LoadFontSet(nil, "", "")
	if err != nil {
		t.Skipf("no system font available: %v", err)
	}
	doc, _ := dom.Parse(strings.NewReader(`<body><iframe src="nope.html"></iframe></body>`))
	_, err = Layout(doc, css.ComputeStyles(doc, nil), fonts, Options{PageWidth: 400, PageHeight: 300, BaseDir: t.TempDir()})
	if err == nil {
		t.Fatal("expected error for unreadable iframe src")
	}
}

func TestIframeSelfReference(t *testing.T) {
	fonts, err := text.LoadFontSet(nil, "", "")
	if err != nil {
		t.Skipf("no system font available: %v", err)
	}
	dir := t.TempDir()
	body := `<body><iframe src="self.html"></iframe></body>`
	if err := os.WriteFile(filepath.Join(dir, "self.html"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, _ := dom.Parse(strings.NewReader(body))
	_, err = Layout(doc, css.ComputeStyles(doc, nil), fonts, Options{PageWidth: 400, PageHeight: 300, BaseDir: dir})
	if err == nil {
		t.Fatal("expected error for a self-referencing iframe")
	}
}
