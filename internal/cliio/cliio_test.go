package cliio

import (
	"path/filepath"
	"testing"
)

func TestPlan(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		name   string
		inline bool
		args   []string
		out    string
		pipe   bool
		want   []Job
		bad    bool
	}{
		{name: "missing", bad: true},
		{name: "implicit stdin", pipe: true, want: []Job{{BaseDir: ".", Output: "output.pdf", Stdin: true}}},
		{name: "explicit stdin", args: []string{"-"}, out: "-", want: []Job{{Input: "-", BaseDir: ".", Output: "-", Stdin: true}}},
		{name: "inline ignores pipe", inline: true, pipe: true, out: dir, want: []Job{{BaseDir: ".", Output: filepath.Join(dir, "output.pdf"), Inline: true}}},
		{name: "file ignores pipe", args: []string{"sub/a.html"}, pipe: true, want: []Job{{Input: "sub/a.html", BaseDir: "sub", Output: "sub/a.pdf"}}},
		{name: "directory", args: []string{"sub/a.html", "b.md"}, out: dir, want: []Job{{Input: "sub/a.html", BaseDir: "sub", Output: filepath.Join(dir, "a.pdf")}, {Input: "b.md", BaseDir: ".", Output: filepath.Join(dir, "b.pdf")}}},
		{name: "inline file", inline: true, args: []string{"a"}, bad: true},
		{name: "inline stdin", inline: true, args: []string{"-"}, bad: true},
		{name: "file stdin", args: []string{"a", "-"}, bad: true},
		{name: "batch file output", args: []string{"a", "b"}, out: "out.pdf", bad: true},
		{name: "batch stdout", args: []string{"a", "b"}, out: "-", bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Plan(tc.inline, tc.args, tc.out, tc.pipe)
			if (err != nil) != tc.bad {
				t.Fatalf("error = %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("jobs = %#v", got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("job = %#v, want %#v", got[i], tc.want[i])
				}
			}
		})
	}
}
