package main

import (
	html2pdf "github.com/kujirahand/html2pdf-go"
	"reflect"
	"testing"
)

func TestItalicFlags(t *testing.T) {
	got := reorderArgs([]string{"input", "-font-italic", "italic.ttf", "-font-bold-italic", "bolditalic.ttf"})
	want := []string{"-font-italic", "italic.ttf", "-font-bold-italic", "bolditalic.ttf", "--", "input"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	for _, tc := range []struct {
		u    html2pdf.FontUse
		want string
	}{{html2pdf.FontUse{Italic: true, Path: "i"}, "font: (default) (italic) -> i"}, {html2pdf.FontUse{Italic: true, Bold: true, Path: "bi"}, "font: (default) (bold italic) -> bi"}} {
		if got := fontUseLine(tc.u); got != tc.want {
			t.Errorf("got %q want %q", got, tc.want)
		}
	}
}
