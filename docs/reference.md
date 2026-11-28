# Support Reference

[日本語版 / Japanese](reference-ja.md)

This manual lists what html2pdf-go supports. Everything here is taken from the source code; anything not listed should be treated as **unsupported**.

## 1. Overview

- Only the contents of `<body>` are rendered (or the frames, for a `<frameset>` document; see "Frames"). Everything else (`<head>`, `<title>`, `<meta>`, ...) is hidden.
- Styles come from three places:
  - `<style>` elements anywhere in the document (all of them, in document order).
  - `<link rel="stylesheet" href="...">` elements, in document order together with the `<style>` elements. Only local files are loaded, with relative paths resolved against the base directory; the file must lie inside it unless `AllowExternalFiles` is set (see the `BaseDir` and `AllowExternalFiles` options), must be a regular file, and may be at most 32 MiB. A link whose `href` contains `://` or starts with `data:` is ignored (remote CSS is not fetched), as is a `rel` containing `alternate`. A local file that cannot be read makes the conversion fail with an error.
  - The `style` attribute of an element.
- `@import` is skipped, in external CSS files too.
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
| `page-break` | block, forces a page break after it | Put `<page-break></page-break>` (or just `<page-break>`) where the next page should start. Anything written after an unclosed `<page-break>` is moved out to follow it. See section 9. |
| `frameset` | frame grid | See "Frames" below. |
| `frame` | one grid cell | See "Frames" below. `noframes` content is not drawn. |
| `iframe` | block-level embedded document | See "Inline frames" below. |
| `head`, `style`, `script`, `title`, `meta`, `link`, `noscript`, `template`, `object`, `embed`, `svg` | hidden (`display: none`) | Not drawn at all, including children. |

### Frames

A document that contains a `<frameset>` (the first one is used) is drawn as a single page of frames instead of `<body>`.

- `rows` and `cols` are comma-separated lists. Each token is a number of pixels (× 0.75 points), a percentage of the whole area (`30%`), or a relative length (`*` has weight 1, `2*` weight 2) that shares the space left by the other tokens. A missing attribute means `100%`. The area is the page inside the margins.
- The cells are rows × columns; `frame` elements are assigned to them in document order. Extra cells stay empty and extra frames are ignored.
- `src` of a `frame` is read as a local file relative to the base directory; an unreadable file makes the conversion fail. A `src` that is a URL (starts with `//` or a scheme such as `https:`, `data:`, `about:` or `blob:`), or a missing `src`, gives an empty cell.
- Each frame document is laid out on its own, with the cell size as the page size, no margin, and the same scale. Its own external CSS and images resolve relative to the directory of that file, and a frame document may itself be a frameset.
- Only the first page of a frame document is shown, and its content is clipped to the cell.
- If the `px` and `%` tracks add up to more than the area, they all shrink in proportion to fit it and relative tracks get 0.
- Exceeding the limits (8 nesting levels, 1000 frame documents in total, counting frames, iframes and `srcdoc`) makes the conversion fail.

### Inline frames

An `iframe` that appears as a block-level child is drawn as an embedded document box.

- The box size comes from the CSS `width` / `height`, then from the `width` / `height` attributes (pixels, or a percentage of the containing block's width for `width`; a percentage `height` falls back to the default), and finally from the HTML default of 300x150 px. Margins, padding and borders behave as for other blocks, and the frame document fills the content box.
- `srcdoc` is used as the document when present (it wins over `src`, as in HTML). A local `src` file is read relative to the base directory and an unreadable file makes the conversion fail; a `src` that is a URL (see "Frames") or no source at all gives an empty box.
- The frame document is laid out at the content-box size and only its first page is shown, clipped to the box. Frames and iframes share the limits of 8 nesting levels and 1000 frame documents in total; exceeding either makes the conversion fail.
- An `iframe` inside an inline element such as `span` or `a` is not drawn. An `iframe` that is a direct child of the block flow (including a direct child of `p`) is drawn as a block.
- In a table cell, an `iframe` counts with its resolved width (see the size rules above) when the column widths are computed.

### Block text

| Element | Rendering | Notes |
|---|---|---|
| `p` | block, margin `1em 0` | |
| `h1` – `h6` | block, bold, sized and spaced by the built-in defaults (section 8) | |
| `blockquote` | block, margin `1em 0` | No indentation. |
| `pre` | block, monospace | Uses a monospace font; whitespace and newlines are still collapsed. |
| `code`, `kbd`, `samp`, `tt` | inline, monospace, background `#f0f0f0` | |
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

Only these attributes are read by the implementation. All other attributes (`href`, `alt`, `align`, `bgcolor`, `border`, `cellpadding`, `cellspacing`, `rowspan`, `start`, ...) have no effect. Attribute names are case-insensitive.

| Attribute | Applies to | Effect |
|---|---|---|
| `class` | any element | Matched by `.class` selectors (space-separated list). |
| `id` | any element | Matched by `#id` selectors (case-sensitive). |
| `style` | any element | Inline declarations; highest priority. |
| `src` | `iframe`, `frame` | Local HTML file path, relative to `BaseDir` and confined to it unless `AllowExternalFiles` is set (at most 32 MiB). A URL (starts with `//` or a scheme such as `https:`, `data:`, `about:`) is not loaded and gives an empty box or cell. An unreadable file is an error. For `iframe`, `srcdoc` takes precedence. |
| `srcdoc` | `iframe` | HTML source of the embedded document. Takes precedence over `src`. |
| `rows`, `cols` | `frameset` | Comma-separated track sizes: pixels, a percentage, or a relative length (`*`, `2*`). Tracks of `px` and `%` that exceed the area shrink in proportion to fit it. |
| `src` | `img` | Image file path, or a `data:` URI. Relative paths resolve against `BaseDir` and are confined to it unless `AllowExternalFiles` is set; paths starting with `//` or `\\` are never loaded. The decoded images of one PDF may total at most 100 million pixels; an image over that budget is skipped. `data:` URIs may be base64 (`data:image/png;base64,...`) or percent-encoded (`data:image/gif,...`); the media type, if given, must be `image/*`; PNG, JPEG and GIF are decoded. `http(s)://` URLs (anything containing `://`) are not loaded. |
| `width` | any element (meaningful for `img`, `iframe`, `table`, `td`, `th`, block elements) | Parsed as a CSS length; a bare number means px. Treated as the lowest-priority `width` (overridden by any CSS `width`). Negative or invalid values are ignored. |
| `height` | `iframe` | Plain pixel number only (a percentage is ignored). Treated as the lowest-priority `height` (overridden by any CSS `height`). |
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
| `font-family` | comma-separated family names | Quotes (`'` or `"`) and surrounding whitespace are stripped from each name. Names are tried in order against the family names of installed fonts (case-insensitive, name table only, not file names); the first match is used, in its bold face for bold text. The generic name `monospace` resolves the way a web browser's fixed-font default does, by OS and UI locale: MS Gothic on Japanese Windows, Courier New on English Windows, Osaka-Mono on Japanese macOS, Courier on English macOS, and on Linux the usual fontconfig choices (Noto Sans Mono CJK JP for Japanese; DejaVu Sans Mono or Liberation Mono otherwise). The locale is read from LC_ALL, LC_CTYPE, then LANG; when it cannot be determined, the Japanese and Western candidates for the OS are both tried. If none of those browser defaults is installed, any installed family whose name contains "Mono" is used (Japanese-capable ones preferred), and with no monospace font at all, the base font is used. If the resolved monospace font has no Japanese glyphs, Japanese text set in it is not drawn. The other generic names (`serif`, `sans-serif`, `cursive`, `fantasy`, `system-ui`, `ui-serif`, `ui-sans-serif`, `ui-monospace`, `ui-rounded`) are skipped. With no match, the base font is used. Inherited. |
| `color` | color | Inherited. Also the default border color. |
| `background-color` | color | Drawn on blocks, tables, rows (`tr`) and cells. Also drawn behind inline elements, per line. |
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
| `head`, `style`, `script`, `title`, `meta`, `link`, `noscript`, `template`, `object`, `embed`, `svg` | `display: none` |
| `iframe` | `display: block` |
| `body`, `div`, `p`, `h1`–`h6`, `blockquote`, `pre`, `center`, `section`, `article`, `header`, `footer`, `address`, `page-break`, `ul`, `ol` | `display: block` |
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
| `page-break` | `page-break-after: always` |
| `code`, `kbd`, `samp`, `tt` | `font-family: monospace`, `background-color: #f0f0f0` |
| `pre` | `font-family: monospace` |

The initial font size is 16px (12pt). Elements not covered by these rules (for example `nav`, `main`, `aside`, `figure`, `dl`, `dt`, `dd`, `form`, `hr`, `fieldset`) are inline: they get no line break or margin around them, and `hr` draws nothing.

## 9. Layout behavior notes

**Blocks.** `width` sets the content width; padding and border are added outside it. Vertical margins collapse between adjacent block siblings (the larger one wins); there is no parent/child margin collapsing, and negative margins do not collapse. `height`, `min-*` and `max-*` do not exist (except that `height` works on an `iframe` box), so heights always follow the content. A block with a background or border that fits on one page is kept on one page.

**Inline content.** Inline elements affect only their text: `color`, `font-size`, `font-weight`, underline and `line-height`. Margin, padding and border on inline elements are ignored, but `background-color` is drawn behind the text run (split per line when it wraps). Whitespace (including newlines, even in `pre`) collapses to single spaces; leading spaces in a line are dropped. `&nbsp;` is kept and does not break. Block-level elements inside inline content start a new line.

**Line breaking and kinsoku (Japanese line-break rules).**
- A line breaks at spaces, and between CJK characters (CJK ideographs and kana, Hangul, CJK compatibility and full-width forms, and U+20000–3FFFF).
- Latin words are not split unless a single word is wider than the line.
- Characters that must not start a line: `、。，．・：；？！ー〜）〕］｝」』】〉》` and small kana (`ゃゅょぁぃぅぇぉっャュョァィゥェォッ`), `ゝゞ々`, and `, . : ; ? ! ) ] }`. Up to two of them may hang past the end of the previous line; otherwise the break moves earlier.
- Characters that must not end a line: `（〔［｛「『【〈《` and `( [ {`.
- `word-break`, `overflow-wrap`, `white-space` and hyphenation are not supported.

**Lists.** `ul` items get the marker `• ` regardless of nesting; `list-style` is not supported. `ol` items get `1. `, `2. `, ... numbered by counting the `display: list-item` children of the `ol` up to the item (`start` and `reversed` are not supported). A `li` whose parent is not an `ol` gets a bullet. The marker is drawn once, on the first line of the item, to the left of the item's content box (in the list's padding), using the item's font size, weight and color. An item with no inline content gets no marker.

**Images.** `img` is inline, with its bottom on the text baseline. The default size is the image's pixel size (the file's, or the decoded `data:` URI's) × 0.75. A CSS or attribute `width` sets the width and the height follows the aspect ratio (`height` is ignored). Images over 8192 pixels on a side or 33 million pixels, and unreadable, invalid or unsupported images (including SVG), are skipped silently (no placeholder, no `alt` text).

**Tables.**
- Cells are placed left to right; `colspan` makes a cell span columns. There is no `rowspan`. A table of more than 10000 columns, or a document whose layout work exceeds a fixed budget (for example deeply nested tables), makes the conversion fail.
- Column widths: a cell `width` (including the attribute) fixes its column (content width plus padding and border); `%` widths are used when the table has a `width`, or when an auto-width table has to shrink. Other columns get their max-content width; surplus space is shared equally, and when space is short columns shrink toward their min-content width. Spanning cells only widen columns that are too narrow.
- A table without `width` is as wide as its content, up to the available width. With a `width`, if every column has a fixed width they are scaled to fill it.
- Cell content is always vertically centered. Row height is the tallest cell.
- Each cell draws its own borders; there is no `border-collapse` or `border-spacing`, so adjacent cell borders do not merge.
- A table is centered when its `text-align` (inherited) is `center` and it is narrower than its container. `margin: auto` does not center it.

**Page breaks.**
- Lines of text (inline images sit inside lines), table rows, and bordered or filled boxes that fit on one page are never split across pages. A row or box taller than a page is not split either, and the overflow is not drawn.
- `page-break-after` / `break-after` forces a new page after a block or a table. It is ignored inside table cells and on inline elements and table rows. `page-break-before`, `page-break-inside`, `break-before` and `break-inside` are not supported.
- `<page-break>` is a dedicated element for a forced page break: the content after it starts on the next page. It is built on `page-break-after: always` (section 8), so the same restrictions apply. The parser would nest what follows an unclosed `<page-break>` inside it, so the children are moved to just after the element, keeping their order; `<p>a</p><page-break><p>b</p>` therefore puts `b` on the next page. A `<page-break>` at the start or end of the document, or several in a row, does not create blank pages. A `<page-break>` placed inside an inline element does not break the page.
- A page that would contain no text or images is dropped, so a blank page cannot be produced.

**Text.** The base faces are regular and bold, from one Japanese-capable font found at conversion time (Noto Sans CJK / Noto Sans JP preferred), or from `FontFile` / `FontFileBold` if given. If no bold face exists, bold text is drawn with the regular face. `font-family` can pick other installed fonts by family name (section 7). Installed fonts are indexed on first use by scanning the font directories (`FontDirs`, or the standard ones). For a font collection (`.ttc`) only one face represents the file. There is no italic.

## 10. Not supported

- Fonts and text: `font-style` (italic), `font` shorthand, `letter-spacing`, `word-spacing`, `text-indent`, `text-transform`, `white-space`, `vertical-align`, `text-shadow`, `line-through` / `overline`, `justify` alignment.
- Box: `height` (except on `iframe`), `min-width`, `max-width`, `min-height`, `max-height`, `box-sizing`, `overflow`, `outline`, `box-shadow`, `opacity`, `visibility`, `margin: auto` centering.
- Layout: `float`, `clear`, `position` (`absolute`, `fixed`, `relative`, `sticky`), `z-index`, flex, grid, multi-column, `inline-block` behavior.
- Tables: `rowspan`, `border-collapse`, `border-spacing`, `cellspacing`/`cellpadding`, table `caption`, `colgroup`/`col`, `vertical-align` in cells, `table-layout`, `border-radius` on cells.
- Lists: `list-style`, `list-style-type`, `ol start`, custom markers.
- Backgrounds: `background-image`, gradients, `background-position`, `background-repeat`.
- Selectors and rules: attribute selectors, pseudo-classes, pseudo-elements, `>` `+` `~`, `@media`, `@import`, `@page`, `@font-face`, `!important` priority, `inherit`/`initial`, `var()`, `calc()`.
- Other: remote CSS, remote frames, remote images, SVG, links (`href`), forms, JavaScript, `page-break-before`/`page-break-inside`, headers and footers, page numbers.

## 11. Conversion options

### Library (`html2pdf.Options`)

| Field | Type | Meaning |
|---|---|---|
| `PageSize` | string | `A4` (595.28 × 841.89pt), `A5` (419.53 × 595.28pt) or `Letter` (612 × 792pt); case-insensitive. Empty = `A4`. Other names are an error. |
| `PageWidth`, `PageHeight` | float64 (points) | Custom size. If both are greater than 0 they are used as given and `PageSize` and `Landscape` are ignored. If only one is set, the other comes from `PageSize`. |
| `Landscape` | bool | Swaps width and height of the `PageSize` size. |
| `BaseDir` | string | Directory against which relative paths (images, stylesheets, frame documents) resolve. Unless `AllowExternalFiles` is set, these files must lie inside it. |
| `AllowExternalFiles` | bool | Permits documents to reference files outside `BaseDir`, by absolute path or with `..`. Default false: images, stylesheets and frame documents can only be read from inside `BaseDir` (`..` and symbolic links cannot leave it; absolute paths are accepted only when they point inside `BaseDir`). Paths starting with `//` or `\\` (network paths) are refused in both modes. Only enable it for documents you trust. |
| `FontDirs` | []string | Directories to search for fonts (up to 4 levels deep; `.ttf`, `.ttc`, `.otf`). Only the given directories are searched; empty = standard system font directories. The library reads no environment variable or config file. An error is returned if no usable font is found. Also searched by `font-family` names. |
| `OnFontUse` | func(FontUse) | Called once for each distinct font face resolved during conversion. `FontUse` has `Family` (the matched `font-family` name, normalized for matching, i.e. lower-cased; empty for the default face), `Bold` (bold was requested; if there is no bold face, `Path` is the regular file and `Bold` is still true) and `Path` (the font file). nil = no callback. |
| `FontFile` | string | Path of a font file used as the regular face instead of the automatic choice. If it cannot be read or parsed, `Convert` returns an error. Empty = automatic selection. |
| `FontFileBold` | string | Path of the bold face. If `FontFile` is set without `FontFileBold`, bold text uses the regular face. An unreadable file is an error. |
| `Scale` | float64 | Shrinks (below 1) or enlarges (above 1) the content. 0 or less means 1.0; NaN or infinity is an error. |
| `Margin` | float64 (points) | Page margin on all four sides. 0 = no margin (the `body` still has its own 8px margin). |
| `MarkdownCSS` | string | Extra CSS appended after `DefaultMarkdownCSS`. Used only by `ConvertMarkdown` and `ConvertMarkdownFile`; see [markdown.md](markdown.md). |

Functions: `Convert(src []byte, w io.Writer, opts *Options) error` and `ConvertFile(htmlPath, pdfPath string, opts *Options) error` (nil `opts` is allowed; `ConvertFile` writes the file only on success).

### CLI

```
html2pdf [flags] input.html
html2pdf [flags] -e '<html>'
```

| Flag | Default | Meaning |
|---|---|---|
| `-e` | none | HTML source given directly instead of an input file; it is also written to `output.html` in the current directory. Relative paths resolve against the current directory. Cannot be combined with an input file. |
| `-o` | input name with `.pdf` (`output.pdf` with `-e`) | Output PDF path. |
| `-page-size` | `A4` | `A4`, `A5` or `Letter`. |
| `-page-width` | `0` | Page width in millimetres (`Options.PageWidth`). Overrides the width of `-page-size`; `0` keeps the named size. Must be positive. |
| `-page-height` | `0` | Page height in millimetres (`Options.PageHeight`). Overrides the height of `-page-size`; `0` keeps the named size. Must be positive. |
| `-landscape` | off | Landscape orientation. |
| `-scale` | `1.0` | Content scale factor. |
| `-margin` | `0` | Page margin in points, all sides. |
| `-font` | auto-detect | Regular font file (`Options.FontFile`). |
| `-font-bold` | the regular font | Bold font file (`Options.FontFileBold`). |
| `-font-dirs` | none | Comma-separated directories searched first for fonts. They are added in front of the other font directories (see below). |
| `-allow-external-files` | off | Let the document read files outside its own directory (`Options.AllowExternalFiles`). For trusted documents only. |
| `-format` | auto | Output format: `pdf` or `png`. Without this flag, a `-o` name ending in `.png` selects PNG and anything else selects PDF. With `png`, each page is written as a PNG image; a multi-page document gets `-1`, `-2`, ... inserted before the `.png` extension. |
| `-png-scale` | 2 | Rasterization scale for `-format png`, relative to 72-dpi points (2 = 144 dpi). |
| `-verbose` | off | Print the font directories searched and the fonts used to standard error. |

The CLI collects the font directories and passes them as `Options.FontDirs`, in this order: `-font-dirs`, then the environment variable `HTML2PDF_FONT_DIRS` (the OS path list format: `:` on Unix, `;` on Windows), then the operating system's font directories. Directories collected earlier take priority, and a repeated directory keeps its first position. `-font-dirs` no longer replaces the standard directories; it adds directories that are searched first. With `-verbose`, each directory is printed before conversion as `font dir [flag|env|os]: <path>`, and each resolved font after conversion as `font: <family or (default)> (bold) -> <path>`.

The CLI sets `BaseDir` to the input file's directory. `PageWidth` and `PageHeight` are set from `-page-width` and `-page-height`, converted from millimetres to points. Flags may appear before or after the input file.
