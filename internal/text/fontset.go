package text

import (
	"path/filepath"
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

// FontSet resolves CSS font-family lists to faces. Regular and Bold are the
// base faces used when no listed family is found.
type FontSet struct {
	Regular, Bold *Face

	dirs []string // directories searched by name; empty means the defaults

	once  sync.Once
	index map[string]*family // lower-cased family name -> faces on disk

	mu    sync.Mutex
	faces map[string]*Face // loaded faces by path

	monoMu   sync.Mutex
	monoDone [2]bool // regular, bold: monospace already resolved
	mono     [2]*Face
}

// family is the font files found for one family name.
type family struct {
	regular, bold string // file paths, "" if absent
}

// LoadFontSet builds a FontSet. If regularFile is set it is loaded as the
// base regular face (failure is an error); otherwise the base faces come from
// LoadFonts(dirs). boldFile likewise overrides the base bold face. When
// regularFile is set without boldFile, bold text uses the regular face.
// Families are looked up by name in dirs (or the default directories if dirs
// is empty) on first use.
func LoadFontSet(dirs []string, regularFile, boldFile string) (*FontSet, error) {
	fs := &FontSet{dirs: dirs, faces: map[string]*Face{}}
	if regularFile == "" {
		base, err := LoadFonts(dirs)
		if err != nil {
			return nil, err
		}
		fs.Regular, fs.Bold = base.Regular, base.Bold
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
	return fs, nil
}

// FaceFor returns the face for the first family in families that names an
// installed font (case-insensitively), choosing its bold or regular face.
// The generic name monospace resolves through monospace; other generic names
// such as serif are skipped. If none match it returns the base
// Regular or Bold face.
func (s *FontSet) FaceFor(families []string, bold bool) *Face {
	for _, name := range families {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "monospace" {
			if f := s.monospace(bold); f != nil {
				return f
			}
			continue
		}
		if key == "" || genericFamilies[key] {
			continue
		}
		fam := s.lookup(key)
		if fam == nil {
			continue
		}
		path := fam.regular
		if (bold && fam.bold != "") || path == "" {
			path = fam.bold
		}
		if f := s.load(path); f != nil {
			return f
		}
	}
	if bold {
		return s.Bold
	}
	return s.Regular
}

// monospace returns an installed monospace face, or nil if there is none.
// Candidates are families whose lower-cased name contains "mono". Names
// containing "cjk" or "jp" come first, then the rest, alphabetically within
// each group. The first candidate with Japanese glyphs wins; failing that, the
// first candidate that loads. The result is cached per weight.
func (s *FontSet) monospace(bold bool) *Face {
	w := 0
	if bold {
		w = 1
	}
	s.monoMu.Lock()
	defer s.monoMu.Unlock()
	if s.monoDone[w] {
		return s.mono[w]
	}
	s.once.Do(s.buildIndex)
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
		fam := s.index[k]
		path := fam.regular
		if (bold && fam.bold != "") || path == "" {
			path = fam.bold
		}
		if f := s.load(path); f != nil {
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
		dirs = defaultFontDirs()
	}
	seen := map[string]bool{}
	regScore := map[string]int{}
	boldScore := map[string]int{}
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
// plainer representative of the weight.
func styleScore(f *Face, weight string) int {
	sc := 0
	if strings.Contains(f.subfamily, "oblique") || strings.Contains(f.subfamily, "italic") {
		sc += 4
	}
	plain := f.subfamily == weight ||
		(weight == "regular" && (f.subfamily == "" || f.subfamily == "book"))
	if !plain {
		sc++
	}
	return sc
}
