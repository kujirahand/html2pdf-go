package pdf

import (
	"bytes"
	"testing"
)

func TestRoundedRectPDF(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	p := w.AddPage(595, 842)
	p.SetScale(0.5)
	r := [4]float64{8, 0, 8, 1000} // zero is square; huge radii are clamped
	p.SetFillColor(0.9, 0.9, 0.9)
	p.FillRoundedRect(50, 50, 200, 30, r, r)
	p.SetStrokeColor(0, 0, 0.5)
	p.StrokeRoundedRect(50, 100, 200, 30, 1, r, r)
	p.FillStrokeRoundedRect(50, 150, 200, 30, 1, r, r)
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	checkStructure(t, buf.Bytes())
}
