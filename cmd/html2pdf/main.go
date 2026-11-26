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

func main() {
	out := flag.String("o", "", "output PDF path (default: input with .pdf extension)")
	pageSize := flag.String("page-size", "A4", "page size, e.g. A4")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: html2pdf [flags] input.html")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	input := flag.Arg(0)
	output := *out
	if output == "" {
		output = strings.TrimSuffix(input, filepath.Ext(input)) + ".pdf"
	}

	opts := &html2pdf.Options{
		PageSize: *pageSize,
		BaseDir:  filepath.Dir(input),
	}
	if err := html2pdf.ConvertFile(input, output, opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
