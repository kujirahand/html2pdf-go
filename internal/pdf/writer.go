package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Writer assembles a PDF document in memory and writes it on Close.
// All coordinates in the API are in points with the origin at the top left.
type Writer struct {
	cw           *countWriter
	offsets      []int64 // byte offset per object number; index 0 is the free entry
	pagesID      int
	fonts        []*Font
	pages        []*Page
	images       []*Image
	info         Info
	outlines     []*outline
	outlineStack []*outline
	closed       bool
}

type countWriter struct {
	w   io.Writer
	n   int64
	err error
}

func (c *countWriter) Write(p []byte) (int, error) {
	if c.err != nil {
		return 0, c.err
	}
	n, err := c.w.Write(p)
	c.n += int64(n)
	c.err = err
	return n, err
}

// NewWriter returns a Writer that outputs the document to w on Close.
func NewWriter(w io.Writer) *Writer {
	return &Writer{cw: &countWriter{w: w}, offsets: []int64{0}}
}

func (w *Writer) alloc() int {
	w.offsets = append(w.offsets, -1)
	return len(w.offsets) - 1
}

func (w *Writer) obj(id int, body string) {
	w.offsets[id] = w.cw.n
	fmt.Fprintf(w.cw, "%d 0 obj\n%s\nendobj\n", id, body)
}

// stream writes a stream object; dict holds extra dictionary entries.
func (w *Writer) stream(id int, dict string, data []byte, compress bool) {
	if compress {
		var b bytes.Buffer
		zw := zlib.NewWriter(&b)
		zw.Write(data)
		zw.Close()
		data = b.Bytes()
		dict += " /Filter /FlateDecode"
	}
	w.offsets[id] = w.cw.n
	fmt.Fprintf(w.cw, "%d 0 obj\n<< %s /Length %d >>\nstream\n", id, dict, len(data))
	w.cw.Write(data)
	io.WriteString(w.cw, "\nendstream\nendobj\n")
}

// Close writes all objects, the xref table and the trailer.
func (w *Writer) Close() error {
	if w.closed {
		return w.cw.err
	}
	w.closed = true
	if len(w.pages) == 0 {
		return fmt.Errorf("pdf: document has no pages")
	}

	infoID := w.alloc()
	catalogID := w.alloc()
	w.pagesID = w.alloc()
	for _, f := range w.fonts {
		for _, c := range f.chunks {
			c.objID = w.alloc()
		}
	}
	for _, im := range w.images {
		im.objID = w.alloc()
		if im.alpha != nil {
			im.smaskID = w.alloc()
		}
	}
	for _, p := range w.pages {
		p.objID = w.alloc()
		p.contentID = w.alloc()
	}

	outlineID := w.allocNavigation()
	io.WriteString(w.cw, "%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
	w.writeInfo(infoID)
	catalog := fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R", w.pagesID)
	if outlineID != 0 {
		catalog += fmt.Sprintf(" /Outlines %d 0 R", outlineID)
	}
	w.obj(catalogID, catalog+" >>")
	var kids strings.Builder
	for _, p := range w.pages {
		fmt.Fprintf(&kids, " %d 0 R", p.objID)
	}
	w.obj(w.pagesID, fmt.Sprintf("<< /Type /Pages /Kids [%s ] /Count %d >>", kids.String(), len(w.pages)))
	for _, f := range w.fonts {
		for _, c := range f.chunks {
			w.writeFontChunk(c)
		}
	}
	for _, im := range w.images {
		w.writeImage(im)
	}
	for _, p := range w.pages {
		w.writePage(p)
	}

	w.writeNavigation(outlineID)
	xrefPos := w.cw.n
	fmt.Fprintf(w.cw, "xref\n0 %d\n", len(w.offsets))
	io.WriteString(w.cw, "0000000000 65535 f \n")
	for _, off := range w.offsets[1:] {
		fmt.Fprintf(w.cw, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(w.cw, "trailer\n<< /Size %d /Root %d 0 R /Info %d 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(w.offsets), catalogID, infoID, xrefPos)
	if w.cw.err != nil {
		return fmt.Errorf("pdf: write document: %w", w.cw.err)
	}
	return nil
}

// numP formats v with at most prec decimals, trimming trailing zeros.
func numP(v float64, prec int) string {
	s := strconv.FormatFloat(v, 'f', prec, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	if s == "-0" {
		s = "0"
	}
	return s
}

func num(v float64) string { return numP(v, 4) }
