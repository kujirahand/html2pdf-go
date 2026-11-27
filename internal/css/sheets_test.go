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
	sheets, err := ResolveSheets(doc, dir)
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
	sheets, err := ResolveSheets(doc, dir)
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
	if _, err := ResolveSheets(doc, t.TempDir()); err == nil {
		t.Fatal("expected an error for a missing stylesheet")
	}
}
