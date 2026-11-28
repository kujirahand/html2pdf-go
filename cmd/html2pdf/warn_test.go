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
	bin := buildCLI(t)
	dir := t.TempDir()
	src := filepath.Join(dir, "input.html")
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
		if !strings.Contains(stderr.String(), "html2pdf: warning: unsupported CSS property: font-style") {
			t.Fatal(stderr.String())
		}
		if verbose && !strings.Contains(stderr.String(), "font dir [") {
			t.Fatal(stderr.String())
		}
	}
}
