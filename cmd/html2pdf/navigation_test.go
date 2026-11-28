package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestMetadataAndBookmarkFlags(t *testing.T) {
	bin := buildCLI(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.html")
	html := `<html><head><title>From HTML</title><meta name="author" content="Meta Author"></head><body><h1>Chapter</h1><p><a href="https://example.org/">x</a></p></body></html>`
	if err := os.WriteFile(in, []byte(html), 0o644); err != nil {
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
	for _, want := range []string{"/Title (From HTML)", "/Author (Meta Author)", "/Outlines", "/URI (https://example.org/)"} {
		if !bytes.Contains(pdf, []byte(want)) {
			t.Errorf("default run: missing %s", want)
		}
	}
	pdf = run("-bookmarks=false", "-title", "Flag Title", "-author", "Flag Author", "-subject", "Flag Subject")
	for _, want := range []string{"/Title (Flag Title)", "/Author (Flag Author)", "/Subject (Flag Subject)"} {
		if !bytes.Contains(pdf, []byte(want)) {
			t.Errorf("flag run: missing %s", want)
		}
	}
	if bytes.Contains(pdf, []byte("/Outlines")) {
		t.Error("-bookmarks=false still wrote /Outlines")
	}
}
