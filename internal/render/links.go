package render

import (
	"net/url"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/layout"
	"github.com/kujirahand/html2pdf-go/internal/pdf"
)

func addNavigation(w *pdf.Writer, pages []*layout.Page, pp []*pdf.Page, noBookmarks bool) {
	type target struct {
		page *pdf.Page
		x, y float64
	}
	targets := map[string]target{}
	for i, p := range pages {
		for _, t := range p.Targets {
			if t.ID != "" {
				if _, ok := targets[t.ID]; !ok {
					targets[t.ID] = target{pp[i], t.X, t.Y}
				}
			}
			if !noBookmarks && t.Level > 0 && t.Title != "" {
				w.AddBookmark(t.Title, t.Level, pp[i], t.X, t.Y)
			}
		}
	}
	for i, p := range pages {
		for _, l := range p.Links {
			href := strings.TrimSpace(l.Href)
			if strings.HasPrefix(href, "#") {
				id, err := url.PathUnescape(href[1:])
				if err != nil || id == "" {
					continue
				}
				if t, ok := targets[id]; ok {
					pp[i].AddLink(l.X, l.Y, l.W, l.H, "", t.page, t.x, t.y)
				}
				continue
			}
			if uri := pdf.SafeURI(href); uri != "" {
				pp[i].AddLink(l.X, l.Y, l.W, l.H, uri, nil, 0, 0)
			}
		}
	}
}
