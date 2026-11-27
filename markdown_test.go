package html2pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testMarkdown = "# Title\n\n" +
	"## Section\n\n" +
	"A paragraph with **bold**, `code` and a [link](https://example.com/).\n\n" +
	"- one\n- two\n\n" +
	"1. first\n2. second\n\n" +
	"| Name | Value |\n|---|---|\n| a | 1 |\n\n" +
	"> quoted\n\n" +
	"---\n\n" +
	"```go\nfunc main() {\n\tx  := 1\n\n\tprintln(x)\n}\n```\n"

func TestMarkdownToHTML(t *testing.T) {
	out, err := MarkdownToHTML([]byte(testMarkdown))
	if err != nil {
		t.Fatalf("MarkdownToHTML: %v", err)
	}
	got := string(out)
	for _, want := range []string{
		"<!DOCTYPE html>", `<meta charset="utf-8">`, "<style>", "blockquote {",
		"<h1", "<h2", "<table", "<blockquote", "<strong", "<hr", `<a href="https://example.com/"`,
		"<ul>", "<ol>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output does not contain %q", want)
		}
	}
	wantCode := "<pre>func&nbsp;main()&nbsp;{<br>" +
		"&nbsp;&nbsp;&nbsp;&nbsp;x&nbsp;&nbsp;:=&nbsp;1<br>" +
		"&nbsp;<br>" +
		"&nbsp;&nbsp;&nbsp;&nbsp;println(x)<br>" +
		"}</pre>"
	if !strings.Contains(got, wantCode) {
		t.Errorf("code block not preserved as expected, got:\n%s", got)
	}
	if strings.Contains(got, "<pre><code") {
		t.Errorf("<code> tags inside <pre> should be stripped")
	}
}

func TestPreservePreBlocksWithoutCode(t *testing.T) {
	in := "<p>a b</p><pre>x  y\n\tz\n</pre><p>c d</p><pre>q r</pre>"
	want := "<p>a b</p><pre>x&nbsp;&nbsp;y<br>&nbsp;&nbsp;&nbsp;&nbsp;z</pre><p>c d</p><pre>q&nbsp;r</pre>"
	if got := string(preservePreBlocks([]byte(in))); got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

func TestMarkdownToHTMLIndentedCode(t *testing.T) {
	out, err := MarkdownToHTML([]byte("text\n\n    a  b\n    c\n"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "<pre>a&nbsp;&nbsp;b<br>c</pre>"; !strings.Contains(string(out), want) {
		t.Errorf("output does not contain %q:\n%s", want, out)
	}
}

func TestConvertMarkdown(t *testing.T) {
	skipWithoutFonts(t)
	src := "# 見出し\n\n本文です。**太字**も使えます。\n\n| 項目 | 内容 |\n|---|---|\n| 名前 | 山田 |\n"
	var buf bytes.Buffer
	if err := ConvertMarkdown([]byte(src), &buf, &Options{MarkdownCSS: "p { color: #333333; }"}); err != nil {
		t.Fatalf("ConvertMarkdown: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatalf("output does not start with %%PDF-")
	}
}

func TestConvertMarkdownFile(t *testing.T) {
	skipWithoutFonts(t)
	dir := t.TempDir()
	md := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(md, []byte("# Hello\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := &Options{}
	pdf := filepath.Join(dir, "doc.pdf")
	if err := ConvertMarkdownFile(md, pdf, opts); err != nil {
		t.Fatalf("ConvertMarkdownFile: %v", err)
	}
	if opts.BaseDir != "" {
		t.Errorf("caller's Options was modified: BaseDir = %q", opts.BaseDir)
	}
	b, err := os.ReadFile(pdf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Errorf("output does not start with %%PDF-")
	}

	bad := filepath.Join(dir, "bad.pdf")
	if err := ConvertMarkdownFile(md, bad, &Options{PageSize: "nonexistent"}); err == nil {
		t.Fatal("expected an error for an unknown page size")
	}
	if _, err := os.Stat(bad); !os.IsNotExist(err) {
		t.Errorf("failed conversion left an output file (stat err = %v)", err)
	}
}

func TestMarkdownTaskListMarkers(t *testing.T) {
	html, err := MarkdownToHTML([]byte("- [ ] pending\n- [x] done\n"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(html)
	if strings.Contains(s, "<input") {
		t.Errorf("task list checkbox <input> left in HTML: %s", s)
	}
	for _, want := range []string{"[ ]", "pending", "[x]", "done"} {
		if !strings.Contains(s, want) {
			t.Errorf("HTML does not contain %q: %s", want, s)
		}
	}
}

func TestMarkdownPreRawHTMLCaseAndQuotedAttr(t *testing.T) {
	// Raw HTML <pre> in upper case, with a '>' inside a quoted attribute,
	// must still have its whitespace preserved.
	html, err := MarkdownToHTML([]byte("<PRE title=\"a>b\">x  y\n  z</PRE>\n"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(html)
	if !strings.Contains(s, "x&nbsp;&nbsp;y<br>&nbsp;&nbsp;z") {
		t.Errorf("raw <PRE> whitespace was not preserved: %s", s)
	}
	if strings.Contains(s, "a>b") && !strings.Contains(s, `title="a>b"`) {
		t.Errorf("quoted attribute was mangled: %s", s)
	}
}
