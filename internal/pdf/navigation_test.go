package pdf

import (
	"bytes"
	"strings"
	"testing"
)

func TestPDFString(t *testing.T) {
	for _, tt := range []struct{ in, want string }{{"abc", "(abc)"}, {`a(b)\c`, `(a\(b\)\\c)`}, {"日本", "<FEFF65E5672C>"}, {"😀", "<FEFFD83DDE00>"}, {"a\n", "<FEFF0061000A>"}} {
		if got := pdfString(tt.in); got != tt.want {
			t.Errorf("%q: %s != %s", tt.in, got, tt.want)
		}
	}
}
func TestNavigationAndInfo(t *testing.T) {
	var b bytes.Buffer
	w := NewWriter(&b)
	w.SetInfo(Info{Title: "日本", Creator: "html2pdf-go"})
	p := w.AddPage(100, 200)
	p.SetScale(2)
	p.AddLink(2, 3, 4, 5, "https://example.org", nil, 0, 0)
	p.AddLink(2, 3, 4, 5, "", p, 10, 20)
	w.AddBookmark("First", 1, p, 10, 20)
	w.AddBookmark("Nested", 3, p, 10, 20)
	w.AddBookmark("Next", 1, p, 10, 20)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	for _, want := range []string{"/Info ", "/Title <FEFF65E5672C>", "/Outlines", "/Count 3", "/Title (Nested)", "/Prev ", "/URI (https://example.org)", "/Rect [4 184 12 194]", "/XYZ 20 160 null"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s", want)
		}
	}
}
func TestSafeURI(t *testing.T) {
	for _, tt := range []struct{ in, want string }{{" HTTP://example.org/日 ", "HTTP://example.org/%E6%97%A5"}, {"JavaScript:alert(1)", ""}, {"file:/tmp/x", ""}, {"https://x/\nfoo", ""}, {"mailto:a@example.org", "mailto:a@example.org"}} {
		if got := SafeURI(tt.in); got != tt.want {
			t.Errorf("%q: %q != %q", tt.in, got, tt.want)
		}
	}
}
