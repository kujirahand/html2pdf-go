package html2pdf

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/gif"
	"image/jpeg"
	pngpkg "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/text"
)

func skipWithoutFonts(t *testing.T) {
	t.Helper()
	if _, err := text.LoadFonts(nil); err != nil {
		t.Skipf("no system font available: %v", err)
	}
}

func TestConvertMinimal(t *testing.T) {
	skipWithoutFonts(t)
	var buf bytes.Buffer
	if err := Convert([]byte("<p>hello</p>"), &buf, nil); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatalf("output does not start with %%PDF-: %q", buf.Bytes()[:min(8, buf.Len())])
	}
}

func TestConvertFixtures(t *testing.T) {
	skipWithoutFonts(t)
	for _, name := range []string{"minimal.html", "invoice.html"} {
		src, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := Convert(src, &buf, &Options{BaseDir: "testdata"}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !strings.HasPrefix(buf.String(), "%PDF-") {
			t.Errorf("%s: not a PDF", name)
		}
	}
}

func TestConvertFontFileMissing(t *testing.T) {
	for _, o := range []*Options{
		{FontFile: "/nonexistent/font.ttf"},
		{FontFileBold: "/nonexistent/bold.ttf"},
	} {
		var buf bytes.Buffer
		if err := Convert([]byte("<p>hello</p>"), &buf, o); err == nil {
			t.Errorf("Convert(%+v) succeeded, want error for unreadable font file", o)
		}
	}
}

func TestConvertFontFile(t *testing.T) {
	const font = "/usr/share/fonts/opentype/noto/NotoSerifCJK-Regular.ttc"
	if _, err := os.Stat(font); err != nil {
		t.Skipf("Noto Serif CJK not available: %v", err)
	}
	src := `<style>h1 { font-family: "Noto Serif CJK JP", serif }</style><h1>見出し</h1><p><b>本文</b></p>`
	var buf bytes.Buffer
	if err := Convert([]byte(src), &buf, &Options{FontFile: font}); err != nil {
		t.Fatalf("Convert with FontFile: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Errorf("output is not a PDF")
	}
}

func TestConvertDataURIImage(t *testing.T) {
	skipWithoutFonts(t)
	var png bytes.Buffer
	if err := pngpkg.Encode(&png, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(png.Bytes())
	var buf bytes.Buffer
	if err := Convert([]byte(`<p>hi <img src="`+uri+`"></p>`), &buf, nil); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("/Subtype /Image")) {
		t.Error("PDF does not contain an image object")
	}
}

func TestConvertDataURIImageFormats(t *testing.T) {
	skipWithoutFonts(t)
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var g, j bytes.Buffer
	if err := gif.Encode(&g, img, nil); err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(&j, img, nil); err != nil {
		t.Fatal(err)
	}
	for name, uri := range map[string]string{
		"gif":  "data:image/gif;base64," + base64.StdEncoding.EncodeToString(g.Bytes()),
		"jpeg": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(j.Bytes()),
	} {
		var buf bytes.Buffer
		if err := Convert([]byte(`<p>hi <img src="`+uri+`"></p>`), &buf, nil); err != nil {
			t.Fatalf("%s: Convert: %v", name, err)
		}
		if !bytes.Contains(buf.Bytes(), []byte("/Subtype /Image")) {
			t.Errorf("%s: PDF does not contain an image object", name)
		}
	}
}

func TestPageSize(t *testing.T) {
	w, h, err := (&Options{PageSize: "letter"}).pageSize()
	if err != nil || w != 612 || h != 792 {
		t.Errorf("letter = %v x %v, %v", w, h, err)
	}
	w, h, _ = (&Options{PageSize: "A5", PageWidth: 100}).pageSize()
	if w != 100 || h != 595.28 {
		t.Errorf("explicit width = %v x %v", w, h)
	}
	if _, _, err := (&Options{PageSize: "B9"}).pageSize(); err == nil {
		t.Error("unknown page size should fail")
	}
}

func TestConvertFileExternalCSS(t *testing.T) {
	skipWithoutFonts(t)
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	write("style.css", "p { color: red }")
	in := write("in.html", `<link rel="stylesheet" href="style.css"><link rel="stylesheet" href="http://example.com/x.css"><p>hello</p>`)
	out := filepath.Join(dir, "out.pdf")
	if err := ConvertFile(in, out, &Options{BaseDir: dir}); err != nil {
		t.Fatalf("ConvertFile: %v", err)
	}
	pdf, err := os.ReadFile(out)
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("bad output: %v", err)
	}

	bad := write("bad.html", `<link rel="stylesheet" href="missing.css"><p>hello</p>`)
	badOut := filepath.Join(dir, "bad.pdf")
	if err := ConvertFile(bad, badOut, &Options{BaseDir: dir}); err == nil {
		t.Error("ConvertFile with a missing stylesheet succeeded, want error")
	}
	if _, err := os.Stat(badOut); err == nil {
		t.Error("failed conversion left an output file")
	}
}

func TestConvertFrameset(t *testing.T) {
	skipWithoutFonts(t)
	dir := t.TempDir()
	for name, body := range map[string]string{
		"a.html": "<p>left</p>",
		"b.html": "<p>right</p>",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	src := `<frameset cols="50%,50%"><frame src="a.html"><frame src="b.html"></frameset>`
	var buf bytes.Buffer
	if err := Convert([]byte(src), &buf, &Options{BaseDir: dir}); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Error("output is not a PDF")
	}
}

func TestConvertRepeatThead(t *testing.T) {
	skipWithoutFonts(t)
	var src strings.Builder
	src.WriteString(`<table><thead><tr><th>Heading</th></tr></thead><tbody>`)
	for i := 0; i < 80; i++ {
		src.WriteString(`<tr><td>body row</td></tr>`)
	}
	src.WriteString(`</tbody></table>`)
	var repeated, once bytes.Buffer
	if err := Convert([]byte(src.String()), &repeated, &Options{}); err != nil {
		t.Fatal(err)
	}
	if err := Convert([]byte(src.String()), &once, &Options{NoRepeatThead: true}); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(repeated.Bytes(), []byte("%PDF-")) || bytes.Equal(repeated.Bytes(), once.Bytes()) {
		t.Fatal("expected different PDF output with repeated headings")
	}
}

func TestMarkdownRepeatThead(t *testing.T) {
	skipWithoutFonts(t)
	src := "| Heading |\n|---|\n" + strings.Repeat("| body row |\n", 80)
	var repeated, once bytes.Buffer
	if err := ConvertMarkdown([]byte(src), &repeated, &Options{}); err != nil {
		t.Fatal(err)
	}
	if err := ConvertMarkdown([]byte(src), &once, &Options{NoRepeatThead: true}); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(repeated.Bytes(), once.Bytes()) {
		t.Fatal("Markdown did not pass the repeat option")
	}
}
