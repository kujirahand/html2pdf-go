// Package fontcfg collects the font directories a command-line tool searches.
// The library itself never reads the environment; the CLIs use this package
// to build Options.FontDirs.
package fontcfg

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/text"
)

// EnvVar is the environment variable holding extra font directories, in the
// operating system's path list format.
const EnvVar = "HTML2PDF_FONT_DIRS"

// Dir is a font directory and where it came from.
type Dir struct {
	Path   string
	Origin string // "flag", "env" or "os"
}

// Gather returns the font directories in search order: the comma-separated
// flagDirs, then the HTML2PDF_FONT_DIRS environment variable, then the
// operating system's font directories. Empty entries are dropped and a
// directory that appears more than once (compared after filepath.Clean, so a
// trailing slash or "./" makes no difference) keeps its first occurrence.
func Gather(flagDirs string) []Dir {
	var dirs []Dir
	seen := map[string]bool{}
	add := func(origin string, paths []string) {
		for _, p := range paths {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			p = filepath.Clean(p)
			if seen[p] {
				continue
			}
			seen[p] = true
			dirs = append(dirs, Dir{Path: p, Origin: origin})
		}
	}
	add("flag", strings.Split(flagDirs, ","))
	add("env", filepath.SplitList(os.Getenv(EnvVar)))
	add("os", text.DefaultFontDirs())
	return dirs
}

// Paths returns the paths of dirs, in order.
func Paths(dirs []Dir) []string {
	paths := make([]string, len(dirs))
	for i, d := range dirs {
		paths[i] = d.Path
	}
	return paths
}
