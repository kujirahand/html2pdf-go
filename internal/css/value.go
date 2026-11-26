package css

import (
	"strconv"
	"strings"
)

// Unit is a CSS length unit.
type Unit int

const (
	UnitNone Unit = iota // bare number
	UnitPX
	UnitEM
	UnitREM
	UnitPercent
	UnitPT
	UnitMM
	UnitCM
	UnitIN
)

// Length is a CSS length with its unit.
type Length struct {
	Value float64
	Unit  Unit
}

var unitNames = map[string]Unit{
	"": UnitNone, "px": UnitPX, "em": UnitEM, "rem": UnitREM, "%": UnitPercent,
	"pt": UnitPT, "mm": UnitMM, "cm": UnitCM, "in": UnitIN,
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// ParseLength parses a CSS length or bare number. Malformed numbers such as
// "2.em" are rejected, as in CSS.
func ParseLength(s string) (Length, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	digits := 0
	for i < len(s) && isDigit(s[i]) {
		i++
		digits++
	}
	if i+1 < len(s) && s[i] == '.' && isDigit(s[i+1]) {
		i++
		for i < len(s) && isDigit(s[i]) {
			i++
			digits++
		}
	}
	if digits == 0 {
		return Length{}, false
	}
	v, err := strconv.ParseFloat(s[:i], 64)
	if err != nil {
		return Length{}, false
	}
	u, ok := unitNames[s[i:]]
	if !ok {
		return Length{}, false
	}
	return Length{v, u}, true
}

// Resolve converts l to points. fontSize and rootFontSize are in points;
// percentBase is the reference size for percentages.
func (l Length) Resolve(fontSize, rootFontSize, percentBase float64) float64 {
	switch l.Unit {
	case UnitEM:
		return l.Value * fontSize
	case UnitREM:
		return l.Value * rootFontSize
	case UnitPercent:
		return percentBase * l.Value / 100
	case UnitPT:
		return l.Value
	case UnitMM:
		return l.Value * 72 / 25.4
	case UnitCM:
		return l.Value * 720 / 25.4
	case UnitIN:
		return l.Value * 72
	}
	return l.Value * 0.75 // px and bare numbers
}

// Color is an RGB color with components in 0..1.
type Color struct{ R, G, B float64 }

func rgb8(r, g, b int) Color { return Color{float64(r) / 255, float64(g) / 255, float64(b) / 255} }

var namedColors = map[string]Color{
	"black": rgb8(0, 0, 0), "white": rgb8(255, 255, 255), "navy": rgb8(0, 0, 128),
	"red": rgb8(255, 0, 0), "blue": rgb8(0, 0, 255), "green": rgb8(0, 128, 0),
	"gray": rgb8(128, 128, 128), "grey": rgb8(128, 128, 128), "silver": rgb8(192, 192, 192),
	"maroon": rgb8(128, 0, 0), "purple": rgb8(128, 0, 128), "yellow": rgb8(255, 255, 0),
	"orange": rgb8(255, 165, 0), "lime": rgb8(0, 255, 0), "aqua": rgb8(0, 255, 255),
	"cyan": rgb8(0, 255, 255), "fuchsia": rgb8(255, 0, 255), "magenta": rgb8(255, 0, 255),
	"teal": rgb8(0, 128, 128), "olive": rgb8(128, 128, 0), "brown": rgb8(165, 42, 42),
	"pink": rgb8(255, 192, 203), "lightgray": rgb8(211, 211, 211), "lightgrey": rgb8(211, 211, 211),
	"darkgray": rgb8(169, 169, 169), "darkgrey": rgb8(169, 169, 169),
}

// ParseColor parses a named color, #rgb, #rrggbb, or rgb()/rgba().
func ParseColor(s string) (Color, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if c, ok := namedColors[s]; ok {
		return c, true
	}
	if strings.HasPrefix(s, "#") {
		h := s[1:]
		if len(h) == 3 {
			h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
		}
		if len(h) != 6 {
			return Color{}, false
		}
		v, err := strconv.ParseUint(h, 16, 32)
		if err != nil {
			return Color{}, false
		}
		return rgb8(int(v>>16), int(v>>8&0xff), int(v&0xff)), true
	}
	if strings.HasPrefix(s, "rgb") && strings.HasSuffix(s, ")") {
		open := strings.IndexByte(s, '(')
		if open < 0 {
			return Color{}, false
		}
		args := strings.FieldsFunc(s[open+1:len(s)-1], func(r rune) bool { return r == ',' || r == ' ' || r == '/' })
		if len(args) < 3 {
			return Color{}, false
		}
		var comp [3]float64
		for i := 0; i < 3; i++ {
			a := args[i]
			if strings.HasSuffix(a, "%") {
				v, err := strconv.ParseFloat(a[:len(a)-1], 64)
				if err != nil {
					return Color{}, false
				}
				comp[i] = clamp01(v / 100)
			} else {
				v, err := strconv.ParseFloat(a, 64)
				if err != nil {
					return Color{}, false
				}
				comp[i] = clamp01(v / 255)
			}
		}
		return Color{comp[0], comp[1], comp[2]}, true
	}
	return Color{}, false
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
