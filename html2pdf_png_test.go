package html2pdf

import (
	"bytes"
	"testing"
)

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

func TestConvertPNG(t *testing.T) {
	skipWithoutFonts(t)
	pages, err := ConvertPNG([]byte("<p>hello</p>"), nil, 0)
	if err != nil {
		t.Fatalf("ConvertPNG: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("got %d pages, want 1", len(pages))
	}
	if !bytes.HasPrefix(pages[0], pngSignature) {
		t.Errorf("output does not start with the PNG signature: %q", pages[0][:min(8, len(pages[0]))])
	}
}

func TestConvertPNGPageBreak(t *testing.T) {
	skipWithoutFonts(t)
	pages, err := ConvertPNG([]byte("<p>one</p><page-break><p>two</p>"), nil, 1)
	if err != nil {
		t.Fatalf("ConvertPNG: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
}

func TestConvertMarkdownPNG(t *testing.T) {
	skipWithoutFonts(t)
	pages, err := ConvertMarkdownPNG([]byte("# Hi"), nil, 1)
	if err != nil {
		t.Fatalf("ConvertMarkdownPNG: %v", err)
	}
	if len(pages) != 1 || !bytes.HasPrefix(pages[0], pngSignature) {
		t.Fatalf("got %d pages, want 1 PNG", len(pages))
	}
}
