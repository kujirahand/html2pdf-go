package css

import (
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/dom"
	"github.com/kujirahand/html2pdf-go/internal/filex"
	"github.com/kujirahand/html2pdf-go/internal/warn"
)

// This file is the single place that lists what the engine supports, for the
// -warn / Options.OnWarning feature. Anything not listed here is reported as
// unsupported. When a feature is implemented (for example font-style), add its
// property to `supported` with a value checker, or remove the entry from
// `ineffective` (HTML attributes that are accepted but have no effect).

type valueCheck func(string) bool

func oneOf(values ...string) valueCheck {
	return func(v string) bool {
		for _, x := range values {
			if v == x {
				return true
			}
		}
		return false
	}
}
func length(v string) bool      { _, ok := ParseLength(v); return ok }
func color(v string) bool       { _, ok := ParseColor(v); return ok }
func box(v string) bool         { _, ok := parseBoxLengths(v); return ok }
func nonnegative(v string) bool { l, ok := ParseLength(v); return ok && l.Value >= 0 }
func tokens(check valueCheck, max int) valueCheck {
	return func(v string) bool {
		ts := splitTokens(v)
		if len(ts) == 0 || len(ts) > max {
			return false
		}
		for _, t := range ts {
			if !check(t) {
				return false
			}
		}
		return true
	}
}
func borderWidth(v string) bool { _, ok := parseBorderWidth(v); return ok }
func borderStyle(v string) bool { _, ok := parseBorderStyle(v); return ok }
func border(v string) bool {
	return tokens(func(t string) bool { return borderWidth(t) || borderStyle(t) || color(t) }, 3)(v)
}
func radius(v string) bool {
	parts := strings.Split(v, "/")
	if len(parts) > 2 {
		return false
	}
	for _, p := range parts {
		ls, ok := parseRadiusList(p)
		if !ok || len(ls) > 4 {
			return false
		}
	}
	return true
}

var supported = func() map[string]valueCheck {
	m := map[string]valueCheck{
		"display": oneOf("none", "block", "inline", "list-item", "table", "table-row", "table-cell", "table-row-group", "table-caption"),
		"font-size": func(v string) bool {
			if _, ok := fontSizeKeywords[v]; ok {
				return true
			}
			l, ok := ParseLength(v)
			return ok && l.Unit != UnitNone && l.Value >= 0
		},
		"font-weight": func(v string) bool {
			if oneOf("normal", "bold", "bolder", "lighter")(v) {
				return true
			}
			l, ok := ParseLength(v)
			return ok && l.Unit == UnitNone
		},
		"font-family": func(v string) bool { return len(parseFontFamily(v)) > 0 }, "color": color,
		"background-color": func(v string) bool { return color(v) || v == "none" },
		"background": func(v string) bool {
			for _, t := range splitTokens(v) {
				if color(t) {
					return true
				}
			}
			return v == "none"
		},
		"text-align": oneOf("left", "right", "center", "start", "end"), "text-decoration": oneOf("none", "underline"), "text-decoration-line": oneOf("none", "underline"),
		"line-height": func(v string) bool { return v == "normal" || nonnegative(v) }, "margin": box, "padding": box,
		"width": func(v string) bool { return v == "auto" || nonnegative(v) }, "height": func(v string) bool {
			if v == "auto" {
				return true
			}
			l, ok := ParseLength(v)
			return ok && l.Value >= 0 && l.Unit != UnitPercent
		},
		"page-break-after": oneOf("always", "page", "left", "right", "auto", "avoid"), "break-after": oneOf("always", "page", "left", "right", "auto", "avoid"),
		"border-radius": radius, "border": border, "border-width": tokens(borderWidth, 4), "border-style": tokens(borderStyle, 4), "border-color": tokens(color, 4),
	}
	for _, s := range []string{"top", "right", "bottom", "left"} {
		m["margin-"+s] = func(v string) bool { return v == "auto" || length(v) }
		m["padding-"+s] = m["margin-"+s]
		m["border-"+s] = border
		m["border-"+s+"-width"] = borderWidth
		m["border-"+s+"-style"] = borderStyle
		m["border-"+s+"-color"] = color
	}
	for p := range cornerProps {
		m[p] = func(v string) bool { ls, ok := parseRadiusList(v); return ok && len(ls) <= 2 }
	}
	return m
}()
var ineffective = map[string][]string{
	"td": {"rowspan", "align", "valign", "bgcolor", "nowrap"}, "th": {"rowspan", "align", "valign", "bgcolor", "nowrap"},
	"table": {"border", "cellpadding", "cellspacing", "bgcolor", "align"}, "font": {"color", "face", "size"},
	"img": {"hspace", "vspace", "align", "border"},
}

func checkDecl(d decl, w *warn.Collector) {
	if w == nil {
		return
	}
	check, ok := supported[d.prop]
	if !ok {
		w.Add("css-property", d.prop, "unsupported CSS property: "+d.prop)
		return
	}
	v := strings.ToLower(strings.TrimSpace(d.val))
	if !check(v) {
		key := d.prop + ": " + v
		w.Add("css-value", key, "unsupported CSS value: "+key)
	}
}
func checkAttrs(n *dom.Node, w *warn.Collector) {
	if w == nil {
		return
	}
	for _, a := range ineffective[n.Tag] {
		if _, ok := n.Attr[a]; ok {
			key := n.Tag + "[" + a + "]"
			w.Add("html-attribute", key, "ignored HTML attribute: "+key)
		}
	}
}

// WarnRemoteSheets reports stylesheet references skipped by ResolveSheets.
func WarnRemoteSheets(doc *dom.Document, w *warn.Collector) {
	if w == nil || doc == nil {
		return
	}
	for _, s := range doc.Sheets {
		if s.Href != "" && (IsURL(s.Href) || filex.IsNetworkPath(s.Href)) {
			w.Add("remote-resource", s.Href, "ignored remote stylesheet: "+s.Href)
		}
	}
}
