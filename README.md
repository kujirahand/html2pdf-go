![html2pdf-go — HTML/CSS/Markdown to PDF](docs/html2pdf-go-logo800.png)

# html2pdf-go

[日本語版 README](README-ja.md)

A pure-Go HTML-to-PDF conversion engine. HTML parsing, CSS application, layout, and PDF writing are all implemented. For the supported range of HTML/CSS, see [docs/reference.md](docs/reference.md).

## Tutorial

First, let's convert one HTML file into a PDF with the CLI. It takes just a few minutes.

### 1. Install it

Prebuilt binaries for Linux, macOS and Windows are on the [GitHub Releases page](https://github.com/kujirahand/html2pdf-go/releases). To install the latest release from the command line:

```sh
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/kujirahand/html2pdf-go/main/scripts/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/kujirahand/html2pdf-go/main/scripts/install.ps1 | iex
```

The binaries are unsigned, so downloading them in a browser makes the OS block them (Gatekeeper / SmartScreen); installing with these scripts avoids that, and they verify the SHA-256 checksum before installing. Or, with Go:

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
- See real examples: [examples/](examples/) contains sample HTML and Markdown documents — an address book, a receipt, an event flyer, a concert ticket, a cafe menu, a business card, a calendar, a boarding pass, meeting minutes, a recipe and more, in Japanese and English. Run `just examples` to convert them all into `out/`

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
html2pdf -page-width 91 -page-height 55 input.html   # custom page size in millimetres
html2pdf -scale 0.95 input.html        # shrinks the content to 95%
html2pdf -e '<h1>Hello</h1>'           # HTML given directly; writes output.html and output.pdf
```

`-e <HTML>` converts HTML given directly on the command line, which is handy for debugging. The source is also written to `output.html` in the current directory, and relative image and CSS paths resolve against the current directory. The PDF goes to the `-o` path, or to `output.pdf` if `-o` is omitted. `-e` cannot be combined with an input file.

By default a document can only read files (images, stylesheets, frame documents) from inside its own directory (`BaseDir`): absolute paths, `..` that leaves the directory, and network paths such as `//host/share` are refused. `-allow-external-files` (`Options.AllowExternalFiles` in the library) lifts this for documents you trust.

`-scale` (`Options.Scale` in the library) is the factor that shrinks or enlarges the content (default 1.0). It is the equivalent of wkhtmltopdf's zoom / smart shrinking and Chrome printToPDF's `scale`, and is useful when a document at its real size spills just slightly onto the next page and you want it to fit.

## Markdown to PDF (md2pdf)

`md2pdf` converts a Markdown file into a PDF. It turns the Markdown into HTML with [goldmark](https://github.com/yuin/goldmark) and passes it through the same engine (Markdown → HTML → PDF).

```sh
go install github.com/kujirahand/html2pdf-go/cmd/md2pdf@latest

md2pdf input.md                        # writes input.pdf
md2pdf -o out.pdf -margin 48 input.md
md2pdf -css style.css input.md         # appends your CSS after the default stylesheet
```

It accepts the same flags as `html2pdf` (`-o`, `-page-size`, `-page-width`, `-page-height`, `-landscape`, `-scale`, `-margin`, `-font`, `-font-bold`, `-font-dirs`, `-allow-external-files`, `-verbose`) plus `-css`. Relative image paths resolve against the Markdown file's directory.

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
just make      # builds native CLI binaries in bin/
just fonts     # downloads the IPAex fonts into fonts/ (does nothing if already downloaded)
just samples   # generates the sample PDFs into out/
```

## Supported range and roadmap

See [docs/architecture.md](docs/architecture.md).

For the list of supported HTML elements and CSS, see [docs/reference.md](docs/reference.md) (Japanese: [docs/reference-ja.md](docs/reference-ja.md)).

## License

MIT License. See the `LICENSE` file for details.
