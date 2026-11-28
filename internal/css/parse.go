package css

import (
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/warn"
)

type decl struct{ prop, val string }

// compound is one simple-selector sequence: an optional tag, classes and
// at most one id.
type compound struct {
	tag     string // "" or "*" matches any element
	classes []string
	id      string
	invalid bool // uses unsupported syntax (attribute, pseudo, ...)
}

// selector is a chain of compounds joined by descendant combinators.
type selector struct {
	parts   []compound
	invalid bool
}

type rule struct {
	sels  []selector
	decls []decl
}

func stripComments(s string) string {
	var sb strings.Builder
	for {
		i := strings.Index(s, "/*")
		if i < 0 {
			sb.WriteString(s)
			return sb.String()
		}
		sb.WriteString(s[:i])
		j := strings.Index(s[i+2:], "*/")
		if j < 0 {
			return sb.String()
		}
		s = s[i+2+j+2:]
	}
}

// parseDecls parses "prop: value; ..." declarations.
func parseDecls(s string) []decl {
	var out []decl
	for _, part := range strings.Split(s, ";") {
		i := strings.IndexByte(part, ':')
		if i < 0 {
			continue
		}
		prop := strings.ToLower(strings.TrimSpace(part[:i]))
		val := strings.TrimSpace(part[i+1:])
		if k := strings.Index(strings.ToLower(val), "!important"); k >= 0 {
			val = strings.TrimSpace(val[:k])
		}
		if prop != "" && val != "" {
			out = append(out, decl{prop, val})
		}
	}
	return out
}

func parseSheet(src string) []rule { return parseSheetW(src, nil) }

func parseSheetW(src string, w *warn.Collector) []rule {
	s := stripComments(src)
	var rules []rule
	for {
		s = strings.TrimSpace(s)
		if s == "" {
			return rules
		}
		open := strings.IndexByte(s, '{')
		if s[0] == '@' {
			if w != nil {
				end := strings.IndexFunc(s, func(r rune) bool {
					return !(r == '@' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
				})
				if end < 0 {
					end = len(s)
				}
				key := strings.ToLower(s[:end])
				if key != "@charset" && key != "@namespace" {
					w.Add("css-at-rule", key, "unsupported CSS at-rule: "+key)
				}
			}
			semi := strings.IndexByte(s, ';')
			if semi >= 0 && (open < 0 || semi < open) {
				s = s[semi+1:]
				continue
			}
		}
		if open < 0 {
			return rules
		}
		depth, end := 0, -1
		for i := open; i < len(s); i++ {
			if s[i] == '{' {
				depth++
			} else if s[i] == '}' {
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
		}
		if end < 0 {
			end = len(s)
		}
		prelude := strings.TrimSpace(s[:open])
		body := ""
		if end > open {
			body = s[open+1 : end]
		}
		if end < len(s) {
			s = s[end+1:]
		} else {
			s = ""
		}
		if strings.HasPrefix(prelude, "@") {
			continue
		}
		r := rule{decls: parseDecls(body)}
		if w != nil {
			for _, d := range r.decls {
				checkDecl(d, w)
			}
		}
		for _, sel := range strings.Split(prelude, ",") {
			r.sels = append(r.sels, parseSelector(sel))
		}
		rules = append(rules, r)
	}
}

func parseSelector(s string) selector {
	var sel selector
	for _, f := range strings.Fields(s) {
		c := parseCompound(f)
		if c.invalid {
			sel.invalid = true
		}
		sel.parts = append(sel.parts, c)
	}
	if len(sel.parts) == 0 {
		sel.invalid = true
	}
	return sel
}

func parseCompound(s string) compound {
	var c compound
	if strings.ContainsAny(s, "[]:>+~()") {
		c.invalid = true
		return c
	}
	// Split into the leading tag and ".class" / "#id" fragments.
	start, kind := 0, byte(0)
	flush := func(end int) bool {
		frag := s[start:end]
		switch kind {
		case 0:
			c.tag = strings.ToLower(frag)
		case '.':
			if frag == "" {
				return false
			}
			c.classes = append(c.classes, frag)
		case '#':
			if frag == "" || c.id != "" {
				return false
			}
			c.id = frag
		}
		return true
	}
	for i := 0; i < len(s); i++ {
		if s[i] != '.' && s[i] != '#' {
			continue
		}
		if !flush(i) {
			return compound{invalid: true}
		}
		start, kind = i+1, s[i]
	}
	if !flush(len(s)) {
		return compound{invalid: true}
	}
	return c
}

func (c compound) matches(n *dom.Node) bool {
	if c.invalid || n.IsText() {
		return false
	}
	if c.tag != "" && c.tag != "*" && c.tag != n.Tag {
		return false
	}
	if c.id != "" && c.id != n.Attr["id"] {
		return false
	}
	if len(c.classes) == 0 {
		return true
	}
	have := strings.Fields(n.Attr["class"])
	for _, want := range c.classes {
		found := false
		for _, h := range have {
			if h == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (s selector) matches(n *dom.Node) bool {
	if s.invalid || len(s.parts) == 0 {
		return false
	}
	last := len(s.parts) - 1
	if !s.parts[last].matches(n) {
		return false
	}
	anc := n.Parent
	for i := last - 1; i >= 0; i-- {
		for anc != nil && !s.parts[i].matches(anc) {
			anc = anc.Parent
		}
		if anc == nil {
			return false
		}
		anc = anc.Parent
	}
	return true
}

// specificity returns (id count, class count, element count).
func (s selector) specificity() (int, int, int) {
	var a, b, e int
	for _, c := range s.parts {
		if c.id != "" {
			a++
		}
		b += len(c.classes)
		if c.tag != "" && c.tag != "*" {
			e++
		}
	}
	return a, b, e
}
