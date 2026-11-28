package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMarkdownLinksBookmarksAndFlags(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "md2pdf")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	dir := t.TempDir()
	in := filepath.Join(dir, "in.md")
	if err := os.WriteFile(in, []byte("# 見出し\n\n[site](https://example.org/)\n\n## Sub\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) []byte {
		t.Helper()
		out := filepath.Join(dir, "out.pdf")
		args = append(args, "-o", out, in)
		if b, err := exec.Command(bin, args...).CombinedOutput(); err != nil {
			t.Fatalf("run: %v\n%s", err, b)
		}
		pdf, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		return pdf
	}
	pdf := run()
	// The default title is the first h1, written as UTF-16BE.
	for _, want := range []string{"/URI (https://example.org/)", "/Outlines", "/Title <FEFF898B51FA3057>"} {
		if !bytes.Contains(pdf, []byte(want)) {
			t.Errorf("default run: missing %s", want)
		}
	}
	pdf = run("-bookmarks=false", "-title", "T", "-author", "A", "-subject", "S")
	for _, want := range []string{"/Title (T)", "/Author (A)", "/Subject (S)"} {
		if !bytes.Contains(pdf, []byte(want)) {
			t.Errorf("flag run: missing %s", want)
		}
	}
	if bytes.Contains(pdf, []byte("/Outlines")) {
		t.Error("-bookmarks=false still wrote /Outlines")
	}
}
