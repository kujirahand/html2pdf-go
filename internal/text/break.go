package text

import "strings"

const (
	// Characters that must not start a line.
	noStartChars = "、。，．・：；？！ー〜）〕］｝」』】〉》ゃゅょぁぃぅぇぉっャュョァィゥェォッゝゞ々" + ",.:;?!)]}"
	// Characters that must not end a line.
	noEndChars = "（〔［｛「『【〈《" + "([{"
)

// maxHang is how many line-start-prohibited characters may overflow a line.
const maxHang = 2

func noStart(r rune) bool { return strings.ContainsRune(noStartChars, r) }

func noEnd(r rune) bool { return strings.ContainsRune(noEndChars, r) }

// isWide reports whether r is a CJK-like character that may break on either side.
func isWide(r rune) bool {
	switch {
	case r >= 0x2E80 && r <= 0x9FFF,
		r >= 0xAC00 && r <= 0xD7AF,
		r >= 0xF900 && r <= 0xFAFF,
		r >= 0xFE30 && r <= 0xFE4F,
		r >= 0xFF00 && r <= 0xFFEF,
		r >= 0x20000 && r <= 0x3FFFF:
		return true
	}
	return false
}

// canBreak reports whether a line may break between rs[i-1] and rs[i].
func canBreak(rs []rune, i int) bool {
	prev, next := rs[i-1], rs[i]
	if next == ' ' {
		return false
	}
	if prev == ' ' {
		return true
	}
	if !isWide(prev) && !isWide(next) {
		return false
	}
	return !noStart(next) && !noEnd(prev)
}

// BreakLines splits s into lines no wider than maxWidth points. It breaks at
// spaces and between CJK characters, applying kinsoku rules: characters that
// cannot start a line hang at the end of the previous line, and characters
// that cannot end a line move to the next line.
func BreakLines(f *Face, s string, size float64, maxWidth float64) []string {
	if s == "" {
		return []string{}
	}
	if maxWidth <= 0 {
		return []string{s}
	}
	rs := []rune(s)
	n := len(rs)
	pw := make([]float64, n+1) // prefix widths
	for i, r := range rs {
		pw[i+1] = pw[i] + f.Advance(r, size)
	}
	const eps = 1e-6

	// visibleEnd trims trailing spaces and hanging punctuation from rs[a:b].
	visibleEnd := func(a, b int) int {
		for b > a && rs[b-1] == ' ' {
			b--
		}
		for hang := 0; hang < maxHang && b > a+1 && noStart(rs[b-1]); hang++ {
			b--
		}
		return b
	}

	var lines []string
	start := 0
	for start < n {
		best, over := -1, -1
		for j := start + 1; j <= n; j++ {
			if pw[visibleEnd(start, j)]-pw[start] > maxWidth+eps {
				over = j
				break
			}
			if j == n || canBreak(rs, j) {
				best = j
			}
		}
		if best < 0 {
			// No break opportunity fits: split inside the unit.
			best = over - 1
			if best <= start {
				best = start + 1
			}
			for best > start+1 && best < n && (noStart(rs[best]) || noEnd(rs[best-1])) {
				best--
			}
		}
		t := best
		for t > start && rs[t-1] == ' ' {
			t--
		}
		lines = append(lines, string(rs[start:t]))
		start = best
		for start < n && rs[start] == ' ' {
			start++
		}
	}
	return lines
}
