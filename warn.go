package html2pdf

import (
	"fmt"

	"github.com/kujirahand/html2pdf-go/internal/warn"
)

const (
	WarningKindCSSProperty    = "css-property"
	WarningKindCSSValue       = "css-value"
	WarningKindCSSAtRule      = "css-at-rule"
	WarningKindHTMLAttribute  = "html-attribute"
	WarningKindRemoteResource = "remote-resource"
	WarningKindImageDecode    = "image-decode"
	WarningKindMissingGlyph   = "missing-glyph"
)

// Warning describes occurrences of an unsupported or ignored item.
type Warning struct {
	Kind, Key, Message string
	Count              int
}

// String formats the warning for display.
func (w Warning) String() string {
	s := "warning: " + w.Message
	if w.Count > 1 {
		s += fmt.Sprintf(" (%d times)", w.Count)
	}
	return s
}

// warningCollector returns a collector for opts.OnWarning (nil when unset) and a
// flush func that reports the aggregated warnings.
func warningCollector(o *Options) (*warn.Collector, func()) {
	if o.OnWarning == nil {
		return nil, func() {}
	}
	c := &warn.Collector{}
	return c, func() {
		for _, v := range c.Sorted() {
			o.OnWarning(Warning(v))
		}
	}
}
