package html2pdf

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// TestInternalLinkTargetsOnLaterPage checks that a link on page 1 jumps to
// the page that holds the heading, and that several kinds of anchors resolve.
func TestInternalLinkTargetsOnLaterPage(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`<a href="#late">go</a> <a href="#empty">empty</a> <a href="#named">named</a> <a href="#nope">nope</a>`)
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&sb, "<p>filler %d</p>", i)
	}
	sb.WriteString(`<h2 id="late">Late</h2><p>x<span id="empty"></span> y</p><p><a name="named"></a>Named</p>`)
	var b bytes.Buffer
	if err := Convert([]byte(sb.String()), &b, nil); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	pages := regexp.MustCompile(`(\d+) 0 obj\n<< /Type /Page /Parent`).FindAllStringSubmatch(s, -1)
	if len(pages) < 2 {
		t.Fatalf("want a multi-page document, got %d pages", len(pages))
	}
	if n := strings.Count(s, "/Subtype /Link"); n != 3 {
		t.Errorf("links: got %d, want 3 (the #nope link must be dropped)", n)
	}
	for _, d := range regexp.MustCompile(`/Dest \[(\d+) 0 R /XYZ`).FindAllStringSubmatch(s, -1) {
		if d[1] == pages[0][1] {
			t.Errorf("destination %s should not be the first page", d[1])
		}
	}
}

func TestSkippedHeadingLevelsKeepOutlinesValid(t *testing.T) {
	var b bytes.Buffer
	if err := Convert([]byte(`<h3>deep first</h3><h1>top</h1><h4>jump</h4><h2>mid</h2><h1></h1>`), &b, nil); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	if !strings.Contains(s, "/Type /Outlines") || !strings.Contains(s, "/Count 4") {
		t.Errorf("outline root missing or wrong count (empty heading must be skipped)")
	}
}
