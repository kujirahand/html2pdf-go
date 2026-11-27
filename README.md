# html2pdf-go

[日本語版 README](README-ja.md)

A pure-Go HTML-to-PDF conversion engine. HTML parsing, CSS application, layout, and PDF writing are all implemented. For the supported range of HTML/CSS, see [docs/reference.md](docs/reference.md).

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

## Repository layout

| Path | Contents |
|---|---|
| `html2pdf.go` | Public API (`Convert`, `ConvertFile`, `Options`) |
| `cmd/html2pdf/` | CLI tool |
| `internal/dom` | HTML parsing and the document tree |
| `internal/css` | CSS parsing, the cascade, and computed styles |
| `internal/text` | Font measurement, line breaking and line-break rules (kinsoku) |
| `internal/layout` | Box generation, layout, and pagination |
| `internal/render` | Rendering of the layout result |
| `internal/pdf` | PDF writing |
| `examples/basic/` | Minimal usage example |
| `examples/*.html` | Sample documents (address book, receipt, etc.) |
| `justfile` | Development tasks (test / fonts / samples) |
| `fonts/` | Where `just fonts` downloads the IPAex fonts (the binaries are not tracked) |
| `out/` | Output directory for generated sample PDFs (not tracked) |
| `testdata/` | HTML files for tests |
| `docs/` | Design documents |

## Supported range and roadmap

See [docs/architecture.md](docs/architecture.md).

For the list of supported HTML elements and CSS, see [docs/reference.md](docs/reference.md) (Japanese: [docs/reference-ja.md](docs/reference-ja.md)).

## License

MIT License. See the `LICENSE` file for details.
