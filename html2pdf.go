// Package html2pdf converts HTML+CSS documents into PDF files. It is a pure-Go HTML-to-PDF rendering engine under development.
package html2pdf

import (
	"errors"
	"io"
	"os"
)

// Options configures a conversion.
type Options struct {
	// PageSize is a named page size such as "A4".
	PageSize string
	// PageWidth and PageHeight are in points; 0 means derive from PageSize.
	PageWidth  float64
	PageHeight float64
	// BaseDir is used to resolve relative paths such as images.
	BaseDir string
	// FontDirs lists extra directories to search for fonts.
	FontDirs []string
}

// ErrNotImplemented is returned while the conversion pipeline is unfinished.
var ErrNotImplemented = errors.New("html2pdf: not implemented yet")

// Convert renders the HTML document src as a PDF and writes it to w.
func Convert(src []byte, w io.Writer, opts *Options) error {
	return ErrNotImplemented
}

// ConvertFile reads the HTML file at htmlPath and writes the PDF to pdfPath.
func ConvertFile(htmlPath, pdfPath string, opts *Options) error {
	src, err := os.ReadFile(htmlPath)
	if err != nil {
		return err
	}
	// TODO: create pdfPath and write the output once Convert is implemented.
	return Convert(src, io.Discard, opts)
}
