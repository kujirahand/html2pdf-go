![html2pdf-go — HTML/CSS/Markdown to PDF](docs/html2pdf-go-logo800.png)

# html2pdf-go - HTML/CSS/Markdown→PDF

[日本語版 README](README-ja.md)

A library and CLI for converting HTML/CSS/Markdown to PDF. By supporting the major HTML/CSS tags, it makes it easy to create reasonably good-looking PDFs. Implemented in pure Go, it is a small, handy tool that runs as a single executable with no browser dependency.

## Install

```sh
# Linux / macOS
curl -fsSL https://raw.githubusercontent.com/kujirahand/html2pdf-go/main/scripts/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/kujirahand/html2pdf-go/main/scripts/install.ps1 | iex
```

The binaries are unsigned, so downloading them in a browser makes the OS block them (Gatekeeper / SmartScreen); installing with these scripts avoids that, and they verify the SHA-256 checksum before installing.

## Tutorial

See [docs/tutorial.md](docs/tutorial.md) for a walkthrough that converts one HTML file to PDF with the CLI, and the sample catalog [examples/README.md](examples/README.md) for the kinds of PDFs it can create.

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
html2pdf -format png input.html        # writes input.png (PNG image output)
cat doc.html | html2pdf -o - > doc.pdf
html2pdf examples/*.html
html2pdf -o outdir a.html b.html   # outdir must be an existing directory
```

`-e <HTML>` converts HTML given directly on the command line, which is handy for debugging. The source is also written to `output.html` in the current directory, and relative image and CSS paths resolve against the current directory. The PDF goes to the `-o` path, or to `output.pdf` if `-o` is omitted. `-e` cannot be combined with stdin or an input file.

Both CLIs read stdin when the input is `-`, or when no file is given and stdin is a pipe or redirected file. Stdin uses the current directory as `BaseDir`, defaults to `output.pdf` without `-o`, and does not write `output.html`. Use `-o -` for stdout; logs and errors go to stderr. A single-page PNG can go to stdout; a multi-page PNG cannot. `-e`, stdin, and input files cannot be combined (a pipe is ignored when `-e` or files are given). Multiple files are converted independently; failures are reported on stderr, conversion continues, and the exit status is non-zero if any failed. With multiple inputs, `-o` must name an existing directory. An existing directory also works for a single input, stdin, or `-e`.

By default a document can only read files (images, stylesheets, frame documents) from inside its own directory (`BaseDir`): absolute paths or `..` pointing outside it, and network paths such as `//host/share`, are refused. `-allow-external-files` (`Options.AllowExternalFiles` in the library) lifts this for documents you trust.

With `-format png`, each page is written as a PNG image instead of a PDF (a multi-page document gets `-1`, `-2`, ... before the `.png` extension), rasterized at `-png-scale` times the 72-dpi point size (default 2, i.e. 144 dpi). The format can also be selected by giving `-o` a `.png` name and omitting `-format`. From Go, use `html2pdf.ConvertPNG` / `html2pdf.ConvertMarkdownPNG`, which return one PNG per page.

`-scale` (`Options.Scale` in the library) is the factor that shrinks or enlarges the content (default 1.0). It is the equivalent of wkhtmltopdf's zoom / smart shrinking and Chrome printToPDF's `scale`, and is useful when a document at its real size spills just slightly onto the next page and you want it to fit.

## Markdown to PDF (md2pdf)

`md2pdf` converts a Markdown file into a PDF. It turns the Markdown into HTML with [goldmark](https://github.com/yuin/goldmark) and passes it through the same engine (Markdown → HTML → PDF).

```sh
go install github.com/kujirahand/html2pdf-go/cmd/md2pdf@latest

md2pdf input.md                        # writes input.pdf
md2pdf -o out.pdf -margin 48 input.md
md2pdf -css style.css input.md         # appends your CSS after the default stylesheet
cat notes.md | md2pdf -o - > notes.pdf
md2pdf notes/*.md
```

It accepts the same flags as `html2pdf` (`-o`, `-page-size`, `-page-width`, `-page-height`, `-landscape`, `-scale`, `-margin`, `-font`, `-font-bold`, `-font-dirs`, `-allow-external-files`, `-format`, `-png-scale`, `-verbose`) plus `-css`. Relative image paths resolve against the Markdown file's directory.

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
