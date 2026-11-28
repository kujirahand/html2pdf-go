package html2pdf

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kujirahand/html2pdf-go/internal/css"
	"github.com/kujirahand/html2pdf-go/internal/layout"
	"github.com/kujirahand/html2pdf-go/internal/text"
)

var now = time.Now

func bandMargin(m float64, value string) float64 {
	if value != "" && m < 24 {
		return 24
	}
	return m
}

func addPageBands(pages []*layout.Page, fonts *text.FontSet, opts *Options, title string) {
	if opts.Header == "" && opts.Footer == "" {
		return
	}
	date := now().Local().Format("2006-01-02")
	face := fonts.FaceFor(nil, false)
	for i, page := range pages {
		scale := page.Scale
		if scale <= 0 {
			scale = 1
		}
		// Clip body ink to the reserved bands, including glyphs that extend
		// beyond their line boxes and oversized content.
		top, bottom := 0.0, 0.0
		if opts.Header != "" {
			top = bandMargin(opts.Margin, opts.Header)
		}
		if opts.Footer != "" {
			bottom = bandMargin(opts.Margin, opts.Footer)
		}
		height := math.Max(0, page.Height-top-bottom)
		page.Items = append([]layout.Item{layout.ClipItem{X: 0, Y: top / scale, W: page.Width / scale, H: height / scale}}, page.Items...)
		page.Items = append(page.Items, layout.ClipItem{End: true})
		for _, b := range []struct {
			value string
			top   bool
		}{{opts.Header, true}, {opts.Footer, false}} {
			if b.value == "" {
				continue
			}
			value := strings.NewReplacer("{page}", fmt.Sprint(i+1), "{pages}", fmt.Sprint(len(pages)), "{title}", title, "{date}", date, "{file}", opts.FileName).Replace(b.value)
			value = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ", "\t", " ").Replace(value)
			width := page.Width - 2*opts.Margin
			if width < 0 {
				width = 0
			}
			if face.Measure(value, 9) > width {
				runes := []rune(value)
				if face.Measure("…", 9) > width {
					value = ""
				} else {
					remaining := width - face.Measure("…", 9)
					used, end := 0.0, 0
					for end < len(runes) {
						advance := face.Advance(runes[end], 9)
						if used+advance > remaining {
							break
						}
						used += advance
						end++
					}
					runes = runes[:end]
					value = string(runes) + "…"
				}
			}
			band := bandMargin(opts.Margin, b.value)
			center := band / 2
			if !b.top {
				center = page.Height - band/2
			}
			baseline := center + (face.Ascent(9)-face.Descent(9))/2
			page.Items = append(page.Items, layout.TextItem{Text: value, X: (page.Width - face.Measure(value, 9)) / 2 / scale, Y: baseline / scale, Size: 9 / scale, Face: face, Color: css.Color{R: 0x44 / 255.0, G: 0x44 / 255.0, B: 0x44 / 255.0}})
		}
	}
}
