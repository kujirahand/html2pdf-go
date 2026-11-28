package main

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// buildCLI compiles the CLI into a temp dir and returns the binary path.
func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "html2pdf")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

var mediaBoxRe = regexp.MustCompile(`/MediaBox\s*\[\s*0\s+0\s+([0-9.]+)\s+([0-9.]+)\s*\]`)

func TestPageWidthHeightFlags(t *testing.T) {
	bin := buildCLI(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.html")
	out := filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(in, []byte("<p>card</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if b, err := exec.Command(bin, "-page-width", "91", "-page-height", "55", "-o", out, in).CombinedOutput(); err != nil {
		t.Fatalf("run: %v\n%s", err, b)
	}
	pdf, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	m := mediaBoxRe.FindSubmatch(pdf)
	if m == nil {
		t.Fatal("MediaBox not found")
	}
	w, _ := strconv.ParseFloat(string(m[1]), 64)
	h, _ := strconv.ParseFloat(string(m[2]), 64)
	t.Logf("MediaBox: %v x %v", w, h)
	if math.Abs(w-257.95) > 1 || math.Abs(h-155.91) > 1 {
		t.Errorf("MediaBox = %v x %v, want about 257.95 x 155.91", w, h)
	}
}

func TestNegativePageWidthFails(t *testing.T) {
	bin := buildCLI(t)
	dir := t.TempDir()
	in := filepath.Join(dir, "in.html")
	if err := os.WriteFile(in, []byte("<p>x</p>"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := exec.Command(bin, "-page-width", "-5", "-o", filepath.Join(dir, "o.pdf"), in).Run()
	ee, ok := err.(*exec.ExitError)
	if !ok || ee.ExitCode() != 2 {
		t.Fatalf("want exit 2, got %v", err)
	}
}
