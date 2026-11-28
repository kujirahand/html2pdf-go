package text

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// genericFamilies are the CSS generic family names. They never match a font
// by name; FaceFor skips them.
var genericFamilies = map[string]bool{
	"serif": true, "sans-serif": true, "monospace": true, "cursive": true, "fantasy": true,
	"system-ui": true, "ui-serif": true, "ui-sans-serif": true, "ui-monospace": true, "ui-rounded": true,
}

// FontSet resolves CSS font-family lists to faces, falling back to the base slots.
type FontSet struct {
	Regular, Bold, Italic, BoldItalic *Face

	dirs []string // directories searched by name; empty means the defaults

	// autoItalicFamily is the lower-cased base regular family whose italic
	// faces fill the empty Italic / BoldItalic slots; "" disables that.
	autoItalicFamily                   string
	explicitItalic, explicitBoldItalic bool
	italicOnce                         sync.Once

	once  sync.Once
	index map[string]*family // lower-cased family name -> faces on disk

	mu    sync.Mutex
	faces map[string]*Face // loaded faces by path

	// OnUse, if set, is called once for each distinct (family, bold, italic, path)
	// that FaceFor returns. family is the matched index family name, or ""
	// for the base faces. Calls are serialized.
	OnUse func(family string, bold, italic bool, path string)

	useMu sync.Mutex
	used  map[useKey]bool

	monoMu   sync.Mutex
	monoDone [4]bool // cached per weight and slant
	mono     [4]*Face
}

type useKey struct {
	family string
	bold   bool
	italic bool
	path   string
}

// report records the face f returned for family (see OnUse), once per
// distinct combination, and returns f.
func (s *FontSet) report(family string, bold, italic bool, f *Face) *Face {
	if s.OnUse == nil || f == nil || f.path == "" {
		return f
	}
	k := useKey{family, bold, italic, f.path}
	s.useMu.Lock()
	defer s.useMu.Unlock()
	if s.used[k] {
		return f
	}
	if s.used == nil {
		s.used = map[useKey]bool{}
	}
	s.used[k] = true
	s.OnUse(family, bold, italic, f.path)
	return f
}

// family is the font files found for one family name.
type family struct {
	regular, bold, italic, boldItalic string // file paths, "" if absent
}

// LoadFontSet builds a FontSet. If regularFile is set it is loaded as the
// base regular face (failure is an error); otherwise the base faces come from
// LoadFonts(dirs). boldFile likewise overrides the base bold face. When
// regularFile is set without boldFile, bold text uses the regular face.
// Italic slots are searched in the base regular family unless regularFile is set.
// Explicit italic paths override those slots. Families are indexed in dirs
// (or the default directories if dirs is empty).
func LoadFontSet(dirs []string, regularFile, boldFile, italicFile, boldItalicFile string) (*FontSet, error) {
	fs := &FontSet{dirs: dirs, faces: map[string]*Face{}}
	if regularFile == "" {
		base, err := LoadFonts(dirs)
		if err != nil {
			return nil, err
		}
		fs.Regular, fs.Bold = base.Regular, base.Bold
		// The italic slots are resolved on first use (see baseItalics), so
		// documents without italic text do not pay for indexing the fonts.
		fs.autoItalicFamily = strings.ToLower(fs.Regular.family)
	} else {
		f, err := loadFace(regularFile)
		if err != nil {
			return nil, err
		}
		fs.Regular, fs.Bold = f, f
	}
	if boldFile != "" {
		f, err := loadFace(boldFile)
		if err != nil {
			return nil, err
		}
		fs.Bold = f
	}
	if italicFile != "" {
		f, err := loadFace(italicFile)
		if err != nil {
			return nil, err
		}
		fs.Italic = f
	}
	if boldItalicFile != "" {
		f, err := loadFace(boldItalicFile)
		if err != nil {
			return nil, err
		}
		fs.BoldItalic = f
	}
	fs.explicitItalic = italicFile != ""
	fs.explicitBoldItalic = boldItalicFile != ""
	return fs, nil
}

// baseItalics fills the Italic and BoldItalic slots that were not given
// explicitly from the base regular family, once, the first time an italic
// face is requested.
func (s *FontSet) baseItalics() {
	s.italicOnce.Do(func() {
		if s.autoItalicFamily == "" {
			return
		}
		fam := s.lookup(s.autoItalicFamily)
		if fam == nil {
			return
		}
		if !s.explicitItalic {
			s.Italic = s.load(fam.italic)
		}
		if !s.explicitBoldItalic {
			s.BoldItalic = s.load(fam.boldItalic)
		}
	})
}

// BaseFace resolves the requested style using the base slots.
func (s *FontSet) BaseFace(bold, italic bool) *Face {
	if italic {
		s.baseItalics()
	}
	if bold {
		if italic && s.BoldItalic != nil {
			return s.BoldItalic
		}
		if s.Bold != nil {
			return s.Bold
		}
		return s.Regular
	}
	if italic && s.Italic != nil {
		return s.Italic
	}
	return s.Regular
}

func (f *family) path(bold, italic bool) string {
	if bold {
		if italic && f.boldItalic != "" {
			return f.boldItalic
		}
		if f.bold != "" {
			return f.bold
		}
		return f.regular
	}
	if italic && f.italic != "" {
		return f.italic
	}
	if f.regular != "" {
		return f.regular
	}
	return f.bold
}

// FaceFor resolves a family and style, using base faces when no family matches.
func (s *FontSet) FaceFor(families []string, bold, italic bool) *Face {
	for _, name := range families {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "monospace" {
			if f := s.monospace(bold, italic); f != nil {
				return s.report(key, bold, italic, f)
			}
			continue
		}
		if key == "" || genericFamilies[key] {
			continue
		}
		if fam := s.lookup(key); fam != nil {
			if f := s.load(fam.path(bold, italic)); f != nil {
				return s.report(key, bold, italic, f)
			}
		}
	}
	return s.report("", bold, italic, s.BaseFace(bold, italic))
}

type monoLocaleClass int

const (
	monoLocaleUnknown monoLocaleClass = iota
	monoLocaleJapanese
	monoLocaleWestern
)

// detectMonoLocaleClass classifies the process locale from LC_ALL, LC_CTYPE,
// then LANG. An explicitly configured Japanese locale receives Japanese
// browser-style defaults; other non-C/POSIX locales receive Western defaults.
func detectMonoLocaleClass() monoLocaleClass {
	locale := ""
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			locale = strings.ToLower(value)
			break
		}
	}
	if locale == "" ||
		locale == "c" ||
		locale == "posix" ||
		strings.HasPrefix(locale, "c.") ||
		strings.HasPrefix(locale, "posix") {
		return monoLocaleUnknown
	}
	if strings.HasPrefix(locale, "ja") {
		return monoLocaleJapanese
	}
	return monoLocaleWestern
}

// preferredMonoNames returns the fixed-font defaults a web browser would
// use for an OS and locale class, modelled on Chrome's per-locale settings:
// MS Gothic on Japanese Windows, Courier New on English Windows, Osaka-Mono
// on Japanese macOS, Courier on English macOS, and the usual fontconfig
// choices on Linux (Noto Sans Mono CJK JP for Japanese, DejaVu/Liberation
// Mono for Western). The names are lower-cased index keys, tried in order.
//
// When the locale class is unknown (no locale in the environment, common
// for GUI apps and daemons), the lists are merged in the order that is
// harmless on that OS: Windows tries Courier New first because it always
// ships with Windows while MS Gothic may not; macOS tries the Japanese
// list first because Osaka-Mono rarely exists on non-Japanese Macs; other
// systems try Noto Sans Mono CJK JP first, then the Western list, then the
// remaining Japanese candidates, without duplicates.
func preferredMonoNames(goos string, class monoLocaleClass) []string {
	japanese := map[string][]string{
		"windows": {
			"ms gothic", "ｍｓ ゴシック", "biz udgothic", "biz udゴシック",
			"ms ui gothic",
		},
		"darwin": {
			"osaka-mono", "osaka-等幅", "osaka－等幅", "osaka",
			"noto sans mono cjk jp", "hiragino kaku gothic pron",
			"hiragino kaku gothic pro",
		},
		"other": {
			"noto sans mono cjk jp", "ipamona pgothic", "ipamona gothic",
			"ipa mona pgothic", "takao gothic", "vl gothic", "biz udgothic",
			"ms gothic", "osaka-mono",
		},
	}
	western := map[string][]string{
		"windows": {"courier new", "consolas", "lucida console", "courier"},
		"darwin":  {"courier", "menlo", "monaco", "courier new"},
		"other": {
			"dejavu sans mono", "liberation mono", "noto sans mono",
			"ubuntu mono", "courier new", "courier",
		},
	}

	platform := goos
	if platform != "windows" && platform != "darwin" {
		platform = "other"
	}

	copyNames := func(names []string) []string {
		return append([]string(nil), names...)
	}
	mergeUnique := func(groups ...[]string) []string {
		seen := make(map[string]bool)
		var names []string
		for _, group := range groups {
			for _, name := range group {
				if !seen[name] {
					seen[name] = true
					names = append(names, name)
				}
			}
		}
		return names
	}

	switch class {
	case monoLocaleJapanese:
		return copyNames(japanese[platform])
	case monoLocaleWestern:
		return copyNames(western[platform])
	default:
		switch platform {
		case "windows":
			return mergeUnique(western["windows"], japanese["windows"])
		case "darwin":
			return mergeUnique(japanese["darwin"], western["darwin"])
		default:
			return mergeUnique(
				[]string{"noto sans mono cjk jp"},
				western["other"],
				japanese["other"][1:],
			)
		}
	}
}

// monospace returns an installed monospace face, or nil if there is none.
// It first tries the browser-style defaults for the current OS and locale
// (see preferredMonoNames). If none of them is installed, it falls back to
// a generic scan of families whose lower-cased name contains "mono": names
// containing "cjk" or "jp" come first, then the rest alphabetically, and a
// face with Japanese glyphs is preferred. The result is cached per weight.
func (s *FontSet) monospace(bold, italic bool) *Face {
	w := 0
	if bold {
		w = 1
	}
	if italic {
		w += 2
	}
	s.monoMu.Lock()
	defer s.monoMu.Unlock()
	if s.monoDone[w] {
		return s.mono[w]
	}
	s.once.Do(s.buildIndex)

	loadFamily := func(fam *family) *Face {
		if fam == nil {
			return nil
		}
		path := fam.path(bold, italic)
		if path == "" {
			return nil
		}
		return s.load(path)
	}

	for _, name := range preferredMonoNames(runtime.GOOS, detectMonoLocaleClass()) {
		if f := loadFamily(s.index[name]); f != nil {
			s.mono[w], s.monoDone[w] = f, true
			return f
		}
	}

	var names []string
	for k := range s.index {
		if strings.Contains(k, "mono") {
			names = append(names, k)
		}
	}
	tier := func(k string) int {
		if strings.Contains(k, "cjk") || strings.Contains(k, "jp") {
			return 0
		}
		return 1
	}
	sort.Slice(names, func(i, j int) bool {
		if ti, tj := tier(names[i]), tier(names[j]); ti != tj {
			return ti < tj
		}
		return names[i] < names[j]
	})
	var faces []*Face
	for _, k := range names {
		if f := loadFamily(s.index[k]); f != nil {
			faces = append(faces, f)
		}
	}
	var res *Face
	for _, f := range faces {
		if hasJapaneseGlyphs(f) {
			res = f
			break
		}
	}
	if res == nil && len(faces) > 0 {
		res = faces[0]
	}
	s.mono[w], s.monoDone[w] = res, true
	return res
}

func (s *FontSet) lookup(key string) *family {
	s.once.Do(s.buildIndex)
	return s.index[key]
}

// load returns the face at path, caching it; nil if it cannot be loaded.
func (s *FontSet) load(path string) *Face {
	s.mu.Lock()
	defer s.mu.Unlock()
	if f, ok := s.faces[path]; ok {
		return f
	}
	f, err := loadFace(path)
	if err != nil {
		f = nil
	}
	s.faces[path] = f
	return f
}

// buildIndex scans the font directories and records the family name and
// boldness of every font file. For a collection, the face ParseFace picks
// stands for the whole file. Within a family and weight, the most plain
// face wins: an upright face beats an oblique/italic one, and a face whose
// subfamily is exactly the weight ("regular"/"book", resp. "bold") beats
// other weights that share the slot (e.g. a Black face is not bold-flagged
// but must not become the family's regular face).
func (s *FontSet) buildIndex() {
	s.index = map[string]*family{}
	dirs := s.dirs
	if len(dirs) == 0 {
		dirs = DefaultFontDirs()
	}
	seen := map[string]bool{}
	regScore := map[string]int{}
	boldScore := map[string]int{}
	italicScore := map[string]int{}
	boldItalicScore := map[string]int{}
	for _, d := range dirs {
		for _, p := range listFontFiles(d) {
			if abs, err := filepath.Abs(p); err == nil {
				p = abs
			}
			if seen[p] {
				continue
			}
			seen[p] = true
			f, err := loadFace(p)
			if err != nil || f.family == "" {
				continue
			}
			key := strings.ToLower(f.family)
			fam := s.index[key]
			if fam == nil {
				fam = &family{}
				s.index[key] = fam
			}
			isItalic := strings.Contains(f.subfamily, "italic") || strings.Contains(f.subfamily, "oblique")
			if isItalic {
				if f.bold {
					if sc := styleScore(f, "bold"); fam.boldItalic == "" || sc < boldItalicScore[key] {
						fam.boldItalic, boldItalicScore[key] = p, sc
					}
				} else {
					if sc := styleScore(f, "regular"); fam.italic == "" || sc < italicScore[key] {
						fam.italic, italicScore[key] = p, sc
					}
				}
				continue
			}
			if f.bold {
				if sc := styleScore(f, "bold"); fam.bold == "" || sc < boldScore[key] {
					fam.bold, boldScore[key] = p, sc
				}
			} else if sc := styleScore(f, "regular"); fam.regular == "" || sc < regScore[key] {
				fam.regular, regScore[key] = p, sc
			}
		}
	}
}

// styleScore ranks a face for a family's regular or bold slot; lower is a
// plainer representative of the weight. An exact-weight face wins; other
// weights are ordered by their distance from the slot's weight so that,
// when a family has no exact face, a Medium or Light is picked rather
// than whichever of e.g. Medium and Black happens to be scanned first.
func styleScore(f *Face, weight string) int {
	sc := 0
	sub := f.subfamily
	if strings.Contains(sub, "oblique") || strings.Contains(sub, "italic") {
		sc += 4
	}
	plain := sub == weight ||
		(weight == "regular" && (sub == "" || sub == "book" || sub == "roman" ||
			strings.HasPrefix(sub, "regular "))) ||
		(weight == "bold" && strings.HasPrefix(sub, "bold "))
	if plain {
		return sc
	}
	sc++
	switch {
	case strings.Contains(sub, "black") || strings.Contains(sub, "heavy"):
		sc += 3
	case strings.Contains(sub, "extrabold") || strings.Contains(sub, "ultrabold"):
		sc += 2
	case strings.Contains(sub, "semibold") || strings.Contains(sub, "demibold") ||
		strings.Contains(sub, "medium"):
		sc += 2
		if weight == "bold" {
			sc--
		}
	case strings.Contains(sub, "light") || strings.Contains(sub, "thin"):
		sc++
	}
	if weight == "bold" && !strings.Contains(sub, "bold") {
		// In the bold slot only via the OS/2 bold bit, with a subfamily
		// that does not claim bold at all: rank it below true bold
		// designs and their near neighbours.
		sc += 2
	}
	return sc
}
