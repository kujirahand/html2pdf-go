package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/dom"
)

func layoutTable(t *testing.T, src string, opt Options) []*Page {
	t.Helper()
	doc, err := dom.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	opt.PageWidth, opt.PageHeight, opt.BaseDir = 595.28, 841.89, "../../testdata"
	pages, err := Layout(doc, css.ComputeStyles(doc, doc.StylesheetText), invoiceFonts(t), opt)
	if err != nil {
		t.Fatal(err)
	}
	return pages
}
func tableTexts(p *Page) map[string]TextItem {
	out := map[string]TextItem{}
	for _, it := range p.Items {
		if v, ok := it.(TextItem); ok {
			out[v.Text] = v
		}
	}
	return out
}
func TestRowspanGrid(t *testing.T) {
	p := layoutTable(t, `<table><tr><td rowspan="2">set</td><td>1</td><td>1000</td></tr><tr><td>2</td><td>2000</td></tr></table>`, Options{})[0]
	x := tableTexts(p)
	if x["1"].X != x["2"].X || x["1000"].X != x["2000"].X || !(x["set"].Y > x["1"].Y && x["set"].Y < x["2"].Y) {
		t.Fatalf("grid: %+v", x)
	}
	p = layoutTable(t, `<table><tr><td>A</td><td rowspan="2">B</td><td>C</td></tr><tr><td colspan="2">D</td></tr></table>`, Options{})[0]
	x = tableTexts(p)
	if x["D"].X < x["C"].X {
		t.Fatalf("overlapping colspan was not shifted: %+v", x)
	}
}
func TestRowspanGroupAndHeight(t *testing.T) {
	for _, value := range []string{"0", "-3", "abc"} {
		p := layoutTable(t, `<table><tr><td rowspan="`+value+`">A</td><td>B</td></tr><tr><td>C</td></tr></table>`, Options{})[0]
		x := tableTexts(p)
		if x["A"].X != x["C"].X {
			t.Fatalf("rowspan %s: %+v", value, x)
		}
	}
	p := layoutTable(t, `<table><thead><tr><td rowspan="99999">A</td><td>B</td></tr></thead><tbody><tr><td>C</td></tr></tbody></table>`, Options{})[0]
	x := tableTexts(p)
	if x["A"].X != x["C"].X {
		t.Fatalf("group boundary: %+v", x)
	}
	p = layoutTable(t, `<table><tr><td rowspan="2" style="background-color: red">A<br>B<br>C<br>D<br>E</td><td>X</td></tr><tr><td>Y</td></tr><tr><td>Z</td></tr></table>`, Options{})[0]
	x = tableTexts(p)
	if x["Z"].Y-x["X"].Y < 40 {
		t.Fatalf("span did not extend rows: %+v", x)
	}
	var rect bool
	for _, it := range p.Items {
		if v, ok := it.(RectItem); ok && v.Fill != nil && v.H > 40 {
			rect = true
		}
	}
	if !rect {
		t.Fatal("missing full-height cell background")
	}
}
func TestTheadRepeatAndRowspanBlocks(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<table><thead><tr><th>No</th><th>Heading</th></tr></thead><tbody>`)
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&b, `<tr><td rowspan="2">S%03d</td><td>A%03d</td></tr><tr><td>B%03d</td></tr>`, i, i, i)
	}
	b.WriteString(`</tbody></table>`)
	for _, disabled := range []bool{false, true} {
		pages := layoutTable(t, b.String(), Options{NoRepeatThead: disabled, MarginTop: 30, MarginBottom: 30})
		if len(pages) < 3 {
			t.Fatalf("pages: %d", len(pages))
		}
		seen := map[string]int{}
		last := -1
		for pi, p := range pages {
			first := ""
			texts := tableTexts(p)
			for _, it := range p.Items {
				v, ok := it.(TextItem)
				if !ok {
					continue
				}
				if first == "" {
					first = v.Text
				}
				if v.Y < 30 || v.Y > 811.89 {
					t.Fatalf("page %d y=%v", pi, v.Y)
				}
				seen[v.Text]++
				if strings.HasPrefix(v.Text, "S") {
					var n int
					fmt.Sscanf(v.Text, "S%d", &n)
					if n != last+1 {
						t.Fatalf("order %s after %d on page %d", v.Text, last, pi)
					}
					last = n
					if _, ok := texts[fmt.Sprintf("A%03d", n)]; !ok {
						t.Fatal("split group A", n)
					}
					if _, ok := texts[fmt.Sprintf("B%03d", n)]; !ok {
						t.Fatal("split group B", n)
					}
				}
			}
			if pi == 0 || !disabled {
				if first != "No" {
					t.Fatalf("page %d first %q", pi, first)
				}
			} else if _, ok := texts["No"]; ok {
				t.Fatal("unexpected repeated heading")
			}
		}
		if seen["No"] != map[bool]int{true: 1, false: len(pages)}[disabled] || last != 79 {
			t.Fatalf("head count %d last %d", seen["No"], last)
		}
		for i := 0; i < 80; i++ {
			for _, prefix := range []string{"S", "A", "B"} {
				key := fmt.Sprintf("%s%03d", prefix, i)
				if seen[key] != 1 {
					t.Fatalf("%s count %d", key, seen[key])
				}
			}
		}
	}
}
func TestTableOversizeAndColumnLimit(t *testing.T) {
	layoutTable(t, `<table><tr><td rowspan="2">`+strings.Repeat("line<br>", 100)+`</td><td>A</td></tr><tr><td>B</td></tr></table>`, Options{})
	src := `<table><tr>` + strings.Repeat(`<td colspan="1000">x</td>`, 11) + `</tr></table>`
	doc, err := dom.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Layout(doc, css.ComputeStyles(doc, doc.StylesheetText), invoiceFonts(t), Options{PageWidth: 595.28, PageHeight: 841.89})
	if err == nil || !strings.Contains(err.Error(), "too many columns") {
		t.Fatalf("error: %v", err)
	}
}

func TestRowspanEmptyRowAndNoThead(t *testing.T) {
	p := layoutTable(t, `<table><tr><td rowspan="2">A</td><td>X</td></tr><tr></tr><tr><td>C</td></tr></table>`, Options{})[0]
	x := tableTexts(p)
	if x["A"].X != x["C"].X {
		t.Fatalf("empty row was not counted: %+v", x)
	}
	var src strings.Builder
	src.WriteString(`<table>`)
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&src, `<tr><td>R%03d</td></tr>`, i)
	}
	src.WriteString(`</table>`)
	pages := layoutTable(t, src.String(), Options{MarginTop: 30, MarginBottom: 30})
	if len(pages) < 2 {
		t.Fatal("expected multiple pages")
	}
	for _, p := range pages {
		for _, it := range p.Items {
			if v, ok := it.(TextItem); ok && !strings.HasPrefix(v.Text, "R") {
				t.Fatalf("unexpected text %q", v.Text)
			}
		}
	}
}
