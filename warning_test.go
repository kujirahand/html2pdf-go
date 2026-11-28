package html2pdf

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
func TestConversionWarnings(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.png"), []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	src := []byte(`<style>@media print { p { ignored: yes } } p { font-style: italic; display: flex } .none { font-style: normal }</style><link rel="stylesheet" href="https://example.test/a.css"><table><tr><td rowspan="2"><img src="https://example.test/a.png"><img src="bad.png">&#x10ffff;</td></tr></table>`)
	var got []Warning
	opts := &Options{BaseDir: dir, OnWarning: func(w Warning) { got = append(got, w) }}
	var with, without bytes.Buffer
	if err := Convert(src, &with, opts); err != nil {
		t.Fatal(err)
	}
	if err := Convert(src, &without, &Options{BaseDir: dir}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(with.Bytes(), without.Bytes()) {
		t.Fatal("warnings changed PDF")
	}
	want := map[string]int{"css-property:font-style": 2, "css-value:display: flex": 1, "css-at-rule:@media": 1, "html-attribute:td[rowspan]": 1, "remote-resource:https://example.test/a.css": 1, "remote-resource:https://example.test/a.png": 1, "image-decode:bad.png": 1, "missing-glyph:U+10FFFF": 1}
	for _, w := range got {
		delete(want, w.Kind+":"+w.Key)
		if w.Key == "font-style" && w.Count != 2 {
			t.Fatal(w)
		}
	}
	if len(want) > 0 {
		t.Fatalf("missing %v; got %+v", want, got)
	}
	got = nil
	if err := Convert(src, failWriter{}, opts); err == nil || len(got) == 0 {
		t.Fatalf("error or warnings missing: %v %+v", err, got)
	}
	got = nil
	if _, err := ConvertPNG(src, opts, 0.2); err != nil || len(got) == 0 {
		t.Fatalf("PNG: %v %+v", err, got)
	}
	got = nil
	opts.MarkdownCSS = "p { font-style: italic }"
	if err := ConvertMarkdown([]byte("hello"), &bytes.Buffer{}, opts); err != nil || len(got) != 1 || got[0].Key != "font-style" {
		t.Fatalf("Markdown: %v %+v", err, got)
	}
	got = nil
	if err := Convert([]byte("<p>hello</p>"), &bytes.Buffer{}, opts); err != nil || len(got) != 0 {
		t.Fatalf("clean: %v %+v", err, got)
	}
	if s := (Warning{Message: "example", Count: 3}).String(); !strings.Contains(s, "(3 times)") {
		t.Fatal(s)
	}
}
