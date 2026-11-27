package pdf

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"unicode/utf16"

	"github.com/kujirahand/html2pdf-go/internal/text"
)

// chunkSize is the number of glyphs per Type3 font object (single-byte codes).
const chunkSize = 256

// Font is a text.Face embedded as Type3 fonts, one per chunk of 256 glyphs.
type Font struct {
	id     int
	face   *text.Face
	chunks []*fontChunk
	index  map[rune]glyphRef
}

type glyphRef struct {
	chunk *fontChunk
	code  byte
}

// fontChunk is one Type3 font object; code i maps to gids[i] and runes[i].
type fontChunk struct {
	font  *Font
	name  string // resource name
	gids  []uint16
	runes []rune
	objID int
}

// NewFont registers f for embedding. Only glyphs drawn via ShowText are embedded.
func (w *Writer) NewFont(f *text.Face) *Font {
	ft := &Font{id: len(w.fonts), face: f, index: make(map[rune]glyphRef)}
	w.fonts = append(w.fonts, ft)
	return ft
}

// code returns the chunk and single-byte code used to draw r.
func (f *Font) code(r rune) (*fontChunk, byte) {
	if g, ok := f.index[r]; ok {
		return g.chunk, g.code
	}
	var c *fontChunk
	if n := len(f.chunks); n > 0 && len(f.chunks[n-1].gids) < chunkSize {
		c = f.chunks[n-1]
	} else {
		c = &fontChunk{font: f, name: fmt.Sprintf("F%d_%d", f.id, len(f.chunks))}
		f.chunks = append(f.chunks, c)
	}
	code := byte(len(c.gids))
	c.gids = append(c.gids, f.face.GlyphID(r))
	c.runes = append(c.runes, r)
	f.index[r] = glyphRef{chunk: c, code: code}
	return c, code
}

func (w *Writer) writeFontChunk(c *fontChunk) {
	face := c.font.face
	upem := face.UnitsPerEm()
	advance := func(gid uint16) float64 {
		if a, ok := face.GlyphAdvanceUnits(gid); ok {
			return a
		}
		return upem * 0.5
	}

	procs := make(map[uint16]int)
	var procOrder []uint16
	bbox := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	for _, gid := range c.gids {
		if _, ok := procs[gid]; ok {
			continue
		}
		id := w.alloc()
		procs[gid] = id
		procOrder = append(procOrder, gid)
		data, box, has := glyphProc(face, gid, advance(gid))
		w.stream(id, "", data, false)
		if has {
			bbox[0] = math.Min(bbox[0], box[0])
			bbox[1] = math.Min(bbox[1], box[1])
			bbox[2] = math.Max(bbox[2], box[2])
			bbox[3] = math.Max(bbox[3], box[3])
		}
	}
	if math.IsInf(bbox[0], 0) {
		bbox = [4]float64{}
	}

	var charProcs, diffs, widths strings.Builder
	for _, gid := range procOrder {
		fmt.Fprintf(&charProcs, " /g%d %d 0 R", gid, procs[gid])
	}
	diffs.WriteString("0")
	for i, gid := range c.gids {
		fmt.Fprintf(&diffs, " /g%d", gid)
		if i > 0 {
			widths.WriteByte(' ')
		}
		widths.WriteString(numP(advance(gid), 3))
	}

	toUni := w.alloc()
	w.stream(toUni, "", []byte(toUnicodeCMap(c.runes)), false)

	w.obj(c.objID, fmt.Sprintf("<< /Type /Font /Subtype /Type3 /FontBBox [%s %s %s %s] /FontMatrix [%s 0 0 %s 0 0] "+
		"/CharProcs <<%s >> /Encoding << /Type /Encoding /Differences [%s] >> "+
		"/FirstChar 0 /LastChar %d /Widths [%s] /ToUnicode %d 0 R >>",
		numP(math.Floor(bbox[0]), 0), numP(math.Floor(bbox[1]), 0),
		numP(math.Ceil(bbox[2]), 0), numP(math.Ceil(bbox[3]), 0),
		numP(1/upem, 10), numP(1/upem, 10),
		charProcs.String(), diffs.String(), len(c.gids)-1, widths.String(), toUni))
}

// glyphProc builds the content stream of a glyph in font units.
// It returns the control-point bounding box and whether the glyph has a path.
func glyphProc(face *text.Face, gid uint16, adv float64) (data []byte, box [4]float64, has bool) {
	var path bytes.Buffer
	var cx, cy float64
	box = [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	pt := func(p text.Point) string {
		box[0] = math.Min(box[0], p.X)
		box[1] = math.Min(box[1], p.Y)
		box[2] = math.Max(box[2], p.X)
		box[3] = math.Max(box[3], p.Y)
		return num(p.X) + " " + num(p.Y)
	}
	for _, s := range face.GlyphOutline(gid) {
		p := s.Points
		switch s.Type {
		case text.SegMoveTo:
			fmt.Fprintf(&path, "%s m\n", pt(p[0]))
			cx, cy = p[0].X, p[0].Y
		case text.SegLineTo:
			fmt.Fprintf(&path, "%s l\n", pt(p[0]))
			cx, cy = p[0].X, p[0].Y
		case text.SegQuadTo:
			// Elevate the quadratic Bezier to a cubic.
			c1 := text.Point{X: cx + 2*(p[0].X-cx)/3, Y: cy + 2*(p[0].Y-cy)/3}
			c2 := text.Point{X: p[1].X + 2*(p[0].X-p[1].X)/3, Y: p[1].Y + 2*(p[0].Y-p[1].Y)/3}
			fmt.Fprintf(&path, "%s %s %s c\n", pt(c1), pt(c2), pt(p[1]))
			cx, cy = p[1].X, p[1].Y
		case text.SegCubeTo:
			fmt.Fprintf(&path, "%s %s %s c\n", pt(p[0]), pt(p[1]), pt(p[2]))
			cx, cy = p[2].X, p[2].Y
		}
	}
	has = path.Len() > 0
	if has {
		path.WriteString("f\n")
	} else {
		box = [4]float64{}
	}
	// d1 declares an uncolored glyph, so it is painted with the current fill color.
	var out bytes.Buffer
	fmt.Fprintf(&out, "%s 0 %s %s %s %s d1\n", numP(adv, 3),
		numP(math.Floor(box[0]), 0), numP(math.Floor(box[1]), 0),
		numP(math.Ceil(box[2]), 0), numP(math.Ceil(box[3]), 0))
	out.Write(path.Bytes())
	return out.Bytes(), box, has
}

// toUnicodeCMap maps single-byte codes (index into runes) to Unicode.
func toUnicodeCMap(runes []rune) string {
	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n" +
		"/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n" +
		"/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n" +
		"1 begincodespacerange\n<00> <FF>\nendcodespacerange\n")
	for start := 0; start < len(runes); start += 100 {
		end := min(start+100, len(runes))
		fmt.Fprintf(&b, "%d beginbfchar\n", end-start)
		for i := start; i < end; i++ {
			fmt.Fprintf(&b, "<%02X> <", i)
			for _, u := range utf16.Encode([]rune{runes[i]}) {
				fmt.Fprintf(&b, "%04X", u)
			}
			b.WriteString(">\n")
		}
		b.WriteString("endbfchar\n")
	}
	b.WriteString("endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n")
	return b.String()
}
