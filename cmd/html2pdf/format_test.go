package main

import "testing"

func TestResolveFormat(t *testing.T) {
	cases := []struct {
		format, output, want string
		wantErr              bool
	}{
		{"", "a.png", "png", false},
		{"", "a.PNG", "png", false},
		{"", "a.pdf", "pdf", false},
		{"", "a", "pdf", false},
		{"", "a.txt", "pdf", false},
		{"png", "a.pdf", "png", false},
		{"pdf", "a.png", "pdf", false},
		{"gif", "a.png", "", true},
		{"gif", "a.pdf", "", true},
	}
	for _, c := range cases {
		got, err := resolveFormat(c.format, c.output)
		if c.wantErr {
			if err == nil {
				t.Errorf("resolveFormat(%q, %q) = %q, want error", c.format, c.output, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("resolveFormat(%q, %q) = %q, %v; want %q", c.format, c.output, got, err, c.want)
		}
	}
}
