package css

import "testing"

func TestBorderRadius(t *testing.T) {
	px := func(v float64) Length { return Length{v, UnitPX} }
	pc := Length{50, UnitPercent}
	tests := []struct {
		decl   string
		wx, wy [4]Length
	}{
		{"border-radius: 4px", [4]Length{px(4), px(4), px(4), px(4)}, [4]Length{px(4), px(4), px(4), px(4)}},
		{"border-radius: 1px 2px", [4]Length{px(1), px(2), px(1), px(2)}, [4]Length{px(1), px(2), px(1), px(2)}},
		{"border-radius: 1px 2px 3px", [4]Length{px(1), px(2), px(3), px(2)}, [4]Length{px(1), px(2), px(3), px(2)}},
		{"border-radius: 1px 2px 3px 4px", [4]Length{px(1), px(2), px(3), px(4)}, [4]Length{px(1), px(2), px(3), px(4)}},
		{"border-radius: 1px 2px / 3px", [4]Length{px(1), px(2), px(1), px(2)}, [4]Length{px(3), px(3), px(3), px(3)}},
		{"border-radius: 10px/5px 6px 7px 8px", [4]Length{px(10), px(10), px(10), px(10)}, [4]Length{px(5), px(6), px(7), px(8)}},
		{"border-radius: 50%", [4]Length{pc, pc, pc, pc}, [4]Length{pc, pc, pc, pc}},
	}
	for _, tc := range tests {
		doc, cs := compute(t, `<div style="`+tc.decl+`">x</div>`)
		st := cs.Style(find(doc.Body, "div"))
		if st.BorderRadiusX != tc.wx || st.BorderRadiusY != tc.wy {
			t.Errorf("%q: got X=%v Y=%v, want X=%v Y=%v", tc.decl, st.BorderRadiusX, st.BorderRadiusY, tc.wx, tc.wy)
		}
	}
}

func TestBorderRadiusLonghand(t *testing.T) {
	doc, cs := compute(t, `<div style="border-top-left-radius: 2mm; border-top-right-radius: 4px 8px;
		border-bottom-right-radius: 1em; border-bottom-left-radius: 1rem">x</div>`)
	st := cs.Style(find(doc.Body, "div"))
	wantX := [4]Length{{2, UnitMM}, {4, UnitPX}, {1, UnitEM}, {1, UnitREM}}
	wantY := [4]Length{{2, UnitMM}, {8, UnitPX}, {1, UnitEM}, {1, UnitREM}}
	if st.BorderRadiusX != wantX || st.BorderRadiusY != wantY {
		t.Errorf("got X=%v Y=%v, want X=%v Y=%v", st.BorderRadiusX, st.BorderRadiusY, wantX, wantY)
	}
}
