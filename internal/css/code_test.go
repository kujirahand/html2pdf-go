package css

import (
	"reflect"
	"testing"
)

func TestCodeAndPreUA(t *testing.T) {
	doc, cs := compute(t, `<body><p><code>x</code></p><pre>y</pre></body>`)
	code := cs.Style(find(doc.Body, "code"))
	if !reflect.DeepEqual(code.FontFamily, []string{"monospace"}) {
		t.Errorf("code font-family = %q, want [monospace]", code.FontFamily)
	}
	bg := code.BackgroundColor
	if bg == nil || *bg != rgb8(0xf0, 0xf0, 0xf0) {
		t.Errorf("code background = %v, want #f0f0f0", bg)
	}
	if got := cs.Style(find(doc.Body, "pre")).FontFamily; !reflect.DeepEqual(got, []string{"monospace"}) {
		t.Errorf("pre font-family = %q, want [monospace]", got)
	}
}
