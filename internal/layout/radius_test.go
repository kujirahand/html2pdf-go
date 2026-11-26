package layout

import "testing"

func rectItems(pg *Page) (fill, stroke *RectItem) {
	for _, it := range pg.Items {
		if r, ok := it.(RectItem); ok {
			r := r
			if r.Fill != nil {
				fill = &r
			}
			if r.Stroke != nil {
				stroke = &r
			}
		}
	}
	return fill, stroke
}

func TestBorderRadiusItems(t *testing.T) {
	pages := run(t, `<div style="background: #eee; border: 1px solid black; border-radius: 8px; padding: 10px">hello</div>`)
	fill, stroke := rectItems(pages[0])
	if fill == nil || stroke == nil {
		t.Fatalf("want rounded fill and stroke RectItems, got fill=%v stroke=%v", fill, stroke)
	}
	// 8px = 6pt.
	want := [4]float64{6, 6, 6, 6}
	if fill.RadiusX != want || fill.RadiusY != want {
		t.Errorf("fill radii = %v / %v, want %v", fill.RadiusX, fill.RadiusY, want)
	}
	// The stroke runs along the middle of the border, so its radius is smaller.
	if stroke.RadiusX[0] <= 0 || stroke.RadiusX[0] >= fill.RadiusX[0] {
		t.Errorf("stroke radius %v should be positive and below fill radius %v", stroke.RadiusX[0], fill.RadiusX[0])
	}
}

func TestBorderRadiusScalesDown(t *testing.T) {
	pages := run(t, `<div style="background: #eee; width: 40px; border-radius: 50px">x</div>`)
	fill, _ := rectItems(pages[0])
	if fill == nil {
		t.Fatal("no fill item")
	}
	if fill.RadiusX[0]*2 > fill.W+1e-6 || fill.RadiusY[0]*2 > fill.H+1e-6 {
		t.Errorf("radii %v / %v exceed box %vx%v", fill.RadiusX, fill.RadiusY, fill.W, fill.H)
	}
}

func TestNoBorderRadiusStaysSquare(t *testing.T) {
	pages := run(t, `<div style="background: #eee; border: 1px solid black">hello</div>`)
	for _, it := range pages[0].Items {
		if r, ok := it.(RectItem); ok && (r.RadiusX != [4]float64{} || r.RadiusY != [4]float64{}) {
			t.Errorf("unexpected radius on %+v", r)
		}
	}
}
