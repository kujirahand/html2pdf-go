# Using html2pdf-go as a Go library

[日本語版](library-ja.md)

## 1. Getting started

```sh
go get github.com/kujirahand/html2pdf-go
```

```go
package main

import (
	"bytes"
	"log"
	"os"

	html2pdf "github.com/kujirahand/html2pdf-go"
)

func main() {
	var buf bytes.Buffer
	src := []byte("<h1>Hello</h1><p>こんにちは</p>")
	if err := html2pdf.Convert(src, &buf, &html2pdf.Options{PageSize: "A4"}); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("out.pdf", buf.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
}
```

For the full list of `Options` fields, see [reference.md](reference.md).

## 2. Converting HTML

`html2pdf.Convert` takes HTML bytes and writes the PDF to an `io.Writer`. To get PNG images instead of a PDF, use `html2pdf.ConvertPNG(src, opts, scale)`, which returns one PNG per page. Each page is rasterized at `scale` times the 72-dpi point size (a value of 0 or less means 2, i.e. 144 dpi).

## 3. Converting Markdown

`ConvertMarkdown` turns Markdown into HTML with [goldmark](https://github.com/yuin/goldmark) and passes it through the same engine (Markdown → HTML → PDF). `ConvertMarkdownPNG(src, opts, scale)` is the PNG counterpart and returns one PNG per page.

```go
// Markdown bytes to PDF bytes
err := html2pdf.ConvertMarkdown(src, &buf, &html2pdf.Options{PageSize: "A4", MarkdownCSS: "h1 { color: #205090; }"})

// Markdown file to PDF file (BaseDir defaults to the Markdown file's directory)
err = html2pdf.ConvertMarkdownFile("input.md", "out.pdf", nil)

// Just the HTML document, with the default stylesheet embedded
htmlDoc, err := html2pdf.MarkdownToHTML(src)
```

`Options.MarkdownCSS` is appended after the default stylesheet. Relative image paths resolve against the Markdown file's directory. For the supported Markdown and its limitations, see [markdown.md](markdown.md).

## 4. Options worth knowing

- `BaseDir` and `AllowExternalFiles`: by default a document can only read files (images, stylesheets, frame documents) from inside its own directory (`BaseDir`). Absolute paths or `..` pointing outside it, and network paths such as `//host/share`, are refused. Setting `Options.AllowExternalFiles` lifts this for documents you trust.
- `Scale`: the factor that shrinks or enlarges the content (default 1.0). It is the equivalent of wkhtmltopdf's zoom / smart shrinking and Chrome printToPDF's `scale`, and is useful when a document at its real size spills just slightly onto the next page and you want it to fit.
