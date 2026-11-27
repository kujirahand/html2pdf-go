package dom

import (
	"strings"
	"testing"
)

func childTags(n *Node) string {
	var out []string
	for _, c := range n.Children {
		if c.IsText() {
			out = append(out, "#"+strings.TrimSpace(c.Text))
		} else {
			out = append(out, c.Tag)
		}
	}
	return strings.Join(out, ",")
}

func TestPageBreakChildrenHoisted(t *testing.T) {
	doc, err := Parse(strings.NewReader("<body><p>a</p><page-break><p>b</p><p>c</p></body>"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := childTags(doc.Body), "p,page-break,p,p"; got != want {
		t.Fatalf("children = %s, want %s", got, want)
	}
	for _, c := range doc.Body.Children {
		if c.Parent != doc.Body {
			t.Errorf("%s has wrong parent", c.Tag)
		}
		if c.Tag == "page-break" && len(c.Children) != 0 {
			t.Error("page-break still has children")
		}
	}
}

func TestPageBreakConsecutiveAndNested(t *testing.T) {
	doc, err := Parse(strings.NewReader("<body><div><page-break><page-break><p>b</p></div><p>c</p></body>"))
	if err != nil {
		t.Fatal(err)
	}
	div := doc.Body.Children[0]
	if got, want := childTags(div), "page-break,page-break,p"; got != want {
		t.Fatalf("div children = %s, want %s", got, want)
	}
}
