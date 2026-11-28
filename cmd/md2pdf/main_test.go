package main

import (
	"reflect"
	"testing"
)

func TestReorderHeaderFooter(t *testing.T) {
	got := reorderArgs([]string{"input.md", "-header", "{title}", "-footer", "{page} / {pages}"})
	want := []string{"-header", "{title}", "-footer", "{page} / {pages}", "--", "input.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q; want %q", got, want)
	}
}
