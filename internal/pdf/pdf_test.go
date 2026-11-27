package pdf

import (
	"bytes"
	"compress/zlib"
	"image"
	"image/color"
	"image/png"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/text"
)

func checkStructure(t *testing.T, out []byte) {
	t.Helper()
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Errorf("output does not start with %%PDF-")
	}
	if !bytes.HasSuffix(bytes.TrimSpace(out), []byte("%%EOF")) {
		t.Errorf("output does not end with %%%%EOF")
	}
	i := bytes.LastIndex(out, []byte("startxref\n"))
	if i < 0 {
		t.Fatal("missing startxref")
	}
	rest := strings.Fields(string(out[i+len("startxref\n"):]))
	pos, err := strconv.Atoi(rest[0])
	if err != nil {
		t.Fatalf("bad startxref offset %q: %v", rest[0], err)
	}
	if !bytes.HasPrefix(out[pos:], []byte("xref\n")) {
		t.Errorf("startxref offset %d does not point at the xref table", pos)
	}
}

func TestTextPDF(t *testing.T) {
	fonts, err := text.LoadFonts(nil)
	if err != nil {
		t.Skipf("no system font available: %v", err)
	}
	var buf bytes.Buffer
	w := NewWriter(&buf)
	f := w.NewFont(fonts.Regular)
	p := w.AddPage(595, 842)
	p.SetFillColor(0.9, 0.9, 0.9)
	p.FillRect(50, 50, 200, 30)
	p.SetFillColor(0, 0, 0)
	p.ShowText(f, 14, 56, 70, "Hello, こんにちは")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out := buf.Bytes()
	checkStructure(t, out)
	for _, want := range []string{"/Type /Page ", "/Type /Pages", "/Subtype /Type3", "/ToUnicode", "/F0_0"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("output lacks %q", want)
		}
	}
}

// contentStreams returns the inflated content of every Flate stream in out.
func contentStreams(out []byte) string {
	var sb strings.Builder
	rest := out
	for {
		i := bytes.Index(rest, []byte("stream\n"))
		if i < 0 {
			break
		}
		rest = rest[i+len("stream\n"):]
		j := bytes.Index(rest, []byte("\nendstream"))
		if j < 0 {
			break
		}
		if zr, err := zlib.NewReader(bytes.NewReader(rest[:j])); err == nil {
			data, _ := io.ReadAll(zr)
			sb.Write(data)
		}
		rest = rest[j:]
	}
	return sb.String()
}

func TestDrawStyledLine(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	p := w.AddPage(595, 842)
	p.DrawStyledLine(10, 10, 100, 10, 2, "dashed")
	p.DrawStyledLine(10, 20, 100, 20, 2, "dotted")
	p.DrawStyledLine(10, 30, 100, 30, 2, "solid")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	cs := contentStreams(buf.Bytes())
	for _, want := range []string{"[6 4] 0 d", "1 J", "[0 4] 0 d"} {
		if !strings.Contains(cs, want) {
			t.Errorf("content stream lacks %q:\n%s", want, cs)
		}
	}
	if n := strings.Count(cs, " d\n"); n != 2 {
		t.Errorf("got %d dash operators, want 2 (solid line must not set one)", n)
	}
	if strings.Count(cs, "q\n") != strings.Count(cs, "Q\n") {
		t.Errorf("unbalanced q/Q:\n%s", cs)
	}
}

func TestFontChunking(t *testing.T) {
	fonts, err := text.LoadFonts(nil)
	if err != nil {
		t.Skipf("no system font available: %v", err)
	}
	var buf bytes.Buffer
	w := NewWriter(&buf)
	f := w.NewFont(fonts.Regular)
	p := w.AddPage(595, 842)
	var sb strings.Builder
	for r := rune(0x4E00); r < 0x4E00+300; r++ {
		sb.WriteRune(r)
	}
	p.ShowText(f, 10, 10, 20, sb.String())
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	checkStructure(t, buf.Bytes())
	if !bytes.Contains(buf.Bytes(), []byte("/F0_1")) {
		t.Error("expected a second font chunk /F0_1")
	}
}

func TestPNGImage(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.Set(0, 0, color.NRGBA{R: 255, A: 255})
	src.Set(1, 0, color.NRGBA{G: 255, A: 128})
	var png1 bytes.Buffer
	if err := png.Encode(&png1, src); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := NewWriter(&buf)
	img, err := w.AddPNGImage(png1.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	p := w.AddPage(200, 200)
	p.DrawImage(img, 10, 10, 50, 50)
	p.SetStrokeColor(0, 0, 0)
	p.StrokeRect(10, 10, 50, 50, 1)
	p.DrawLine(0, 0, 100, 100, 0.5)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	checkStructure(t, buf.Bytes())
	for _, want := range []string{"/Subtype /Image", "/SMask", "/XObject"} {
		if !bytes.Contains(buf.Bytes(), []byte(want)) {
			t.Errorf("output lacks %q", want)
		}
	}
	if _, err := NewWriter(&bytes.Buffer{}).AddPNGImage([]byte("not a png")); err == nil {
		t.Error("expected error for invalid PNG")
	}
}
