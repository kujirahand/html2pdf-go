# Building PDFs from HTML in Pure Go

Do you really need a headless browser or an external command just to make a PDF? This article shows how to generate PDFs from HTML with a pure Go library, and how to choose among the options.

## Introduction

When you generate invoices or reports automatically, three approaches are common.

1. Have a headless browser print the page
2. Call an external command such as `wkhtmltopdf`
3. Generate the file directly with a pure Go library

The third approach ships as a single binary and keeps container images small.

## Basic Usage

Pass an HTML string to the `Convert` function and you get the PDF bytes back.

```go
package main

import (
	"log"
	"os"

	"example.com/pdfkit"
)

func main() {
	html := `<h1>Hello</h1><p>This PDF was made with Go.</p>`

	pdf, err := pdfkit.Convert(html, pdfkit.Options{
		PageSize: "A4",
		FontDirs: []string{"./fonts"},
	})
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("out.pdf", pdf, 0o644); err != nil {
		log.Fatal(err)
	}
}
```

Point `Options.FontDirs` at a folder of fonts and text in other scripts, such as Japanese, renders as-is.

> **Note:** The library does not run JavaScript. For dynamic pages, render the HTML first and then pass it in.

## Comparing the Approaches

| Approach | Ease of distribution | Fidelity | JavaScript | Speed |
|---|---|---|---|---|
| Headless browser | Low (large dependency) | High | Supported | Slower |
| External command | Medium (needs installing) | Medium | Partial | Average |
| Pure Go library | High (single binary) | High within its scope | Not supported | Fast |

## Summary

- For template-driven documents, pure Go is simple and stable.
- For pages that need complex CSS or JavaScript, the browser approach fits better.
- Whichever you pick, start with a small template and expand from there.

*The package name `pdfkit` and the code in this article are fictional and for illustration only.*
