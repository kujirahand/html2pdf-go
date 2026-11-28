package layout

import (
	"fmt"
	"unicode"

	"github.com/kujirahand/html2pdf-go/internal/text"
	"github.com/kujirahand/html2pdf-go/internal/warn"
)

// WarnMissingGlyphs checks only final drawing items, not repeated layout measurements.
func WarnMissingGlyphs(pages []*Page, fonts *text.FontSet, w *warn.Collector) {
	if w == nil {
		return
	}
	type key struct {
		face *text.Face
		r    rune
	}
	cache := map[key]bool{}
	for _, p := range pages {
		for _, item := range p.Items {
			t, ok := item.(TextItem)
			if !ok {
				continue
			}
			face := t.Face
			if face == nil {
				face = fonts.Regular
				if t.Bold {
					face = fonts.Bold
				}
			}
			if face == nil {
				continue
			}
			for _, r := range t.Text {
				if unicode.IsSpace(r) || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
					continue
				}
				k := key{face, r}
				missing, ok := cache[k]
				if !ok {
					missing = face.GlyphID(r) == 0
					cache[k] = missing
				}
				if missing {
					id := fmt.Sprintf("U+%04X", r)
					w.Add("missing-glyph", id, fmt.Sprintf("missing glyph: %s (%c)", id, r))
				}
			}
		}
	}
}
