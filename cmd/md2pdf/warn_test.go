package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWarnFlag(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "md2pdf")
	if b, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatal(err, string(b))
	}
	src := filepath.Join(dir, "input.md")
	if err := os.WriteFile(src, []byte(`<p style="font-style: italic">hello</p>`), 0600); err != nil {
		t.Fatal(err)
	}
	for _, verbose := range []bool{false, true} {
		args := []string{"-warn", "-o", filepath.Join(dir, "out.pdf")}
		if verbose {
			args = append(args, "-verbose")
		}
		args = append(args, src)
		cmd := exec.Command(bin, args...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Fatal(err, stderr.String())
		}
		if !strings.Contains(stderr.String(), "md2pdf: warning: unsupported CSS property: font-style") {
			t.Fatal(stderr.String())
		}
		if verbose && !strings.Contains(stderr.String(), "font dir [") {
			t.Fatal(stderr.String())
		}
	}
}
