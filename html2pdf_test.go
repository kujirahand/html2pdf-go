package html2pdf

import (
	"bytes"
	"os"
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
