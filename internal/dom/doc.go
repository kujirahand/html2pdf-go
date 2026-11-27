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

// Document is a parsed HTML document.
type Document struct {
	Body *Node
	// StylesheetText holds the contents of all <style> elements in document order.
	StylesheetText []string
}

// Parse reads HTML from r and builds a Document.
func Parse(r io.Reader) (*Document, error) {
	root, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	doc := &Document{}
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
				case "style":
					var sb strings.Builder
					for t := c.FirstChild; t != nil; t = t.NextSibling {
						if t.Type == html.TextNode {
							sb.WriteString(t.Data)
						}
					}
					doc.StylesheetText = append(doc.StylesheetText, sb.String())
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
	return doc, nil
}
