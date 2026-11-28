// Command html2pdf converts an HTML file into a PDF file.
package main

import (
	"bytes"
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
var valueFlags = map[string]bool{"e": true, "o": true, "page-size": true, "page-width": true, "page-height": true, "scale": true, "margin": true, "font": true, "font-bold": true, "font-dirs": true, "format": true, "png-scale": true, "header": true, "footer": true}

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

// convertString converts HTML source held in memory and writes the PDF to
// path only on success, like ConvertFile.
func convertString(src, path string, opts *html2pdf.Options) error {
	var buf bytes.Buffer
	if err := html2pdf.Convert([]byte(src), &buf, opts); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// pngPaths returns the file names for n PNG pages: output with its extension
// replaced by .png for a single page, or with -1, -2, ... inserted before
// .png for several.
func pngPaths(output string, n int) []string {
	base := strings.TrimSuffix(output, filepath.Ext(output))
	if n == 1 {
		return []string{base + ".png"}
	}
	paths := make([]string, n)
	for i := range paths {
		paths[i] = fmt.Sprintf("%s-%d.png", base, i+1)
	}
	return paths
}

// writePNGs writes the pages to the files named by pngPaths.
func writePNGs(output string, pages [][]byte) error {
	for i, path := range pngPaths(output, len(pages)) {
		if err := os.WriteFile(path, pages[i], 0o644); err != nil {
			return err
		}
	}
	return nil
}

// resolveFormat decides the output format: an explicit -format wins;
// without one, a .png output name selects PNG and anything else PDF.
func resolveFormat(format, output string) (string, error) {
	switch format {
	case "pdf", "png":
		return format, nil
	case "":
		if strings.EqualFold(filepath.Ext(output), ".png") {
			return "png", nil
		}
		return "pdf", nil
	default:
		return "", fmt.Errorf("unknown -format %q (want pdf or png)", format)
	}
}

// mmToPt converts millimetres to PDF points.
func mmToPt(mm float64) float64 { return mm * 72 / 25.4 }

func main() {
	out := flag.String("o", "", "output path (default: input with .pdf extension, or output.pdf with -e; a .png extension selects PNG output unless -format is given)")
	format := flag.String("format", "", "output format: pdf or png (default: png when the -o name ends in .png, otherwise pdf; png writes one file per page, name-1.png, name-2.png, ...)")
	pngScale := flag.Float64("png-scale", 2, "rasterization scale for -format png (2 = 144 dpi)")
	pageSize := flag.String("page-size", "A4", "page size, e.g. A4")
	pageWidth := flag.Float64("page-width", 0, "page width in millimetres (overrides -page-size width; 0 keeps the named size)")
	pageHeight := flag.Float64("page-height", 0, "page height in millimetres (overrides -page-size height; 0 keeps the named size)")
	landscape := flag.Bool("landscape", false, "landscape orientation (swap page width and height)")
	scale := flag.Float64("scale", 1.0, "content scale factor, e.g. 0.95 to shrink slightly")
	margin := flag.Float64("margin", 0, "page margin in points, applied to all sides")
	header := flag.String("header", "", "centered header template")
	footer := flag.String("footer", "", "centered footer template")
	fontFile := flag.String("font", "", "regular font file (default: auto-detect a Japanese font)")
	fontBold := flag.String("font-bold", "", "bold font file (default: the regular font)")
	fontDirs := flag.String("font-dirs", "", "comma-separated directories searched first for fonts")
	verbose := flag.Bool("verbose", false, "print the font directories searched and the fonts used to stderr")
	allowExternal := flag.Bool("allow-external-files", false, "let the document read files outside its own directory (absolute paths, ..); for trusted documents only")
	showVersion := flag.Bool("version", false, "print version and exit")
	htmlSrc := flag.String("e", "", "HTML source given directly instead of an input file; it is also written to output.html")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: html2pdf [flags] input.html")
		fmt.Fprintln(os.Stderr, "       html2pdf [flags] -e '<html>'")
		flag.PrintDefaults()
	}
	flag.CommandLine.Parse(reorderArgs(os.Args[1:]))

	if *showVersion {
		fmt.Println("html2pdf " + version)
		return
	}

	if *format != "" && *format != "pdf" && *format != "png" {
		fmt.Fprintf(os.Stderr, "html2pdf: unknown -format %q (want pdf or png)\n", *format)
		os.Exit(2)
	}

	// -e counts as given even with an empty value, so presence is what
	// selects inline mode, not the content.
	inline := false
	flag.CommandLine.Visit(func(f *flag.Flag) {
		if f.Name == "e" {
			inline = true
		}
	})
	if inline && flag.NArg() > 0 || !inline && flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	var input, baseDir string
	output := *out
	if inline {
		// Keep a copy of the source for debugging; conversion uses the string itself.
		input = "output.html"
		baseDir = "."
		if err := os.WriteFile(input, []byte(*htmlSrc), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if output == "" {
			output = "output.pdf"
		}
	} else {
		input = flag.Arg(0)
		baseDir = filepath.Dir(input)
		if output == "" {
			output = strings.TrimSuffix(input, filepath.Ext(input)) + ".pdf"
		}
	}

	if *pageWidth < 0 || *pageHeight < 0 {
		fmt.Fprintln(os.Stderr, "html2pdf: -page-width and -page-height must be positive")
		os.Exit(2)
	}

	dirs := fontcfg.Gather(*fontDirs)
	var uses []html2pdf.FontUse
	if *verbose {
		for _, d := range dirs {
			fmt.Fprintf(os.Stderr, "font dir [%s]: %s\n", d.Origin, d.Path)
		}
	}

	opts := &html2pdf.Options{
		PageSize:  *pageSize,
		Landscape: *landscape,
		BaseDir:   baseDir,
		Scale:     *scale,

		AllowExternalFiles: *allowExternal,
		Margin:             *margin,
		Header:             *header,
		Footer:             *footer,
		FontFile:           *fontFile,
		FontDirs:           fontcfg.Paths(dirs),

		FontFileBold: *fontBold,
	}
	if !inline {
		opts.FileName = filepath.Base(input)
	}
	if *pageWidth > 0 {
		opts.PageWidth = mmToPt(*pageWidth)
	}
	if *pageHeight > 0 {
		opts.PageHeight = mmToPt(*pageHeight)
	}
	if *verbose {
		opts.OnFontUse = func(u html2pdf.FontUse) { uses = append(uses, u) }
	}
	outFormat, ferr := resolveFormat(*format, output)
	if ferr != nil {
		fmt.Fprintf(os.Stderr, "html2pdf: %v\n", ferr)
		os.Exit(2)
	}
	var err error
	if outFormat == "png" {
		src := []byte(*htmlSrc)
		if !inline {
			src, err = os.ReadFile(input)
		}
		if err == nil {
			var pages [][]byte
			if pages, err = html2pdf.ConvertPNG(src, opts, *pngScale); err == nil {
				err = writePNGs(output, pages)
			}
		}
	} else if inline {
		err = convertString(*htmlSrc, output, opts)
	} else {
		err = html2pdf.ConvertFile(input, output, opts)
	}
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
