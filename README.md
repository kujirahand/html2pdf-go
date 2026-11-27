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
- Start a new page: put `<page-break></page-break>` where the next page should begin, for example `<p>Page 1</p><page-break></page-break><p>Page 2</p>`. Leading, trailing, and repeated `<page-break>` elements do not create blank pages
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
