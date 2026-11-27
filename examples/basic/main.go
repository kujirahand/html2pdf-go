// Command basic converts a small HTML string to out.pdf.
package main

import (
	"bytes"
	"fmt"
	"os"

	html2pdf "github.com/kujirahand/html2pdf-go"
)

const src = `<html><body>
<h1>Hello, html2pdf-go</h1>
<p>こんにちは、PDF。</p>
</body></html>`

func main() {
	var buf bytes.Buffer
	err := html2pdf.Convert([]byte(src), &buf, &html2pdf.Options{PageSize: "A4"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile("out.pdf", buf.Bytes(), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
