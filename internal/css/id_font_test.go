package css

import (
	"reflect"
	"testing"
)

func TestParseCompoundID(t *testing.T) {
	tests := []struct {
		in      string
		want    compound
		invalid bool
	}{
		{in: "#main", want: compound{id: "main"}},
		{in: "div#main", want: compound{tag: "div", id: "main"}},
		{in: "div.note#main", want: compound{tag: "div", classes: []string{"note"}, id: "main"}},
		{in: ".a#b", want: compound{classes: []string{"a"}, id: "b"}},
		{in: "#b.a", want: compound{classes: []string{"a"}, id: "b"}},
		{in: "*#x", want: compound{tag: "*", id: "x"}},
		{in: "#", invalid: true},
		{in: "div#", invalid: true},
		{in: "#a#b", invalid: true},
		{in: "#a.", invalid: true},
		{in: "a..b", invalid: true},
		{in: "#a:hover", invalid: true},
		{in: "#a[x]", invalid: true},
	}
	for _, tt := range tests {
		got := parseCompound(tt.in)
		if got.invalid != tt.invalid {
			t.Errorf("parseCompound(%q).invalid = %v, want %v", tt.in, got.invalid, tt.invalid)
			continue
		}
		if !tt.invalid && !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseCompound(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestIDSelectorMatching(t *testing.T) {
	doc, cs := compute(t, `<style>
#main { color: red }
div#main { margin-left: 7px }
p#main { margin-left: 9px }
.note#main { padding-left: 3px }
#other { color: blue }
</style><body><div id="main" class="note">a</div><div id="x">b</div></body>`)
	red, _ := ParseColor("red")
	divs := doc.Body.Children
	a, b := cs.Style(divs[0]), cs.Style(divs[1])
	if a.Color != red {
		t.Errorf("#main color = %v, want red", a.Color)
	}
	if a.Margin[Left].Value != 7 {
		t.Errorf("div#main margin-left = %v, want 7 (p#main must not match)", a.Margin[Left])
	}
	if a.Padding[Left].Value != 3 {
		t.Errorf(".note#main padding-left = %v, want 3", a.Padding[Left])
	}
	if b.Color == red || b.Margin[Left].Value != 0 {
		t.Errorf("rules for #main leaked onto #x: %+v", b)
	}
}

func TestIDSpecificityBeatsClass(t *testing.T) {
	// The class rule comes later and has more classes, but the id wins.
	doc, cs := compute(t, `<style>
#a { color: red }
.x.y.z { color: blue }
</style><body><p id="a" class="x y z">t</p></body>`)
	red, _ := ParseColor("red")
	if got := cs.Style(find(doc.Body, "p")).Color; got != red {
		t.Errorf("color = %v, want red (id beats classes)", got)
	}
}

func TestSpecificityTriple(t *testing.T) {
	tests := []struct {
		sel     string
		a, b, c int
	}{
		{"p", 0, 0, 1},
		{".a", 0, 1, 0},
		{"#a", 1, 0, 0},
		{"div#a.b", 1, 1, 1},
		{"ul li #a", 1, 0, 2},
		{"*", 0, 0, 0},
	}
	for _, tt := range tests {
		a, b, c := parseSelector(tt.sel).specificity()
		if a != tt.a || b != tt.b || c != tt.c {
			t.Errorf("specificity(%q) = (%d,%d,%d), want (%d,%d,%d)", tt.sel, a, b, c, tt.a, tt.b, tt.c)
		}
	}
}

func TestInvalidIDSelectorIgnored(t *testing.T) {
	doc, cs := compute(t, `<style>#{ color: red } p#{ color: red }</style><body><p id="a">t</p></body>`)
	red, _ := ParseColor("red")
	if got := cs.Style(find(doc.Body, "p")).Color; got == red {
		t.Errorf("rule with an empty id name applied")
	}
}

func TestFontFamilyParse(t *testing.T) {
	tests := []struct {
		val  string
		want []string
	}{
		{`"Noto Serif CJK JP", serif`, []string{"Noto Serif CJK JP", "serif"}},
		{`'Foo Bar',Baz`, []string{"Foo Bar", "Baz"}},
		{`  Arial ,  "Helvetica Neue"  , sans-serif `, []string{"Arial", "Helvetica Neue", "sans-serif"}},
		{`serif`, []string{"serif"}},
	}
	for _, tt := range tests {
		if got := parseFontFamily(tt.val); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseFontFamily(%q) = %q, want %q", tt.val, got, tt.want)
		}
	}
}

func TestFontFamilyInherited(t *testing.T) {
	doc, cs := compute(t, `<style>
body { font-family: "A B", serif }
em { font-family: Mono }
</style><body><p>x <span>y</span> <em>z <b>w</b></em></p></body>`)
	want := func(tag string, fam ...string) {
		t.Helper()
		if got := cs.Style(find(doc.Body, tag)).FontFamily; !reflect.DeepEqual(got, fam) {
			t.Errorf("<%s> font-family = %q, want %q", tag, got, fam)
		}
	}
	want("p", "A B", "serif")
	want("span", "A B", "serif")
	want("em", "Mono")
	want("b", "Mono")
}
