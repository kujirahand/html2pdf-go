package css

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/dom"
)

func findTag(n *dom.Node, tag string) *dom.Node {
	if n == nil {
		return nil
	}
	if n.Tag == tag {
		return n
	}
	for _, c := range n.Children {
		if f := findTag(c, tag); f != nil {
			return f
		}
	}
	return nil
}

// The text read from a linked stylesheet must actually reach the cascade:
// emptying it (a plausible regression) has to change computed styles.
func TestResolveSheetsAppliesExternalCSS(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte("p { font-size: 32pt; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := dom.Parse(strings.NewReader(
		`<html><head><link rel="stylesheet" href="style.css"></head><body><p>x</p></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	sheets, err := ResolveSheets(doc, dir, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	styles := ComputeStyles(doc, sheets)
	p := findTag(doc.Body, "p")
	if p == nil {
		t.Fatal("no <p> found")
	}
	if got := styles.Style(p).FontSize.Resolve(12, 12, 0); got != 32 {
		t.Fatalf("font-size = %v, want 32 (from the external sheet)", got)
	}
}

// A <style> block that comes after a <link> wins at equal specificity,
// because sheets apply in document order.
func TestResolveSheetsDocumentOrder(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.css"), []byte("p { font-size: 20pt; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := dom.Parse(strings.NewReader(
		`<html><head><link rel="stylesheet" href="a.css"><style>p { font-size: 10pt; }</style></head><body><p>x</p></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	sheets, err := ResolveSheets(doc, dir, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	styles := ComputeStyles(doc, sheets)
	p := findTag(doc.Body, "p")
	if got := styles.Style(p).FontSize.Resolve(12, 12, 0); got != 10 {
		t.Fatalf("font-size = %v, want 10 (the later <style> wins)", got)
	}
}

func TestResolveSheetsMissingFile(t *testing.T) {
	doc, err := dom.Parse(strings.NewReader(
		`<html><head><link rel="stylesheet" href="nope.css"></head><body><p>x</p></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if _, err := ResolveSheets(doc, dir, dir, false); err == nil {
		t.Fatal("expected an error for a missing stylesheet")
	}
}

// Links with a scheme, or protocol-relative ones, are URLs, not files: they
// are ignored rather than read from disk.
func TestResolveSheetsIgnoresURLs(t *testing.T) {
	for _, href := range []string{
		"//example.com/x.css",
		"https://example.com/x.css",
		"data:text/css,p{color:red}",
		"about:blank",
		"blob:abc",
	} {
		doc, err := dom.Parse(strings.NewReader(
			`<html><head><link rel="stylesheet" href="` + href + `"></head><body><p>x</p></body></html>`))
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		sheets, err := ResolveSheets(doc, dir, dir, false)
		if err != nil {
			t.Errorf("%s: unexpected error %v", href, err)
		}
		if len(sheets) != 0 {
			t.Errorf("%s: got %d sheets, want 0", href, len(sheets))
		}
	}
}

// A stylesheet outside baseDir, reached with ".." or an absolute path, is
// refused unless external files are allowed.
func TestResolveSheetsConfinement(t *testing.T) {
	root := t.TempDir()
	base := filepath.Join(root, "doc")
	if err := os.Mkdir(base, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(root, "outside.css")
	if err := os.WriteFile(outside, []byte("p { font-size: 32pt; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, href := range []string{"../outside.css", outside} {
		doc, err := dom.Parse(strings.NewReader(
			`<html><head><link rel="stylesheet" href="` + href + `"></head><body><p>x</p></body></html>`))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ResolveSheets(doc, base, base, false); err == nil {
			t.Errorf("href %q outside baseDir was read by default", href)
		}
		sheets, err := ResolveSheets(doc, base, base, true)
		if err != nil || len(sheets) != 1 || !strings.Contains(sheets[0], "32pt") {
			t.Errorf("href %q with allowExternal: %v, %v", href, sheets, err)
		}
	}
}

// A stylesheet that is not a regular file (here /dev/zero) is an error, not
// an unbounded read.
func TestResolveSheetsRejectsDevice(t *testing.T) {
	if _, err := os.Stat("/dev/zero"); err != nil {
		t.Skip("no /dev/zero")
	}
	doc, err := dom.Parse(strings.NewReader(
		`<html><head><link rel="stylesheet" href="/dev/zero"></head><body><p>x</p></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveSheets(doc, t.TempDir(), t.TempDir(), true); err == nil {
		t.Fatal("expected an error for a device file")
	}
}
