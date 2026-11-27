// Command md2pdf converts a Markdown file into a PDF file.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	html2pdf "github.com/kujirahand/html2pdf-go"
	"github.com/kujirahand/html2pdf-go/internal/fontcfg"
)

// version is stamped at build time with -ldflags "-X main.version=...".
var version = "dev"

// fontUseLine formats a resolved font for -verbose output.
func fontUseLine(u html2pdf.FontUse) string {
	family := u.Family
	if family == "" {
		family = "(default)"
	}
	if u.Bold {
		family += " (bold)"
	}
	return "font: " + family + " -> " + u.Path
}

// valueFlags lists the flags that take a value (so "-o out.pdf" moves as a pair).
var valueFlags = map[string]bool{"o": true, "page-size": true, "scale": true, "margin": true, "font": true, "font-bold": true, "font-dirs": true, "css": true}

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
	fontDirs := flag.String("font-dirs", "", "comma-separated directories searched first for fonts")
	verbose := flag.Bool("verbose", false, "print the font directories searched and the fonts used to stderr")
	cssFile := flag.String("css", "", "CSS file appended after the default Markdown stylesheet")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: md2pdf [flags] input.md")
		flag.PrintDefaults()
	}
	flag.CommandLine.Parse(reorderArgs(os.Args[1:]))

	if *showVersion {
		fmt.Println("md2pdf " + version)
		return
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	input := flag.Arg(0)
	output := *out
	if output == "" {
		output = strings.TrimSuffix(input, filepath.Ext(input)) + ".pdf"
	}

	dirs := fontcfg.Gather(*fontDirs)
	var uses []html2pdf.FontUse
	if *verbose {
		for _, d := range dirs {
			fmt.Fprintf(os.Stderr, "font dir [%s]: %s\n", d.Origin, d.Path)
		}
	}

	var extraCSS string
	if *cssFile != "" {
		b, err := os.ReadFile(*cssFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		extraCSS = string(b)
	}

	opts := &html2pdf.Options{
		PageSize:  *pageSize,
		Landscape: *landscape,
		BaseDir:   filepath.Dir(input),
		Scale:     *scale,
		Margin:    *margin,
		FontFile:  *fontFile,
		FontDirs:  fontcfg.Paths(dirs),

		FontFileBold: *fontBold,
		MarkdownCSS:  extraCSS,
	}
	if *verbose {
		opts.OnFontUse = func(u html2pdf.FontUse) { uses = append(uses, u) }
	}
	err := html2pdf.ConvertMarkdownFile(input, output, opts)
	if *verbose {
		for _, u := range uses {
			fmt.Fprintln(os.Stderr, fontUseLine(u))
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
