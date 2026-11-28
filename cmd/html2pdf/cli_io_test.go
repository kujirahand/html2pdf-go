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
	bin := buildCLI(t)
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
	source := "<p>x</p>"
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
	if _, err := os.Stat(filepath.Join(dir, "output.html")); !os.IsNotExist(err) {
		t.Fatalf("stdin wrote output.html: %v", err)
	}
	if _, _, code := run(""); code != 2 {
		t.Fatalf("missing input: %d", code)
	}
	out, stderr, code := run(source, "-verbose", "-o", "-")
	if code != 0 || !bytes.HasPrefix(out, []byte("%PDF")) || bytes.Contains(stderr, []byte("%PDF")) {
		t.Fatalf("stdout: %d %q", code, stderr)
	}
	for _, name := range []string{"a.html", "b.html"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, e, c := run("", "a.html", "b.html"); c != 0 {
		t.Fatalf("batch: %d %s", c, e)
	}
	for _, name := range []string{"a.pdf", "b.pdf"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, c := run("", "-o", "file.pdf", "a.html", "b.html"); c != 2 {
		t.Fatalf("batch output: %d", c)
	}
	if _, _, c := run("", "a.html", "-"); c != 2 {
		t.Fatalf("mixed stdin: %d", c)
	}
	if err := os.Mkdir(filepath.Join(dir, "out"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, e, c := run("", "-o", "out", "a.html", "b.html"); c != 0 {
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
	out, stderr, code = run("<p>one</p><page-break><p>two</p>", "-format", "png", "-o", "-")
	if code != 1 || len(out) != 0 || !bytes.Contains(stderr, []byte("multi-page PNG")) {
		t.Fatalf("multi png: %d %q", code, stderr)
	}
	if out, _, c := run("", "-o", "-", "missing.html"); c != 1 || len(out) != 0 {
		t.Fatalf("failed stdout: %d, %d bytes", c, len(out))
	}
	if err := os.Remove(filepath.Join(dir, "a.pdf")); err != nil {
		t.Fatal(err)
	}
	if _, e, c := run("", "missing.html", "a.html"); c != 1 || !bytes.Contains(e, []byte("missing.html")) {
		t.Fatalf("batch failure: %d %s", c, e)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.pdf")); err != nil {
		t.Fatalf("batch did not continue: %v", err)
	}
	for _, arg := range []string{"-", "a.html"} {
		if _, _, c := run("", "-e", "<p>x</p>", arg); c != 2 {
			t.Fatalf("inline conflict: %d", c)
		}
	}
	if _, e, c := run("", "-e", "<p>y</p>", "-o", "inline.pdf"); c != 0 {
		t.Fatalf("inline: %d %s", c, e)
	}
	if _, err := os.Stat(filepath.Join(dir, "output.html")); err != nil {
		t.Fatalf("-e did not write output.html: %v", err)
	}
}
