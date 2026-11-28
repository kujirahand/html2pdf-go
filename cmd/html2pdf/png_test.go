package main

import (
	"reflect"
	"testing"
)

func TestPNGPaths(t *testing.T) {
	for _, c := range []struct {
		output string
		n      int
		want   []string
	}{
		{"name.pdf", 1, []string{"name.png"}},
		{"dir/name.pdf", 3, []string{"dir/name-1.png", "dir/name-2.png", "dir/name-3.png"}},
		{"name.png", 2, []string{"name-1.png", "name-2.png"}},
		{"name", 1, []string{"name.png"}},
	} {
		if got := pngPaths(c.output, c.n); !reflect.DeepEqual(got, c.want) {
			t.Errorf("pngPaths(%q, %d) = %v, want %v", c.output, c.n, got, c.want)
		}
	}
}
