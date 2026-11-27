package text

import (
	"path/filepath"
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
// Generic names such as serif are skipped. If none match it returns the base
// Regular or Bold face.
func (s *FontSet) FaceFor(families []string, bold bool) *Face {
	for _, name := range families {
		key := strings.ToLower(strings.TrimSpace(name))
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
// stands for the whole file. The first file for a family and style wins.
func (s *FontSet) buildIndex() {
	s.index = map[string]*family{}
	dirs := s.dirs
	if len(dirs) == 0 {
		dirs = defaultFontDirs()
	}
	seen := map[string]bool{}
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
				if fam.bold == "" {
					fam.bold = p
				}
			} else if fam.regular == "" {
				fam.regular = p
			}
		}
	}
}
