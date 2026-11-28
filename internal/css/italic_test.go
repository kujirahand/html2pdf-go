package css

import "testing"

func TestItalicStyles(t *testing.T) {
	doc, cs := compute(t, `<p><em><i>nested</i><span style="font-style:normal">normal</span><b style="font-style:invalid">inherited</b></em></p><cite>x</cite><dfn>x</dfn><var>x</var><address>x</address><div style="font-style:oblique"><small>x</small></div>`)
	for _, tag := range []string{"em", "i", "cite", "dfn", "var", "address", "b"} {
		if !cs.Style(find(doc.Body, tag)).Italic() {
			t.Errorf("%s should be italic", tag)
		}
	}
	for _, tag := range []string{"p", "span"} {
		if cs.Style(find(doc.Body, tag)).FontStyle != "normal" {
			t.Errorf("%s should be normal", tag)
		}
	}
	if cs.Style(find(doc.Body, "small")).FontStyle != "oblique" {
		t.Error("oblique should inherit")
	}
}
