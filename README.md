# html2pdf-go

[日本語版 README](README-ja.md)

A pure-Go HTML-to-PDF conversion engine. HTML parsing, CSS application, layout, and PDF writing are all implemented. For the supported range of HTML/CSS, see [docs/reference.md](docs/reference.md).

## Tutorial

First, let's convert one HTML file into a PDF with the CLI. It takes just a few minutes.

### 1. Install it

```sh
go install github.com/kujirahand/html2pdf-go/cmd/html2pdf@latest
```

### 2. Create an HTML file

Create a file named `hello.html` with the following content. It is a minimal example with a heading, a paragraph, a list, a table, and styling in a `<style>` element.

```html
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<style>
  body { font-size: 12pt; margin: 48px; }
  h1 { font-size: 20pt; color: #205090; }
  table { border: 1px solid #333; }
  th, td { border: 1px solid #333; padding: 4px 8px; }
</style>
</head>
<body>
<h1>My first PDF</h1>
<p>This PDF was made with html2pdf-go. Japanese text such as こんにちは works as-is, too.</p>
<ul>
  <li>Headings and paragraphs</li>
  <li>Lists</li>
  <li>Tables</li>
</ul>
<table>
  <tr><th>Item</th><th>Value</th></tr>
  <tr><td>Page size</td><td>A4</td></tr>
  <tr><td>Font</td><td>Chosen automatically</td></tr>
</table>
</body>
</html>
```

### 3. Convert it to PDF

```sh
html2pdf hello.html
```

This creates `hello.pdf` in the same folder. Open it and you should see a PDF with a large blue heading and a table with borders.

To change the output file or the page settings, pass options:

```sh
html2pdf -o out.pdf -page-size A4 -margin 48 hello.html   # output file and margins
html2pdf -scale 0.95 hello.html                            # shrink everything to 95%
```

### 4. Next steps

- Change the appearance: edit the contents of the `<style>` element. The available HTML elements and CSS are listed in [docs/reference.md](docs/reference.md)
- Use it from a Go program: see "Using it as a library" below
- See real examples: [examples/](examples/) contains sample HTML such as an address book and a receipt

## Using it as a library

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

## Using the CLI

```sh
go install github.com/kujirahand/html2pdf-go/cmd/html2pdf@latest

html2pdf input.html                    # writes input.pdf
html2pdf -o out.pdf -page-size A4 input.html
html2pdf -scale 0.95 input.html        # shrinks the content to 95%
```

`-scale` (`Options.Scale` in the library) is the factor that shrinks or enlarges the content (default 1.0). It is the equivalent of wkhtmltopdf's zoom / smart shrinking and Chrome printToPDF's `scale`, and is useful when a document at its real size spills just slightly onto the next page and you want it to fit.

## Markdown to PDF (md2pdf)

`md2pdf` converts a Markdown file into a PDF. It turns the Markdown into HTML with [goldmark](https://github.com/yuin/goldmark) and passes it through the same engine (Markdown → HTML → PDF).

```sh
go install github.com/kujirahand/html2pdf-go/cmd/md2pdf@latest

md2pdf input.md                        # writes input.pdf
md2pdf -o out.pdf -margin 48 input.md
md2pdf -css style.css input.md         # appends your CSS after the default stylesheet
```

It accepts the same flags as `html2pdf` (`-o`, `-page-size`, `-landscape`, `-scale`, `-margin`, `-font`, `-font-bold`, `-font-dirs`) plus `-css`. Relative image paths resolve against the Markdown file's directory.

From a Go program:

```go
// Markdown bytes to PDF bytes
err := html2pdf.ConvertMarkdown(src, &buf, &html2pdf.Options{PageSize: "A4", MarkdownCSS: "h1 { color: #205090; }"})

// Markdown file to PDF file (BaseDir defaults to the Markdown file's directory)
err = html2pdf.ConvertMarkdownFile("input.md", "out.pdf", nil)

// Just the HTML document, with the default stylesheet embedded
htmlDoc, err := html2pdf.MarkdownToHTML(src)
```

Supported Markdown: headings, paragraphs, bold, lists (including nested ones), task lists, blockquotes, fenced and indented code blocks, GFM tables, horizontal rules, links, images, and raw HTML.

Limitations:

- There is no italic face, so emphasis is drawn as regular text.
- Code blocks keep their line breaks and indentation (through a transform of `<pre>` blocks), but they are not set in a monospace font.
- Raw HTML in the Markdown passes through unchanged, so only convert input you trust.

See [docs/markdown.md](docs/markdown.md) for details.

## Development

Development tasks use the [just](https://github.com/casey/just) task runner. Running `just` with no arguments lists the recipes.

```sh
just test      # go test ./...
just fonts     # downloads the IPAex fonts into fonts/ (does nothing if already downloaded)
just samples   # generates the sample PDFs into out/
```

## Supported range and roadmap

See [docs/architecture.md](docs/architecture.md).

For the list of supported HTML elements and CSS, see [docs/reference.md](docs/reference.md) (Japanese: [docs/reference-ja.md](docs/reference-ja.md)).

## License

MIT License. See the `LICENSE` file for details.
