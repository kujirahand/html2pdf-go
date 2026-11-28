package css

import (
	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/warn"
	"strings"
	"testing"
)

func TestWarnings(t *testing.T) {
	doc, err := dom.Parse(strings.NewReader(`<style>@charset "utf-8"; @media print { p { nope: 1 } } @import "x"; .no { font-style: italic; display: flex } p { font-style: italic }</style><p style="font-style: italic">a</p><table><tr><td rowspan="2">b</td></tr></table>`))
	if err != nil {
		t.Fatal(err)
	}
	w := new(warn.Collector)
	sheets, err := ResolveSheets(doc, "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	ComputeStylesW(doc, sheets, w)
	got := w.Sorted()
	counts := map[string]int{}
	for _, v := range got {
		counts[v.Kind+":"+v.Key] = v.Count
	}
	for k, n := range map[string]int{"css-at-rule:@media": 1, "css-at-rule:@import": 1, "css-property:font-style": 3, "css-value:display: flex": 1, "html-attribute:td[rowspan]": 1} {
		if counts[k] != n {
			t.Errorf("%s: %+v", k, got)
		}
	}
	if len(got) != 5 {
		t.Fatalf("unexpected warnings: %+v", got)
	}
}
func TestSupportedSamples(t *testing.T) {
	for prop, check := range supported {
		sample := "1px"
		switch {
		case prop == "display":
			sample = "block"
		case prop == "font-weight":
			sample = "bold"
		case prop == "font-family":
			sample = "serif"
		case prop == "text-align":
			sample = "left"
		case strings.HasPrefix(prop, "text-decoration"):
			sample = "underline"
		case strings.Contains(prop, "color") || prop == "background":
			sample = "red"
		case strings.HasSuffix(prop, "style"):
			sample = "solid"
		case strings.HasPrefix(prop, "break-") || strings.HasPrefix(prop, "page-break-"):
			sample = "always"
		}
		if !check(sample) {
			t.Errorf("%s rejects %s", prop, sample)
		}
	}
	w := new(warn.Collector)
	parseSheetW(uaSheet, w)
	parseSheetW("p { color: red; margin: 1em; display: block }", w)
	if got := w.Sorted(); len(got) > 0 {
		t.Fatalf("%+v", got)
	}
}
