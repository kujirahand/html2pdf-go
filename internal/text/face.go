package text

import (
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// SegType is the kind of an outline segment.
type SegType int

// Outline segment kinds.
const (
	SegMoveTo SegType = iota // 1 point
	SegLineTo                // 1 point
	SegQuadTo                // 2 points: control, end
	SegCubeTo                // 3 points: control 1, control 2, end
)

// Point is a coordinate in font units, Y up.
type Point struct{ X, Y float64 }

// Segment is one outline path segment.
type Segment struct {
	Type   SegType
	Points []Point
}

type advEntry struct {
	w  float64
	ok bool
}

// Face is a single font face. It is safe for concurrent use.
type Face struct {
	sf      *sfnt.Font
	mu      sync.Mutex // guards buf and adv
	buf     sfnt.Buffer
	upem    float64
	ppem    fixed.Int26_6 // ppem == upem, so 26.6 values are font units * 64
	ascent  float64       // font units
	descent float64       // font units, positive
	adv     map[uint16]advEntry
}

func newFace(sf *sfnt.Font) *Face {
	upem := float64(sf.UnitsPerEm())
	if upem <= 0 {
		upem = 1000
	}
	f := &Face{
		sf:   sf,
		upem: upem,
		ppem: fixed.Int26_6(int(upem) << 6),
		adv:  make(map[uint16]advEntry),
	}
	if m, err := sf.Metrics(&f.buf, f.ppem, font.HintingNone); err == nil && m.Ascent > 0 {
		f.ascent = float64(m.Ascent) / 64
		f.descent = float64(m.Descent) / 64
	} else {
		f.ascent = 0.88 * upem
		f.descent = 0.12 * upem
	}
	return f
}

// UnitsPerEm returns the font design units per em.
func (f *Face) UnitsPerEm() float64 { return f.upem }

// Ascent returns the ascent in points at the given font size.
func (f *Face) Ascent(size float64) float64 { return f.ascent / f.upem * size }

// Descent returns the (positive) descent in points at the given font size.
func (f *Face) Descent(size float64) float64 { return f.descent / f.upem * size }

// GlyphID returns the glyph index for r, or 0 (.notdef) if the font has no glyph.
func (f *Face) GlyphID(r rune) uint16 {
	f.mu.Lock()
	defer f.mu.Unlock()
	g, err := f.sf.GlyphIndex(&f.buf, r)
	if err != nil {
		return 0
	}
	return uint16(g)
}

// GlyphAdvanceUnits returns the advance of a glyph in font units.
// ok is false if the advance cannot be determined.
func (f *Face) GlyphAdvanceUnits(gid uint16) (w float64, ok bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if e, hit := f.adv[gid]; hit {
		return e.w, e.ok
	}
	e := advEntry{}
	a, err := f.sf.GlyphAdvance(&f.buf, sfnt.GlyphIndex(gid), f.ppem, font.HintingNone)
	if err == nil {
		e.w = float64(a) / 64
		e.ok = !(gid == 0 && e.w <= 0)
	}
	f.adv[gid] = e
	return e.w, e.ok
}

// Advance returns the advance width of r in points at the given size.
// Missing glyphs use the .notdef width, or size*0.5 if that is unavailable.
func (f *Face) Advance(r rune, size float64) float64 {
	w, ok := f.GlyphAdvanceUnits(f.GlyphID(r))
	if !ok {
		return size * 0.5
	}
	return w / f.upem * size
}

// Measure returns the width of s in points at the given size.
func (f *Face) Measure(s string, size float64) float64 {
	var w float64
	for _, r := range s {
		w += f.Advance(r, size)
	}
	return w
}

// Outline returns the glyph outline of r in font units, Y up.
func (f *Face) Outline(r rune) []Segment {
	return f.GlyphOutline(f.GlyphID(r))
}

// GlyphOutline returns the outline of a glyph in font units, Y up.
func (f *Face) GlyphOutline(gid uint16) []Segment {
	f.mu.Lock()
	defer f.mu.Unlock()
	segs, err := f.sf.LoadGlyph(&f.buf, sfnt.GlyphIndex(gid), f.ppem, nil)
	if err != nil {
		return nil
	}
	out := make([]Segment, 0, len(segs))
	for _, s := range segs {
		var t SegType
		var n int
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			t, n = SegMoveTo, 1
		case sfnt.SegmentOpLineTo:
			t, n = SegLineTo, 1
		case sfnt.SegmentOpQuadTo:
			t, n = SegQuadTo, 2
		case sfnt.SegmentOpCubeTo:
			t, n = SegCubeTo, 3
		default:
			continue
		}
		pts := make([]Point, n)
		for i := range pts {
			// sfnt returns Y pointing down; flip to Y up.
			pts[i] = Point{
				X: float64(s.Args[i].X) / 64,
				Y: -float64(s.Args[i].Y) / 64,
			}
		}
		out = append(out, Segment{Type: t, Points: pts})
	}
	return out
}
