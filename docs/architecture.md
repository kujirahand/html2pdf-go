# Architecture

[日本語版](architecture-ja.md)

html2pdf-go is an engine that converts HTML+CSS into PDF using pure Go only. The pipeline below is implemented; the supported range and limitations are described later in this document.

## Pipeline

```
HTML
  │
  ▼
dom      Parse HTML and build the document tree
  │
  ▼
css      Apply stylesheets (selector matching → cascade → computed styles)
  │
  ▼
layout   Box tree generation → block/table layout → pagination
  │
  ▼
render   Convert the laid-out pages into drawing operations
  │
  ▼
pdf      Write PDF objects and content streams
  │
  ▼
PDF
```

The `text` package is used by `layout` and `render` for font measurement and line breaking.

## Package responsibilities

| Package | Responsibility |
|---|---|
| `internal/dom` | Parses HTML and builds the styled document tree (will use `x/net/html` in the future) |
| `internal/css` | CSS parsing, selector matching, the cascade, and computed-style generation |
| `internal/text` | Font loading and measurement, and text shaping (line breaking and line-break rules / kinsoku) |
| `internal/layout` | Box tree generation, block/table layout, and pagination |
| `internal/datauri` | Decodes `data:` URIs (base64 / percent-encoded); used by `layout` and `render` as an image source |
| `internal/render` | Renders laid-out pages via the `pdf` package |
| `internal/pdf` | PDF file writing (objects, content streams, and font/image embedding) |

Layout maps link rectangles and ID and heading positions to pages. Rendering resolves internal links, and the PDF writer emits annotations, bookmarks, and document information. These use the same scale and Y-axis inversion as drawing.

## Scale (shrink / enlarge factor)

With `Options.Scale` (`-scale` in the CLI) you can shrink or enlarge the whole content. It is the equivalent of wkhtmltopdf's zoom / smart shrinking and Chrome printToPDF's `scale`. Values of 0 or less are treated as 1.0.

- `layout` lays out on a virtual page of "real page size / Scale", and the returned `Page` has its `Width`/`Height` restored to the real size and carries the `Scale`
- `render` passes `Page.Scale` to `pdf.Page.SetScale`, and `pdf` multiplies coordinates, sizes, line widths, and font sizes by the Scale when drawing (the Y-axis flip uses the real page height)
- When Scale is 1, the result is the same as before

## Supported range for now

For details of the implemented supported range, see [reference.md](reference.md) (Japanese: [reference-ja.md](reference-ja.md)).

It is based on measurements of invoice HTML.

- **Tags**: `h1`, `p`, `div`, `br`, `table`, `tr`, `th`, `td`, `img`, `center`, `ul`, `li`, `ol` (`ul`, `li`, `ol` are not used in invoices, but are included in the initial supported range because they are essential for general documents)
- **Selectors**: `*`, element, `.class`, `#id`, and compounds of them (such as `element.class`, `element#id`, `.class#id`), and descendants (such as `.total td`). Specificity is a triple of (number of IDs, number of classes, number of elements)
- **Properties**: `font-size`, `font-weight`, `font-family` (specified via `FontFile` / `FontFileBold`, or by the family name of an installed font), `color`, `background-color`, `text-align`, `text-decoration`, `line-height`, `margin` (and each side), `padding` (and each side), `border` (and each side; styles are solid / dashed / dotted / double, and corner rounding uses `border-radius` and the longhands for each corner), `width`, `page-break-after`
- **Units**: The units supported for length-like properties (such as `width`, `margin`, `padding`, `font-size`) are `px`, `em`, `rem`, `%`, `pt`, `mm`, `cm`, `in`, and a unit-less `0`
  - Conversion basis: 1px = 0.75pt (96dpi → 72pt), 1in = 25.4mm = 72pt
  - `em` is relative to the element's own `font-size` (except that a `font-size` specification itself is relative to the parent element's `font-size`), and `rem` is relative to the root element's `font-size`
  - The basis of `%` differs per property (for `width`-like properties it is the width of the containing block)

## Limitations

- A table row or image taller than one page is not split; the overflowing part is not drawn
- Processing time for nested tables grows exponentially with depth, so deeply nested tables are not a good fit
- Table column widths use automatic layout; in a table with no `width` specified, a cell's `%` specification is not used when the column widths fit in total, and is used only when shrinking is required
- Inside a block with `text-align: center`, a table is centered (a difference from browsers)

## Non-goals (initial)

- JavaScript
- flex
- grid
- float
- position

## Roadmap

| Version | Contents |
|---|---|
| v0 | Text-only rendering |
| v1 | Blocks and borders (including list `ul`/`ol`/`li` markers, numbers, and indentation) |
| v2 | Tables |
| v3 | Images and page breaks |
