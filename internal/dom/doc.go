// Package dom parses HTML into a small node tree built on golang.org/x/net/html.
package dom

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// Node is an element or text node.
type Node struct {
	// Tag is the lower-case element name; empty for text nodes.
	Tag      string
	Attr     map[string]string
	Children []*Node
	Parent   *Node
	// Text is the character data of a text node.
	Text string
}

// IsText reports whether n is a text node.
func (n *Node) IsText() bool { return n.Tag == "" }

// SheetRef is one stylesheet source: a <style> element or a <link
// rel="stylesheet">.
type SheetRef struct {
	// Href is the link target; empty for a <style> element.
	Href string
	// Text is the contents of a <style> element; empty for a link.
	Text string
}

// Document is a parsed HTML document.
type Document struct {
	Body *Node
	// Title is the text of the first title element.
	Title string
	// Frameset is the first <frameset> element, or nil for a normal document.
	Frameset *Node
	// StylesheetText holds the contents of all <style> elements in document order.
	StylesheetText []string
	// Sheets lists the <style> elements and stylesheet <link> elements in
	// document order.
	Sheets []SheetRef
}

// isStylesheetLink reports whether attr describes a <link> that loads an
// author stylesheet: rel contains "stylesheet" but not "alternate", and href
// is present.
func isStylesheetLink(attr map[string]string) bool {
	if strings.TrimSpace(attr["href"]) == "" {
		return false
	}
	var sheet, alt bool
	for _, tok := range strings.Fields(strings.ToLower(attr["rel"])) {
		switch tok {
		case "stylesheet":
			sheet = true
		case "alternate":
			alt = true
		}
	}
	return sheet && !alt
}

// Parse reads HTML from r and builds a Document.
func Parse(r io.Reader) (*Document, error) {
	root, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	doc := &Document{}
	titleFound := false
	var walk func(h *html.Node, parent *Node)
	walk = func(h *html.Node, parent *Node) {
		for c := h.FirstChild; c != nil; c = c.NextSibling {
			switch c.Type {
			case html.TextNode:
				if parent != nil {
					parent.Children = append(parent.Children, &Node{Text: c.Data, Parent: parent})
				}
			case html.ElementNode:
				switch c.Data {
				case "title":
					if !titleFound {
						titleFound = true
						for t := c.FirstChild; t != nil; t = t.NextSibling {
							if t.Type == html.TextNode {
								doc.Title += t.Data
							}
						}
					}
				case "style":
					var sb strings.Builder
					for t := c.FirstChild; t != nil; t = t.NextSibling {
						if t.Type == html.TextNode {
							sb.WriteString(t.Data)
						}
					}
					doc.StylesheetText = append(doc.StylesheetText, sb.String())
					doc.Sheets = append(doc.Sheets, SheetRef{Text: sb.String()})
					continue
				case "script":
					continue
				}
				n := &Node{Tag: c.Data, Attr: map[string]string{}, Parent: parent}
				for _, a := range c.Attr {
					n.Attr[strings.ToLower(a.Key)] = a.Val
				}
				if c.Data == "body" && doc.Body == nil {
					doc.Body = n
				}
				if c.Data == "frameset" && doc.Frameset == nil {
					doc.Frameset = n
				}
				if c.Data == "link" && isStylesheetLink(n.Attr) {
					doc.Sheets = append(doc.Sheets, SheetRef{Href: strings.TrimSpace(n.Attr["href"])})
				}
				if parent != nil {
					parent.Children = append(parent.Children, n)
				}
				walk(c, n)
			}
			// Comments, doctypes, and other node types are dropped.
		}
	}
	walk(root, nil)
	if doc.Body == nil {
		doc.Body = &Node{Tag: "body", Attr: map[string]string{}}
	}
	hoistPageBreakChildren(doc.Body)
	return doc, nil
}

// hoistPageBreakChildren moves the children of every <page-break> element
// to just after it, keeping their order. The HTML parser treats the unknown
// element as a container, so <page-break><p>b</p> would otherwise nest the
// paragraph inside it; hoisting makes the element behave as if it were void.
func hoistPageBreakChildren(n *Node) {
	for i := 0; i < len(n.Children); i++ {
		c := n.Children[i]
		if c.Tag == "page-break" && len(c.Children) > 0 {
			moved := c.Children
			c.Children = nil
			rest := append([]*Node{}, n.Children[i+1:]...)
			n.Children = append(append(n.Children[:i+1], moved...), rest...)
			for _, m := range moved {
				m.Parent = n
			}
		}
	}
	for _, c := range n.Children {
		hoistPageBreakChildren(c)
	}
}
