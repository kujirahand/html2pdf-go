package text

import (
	"strings"
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
	lineAsc float64       // line-box ascent in font units (OS/2 typo when usable)
	lineDsc float64       // line-box descent in font units, positive
	adv     map[uint16]advEntry
	family  string // typographic family name, "" if the font has none
	bold    bool   // subfamily says Bold, or OS/2 fsSelection has the bold bit
}

// newFace wraps sf. data and off locate the face's table directory (off is
// non-zero for TTC members) so OS/2 typo metrics can be read.
func newFace(sf *sfnt.Font, data []byte, off int) *Face {
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
	f.lineAsc, f.lineDsc = f.ascent, f.descent
	if a, d, ok := typoMetrics(data, off); ok {
		f.lineAsc, f.lineDsc = float64(a), float64(d)
	}
	f.family, f.bold = faceNames(sf, data, off)
	return f
}

// Name IDs of the typographic family and subfamily (the "preferred" names
// that group fonts into families wider than four styles).
const (
	nameIDTypographicFamily    sfnt.NameID = 16
	nameIDTypographicSubfamily sfnt.NameID = 17
)

// faceNames returns the family name (NameID 16, else 1) and whether the face
// is bold: the subfamily (NameID 17, else 2) contains "Bold", or OS/2
// fsSelection has the bold bit (0x20).
func faceNames(sf *sfnt.Font, data []byte, off int) (family string, bold bool) {
	var b sfnt.Buffer
	pick := func(ids ...sfnt.NameID) string {
		for _, id := range ids {
			if s, err := sf.Name(&b, id); err == nil && s != "" {
				return s
			}
		}
		return ""
	}
	family = pick(nameIDTypographicFamily, sfnt.NameIDFamily)
	bold = strings.Contains(strings.ToLower(pick(nameIDTypographicSubfamily, sfnt.NameIDSubfamily)), "bold")
	if o, l, ok := findTable(data, off, "OS/2"); ok && l >= 64 && o+64 <= len(data) {
		bold = bold || data[o+63]&0x20 != 0 // low byte of fsSelection at offset 62
	}
	return family, bold
}

// FamilyName returns the font family name from the name table (the
// typographic family if present), or "" if the font has none.
func (f *Face) FamilyName() string { return f.family }

// findTable locates the table with the given tag in the face whose table
// directory starts at off.
func findTable(data []byte, off int, tag string) (o, l int, ok bool) {
	u16 := func(p int) int { return int(data[p])<<8 | int(data[p+1]) }
	u32 := func(p int) int {
		return int(uint32(data[p])<<24 | uint32(data[p+1])<<16 | uint32(data[p+2])<<8 | uint32(data[p+3]))
	}
	if off < 0 || off+12 > len(data) {
		return 0, 0, false
	}
	n := u16(off + 4)
	for i := 0; i < n; i++ {
		rec := off + 12 + i*16
		if rec+16 > len(data) {
			break
		}
		if string(data[rec:rec+4]) == tag {
			return u32(rec + 8), u32(rec + 12), true
		}
	}
	return 0, 0, false
}

// typoMetrics reads sTypoAscender/sTypoDescender from the OS/2 table of the
// face whose table directory starts at off. ok is false when there is no
// OS/2 table or the ascender is not positive. The returned descender is
// positive.
//
// The typo metrics are used for line layout regardless of the fsSelection
// USE_TYPO_METRICS bit: for CJK fonts such as Noto Sans CJK the hhea/win
// metrics span 1.448em, which makes the CSS "normal" line height far wider
// than the values document layouts are designed for, while the typo
// metrics (880/120 = 1.0em) match the glyph design box. This follows the
// CSS recommendation that "normal" line height sit roughly between 1.0
// and 1.2.
func typoMetrics(data []byte, off int) (asc, desc int, ok bool) {
	o, l, found := findTable(data, off, "OS/2")
	if !found || l < 72 || o < 0 || o+72 > len(data) {
		return 0, 0, false
	}
	u16 := func(p int) int { return int(data[p])<<8 | int(data[p+1]) }
	asc, desc = int(int16(u16(o+68))), -int(int16(u16(o+70)))
	if asc <= 0 {
		return 0, 0, false
	}
	return asc, max(desc, 0), true
}

// UnitsPerEm returns the font design units per em.
func (f *Face) UnitsPerEm() float64 { return f.upem }

// Ascent returns the ascent in points at the given font size.
func (f *Face) Ascent(size float64) float64 { return f.ascent / f.upem * size }

// Descent returns the (positive) descent in points at the given font size.
func (f *Face) Descent(size float64) float64 { return f.descent / f.upem * size }

// LineAscent returns the line-box ascent in points: OS/2 sTypoAscender when
// available, otherwise Ascent.
func (f *Face) LineAscent(size float64) float64 { return f.lineAsc / f.upem * size }

// LineDescent returns the (positive) line-box descent in points: OS/2
// sTypoDescender when available, otherwise Descent.
func (f *Face) LineDescent(size float64) float64 { return f.lineDsc / f.upem * size }

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
