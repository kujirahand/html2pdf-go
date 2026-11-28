package html2pdf

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestPDFNavigationMetadata(t *testing.T) {
	html := `<html><head><title>日本</title><meta NAME="AUTHOR" content="Original"><meta name="description" content="Description"><meta name="keywords" content="keys"></head><body><a href="https://example.org/日">external</a><a href="javascript:alert(1)">bad</a><a href=" ">empty</a><a href="#nope">missing</a><a href="#sec">go</a><page-break></page-break><h1 id="sec">Section</h1><h3>Sub</h3></body></html>`
	var b bytes.Buffer
	if err := Convert([]byte(html), &b, &Options{Title: "Override", Author: "Alice", Subject: "Subject"}); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	for _, want := range []string{"/Annots", "/URI (https://example.org/%E6%97%A5)", "/Outlines", "/Title (Override)", "/Author (Alice)", "/Subject (Subject)", "/Keywords (keys)", "/Info "} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s", want)
		}
	}
	if n := strings.Count(s, "/Subtype /Link"); n != 2 {
		t.Errorf("links: %d", n)
	}
	m := regexp.MustCompile(`(?s)(\d+) 0 obj\n<< /Type /Page /Parent[^\n]+`).FindAllStringSubmatch(s, -1)
	d := regexp.MustCompile(`/Dest \[(\d+) 0 R /XYZ`).FindStringSubmatch(s)
	if len(m) < 2 || len(d) < 2 || d[1] != m[1][1] {
		t.Errorf("destination page: %v pages: %v", d, m)
	}
	b.Reset()
	if err := Convert([]byte(html), &b, &Options{NoBookmarks: true}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "/Outlines") || !strings.Contains(b.String(), "/Title <FEFF65E5672C>") {
		t.Error("bookmarks or Unicode title")
	}
}
func TestMarkdownNavigationTitle(t *testing.T) {
	var b bytes.Buffer
	if err := ConvertMarkdown([]byte("# Heading\n\n[link](https://example.org)"), &b, nil); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"/URI (https://example.org)", "/Title (Heading)", "/Outlines"} {
		if !strings.Contains(b.String(), s) {
			t.Errorf("missing %s", s)
		}
	}
}

func TestWrappedLinkAndEmptyMetadata(t *testing.T) {
	var b bytes.Buffer
	if err := Convert([]byte(`<a href="https://example.org">one two three four five six seven eight nine ten eleven twelve thirteen fourteen</a>`), &b, &Options{PageWidth: 80, PageHeight: 55}); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	if n := strings.Count(s, "/Subtype /Link"); n < 2 {
		t.Errorf("expected wrapped link fragments, got %d", n)
	}
	if !strings.Contains(s, "/Creator (html2pdf-go)") || !strings.Contains(s, "/Producer (html2pdf-go)") {
		t.Error("missing default metadata")
	}
}
