# Tutorial

[日本語版](tutorial-ja.md)

First, let's convert one HTML file into a PDF with the CLI.

## 1. Install it

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

## 2. Create an HTML file

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

## 3. Convert it to PDF

```sh
html2pdf hello.html
```

This creates `hello.pdf` in the same folder. Open it and you should see a PDF with a large blue heading and a table with borders.

To change the output file or the page settings, pass options:

```sh
html2pdf -o out.pdf -page-size A4 -margin 48 hello.html   # output file and margins
html2pdf -scale 0.95 hello.html                            # shrink everything to 95%
```

## 4. Next steps

- Change the appearance: edit the contents of the `<style>` element. The available HTML elements and CSS are listed in [reference.md](reference.md)
- Start a new page: put `<page-break></page-break>` where the next page should begin, for example `<p>Page 1</p><page-break></page-break><p>Page 2</p>`. Leading, trailing, and repeated `<page-break>` elements do not create blank pages
- Use it from a Go program: see the [library guide](library.md)
- See real examples: [examples/](../examples/) contains sample HTML and Markdown documents — an address book, a receipt, an event flyer, a concert ticket, a cafe menu, a business card, a calendar, a boarding pass, meeting minutes, a recipe and more, in Japanese and English. Run `just examples` to convert them all into `examples/pdf/`
