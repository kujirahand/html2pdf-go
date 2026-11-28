package html2pdf

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// DefaultMarkdownCSS is the stylesheet embedded in documents produced by
// MarkdownToHTML. It only uses CSS that the engine supports. Extra CSS given
// in Options.MarkdownCSS is appended after it, so it overrides these rules.
const DefaultMarkdownCSS = `
body { font-size: 11pt; line-height: 1.6; color: #222222; }
h1 { font-size: 22pt; margin: 0.8em 0 0.5em 0; padding-bottom: 4px; border-bottom: 2px solid #2f5496; }
h2 { font-size: 17pt; margin: 1em 0 0.5em 0; padding-bottom: 3px; border-bottom: 1px solid #999999; }
h3 { font-size: 14pt; margin: 1em 0 0.4em 0; }
h4 { font-size: 12pt; margin: 1em 0 0.4em 0; }
h5 { font-size: 11pt; margin: 1em 0 0.4em 0; }
h6 { font-size: 10pt; margin: 1em 0 0.4em 0; color: #555555; }
p { margin: 0.6em 0; }
a { color: #1a5fb4; }
ul, ol { margin: 0.6em 0; padding-left: 28px; }
li { margin: 0.2em 0; }
blockquote { margin: 0.8em 0; padding: 6px 14px; border-left: 4px solid #aab4c8; background-color: #f3f5f9; color: #444444; }
pre { margin: 0.8em 0; padding: 10px 12px; background-color: #f0f0f0; font-size: 10pt; line-height: 1.4; }
code { background-color: #f0f0f0; font-size: 10pt; }
table { margin: 0.8em 0; border: 1px solid #999999; width: 100%; }
th, td { border: 1px solid #999999; padding: 4px 8px; }
th { background-color: #e8ecf4; }
hr { display: block; margin: 1.2em 0; border-top: 1px solid #999999; }
`

// MarkdownToHTML converts the Markdown document src (CommonMark plus the
// GitHub Flavored Markdown tables, strikethrough and task lists) into a
// complete HTML document that embeds DefaultMarkdownCSS. Raw HTML in the
// source passes through unchanged, so only convert trusted input. Raw HTML
// can reference local files (images, stylesheets, frames); when converting,
// these are confined to Options.BaseDir unless Options.AllowExternalFiles is
// set.
func MarkdownToHTML(src []byte) ([]byte, error) {
	return markdownDocument(src, "")
}

// markdownDocument renders src as an HTML document, appending extraCSS after
// DefaultMarkdownCSS.
func markdownDocument(src []byte, extraCSS string) ([]byte, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
	var body bytes.Buffer
	if err := md.Convert(src, &body); err != nil {
		return nil, err
	}

	var doc bytes.Buffer
	doc.WriteString("<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n<style>")
	doc.WriteString(DefaultMarkdownCSS)
	if extraCSS != "" {
		doc.WriteString("\n")
		doc.WriteString(extraCSS)
	}
	doc.WriteString("</style>\n</head>\n<body>\n")
	doc.Write(replaceTaskListChecks(preservePreBlocks(body.Bytes())))
	doc.WriteString("</body>\n</html>\n")
	return doc.Bytes(), nil
}

// replaceTaskListChecks rewrites GFM task-list checkboxes as text markers,
// because the engine does not render form controls: without this, checked
// and unchecked items would look exactly the same in the PDF.
func replaceTaskListChecks(htmlSrc []byte) []byte {
	s := string(htmlSrc)
	s = strings.ReplaceAll(s, `<input checked="" disabled="" type="checkbox">`, "[x]")
	s = strings.ReplaceAll(s, `<input disabled="" type="checkbox">`, "[ ]")
	return []byte(s)
}

// indexFold returns the index of the first ASCII case-insensitive match of
// sub in s at or after from, or -1.
func indexFold(s, sub string, from int) int {
	for i := from; i+len(sub) <= len(s); i++ {
		ok := true
		for j := 0; j < len(sub); j++ {
			a, b := s[i+j], sub[j]
			if 'A' <= a && a <= 'Z' {
				a += 'a' - 'A'
			}
			if 'A' <= b && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

// tagEndIndex returns the index of the '>' that closes the tag starting at
// or before from, skipping '>' inside quoted attribute values, or -1.
func tagEndIndex(s string, from int) int {
	var quote byte
	for i := from; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
		} else if c == '>' {
			return i
		}
	}
	return -1
}

// preservePreBlocks rewrites every <pre>...</pre> block so that its line
// breaks and spaces survive the layout engine, which collapses whitespace:
// lines are joined with <br>, tabs become four spaces and every space becomes
// &nbsp;. Empty lines become a lone &nbsp; so that they keep their height.
func preservePreBlocks(htmlSrc []byte) []byte {
	const (
		preOpen   = "<pre"
		preClose  = "</pre>"
		codeOpen  = "<code"
		codeClose = "</code>"
	)
	s := string(htmlSrc)
	var out strings.Builder
	for {
		start := indexFold(s, preOpen, 0)
		if start < 0 {
			break
		}
		// The tag must be exactly <pre> or <pre ...>, not e.g. <prefix>.
		tagEnd := tagEndIndex(s, start)
		if tagEnd < 0 {
			break
		}
		if name := s[start+len(preOpen) : tagEnd]; name != "" && name[0] != ' ' && name[0] != '\n' && name[0] != '\t' && name[0] != '\r' {
			out.WriteString(s[:tagEnd+1])
			s = s[tagEnd+1:]
			continue
		}
		end := indexFold(s, preClose, tagEnd)
		if end < 0 {
			break
		}

		inner := s[tagEnd+1 : end]
		if indexFold(inner, codeOpen, 0) == 0 {
			if i := tagEndIndex(inner, 0); i >= 0 {
				inner = inner[i+1:]
			}
		}
		if len(inner) >= len(codeClose) && indexFold(inner[len(inner)-len(codeClose):], codeClose, 0) == 0 {
			inner = inner[:len(inner)-len(codeClose)]
		}
		inner = strings.ReplaceAll(inner, "\r", "")

		lines := strings.Split(inner, "\n")
		if len(lines) > 1 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		for i, line := range lines {
			line = strings.ReplaceAll(line, "\t", "    ")
			line = strings.ReplaceAll(line, " ", "&nbsp;")
			if line == "" {
				line = "&nbsp;"
			}
			lines[i] = line
		}

		out.WriteString(s[:tagEnd+1])
		out.WriteString(strings.Join(lines, "<br>"))
		out.WriteString(preClose)
		s = s[end+len(preClose):]
	}
	out.WriteString(s)
	return []byte(out.String())
}

// ConvertMarkdown renders the Markdown document src as a PDF and writes it to
// w. It converts the Markdown to HTML with MarkdownToHTML and then calls
// Convert. If opts.MarkdownCSS is set it is appended after DefaultMarkdownCSS.
func ConvertMarkdown(src []byte, w io.Writer, opts *Options) error {
	if opts == nil {
		opts = &Options{}
	}
	doc, err := markdownDocument(src, opts.MarkdownCSS)
	if err != nil {
		return err
	}
	return Convert(doc, w, opts)
}

// ConvertMarkdownFile reads the Markdown file at mdPath and writes the PDF to
// pdfPath. Relative image paths are resolved against opts.BaseDir, which
// defaults to the Markdown file's directory.
func ConvertMarkdownFile(mdPath, pdfPath string, opts *Options) error {
	src, err := os.ReadFile(mdPath)
	if err != nil {
		return err
	}
	// Work on a copy so the caller's Options is never modified.
	var o Options
	if opts != nil {
		o = *opts
	}
	if o.BaseDir == "" {
		o.BaseDir = filepath.Dir(mdPath)
	}
	// Render into memory first so a failed conversion leaves no partial file.
	var buf bytes.Buffer
	if err := ConvertMarkdown(src, &buf, &o); err != nil {
		return err
	}
	return os.WriteFile(pdfPath, buf.Bytes(), 0o644)
}
