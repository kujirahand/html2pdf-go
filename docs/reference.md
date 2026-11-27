# Support Reference

[日本語版 / Japanese](reference-ja.md)

This manual lists what html2pdf-go supports. Everything here is taken from the source code; anything not listed should be treated as **unsupported**.

## 1. Overview

- Only the contents of `<body>` are rendered. Everything else (`<head>`, `<title>`, `<meta>`, ...) is hidden.
- Styles come from exactly two places:
  - `<style>` elements anywhere in the document (all of them, in document order).
  - The `style` attribute of an element.
- `<link rel="stylesheet">` is **not** loaded: external CSS files are ignored. `@import` is skipped too.
- `<script>` is dropped, and no JavaScript runs.
- CSS comments (`/* ... */`) are stripped. Unknown properties and invalid values are silently ignored.
- At-rule blocks (`@media`, `@page`, `@font-face`, `@keyframes`, ...) are skipped entirely, including the rules nested inside them.
- `!important` is stripped from the value but has **no effect** on priority.
- The keywords `inherit`, `initial` and `unset`, `var()` and `calc()` are not supported.

## 2. Supported HTML elements

Elements that are not listed here, and unknown elements, have `display: inline` and are drawn as inline text (their text content is shown; no special styling).

### Document structure

| Element | Rendering | Notes |
|---|---|---|
| `body` | block, margin 8px | The root of rendering. |
| `div`, `section`, `article`, `header`, `footer`, `address` | block | `address` is not italic. |
| `center` | block, `text-align: center` | |
| `head`, `style`, `script`, `title`, `meta`, `link`, `noscript`, `iframe`, `template`, `object`, `embed`, `svg` | hidden (`display: none`) | Not drawn at all, including children. |

### Block text

| Element | Rendering | Notes |
|---|---|---|
| `p` | block, margin `1em 0` | |
| `h1` – `h6` | block, bold, sized and spaced by the built-in defaults (section 8) | |
| `blockquote` | block, margin `1em 0` | No indentation. |
| `pre` | block | No monospace font; whitespace and newlines are still collapsed. |
| `br` | forced line break | |
| `b`, `strong` | bold | |
| `u`, `ins` | underline | |
| `a` | blue, underlined | `href` is not used (no link). |

### Lists

| Element | Rendering | Notes |
|---|---|---|
| `ul`, `ol` | block, margin `1em 0`, `padding-left: 40px` | Margin is 0 when nested in another list. |
| `li` | list item with a marker | See section 9 for marker rules. |

### Tables

| Element | Rendering | Notes |
|---|---|---|
| `table` | table | Nested tables work. |
| `thead`, `tbody`, `tfoot` | row group | All three behave the same; rows are laid out in source order (`tfoot` is not moved to the bottom). |
| `tr` | table row | |
| `td` | table cell, padding 1px | |
| `th` | table cell, padding 1px, bold, centered | |
| `caption` | — | Not drawn (ignored by the table layout). |
| `colgroup`, `col` | — | Ignored. |

Text or non-row elements placed directly inside `table`, a row group or `tr` are not drawn.

### Images

| Element | Rendering | Notes |
|---|---|---|
| `img` | inline image | Uses `src`. PNG, JPEG and GIF. See section 9. |

### Hidden elements

See the `display: none` list under "Document structure". Any element can also be hidden with `display: none` in CSS.

## 3. Supported attributes

Only these attributes are read by the implementation. All other attributes (`href`, `alt`, `height`, `align`, `bgcolor`, `border`, `cellpadding`, `cellspacing`, `rowspan`, `start`, ...) have no effect. Attribute names are case-insensitive.

| Attribute | Applies to | Effect |
|---|---|---|
| `class` | any element | Matched by `.class` selectors (space-separated list). |
| `id` | any element | Matched by `#id` selectors (case-sensitive). |
| `style` | any element | Inline declarations; highest priority. |
| `src` | `img` | Image file path. Relative paths resolve against `BaseDir`. `http(s)://` URLs (anything containing `://`) and `data:` URIs are not loaded. |
| `width` | any element (meaningful for `img`, `table`, `td`, `th`, block elements) | Parsed as a CSS length; a bare number means px. Treated as the lowest-priority `width` (overridden by any CSS `width`). Negative or invalid values are ignored. |
| `colspan` | `td`, `th` | Number of columns spanned. Values of 1 or less, or non-numeric values, mean 1. Capped at 1000. |

## 4. CSS selectors

| Supported | Example |
|---|---|
| Universal | `*` |
| Element (case-insensitive) | `p` |
| Class (case-sensitive) | `.note` |
| Element + class | `p.note` |
| Multiple classes (compound) | `.a.b`, `td.a.b` |
| ID (case-sensitive; at most one per compound) | `#main` |
| Element + ID, class + ID | `div#main`, `.a#b`, `div.note#main` |
| Descendant (whitespace) | `.total td`, `div p span` |
| Selector list (comma) | `h1, h2, .title` |

| Not supported | Example |
|---|---|
| Attribute selector | `[type=text]` |
| Pseudo-classes / pseudo-elements | `:first-child`, `::before` |
| Child and sibling combinators | `>`, `+`, `~` |

A compound selector that contains any of `[ ] : > + ~ ( )` is **invalid** and never matches. So is one with an empty fragment (`#`, `div#`, `a..b`) or more than one `#` (`#a#b`). Invalid selectors are not an error: in a comma list, the other selectors still work. Note that `div>p` and `div > p` are both invalid, because the `>` makes a part invalid.

## 5. Cascade

Declarations are applied in this order; later steps override earlier ones:

1. The `width` attribute hint.
2. The built-in (UA) stylesheet (section 8).
3. Author stylesheets (`<style>` elements), sorted by specificity, then by source order (a later rule wins on a tie). Any author rule beats any UA rule regardless of specificity.
4. The inline `style` attribute.

Within one rule, later declarations win. `!important` is ignored.

**Specificity** is a triple compared left to right:

1. The number of IDs in the selector (summed over all compounds).
2. The number of class names (summed over all compounds).
3. The number of element names (summed over all compounds). `*` counts as 0.

Example: `#a` = (1, 0, 0); `div#a.b` = (1, 1, 1); `td.a.b` = (0, 2, 1); `.total td` = (0, 1, 1); `p` = (0, 0, 1). An ID always beats any number of classes.

**Inheritance.** Only these properties are inherited: `font-size`, `font-weight`, `font-family`, `color`, `text-align`, `text-decoration`, `line-height`. All others start from their initial value on every element.

## 6. Values

### Lengths

| Unit | Meaning |
|---|---|
| `px` | 0.75pt (1/96 in) |
| `pt` | 1/72 in |
| `mm` | millimetre |
| `cm` | centimetre |
| `in` | inch (72pt) |
| `em` | The element's own `font-size`; for the `font-size` property, the parent's `font-size` |
| `rem` | The root font size (16px = 12pt) |
| `%` | Depends on the property: containing block width for `width`, `margin`, `padding` and border widths; parent `font-size` for `font-size`; the font size for `line-height`; the box width / height for `border-radius` (X / Y) |
| no unit | Treated as px (e.g. `margin: 5`), except where noted: `line-height` (multiplier), `font-weight` (weight), and `font-size` (rejected) |

Other units (`vw`, `vh`, `ex`, `pc`, `ch`, ...) are not supported. A leading `+` or `-` is accepted; malformed numbers such as `2.em` are rejected.

### Colors

| Form | Notes |
|---|---|
| `#rgb`, `#rrggbb` | Hex. `#rgba` and `#rrggbbaa` are rejected. |
| `rgb(r, g, b)`, `rgba(r, g, b, a)` | Components are 0–255 or percentages, separated by commas, spaces or `/`; values are clamped. The alpha value is **ignored**. |
| Named colors | `black`, `white`, `navy`, `red`, `blue`, `green`, `gray`, `grey`, `silver`, `maroon`, `purple`, `yellow`, `orange`, `lime`, `aqua`, `cyan`, `fuchsia`, `magenta`, `teal`, `olive`, `brown`, `pink`, `lightgray`, `lightgrey`, `darkgray`, `darkgrey` |

Not supported: `hsl()`, `transparent` as a color value, `currentColor`, and all other CSS color names. The default text color is black (the initial `color`).

## 7. Supported CSS properties

Any property not listed here is ignored.

| Property | Values | Notes |
|---|---|---|
| `display` | any keyword | Layout recognizes `none`, `block`, `list-item`, `table`, `table-row-group`, `table-row`, `table-cell`, `table-caption`. Every other value (`inline`, `inline-block`, `flex`, `grid`, ...) is treated as inline. Not validated. |
| `font-size` | length, or `xx-small` `x-small` `small` `medium` `large` `x-large` `xx-large` `smaller` `larger` | Keywords: 9, 10, 13, 16, 18, 24, 32px; `smaller` = 0.8333em, `larger` = 1.2em. A unitless number or negative value is ignored. Inherited. |
| `font-weight` | `bold`, `bolder`, `normal`, `lighter`, number | `bold`/`bolder` = 700, `normal`/`lighter` = 400. A number of 700 or more is drawn bold, otherwise regular (only two faces exist). Inherited. |
| `font-family` | comma-separated family names | Quotes (`'` or `"`) and surrounding whitespace are stripped from each name. Names are tried in order against the family names of installed fonts (case-insensitive, name table only, not file names); the first match is used, in its bold face for bold text. Generic names (`serif`, `sans-serif`, `monospace`, `cursive`, `fantasy`, `system-ui`, `ui-serif`, `ui-sans-serif`, `ui-monospace`, `ui-rounded`) are skipped. With no match, the base font is used. Inherited. |
| `color` | color | Inherited. Also the default border color. |
| `background-color` | color | Drawn on blocks, tables, rows (`tr`) and cells. Not drawn on inline elements. |
| `background` | shorthand | Only the first color token is used; everything else (images, position, ...) is ignored. The values `transparent` and `none` clear the background. |
| `text-align` | `left`, `right`, `center` | `justify` and `start` become `left`; `end` becomes `right`. Inherited. Also centers a table narrower than its container. |
| `text-decoration`, `text-decoration-line` | contains `underline` or `none` | Only underline is supported; `line-through` and `overline` are ignored. Inherited. |
| `line-height` | `normal`, number, length | A number is a multiplier of the font size; a length (including `%`) is absolute. Inherited. |
| `margin` | 1–4 lengths, `auto` | `auto` is treated as 0 (no automatic centering). Negative values allowed. |
| `margin-top`, `margin-right`, `margin-bottom`, `margin-left` | length, `auto` | Longhands of `margin`. |
| `padding` | 1–4 lengths | Negative values are treated as 0. |
| `padding-top`, `padding-right`, `padding-bottom`, `padding-left` | length | Longhands of `padding`. |
| `width` | length, `auto` | The content width of a block; for a table it includes the table's padding and border. Percentages refer to the containing block width. |
| `border` | width, style and color in any order | Sets all four sides. Missing width = `medium` (3px); missing color = the element's `color`; missing style = `none`, so **`border: 1px red` draws nothing**. |
| `border-top`, `border-right`, `border-bottom`, `border-left` | same as `border` | One side. |
| `border-width` | 1–4 of `thin` (1px), `medium` (3px), `thick` (5px), length | |
| `border-style` | 1–4 of `none`, `hidden`, `solid`, `dashed`, `dotted`, `double`, `groove`, `ridge`, `inset`, `outset` | `hidden` = `none`; `groove`, `ridge`, `inset`, `outset` are drawn as `solid`. If any value is invalid the whole declaration is ignored. |
| `border-color` | 1–4 colors | If any value is invalid the whole declaration is ignored. |
| `border-top-width`, `border-right-width`, `border-bottom-width`, `border-left-width` | width | Per-side longhands of `border-width`. |
| `border-top-style`, ... `border-left-style` | style | Per-side longhands of `border-style`. |
| `border-top-color`, ... `border-left-color` | color | Per-side longhands of `border-color`. |
| `border-radius` | 1–4 lengths, optionally `/` and 1–4 more | Horizontal radii, then vertical radii after `/` (default: same as horizontal). Order: top-left, top-right, bottom-right, bottom-left. Negative values are ignored. Percentages refer to the box width (X) and height (Y). |
| `border-top-left-radius`, `border-top-right-radius`, `border-bottom-right-radius`, `border-bottom-left-radius` | length [length] | Per-corner longhands of `border-radius`. |
| `page-break-after`, `break-after` | `always`, `page`, `left`, `right`, `auto`, `avoid` | The first four force a page break after the element; `auto` and `avoid` cancel it. Other values are ignored. Not inherited. |

**Shorthand and longhand.** Shorthands that take 1–4 values (`margin`, `padding`, `border-width`, `border-style`, `border-color`) expand in the order top, right, bottom, left: one value for all sides; two = top/bottom, left/right; three = top, left/right, bottom. `border` sets width, style and color of all sides, overriding earlier longhands; a later longhand overrides one aspect of one side. `border-radius` expands in the order top-left, top-right, bottom-right, bottom-left in the same way.

**Border and radius drawing.** Each side is drawn as a line, centered in its border width. With a radius, a uniform border (same width and color on all four sides, all `solid`) is drawn as one rounded outline. If widths, colors or styles differ, or any side is `dashed`, `dotted` or `double`, the four straight lines are drawn and the radii only shape the background. `double` is drawn as two lines of one third of the width. Radii are scaled down when they do not fit the box. Table cells do not support `border-radius`.

## 8. Built-in (UA) defaults

| Selector | Default |
|---|---|
| `head`, `style`, `script`, `title`, `meta`, `link`, `noscript`, `iframe`, `template`, `object`, `embed`, `svg` | `display: none` |
| `body`, `div`, `p`, `h1`–`h6`, `blockquote`, `pre`, `center`, `section`, `article`, `header`, `footer`, `address`, `ul`, `ol` | `display: block` |
| `li` | `display: list-item` |
| `table` | `display: table` |
| `thead`, `tbody`, `tfoot` | `display: table-row-group` |
| `tr` | `display: table-row` |
| `td`, `th` | `display: table-cell` |
| `caption` | `display: table-caption` (not drawn) |
| `body` | `margin: 8px` |
| `p`, `blockquote` | `margin: 1em 0` |
| `h1` | `font-size: 2em`, bold, `margin: 0.67em 0` |
| `h2` | `font-size: 1.5em`, bold, `margin: 0.83em 0` |
| `h3` | `font-size: 1.17em`, bold, `margin: 1em 0` |
| `h4` | bold (1em), `margin: 1.33em 0` |
| `h5` | `font-size: 0.83em`, bold, `margin: 1.67em 0` |
| `h6` | `font-size: 0.67em`, bold, `margin: 2.33em 0` |
| `ul`, `ol` | `margin: 1em 0`, `padding-left: 40px` |
| `ul ul`, `ul ol`, `ol ul`, `ol ol` | `margin: 0` |
| `b`, `strong` | bold |
| `u`, `ins` | underline |
| `a` | `color: blue`, underline |
| `center` | `text-align: center` |
| `th` | bold, `text-align: center` |
| `td`, `th` | `padding: 1px` |

The initial font size is 16px (12pt). Elements not covered by these rules (for example `nav`, `main`, `aside`, `figure`, `dl`, `dt`, `dd`, `form`, `hr`, `fieldset`) are inline: they get no line break or margin around them, and `hr` draws nothing.

## 9. Layout behavior notes

**Blocks.** `width` sets the content width; padding and border are added outside it. Vertical margins collapse between adjacent block siblings (the larger one wins); there is no parent/child margin collapsing, and negative margins do not collapse. `height`, `min-*` and `max-*` do not exist, so heights always follow the content. A block with a background or border that fits on one page is kept on one page.

**Inline content.** Inline elements affect only their text: `color`, `font-size`, `font-weight`, underline and `line-height`. Margin, padding, border and background on inline elements are ignored. Whitespace (including newlines, even in `pre`) collapses to single spaces; leading spaces in a line are dropped. `&nbsp;` is kept and does not break. Block-level elements inside inline content start a new line.

**Line breaking and kinsoku (Japanese line-break rules).**
- A line breaks at spaces, and between CJK characters (CJK ideographs and kana, Hangul, CJK compatibility and full-width forms, and U+20000–3FFFF).
- Latin words are not split unless a single word is wider than the line.
- Characters that must not start a line: `、。，．・：；？！ー〜）〕］｝」』】〉》` and small kana (`ゃゅょぁぃぅぇぉっャュョァィゥェォッ`), `ゝゞ々`, and `, . : ; ? ! ) ] }`. Up to two of them may hang past the end of the previous line; otherwise the break moves earlier.
- Characters that must not end a line: `（〔［｛「『【〈《` and `( [ {`.
- `word-break`, `overflow-wrap`, `white-space` and hyphenation are not supported.

**Lists.** `ul` items get the marker `• ` regardless of nesting; `list-style` is not supported. `ol` items get `1. `, `2. `, ... numbered by counting the `display: list-item` children of the `ol` up to the item (`start` and `reversed` are not supported). A `li` whose parent is not an `ol` gets a bullet. The marker is drawn once, on the first line of the item, to the left of the item's content box (in the list's padding), using the item's font size, weight and color. An item with no inline content gets no marker.

**Images.** `img` is inline, with its bottom on the text baseline. The default size is the file's pixel size × 0.75. A CSS or attribute `width` sets the width and the height follows the aspect ratio (`height` is ignored). Files over 8192 pixels on a side or 33 million pixels, and unreadable or unsupported files, are skipped silently (no placeholder, no `alt` text).

**Tables.**
- Cells are placed left to right; `colspan` makes a cell span columns. There is no `rowspan`.
- Column widths: a cell `width` (including the attribute) fixes its column (content width plus padding and border); `%` widths are used when the table has a `width`, or when an auto-width table has to shrink. Other columns get their max-content width; surplus space is shared equally, and when space is short columns shrink toward their min-content width. Spanning cells only widen columns that are too narrow.
- A table without `width` is as wide as its content, up to the available width. With a `width`, if every column has a fixed width they are scaled to fill it.
- Cell content is always vertically centered. Row height is the tallest cell.
- Each cell draws its own borders; there is no `border-collapse` or `border-spacing`, so adjacent cell borders do not merge.
- A table is centered when its `text-align` (inherited) is `center` and it is narrower than its container. `margin: auto` does not center it.

**Page breaks.**
- Lines of text (inline images sit inside lines), table rows, and bordered or filled boxes that fit on one page are never split across pages. A row or box taller than a page is not split either, and the overflow is not drawn.
- `page-break-after` / `break-after` forces a new page after a block or a table. It is ignored inside table cells and on inline elements and table rows. `page-break-before`, `page-break-inside`, `break-before` and `break-inside` are not supported.
- A page that would contain no text or images is dropped, so a blank page cannot be produced.

**Text.** The base faces are regular and bold, from one Japanese-capable font found at conversion time (Noto Sans CJK / Noto Sans JP preferred), or from `FontFile` / `FontFileBold` if given. If no bold face exists, bold text is drawn with the regular face. `font-family` can pick other installed fonts by family name (section 7). Installed fonts are indexed on first use by scanning the font directories (`FontDirs`, or the standard ones). For a font collection (`.ttc`) only one face represents the file. There is no italic.

## 10. Not supported

- Fonts and text: `font-style` (italic), `font` shorthand, `letter-spacing`, `word-spacing`, `text-indent`, `text-transform`, `white-space`, `vertical-align`, `text-shadow`, `line-through` / `overline`, `justify` alignment.
- Box: `height`, `min-width`, `max-width`, `min-height`, `max-height`, `box-sizing`, `overflow`, `outline`, `box-shadow`, `opacity`, `visibility`, `margin: auto` centering.
- Layout: `float`, `clear`, `position` (`absolute`, `fixed`, `relative`, `sticky`), `z-index`, flex, grid, multi-column, `inline-block` behavior.
- Tables: `rowspan`, `border-collapse`, `border-spacing`, `cellspacing`/`cellpadding`, table `caption`, `colgroup`/`col`, `vertical-align` in cells, `table-layout`, `border-radius` on cells.
- Lists: `list-style`, `list-style-type`, `ol start`, custom markers.
- Backgrounds: `background-image`, gradients, `background-position`, `background-repeat`.
- Selectors and rules: attribute selectors, pseudo-classes, pseudo-elements, `>` `+` `~`, `@media`, `@import`, `@page`, `@font-face`, `!important` priority, `inherit`/`initial`, `var()`, `calc()`.
- Other: external CSS files, remote or `data:` images, SVG, links (`href`), forms, JavaScript, `page-break-before`/`page-break-inside`, headers and footers, page numbers.

## 11. Conversion options

### Library (`html2pdf.Options`)

| Field | Type | Meaning |
|---|---|---|
| `PageSize` | string | `A4` (595.28 × 841.89pt), `A5` (419.53 × 595.28pt) or `Letter` (612 × 792pt); case-insensitive. Empty = `A4`. Other names are an error. |
| `PageWidth`, `PageHeight` | float64 (points) | Custom size. If both are greater than 0 they are used as given and `PageSize` and `Landscape` are ignored. If only one is set, the other comes from `PageSize`. |
| `Landscape` | bool | Swaps width and height of the `PageSize` size. |
| `BaseDir` | string | Directory against which relative image paths resolve. |
| `FontDirs` | []string | Directories to search for fonts (up to 4 levels deep; `.ttf`, `.ttc`, `.otf`). Empty = standard system font directories. An error is returned if no usable font is found. Also searched by `font-family` names. |
| `FontFile` | string | Path of a font file used as the regular face instead of the automatic choice. If it cannot be read or parsed, `Convert` returns an error. Empty = automatic selection. |
| `FontFileBold` | string | Path of the bold face. If `FontFile` is set without `FontFileBold`, bold text uses the regular face. An unreadable file is an error. |
| `Scale` | float64 | Shrinks (below 1) or enlarges (above 1) the content. 0 or less means 1.0; NaN or infinity is an error. |
| `Margin` | float64 (points) | Page margin on all four sides. 0 = no margin (the `body` still has its own 8px margin). |

Functions: `Convert(src []byte, w io.Writer, opts *Options) error` and `ConvertFile(htmlPath, pdfPath string, opts *Options) error` (nil `opts` is allowed; `ConvertFile` writes the file only on success).

### CLI

```
html2pdf [flags] input.html
```

| Flag | Default | Meaning |
|---|---|---|
| `-o` | input name with `.pdf` | Output PDF path. |
| `-page-size` | `A4` | `A4`, `A5` or `Letter`. |
| `-landscape` | off | Landscape orientation. |
| `-scale` | `1.0` | Content scale factor. |
| `-margin` | `0` | Page margin in points, all sides. |
| `-font` | auto-detect | Regular font file (`Options.FontFile`). |
| `-font-bold` | the regular font | Bold font file (`Options.FontFileBold`). |
| `-font-dirs` | none | Comma-separated directories to search for fonts (`Options.FontDirs`). |

The CLI sets `BaseDir` to the input file's directory. It has no flag for `PageWidth` or `PageHeight`. Flags may appear before or after the input file.
