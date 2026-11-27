package css

import "strings"

// RootFontSize is the root font size in points (16px).
const RootFontSize = 12.0

// Side indexes for the [4] arrays in Style.
const (
	Top = iota
	Right
	Bottom
	Left
)

// Style is the computed style of a node. FontSize is always in points
// (UnitPT); other lengths are kept unresolved so layout can resolve them.
type Style struct {
	Display         string
	FontSize        Length
	FontWeight      int
	Color           Color
	BackgroundColor *Color
	TextAlign       string // left, right, center
	TextDecoration  string // none, underline
	// LineHeight is a multiplier of the font size; LineHeightPt is an absolute
	// height. Both zero means the font's default line height.
	LineHeight     float64
	LineHeightPt   float64
	Margin         [4]Length
	Padding        [4]Length
	BorderWidth    [4]Length
	BorderColor    [4]Color
	BorderStyle    [4]string // "none", "solid", "dashed", "dotted" or "double"
	Width          *Length   // nil means auto
	PageBreakAfter bool
	// BorderRadiusX and BorderRadiusY are the horizontal and vertical corner
	// radii, ordered top-left, top-right, bottom-right, bottom-left.
	BorderRadiusX [4]Length
	BorderRadiusY [4]Length

	borderColorSet [4]bool
	lhLen          *Length
}

// Bold reports whether the font weight is bold.
func (s *Style) Bold() bool { return s.FontWeight >= 700 }

func defaultStyle() *Style {
	s := &Style{
		Display:        "inline",
		FontSize:       Length{RootFontSize, UnitPT},
		FontWeight:     400,
		TextAlign:      "left",
		TextDecoration: "none",
	}
	s.resetBoxes()
	return s
}

func (s *Style) resetBoxes() {
	for i := 0; i < 4; i++ {
		s.BorderWidth[i] = Length{3, UnitPX}
		s.BorderStyle[i] = "none"
	}
}

// inheritFrom returns a new style that carries inherited properties of p.
func inheritFrom(p *Style) *Style {
	s := defaultStyle()
	s.FontSize = p.FontSize
	s.FontWeight = p.FontWeight
	s.Color = p.Color
	s.TextAlign = p.TextAlign
	s.TextDecoration = p.TextDecoration
	s.LineHeight = p.LineHeight
	s.LineHeightPt = p.LineHeightPt
	return s
}

// splitTokens splits on whitespace outside parentheses.
func splitTokens(v string) []string {
	var out []string
	depth, start := 0, -1
	for i := 0; i < len(v); i++ {
		c := v[i]
		switch {
		case c == '(':
			depth++
		case c == ')':
			if depth > 0 {
				depth--
			}
		}
		space := depth == 0 && (c == ' ' || c == '\t' || c == '\n' || c == '\r')
		if space {
			if start >= 0 {
				out = append(out, v[start:i])
				start = -1
			}
		} else if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, v[start:])
	}
	return out
}

// expand4 maps 1-4 values to top, right, bottom, left.
func expand4[T any](v []T) ([4]T, bool) {
	var o [4]T
	switch len(v) {
	case 1:
		o = [4]T{v[0], v[0], v[0], v[0]}
	case 2:
		o = [4]T{v[0], v[1], v[0], v[1]}
	case 3:
		o = [4]T{v[0], v[1], v[2], v[1]}
	case 4:
		o = [4]T{v[0], v[1], v[2], v[3]}
	default:
		return o, false
	}
	return o, true
}

func parseBoxLengths(v string) ([4]Length, bool) {
	var ls []Length
	for _, t := range splitTokens(v) {
		if strings.EqualFold(t, "auto") {
			ls = append(ls, Length{})
			continue
		}
		l, ok := ParseLength(t)
		if !ok {
			return [4]Length{}, false
		}
		ls = append(ls, l)
	}
	return expand4(ls)
}

var sideNames = map[string]int{"top": Top, "right": Right, "bottom": Bottom, "left": Left}

var fontSizeKeywords = map[string]Length{
	"xx-small": {9, UnitPX}, "x-small": {10, UnitPX}, "small": {13, UnitPX}, "medium": {16, UnitPX},
	"large": {18, UnitPX}, "x-large": {24, UnitPX}, "xx-large": {32, UnitPX},
	"smaller": {0.8333, UnitEM}, "larger": {1.2, UnitEM},
}

func parseBorderStyle(t string) (string, bool) {
	switch strings.ToLower(t) {
	case "none", "hidden":
		return "none", true
	case "dashed", "dotted", "double":
		return strings.ToLower(t), true
	case "solid", "groove", "ridge", "inset", "outset":
		return "solid", true
	}
	return "", false
}

func parseBorderWidth(t string) (Length, bool) {
	switch strings.ToLower(t) {
	case "thin":
		return Length{1, UnitPX}, true
	case "medium":
		return Length{3, UnitPX}, true
	case "thick":
		return Length{5, UnitPX}, true
	}
	return ParseLength(t)
}

// setBorderSide applies a border shorthand value to side i.
func (s *Style) setBorderSide(i int, v string) {
	w, st := Length{3, UnitPX}, "none"
	var col Color
	colSet := false
	for _, t := range splitTokens(v) {
		if x, ok := parseBorderStyle(t); ok {
			st = x
		} else if l, ok := parseBorderWidth(t); ok {
			w = l
		} else if c, ok := ParseColor(t); ok {
			col, colSet = c, true
		}
	}
	s.BorderWidth[i], s.BorderStyle[i] = w, st
	s.BorderColor[i], s.borderColorSet[i] = col, colSet
}

// Corner indexes for Style.BorderRadiusX and BorderRadiusY.
const (
	TopLeft = iota
	TopRight
	BottomRight
	BottomLeft
)

// expandCorners maps 1-4 values to top-left, top-right, bottom-right,
// bottom-left.
func expandCorners(v []Length) ([4]Length, bool) {
	var o [4]Length
	switch len(v) {
	case 1:
		o = [4]Length{v[0], v[0], v[0], v[0]}
	case 2:
		o = [4]Length{v[0], v[1], v[0], v[1]}
	case 3:
		o = [4]Length{v[0], v[1], v[2], v[1]}
	case 4:
		o = [4]Length{v[0], v[1], v[2], v[3]}
	default:
		return o, false
	}
	return o, true
}

func parseRadiusList(v string) ([]Length, bool) {
	var ls []Length
	for _, t := range splitTokens(v) {
		l, ok := ParseLength(t)
		if !ok || l.Value < 0 {
			return nil, false
		}
		ls = append(ls, l)
	}
	return ls, len(ls) > 0
}

// setBorderRadius applies the border-radius shorthand: X radii, optionally
// followed by "/" and Y radii (which default to the X radii).
func (s *Style) setBorderRadius(val string) {
	xs, ys := val, ""
	hasY := false
	if i := strings.Index(val, "/"); i >= 0 {
		xs, ys, hasY = val[:i], val[i+1:], true
	}
	xl, ok := parseRadiusList(xs)
	if !ok {
		return
	}
	x, ok := expandCorners(xl)
	if !ok {
		return
	}
	y := x
	if hasY {
		yl, ok := parseRadiusList(ys)
		if !ok {
			return
		}
		if y, ok = expandCorners(yl); !ok {
			return
		}
	}
	s.BorderRadiusX, s.BorderRadiusY = x, y
}

// setCornerRadius applies a border-<corner>-radius longhand: X [Y].
func (s *Style) setCornerRadius(i int, val string) {
	ls, ok := parseRadiusList(val)
	if !ok || len(ls) > 2 {
		return
	}
	s.BorderRadiusX[i], s.BorderRadiusY[i] = ls[0], ls[len(ls)-1]
}

var cornerProps = map[string]int{
	"border-top-left-radius":     TopLeft,
	"border-top-right-radius":    TopRight,
	"border-bottom-right-radius": BottomRight,
	"border-bottom-left-radius":  BottomLeft,
}

// apply applies one declaration. parentFontSize is in points.
func (s *Style) apply(prop, val string, parentFontSize float64) {
	lv := strings.ToLower(val)
	switch prop {
	case "display":
		s.Display = lv
	case "font-size":
		l, ok := fontSizeKeywords[lv]
		if !ok {
			l, ok = ParseLength(val)
			if !ok || l.Unit == UnitNone || l.Value < 0 {
				return
			}
		}
		s.FontSize = Length{l.Resolve(parentFontSize, RootFontSize, parentFontSize), UnitPT}
	case "font-weight":
		switch lv {
		case "bold", "bolder":
			s.FontWeight = 700
		case "normal", "lighter":
			s.FontWeight = 400
		default:
			if l, ok := ParseLength(val); ok && l.Unit == UnitNone {
				s.FontWeight = int(l.Value)
			}
		}
	case "color":
		if c, ok := ParseColor(val); ok {
			s.Color = c
		}
	case "background-color", "background":
		for _, t := range splitTokens(val) {
			if c, ok := ParseColor(t); ok {
				c := c
				s.BackgroundColor = &c
				return
			}
		}
		if lv == "transparent" || lv == "none" {
			s.BackgroundColor = nil
		}
	case "text-align":
		switch lv {
		case "left", "right", "center":
			s.TextAlign = lv
		case "justify", "start":
			s.TextAlign = "left"
		case "end":
			s.TextAlign = "right"
		}
	case "text-decoration", "text-decoration-line":
		if strings.Contains(lv, "underline") {
			s.TextDecoration = "underline"
		} else if strings.Contains(lv, "none") {
			s.TextDecoration = "none"
		}
	case "line-height":
		if lv == "normal" {
			s.LineHeight, s.LineHeightPt, s.lhLen = 0, 0, nil
			return
		}
		l, ok := ParseLength(val)
		if !ok || l.Value < 0 {
			return
		}
		if l.Unit == UnitNone {
			s.LineHeight, s.LineHeightPt, s.lhLen = l.Value, 0, nil
		} else {
			s.lhLen = &l
		}
	case "margin":
		if b, ok := parseBoxLengths(val); ok {
			s.Margin = b
		}
	case "padding":
		if b, ok := parseBoxLengths(val); ok {
			s.Padding = b
		}
	case "width":
		if lv == "auto" {
			s.Width = nil
		} else if l, ok := ParseLength(val); ok && l.Value >= 0 {
			s.Width = &l
		}
	case "page-break-after", "break-after":
		switch lv {
		case "always", "page", "left", "right":
			s.PageBreakAfter = true
		case "auto", "avoid":
			s.PageBreakAfter = false
		}
	case "border-radius":
		s.setBorderRadius(val)
	case "border":
		for i := 0; i < 4; i++ {
			s.setBorderSide(i, val)
		}
	case "border-width":
		var ls []Length
		for _, t := range splitTokens(val) {
			l, ok := parseBorderWidth(t)
			if !ok {
				return
			}
			ls = append(ls, l)
		}
		if b, ok := expand4(ls); ok {
			s.BorderWidth = b
		}
	case "border-style":
		var ss []string
		for _, t := range splitTokens(val) {
			x, ok := parseBorderStyle(t)
			if !ok {
				return
			}
			ss = append(ss, x)
		}
		if b, ok := expand4(ss); ok {
			s.BorderStyle = b
		}
	case "border-color":
		var cs []Color
		for _, t := range splitTokens(val) {
			c, ok := ParseColor(t)
			if !ok {
				return
			}
			cs = append(cs, c)
		}
		if b, ok := expand4(cs); ok {
			s.BorderColor = b
			s.borderColorSet = [4]bool{true, true, true, true}
		}
	default:
		if i, ok := cornerProps[prop]; ok {
			s.setCornerRadius(i, val)
			return
		}
		s.applySide(prop, val)
	}
}

// applySide handles margin-top, border-left-width, and similar per-side properties.
func (s *Style) applySide(prop, val string) {
	parts := strings.Split(prop, "-")
	if len(parts) < 2 {
		return
	}
	i, ok := sideNames[parts[len(parts)-1]]
	if ok && len(parts) == 2 {
		switch parts[0] {
		case "margin", "padding":
			l, ok := ParseLength(val)
			if !ok && strings.EqualFold(val, "auto") {
				l, ok = Length{}, true
			}
			if !ok {
				return
			}
			if parts[0] == "margin" {
				s.Margin[i] = l
			} else {
				s.Padding[i] = l
			}
		case "border":
			s.setBorderSide(i, val)
		}
		return
	}
	// border-<side>-<width|style|color>
	if len(parts) == 3 && parts[0] == "border" {
		i, ok := sideNames[parts[1]]
		if !ok {
			return
		}
		switch parts[2] {
		case "width":
			if l, ok := parseBorderWidth(val); ok {
				s.BorderWidth[i] = l
			}
		case "style":
			if x, ok := parseBorderStyle(val); ok {
				s.BorderStyle[i] = x
			}
		case "color":
			if c, ok := ParseColor(val); ok {
				s.BorderColor[i], s.borderColorSet[i] = c, true
			}
		}
	}
}

// finish resolves values that depend on the final font size or color.
func (s *Style) finish() {
	if s.lhLen != nil {
		fs := s.FontSize.Value
		s.LineHeightPt = s.lhLen.Resolve(fs, RootFontSize, fs)
		s.LineHeight = 0
		s.lhLen = nil
	}
	for i := 0; i < 4; i++ {
		if !s.borderColorSet[i] {
			s.BorderColor[i] = s.Color
		}
	}
}
