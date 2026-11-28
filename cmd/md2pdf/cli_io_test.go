package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInputOutputAndBatch(t *testing.T) {
	bin := buildBatchCLI(t)
	dir := t.TempDir()
	run := func(input string, args ...string) ([]byte, []byte, int) {
		t.Helper()
		c := exec.Command(bin, args...)
		c.Dir = dir
		if input != "" {
			c.Stdin = strings.NewReader(input)
		}
		var out, stderr bytes.Buffer
		c.Stdout = &out
		c.Stderr = &stderr
		err := c.Run()
		code := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				code = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		return out.Bytes(), stderr.Bytes(), code
	}
	source := "# Hi"
	for _, args := range [][]string{{"-o", "explicit.pdf", "-"}, {"-o", "implicit.pdf"}} {
		_, e, code := run(source, args...)
		if code != 0 {
			t.Fatalf("stdin: %d %s", code, e)
		}
	}
	for _, name := range []string{"explicit.pdf", "implicit.pdf"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !bytes.HasPrefix(b, []byte("%PDF")) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, _, code := run(""); code != 2 {
		t.Fatalf("missing input: %d", code)
	}
	out, stderr, code := run(source, "-verbose", "-o", "-")
	if code != 0 || !bytes.HasPrefix(out, []byte("%PDF")) || bytes.Contains(stderr, []byte("%PDF")) {
		t.Fatalf("stdout: %d %q", code, stderr)
	}
	for _, name := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, e, c := run("", "a.md", "b.md"); c != 0 {
		t.Fatalf("batch: %d %s", c, e)
	}
	for _, name := range []string{"a.pdf", "b.pdf"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, c := run("", "-o", "file.pdf", "a.md", "b.md"); c != 2 {
		t.Fatalf("batch output: %d", c)
	}
	if _, _, c := run("", "a.md", "-"); c != 2 {
		t.Fatalf("mixed stdin: %d", c)
	}
	if err := os.Mkdir(filepath.Join(dir, "out"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, e, c := run("", "-o", "out", "a.md", "b.md"); c != 0 {
		t.Fatalf("batch directory: %d %s", c, e)
	}
	for _, name := range []string{"a.pdf", "b.pdf"} {
		if _, err := os.Stat(filepath.Join(dir, "out", name)); err != nil {
			t.Fatal(err)
		}
	}
	out, stderr, code = run(source, "-format", "png", "-o", "-")
	if code != 0 || !bytes.HasPrefix(out, []byte("\x89PNG")) {
		t.Fatalf("png: %d %s", code, stderr)
	}
	out, stderr, code = run("one\n\n<page-break></page-break>\n\ntwo", "-format", "png", "-o", "-")
	if code != 1 || len(out) != 0 || !bytes.Contains(stderr, []byte("multi-page PNG")) {
		t.Fatalf("multi png: %d %q", code, stderr)
	}
	if out, _, c := run("", "-o", "-", "missing.md"); c != 1 || len(out) != 0 {
		t.Fatalf("failed stdout: %d, %d bytes", c, len(out))
	}
	if err := os.Remove(filepath.Join(dir, "a.pdf")); err != nil {
		t.Fatal(err)
	}
	if _, e, c := run("", "missing.md", "a.md"); c != 1 || !bytes.Contains(e, []byte("missing.md")) {
		t.Fatalf("batch failure: %d %s", c, e)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.pdf")); err != nil {
		t.Fatalf("batch did not continue: %v", err)
	}
}

func buildBatchCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "md2pdf")
	if b, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, b)
	}
	return bin
}
