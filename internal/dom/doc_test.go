package dom

import (
	"strings"
	"testing"
)

func TestSheetsDocumentOrder(t *testing.T) {
	doc, err := Parse(strings.NewReader(`<html><head>
<style>a{}</style>
<link rel="stylesheet" href="one.css">
<link rel="alternate stylesheet" href="alt.css">
<link rel="icon" href="icon.png">
<link rel="stylesheet">
<style>b{}</style>
<LINK REL="Stylesheet" HREF="two.css">
</head><body></body></html>`))
	if err != nil {
		t.Fatal(err)
	}
	want := []SheetRef{{Text: "a{}"}, {Href: "one.css"}, {Text: "b{}"}, {Href: "two.css"}}
	if len(doc.Sheets) != len(want) {
		t.Fatalf("Sheets = %+v, want %+v", doc.Sheets, want)
	}
	for i, w := range want {
		if doc.Sheets[i] != w {
			t.Errorf("Sheets[%d] = %+v, want %+v", i, doc.Sheets[i], w)
		}
	}
	if len(doc.StylesheetText) != 2 {
		t.Errorf("StylesheetText = %v, want the two <style> texts", doc.StylesheetText)
	}
}

func TestFrameset(t *testing.T) {
	doc, err := Parse(strings.NewReader(`<frameset cols="50%,50%"><frame src="a.html"><frame src="b.html"></frameset>`))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Frameset == nil {
		t.Fatal("Frameset is nil")
	}
	if got := doc.Frameset.Attr["cols"]; got != "50%,50%" {
		t.Errorf("cols = %q", got)
	}
	doc, _ = Parse(strings.NewReader(`<p>hi</p>`))
	if doc.Frameset != nil {
		t.Error("Frameset set for a normal document")
	}
}
