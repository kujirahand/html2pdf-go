# Markdown to PDF (md2pdf)

[日本語版 / Japanese](markdown-ja.md)

`md2pdf` converts a Markdown file into a PDF with the same engine as `html2pdf`. For the HTML and CSS the engine supports, see [reference.md](reference.md).

## 1. Pipeline

```
Markdown → (goldmark) → HTML → code-block transform → + default CSS → Convert → PDF
```

1. The Markdown is converted to HTML with [goldmark](https://github.com/yuin/goldmark) (CommonMark plus the GitHub Flavored Markdown tables, strikethrough and task lists). Raw HTML in the source passes through unchanged.
2. Every `<pre>` block is rewritten so that line breaks and indentation survive (section 4).
3. The result is wrapped in a complete HTML document whose `<style>` element holds `DefaultMarkdownCSS` followed by your extra CSS, if any.
4. The document is passed to `Convert`, so the usual `Options` (page size, margin, fonts, scale) apply.

Functions:

- `MarkdownToHTML(src []byte) ([]byte, error)` returns the HTML document (with `DefaultMarkdownCSS` only).
- `ConvertMarkdown(src []byte, w io.Writer, opts *Options) error` renders Markdown to a PDF written to `w`.
- `ConvertMarkdownFile(mdPath, pdfPath string, opts *Options) error` reads a file and writes the PDF only on success. `BaseDir` defaults to the Markdown file's directory (a copy of `opts` is used; the caller's value is never modified), so relative image paths resolve against it.

## 2. CSS

`DefaultMarkdownCSS` styles headings (with bottom borders on `h1` and `h2`), paragraphs, links, lists, blockquotes (left border and light background), `pre` and `code` (light gray background), tables (borders on `table`, `th` and `td`, shaded `th`) and `hr`. It uses only CSS that the engine supports.

To override it, supply extra CSS. It is appended after the default, so later rules win on equal specificity:

- CLI: `-css style.css`
- Library: `Options.MarkdownCSS` (used only by `ConvertMarkdown` and `ConvertMarkdownFile`)

```css
h1 { color: #205090; }
body { font-size: 12pt; }
```

## 3. Supported elements

| Markdown | Result |
|---|---|
| `#` to `######` | `h1` to `h6` |
| Paragraphs, `**bold**` | `p`, bold text |
| `*emphasis*` | `em`, drawn as regular text (no italic face) |
| `~~strikethrough~~` | `del`, drawn as regular text |
| `` `code` `` | `code`; the background is not drawn (see section 5) |
| Fenced and indented code blocks | `pre`, with line breaks and indentation preserved |
| `-` / `1.` lists, nested lists | `ul` / `ol` |
| `- [x]` task lists | list items with a `[x]` / `[ ]` text marker instead of a drawn checkbox |
| `>` blockquotes | `blockquote` |
| GFM tables | `table` with borders |
| `---` | `hr`, a horizontal rule (drawn through CSS) |
| `[text](url)` | `a`, blue underlined text (the link is not clickable) |
| `![alt](path)` | `img`; relative paths resolve against `BaseDir` |
| Raw HTML | passed through as-is |

## 4. Code-block whitespace transform

The layout engine collapses all whitespace (including newlines) and has no whitespace-preserving `pre`. Left alone, a code block would be flattened into one line. So each `<pre>` block that goldmark produces is rewritten before conversion:

1. The `<code>` tags inside are removed; the text, which goldmark has already HTML-escaped, is kept.
2. The text is split into lines, and one trailing empty line is dropped.
3. In each line, tabs become 4 spaces and every space becomes `&nbsp;` (U+00A0, which the layout engine keeps and draws at the width of a space). An empty line becomes a single `&nbsp;` so that it keeps its height.
4. The lines are joined with `<br>` and wrapped in `<pre>` again.

## 5. Limitations

- No italic: emphasis is drawn as regular text.
- Links are drawn but are not clickable.
- Long code lines wrap like normal text.
- Raw HTML in the source is passed through (goldmark runs with unsafe rendering enabled). Only convert Markdown you trust.
