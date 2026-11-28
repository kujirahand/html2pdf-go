package html2pdf

import (
	"bytes"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
	"os"
	"path/filepath"
	"testing"
)

func TestItalicOptionsAndMarkdown(t *testing.T) {
	d := t.TempDir()
	paths := make([]string, 3)
	for i, data := range [][]byte{goregular.TTF, goitalic.TTF, gobolditalic.TTF} {
		paths[i] = filepath.Join(d, string(rune('a'+i))+".ttf")
		if err := os.WriteFile(paths[i], data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var uses []FontUse
	opts := &Options{FontFile: paths[0], FontFileItalic: paths[1], FontFileBoldItalic: paths[2], OnFontUse: func(u FontUse) { uses = append(uses, u) }}
	var buf bytes.Buffer
	if err := ConvertMarkdown([]byte("*emphasis* and ***bold italic***"), &buf, opts); err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, u := range uses {
		if u.Italic {
			found[u.Path] = true
		}
	}
	if !found[paths[1]] || !found[paths[2]] {
		t.Fatalf("italic paths: %+v", uses)
	}
	opts.FontFileItalic = ""
	opts.FontFileBoldItalic = ""
	uses = nil
	buf.Reset()
	if err := Convert([]byte("<em>fallback</em>"), &buf, opts); err != nil {
		t.Fatal(err)
	}
	if len(uses) == 0 || uses[0].Path != paths[0] || !uses[0].Italic {
		t.Fatalf("fallback: %+v", uses)
	}
	for _, which := range []bool{false, true} {
		o := Options{FontFile: paths[0]}
		if which {
			o.FontFileBoldItalic = filepath.Join(d, "missing")
		} else {
			o.FontFileItalic = filepath.Join(d, "missing")
		}
		if err := Convert([]byte("x"), &buf, &o); err == nil {
			t.Error("missing italic font accepted")
		}
	}
}

func TestRegularOutputUnaffectedByItalicOptions(t *testing.T) {
	d := t.TempDir()
	regular := filepath.Join(d, "regular.ttf")
	italic := filepath.Join(d, "italic.ttf")
	if err := os.WriteFile(regular, goregular.TTF, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(italic, goitalic.TTF, 0600); err != nil {
		t.Fatal(err)
	}
	var a, b bytes.Buffer
	source := []byte("<p>Plain document</p>")
	if err := Convert(source, &a, &Options{FontFile: regular}); err != nil {
		t.Fatal(err)
	}
	if err := Convert(source, &b, &Options{FontFile: regular, FontFileItalic: italic}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("unused italic changed PDF output")
	}
}
