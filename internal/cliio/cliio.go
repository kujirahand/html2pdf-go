// Package cliio plans CLI input and output paths for html2pdf and md2pdf.
package cliio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoInput is returned when there is no -e flag, no positional input,
// and stdin is a terminal (or otherwise not a pipe).
var ErrNoInput = errors.New("no input specified")

// Job is one conversion: a file, stdin, or html2pdf -e source, and its output path.
type Job struct {
	Input   string // positional path; empty for implicit stdin or -e
	BaseDir string
	Output  string // file path, or "-" for stdout
	Stdin   bool
	Inline  bool // html2pdf -e
}

// StdinPipe reports whether stdin is a pipe or redirected file, not a terminal.
func StdinPipe() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice == 0
}

// Plan builds conversion jobs from the CLI inputs.
// inline is true when html2pdf -e was given. args are the positional inputs.
// output is the -o value (empty if omitted). pipe is whether stdin is not a TTY.
// Combining -e, stdin, and files is an error. Multiple inputs with a
// non-directory -o is an error.
func Plan(inline bool, args []string, output string, pipe bool) ([]Job, error) {
	if inline && len(args) != 0 {
		return nil, fmt.Errorf("-e cannot be combined with input files or stdin")
	}
	for _, arg := range args {
		if arg == "-" && len(args) != 1 {
			return nil, fmt.Errorf("stdin cannot be combined with input files")
		}
	}
	if !inline && len(args) == 0 && !pipe {
		return nil, ErrNoInput
	}
	isDir := false
	if output != "" && output != "-" {
		info, err := os.Stat(output)
		isDir = err == nil && info.IsDir()
	}
	if len(args) > 1 && output != "" && !isDir {
		return nil, fmt.Errorf("-o must be an existing directory with multiple inputs")
	}
	if inline || len(args) == 0 {
		args = []string{""}
	}
	jobs := make([]Job, 0, len(args))
	for _, input := range args {
		j := Job{Input: input, Inline: inline, Stdin: !inline && (input == "" || input == "-"), BaseDir: "."}
		name := "output.pdf"
		if !j.Stdin && !j.Inline {
			j.BaseDir = filepath.Dir(input)
			name = strings.TrimSuffix(input, filepath.Ext(input)) + ".pdf"
		}
		switch {
		case output == "":
			j.Output = name
		case isDir:
			j.Output = filepath.Join(output, filepath.Base(name))
		default:
			j.Output = output
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}
