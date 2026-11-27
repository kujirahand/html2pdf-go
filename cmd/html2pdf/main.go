// Command html2pdf converts an HTML file into a PDF file.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	html2pdf "github.com/kujirahand/html2pdf-go"
)

// valueFlags lists the flags that take a value (so "-o out.pdf" moves as a pair).
var valueFlags = map[string]bool{"o": true, "page-size": true, "scale": true, "margin": true, "font": true, "font-bold": true, "font-dirs": true}

// reorderArgs moves flags (and their values) in front of positional
// arguments, because the flag package stops parsing at the first one.
// Everything after a literal "--" stays positional.
func reorderArgs(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			pos = append(pos, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if !strings.Contains(name, "=") && valueFlags[name] && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(append(flags, "--"), pos...)
}

func main() {
	out := flag.String("o", "", "output PDF path (default: input with .pdf extension)")
	pageSize := flag.String("page-size", "A4", "page size, e.g. A4")
	landscape := flag.Bool("landscape", false, "landscape orientation (swap page width and height)")
	scale := flag.Float64("scale", 1.0, "content scale factor, e.g. 0.95 to shrink slightly")
	margin := flag.Float64("margin", 0, "page margin in points, applied to all sides")
	fontFile := flag.String("font", "", "regular font file (default: auto-detect a Japanese font)")
	fontBold := flag.String("font-bold", "", "bold font file (default: the regular font)")
	fontDirs := flag.String("font-dirs", "", "comma-separated directories to search for fonts (for font-family lookup)")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: html2pdf [flags] input.html")
		flag.PrintDefaults()
	}
	flag.CommandLine.Parse(reorderArgs(os.Args[1:]))

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	input := flag.Arg(0)
	output := *out
	if output == "" {
		output = strings.TrimSuffix(input, filepath.Ext(input)) + ".pdf"
	}

	var dirs []string
	for _, d := range strings.Split(*fontDirs, ",") {
		if d = strings.TrimSpace(d); d != "" {
			dirs = append(dirs, d)
		}
	}

	opts := &html2pdf.Options{
		PageSize:  *pageSize,
		Landscape: *landscape,
		BaseDir:   filepath.Dir(input),
		Scale:     *scale,
		Margin:    *margin,
		FontFile:  *fontFile,
		FontDirs:  dirs,

		FontFileBold: *fontBold,
	}
	if err := html2pdf.ConvertFile(input, output, opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
