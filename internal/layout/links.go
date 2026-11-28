package layout

import (
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/dom"
)

func nodeText(n *dom.Node) string {
	var b strings.Builder
	var walk func(*dom.Node)
	walk = func(n *dom.Node) {
		if n.IsText() {
			b.WriteString(n.Text)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}
