package layout

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
)

// layoutErr lays out src and returns the pages and error.
func layoutErr(t *testing.T, src string, opt Options) ([]*Page, error) {
	t.Helper()
	doc, err := dom.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	opt.PageWidth, opt.PageHeight = 595.28, 841.89
	return Layout(doc, css.ComputeStyles(doc, doc.StylesheetText), invoiceFonts(t), opt)
}

func nestedTables(depth int) string {
	s := "x"
	for i := 0; i < depth; i++ {
		s = "<table><tr><td>" + s + "</td><td>y</td></tr></table>"
	}
	return s
}

func TestLayoutWorkLimit(t *testing.T) {
	saved := maxLayoutWork
	maxLayoutWork = 5000
	defer func() { maxLayoutWork = saved }()
	_, err := layoutErr(t, nestedTables(20), Options{})
	if err == nil || !strings.Contains(err.Error(), "too complex") {
		t.Fatalf("err = %v, want the work limit error", err)
	}
}

func TestLayoutWorkLimitAllowsOrdinaryDocuments(t *testing.T) {
	if _, err := layoutErr(t, nestedTables(3), Options{}); err != nil {
		t.Fatal(err)
	}
}

func TestTableColumnLimit(t *testing.T) {
	row := "<tr>" + strings.Repeat(`<td colspan="1000">x</td>`, maxTableCols/1000+1) + "</tr>"
	_, err := layoutErr(t, "<table>"+row+"</table>", Options{})
	if err == nil || !strings.Contains(err.Error(), "too many columns") {
		t.Fatalf("err = %v, want the column limit error", err)
	}
}

// outsideFixture creates dir/doc (the base directory) next to a sibling
// directory holding a frame document and a PNG.
func outsideFixture(t *testing.T) (base, frame, img string) {
	t.Helper()
	root := t.TempDir()
	base = filepath.Join(root, "doc")
	if err := os.Mkdir(base, 0o755); err != nil {
		t.Fatal(err)
	}
	frame = filepath.Join(root, "secret.html")
	if err := os.WriteFile(frame, []byte("<p>secret</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	img = filepath.Join(root, "secret.png")
	f, err := os.Create(img)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return base, frame, img
}

func TestFrameConfinement(t *testing.T) {
	base, frame, _ := outsideFixture(t)
	for _, src := range []string{frame, "../secret.html"} {
		doc := fmt.Sprintf(`<iframe src="%s" width="200" height="100"></iframe>`, src)
		if _, err := layoutErr(t, doc, Options{BaseDir: base}); err == nil {
			t.Errorf("iframe src %q outside BaseDir was read by default", src)
		}
		if _, err := layoutErr(t, doc, Options{BaseDir: base, AllowExternalFiles: true}); err != nil {
			t.Errorf("iframe src %q with AllowExternalFiles: %v", src, err)
		}
	}
}

func TestImageConfinement(t *testing.T) {
	base, _, img := outsideFixture(t)
	for _, src := range []string{img, "../secret.png"} {
		doc := fmt.Sprintf(`<img src="%s">`, src)
		pages, err := layoutErr(t, doc, Options{BaseDir: base})
		if err != nil {
			t.Fatal(err)
		}
		if n := len(imageItems(pages)); n != 0 {
			t.Errorf("img src %q outside BaseDir: %d image items by default", src, n)
		}
		pages, err = layoutErr(t, doc, Options{BaseDir: base, AllowExternalFiles: true})
		if err != nil {
			t.Fatal(err)
		}
		items := imageItems(pages)
		if len(items) != 1 || items[0].Dir != base {
			t.Errorf("img src %q with AllowExternalFiles: items = %+v", src, items)
		}
	}
}

func TestImageRejectsNetworkPaths(t *testing.T) {
	for _, src := range []string{"//host/share/a.png", `\\host\share\a.png`} {
		pages, err := layoutErr(t, fmt.Sprintf(`<img src="%s">`, src), Options{AllowExternalFiles: true})
		if err != nil {
			t.Fatal(err)
		}
		if n := len(imageItems(pages)); n != 0 {
			t.Errorf("img src %q produced %d image items", src, n)
		}
	}
}
