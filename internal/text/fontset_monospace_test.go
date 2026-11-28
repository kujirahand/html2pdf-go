package text

import (
	"os"
	"testing"
)

// The browser-style preference list must win over the generic "mono" scan:
// "courier new" does not contain "mono", so only the preference list (Linux,
// Western locale: "courier new" is in it) can pick the face registered under
// that name, while the generic scan would pick "zz mono".
func TestMonospacePrefersBrowserDefaultOverGenericScan(t *testing.T) {
	const dejavuMono = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
	const liberationMono = "/usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf"
	if _, err := os.Stat(dejavuMono); err != nil {
		t.Skip("DejaVu Sans Mono not installed")
	}
	if _, err := os.Stat(liberationMono); err != nil {
		t.Skip("Liberation Mono not installed")
	}
	t.Setenv("LC_ALL", "")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LANG", "en_US.UTF-8")

	fs := &FontSet{faces: map[string]*Face{}}
	fs.once.Do(func() {}) // keep buildIndex from replacing the manual index
	fs.index = map[string]*family{
		"courier new": {regular: dejavuMono},
		"zz mono":     {regular: liberationMono},
	}
	got := fs.FaceFor([]string{"monospace"}, false, false)
	if got == nil {
		t.Fatal("monospace did not resolve")
	}
	if got.FamilyName() != "DejaVu Sans Mono" {
		t.Errorf("monospace resolved to %q, want the face registered as \"courier new\" (DejaVu Sans Mono)", got.FamilyName())
	}
}

func TestPreferredMonoNames(t *testing.T) {
	tests := []struct {
		name  string
		goos  string
		class monoLocaleClass
		first string
	}{
		{"windows japanese", "windows", monoLocaleJapanese, "ms gothic"},
		{"darwin japanese", "darwin", monoLocaleJapanese, "osaka-mono"},
		{"windows western", "windows", monoLocaleWestern, "courier new"},
		{"darwin western", "darwin", monoLocaleWestern, "courier"},
		{"linux japanese", "linux", monoLocaleJapanese, "noto sans mono cjk jp"},
		{"windows unknown", "windows", monoLocaleUnknown, "courier new"},
		{"darwin unknown", "darwin", monoLocaleUnknown, "osaka-mono"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := preferredMonoNames(tt.goos, tt.class)
			if len(names) == 0 || names[0] != tt.first {
				t.Fatalf("preferredMonoNames(%q, %v) = %v, want first %q",
					tt.goos, tt.class, names, tt.first)
			}
		})
	}

	windowsUnknown := preferredMonoNames("windows", monoLocaleUnknown)
	foundMSGothic := false
	for _, name := range windowsUnknown {
		if name == "ms gothic" {
			foundMSGothic = true
			break
		}
	}
	if !foundMSGothic {
		t.Fatalf("unknown Windows names %v do not contain ms gothic", windowsUnknown)
	}

	linuxUnknown := preferredMonoNames("linux", monoLocaleUnknown)
	seen := make(map[string]bool)
	for _, name := range linuxUnknown {
		if seen[name] {
			t.Fatalf("unknown Linux names contain duplicate %q: %v", name, linuxUnknown)
		}
		seen[name] = true
	}
}

func TestStyleScore(t *testing.T) {
	regular := styleScore(&Face{subfamily: "regular"}, "regular")
	if got := styleScore(&Face{subfamily: "roman"}, "regular"); got != regular {
		t.Errorf("roman should rank as plain regular: got %d, want %d", got, regular)
	}
	medium := styleScore(&Face{subfamily: "medium"}, "regular")
	black := styleScore(&Face{subfamily: "black"}, "regular")
	if !(medium < black) {
		t.Errorf("medium (%d) should rank before black (%d) for the regular slot", medium, black)
	}
	upright := styleScore(&Face{subfamily: "black"}, "regular")
	oblique := styleScore(&Face{subfamily: "black oblique"}, "regular")
	if !(upright < oblique) {
		t.Errorf("upright (%d) should rank before oblique (%d)", upright, oblique)
	}
	if got := styleScore(&Face{subfamily: "bold"}, "bold"); got != 0 {
		t.Errorf("exact bold should score 0, got %d", got)
	}
	if got := styleScore(&Face{subfamily: "bold italic"}, "bold"); got != 4 {
		t.Errorf("bold italic for the bold slot should score 4 (italic only), got %d", got)
	}
	// A face that reaches the bold slot only via the OS/2 bold bit (its
	// subfamily does not claim bold) ranks below a semibold design.
	bitOnly := styleScore(&Face{subfamily: "regular"}, "bold")
	semi := styleScore(&Face{subfamily: "semibold"}, "bold")
	if !(semi < bitOnly) {
		t.Errorf("semibold (%d) should rank before a bold-bit-only regular (%d) for the bold slot", semi, bitOnly)
	}
}

func TestDetectMonoLocaleClass(t *testing.T) {
	tests := []struct {
		name   string
		lcAll  string
		lcType string
		lang   string
		want   monoLocaleClass
	}{
		{
			name: "japanese LANG",
			lang: "ja_JP.UTF-8",
			want: monoLocaleJapanese,
		},
		{
			name: "western LANG",
			lang: "en_US.UTF-8",
			want: monoLocaleWestern,
		},
		{
			name: "empty locale",
			want: monoLocaleUnknown,
		},
		{
			name: "C locale",
			lang: "C",
			want: monoLocaleUnknown,
		},
		{
			name:  "LC_ALL takes precedence over LANG",
			lcAll: "en_US.UTF-8",
			lang:  "ja_JP.UTF-8",
			want:  monoLocaleWestern,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LC_ALL", tt.lcAll)
			t.Setenv("LC_CTYPE", tt.lcType)
			t.Setenv("LANG", tt.lang)

			if got := detectMonoLocaleClass(); got != tt.want {
				t.Fatalf("detectMonoLocaleClass() = %v, want %v", got, tt.want)
			}
		})
	}
}
