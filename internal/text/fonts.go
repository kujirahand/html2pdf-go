package text

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font/sfnt"
)

// Fonts holds the faces used for rendering.
type Fonts struct{ Regular, Bold *Face }

const maxScanDepth = 4

var (
	regularNames = []string{"NotoSansCJK-Regular.ttc", "NotoSansJP-Regular.ttf"}
	boldNames    = []string{"NotoSansCJK-Bold.ttc", "NotoSansJP-Bold.ttf"}
)

// DefaultFontDirs returns the operating system's standard font directories,
// searched when no directories are given.
func DefaultFontDirs() []string {
	dirs := []string{
		"/usr/share/fonts/opentype/noto",
		"/usr/share/fonts/truetype/noto",
		"/usr/share/fonts",
		"/System/Library/Fonts",
		"/Library/Fonts",
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Library", "Fonts"))
	}
	return append(dirs, `C:\Windows\Fonts`)
}

// LoadFonts finds a Japanese-capable font in dirs (or in default locations
// if dirs is empty). Earlier directories take precedence: Regular comes from
// the first directory with a usable face, and Bold is looked up in that
// directory first, then in the remaining ones. If no bold face is found,
// Bold falls back to Regular.
func LoadFonts(dirs []string) (*Fonts, error) {
	search := dirs
	if len(search) == 0 {
		search = DefaultFontDirs()
	}
	files := make([][]string, len(search))
	for i, d := range search {
		files[i] = listFontFiles(d)
	}
	var reg *Face
	var regPath string
	regDir := -1
	for i, fl := range files {
		if reg, regPath = pickRegular(fl); reg != nil {
			regDir = i
			break
		}
	}
	if reg == nil {
		return nil, fmt.Errorf("text: no usable font found in %v", search)
	}
	bold := pickBold(files[regDir], regPath)
	for i := 0; bold == nil && i < len(files); i++ {
		if i != regDir {
			bold = pickBold(files[i], regPath)
		}
	}
	if bold == nil {
		bold = reg
	}
	return &Fonts{Regular: reg, Bold: bold}, nil
}

// listFontFiles returns font files under dir in lexical order, scanning a few levels deep.
func listFontFiles(dir string) []string {
	var files []string
	root := filepath.Clean(dir)
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if rel, rerr := filepath.Rel(root, p); rerr == nil && rel != "." {
				if strings.Count(rel, string(filepath.Separator))+1 > maxScanDepth {
					return fs.SkipDir
				}
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".ttf", ".ttc", ".otf":
			files = append(files, p)
		}
		return nil
	})
	return files
}

func isBoldName(base string) bool { return strings.Contains(strings.ToLower(base), "bold") }

func isCJKName(base string) bool { return strings.Contains(strings.ToLower(base), "cjk") }

// hasJapaneseGlyphs reports whether f can render Japanese text (checked via 'あ').
func hasJapaneseGlyphs(f *Face) bool { return f.GlyphID('あ') != 0 }

// firstLoadable loads the first file whose base name satisfies match and
// whose face satisfies ok (nil accepts any face).
func firstLoadable(files []string, match func(base string) bool, ok func(*Face) bool) (*Face, string) {
	for _, p := range files {
		if !match(filepath.Base(p)) {
			continue
		}
		if f, err := loadFace(p); err == nil && (ok == nil || ok(f)) {
			return f, p
		}
	}
	return nil, ""
}

// pickRegular prefers well-known Japanese font names; every other candidate
// must actually contain Japanese glyphs, otherwise nil is returned.
func pickRegular(files []string) (*Face, string) {
	for _, name := range regularNames {
		if f, p := firstLoadable(files, func(b string) bool { return b == name }, nil); f != nil {
			return f, p
		}
	}
	if f, p := firstLoadable(files, func(b string) bool { return isCJKName(b) && !isBoldName(b) }, hasJapaneseGlyphs); f != nil {
		return f, p
	}
	if f, p := firstLoadable(files, func(b string) bool { return !isBoldName(b) }, hasJapaneseGlyphs); f != nil {
		return f, p
	}
	return firstLoadable(files, func(string) bool { return true }, hasJapaneseGlyphs)
}

func pickBold(files []string, regPath string) *Face {
	for _, name := range boldNames {
		if f, _ := firstLoadable(files, func(b string) bool { return b == name }, nil); f != nil {
			return f
		}
	}
	// Sibling of a fallback regular face, e.g. Foo-Regular.ttf -> Foo-Bold.ttf.
	if base := filepath.Base(regPath); strings.Contains(base, "Regular") {
		want := filepath.Join(filepath.Dir(regPath), strings.Replace(base, "Regular", "Bold", 1))
		if f, err := loadFace(want); err == nil && hasJapaneseGlyphs(f) {
			return f
		}
	}
	f, _ := firstLoadable(files, func(b string) bool { return isCJKName(b) && isBoldName(b) }, hasJapaneseGlyphs)
	return f
}

func loadFace(path string) (*Face, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("text: read font %s: %w", path, err)
	}
	f, err := ParseFace(data)
	if err != nil {
		return nil, fmt.Errorf("text: font %s: %w", path, err)
	}
	f.path = path
	return f, nil
}

// ParseFace parses TTF/OTF/TTC data. For collections it prefers a face
// whose name contains " JP", else the first face.
func ParseFace(data []byte) (*Face, error) {
	if len(data) >= 4 && string(data[:4]) == "ttcf" {
		c, err := sfnt.ParseCollection(data)
		if err != nil {
			return nil, fmt.Errorf("text: parse collection: %w", err)
		}
		n := c.NumFonts()
		if n == 0 {
			return nil, fmt.Errorf("text: empty font collection")
		}
		var first *sfnt.Font
		firstIdx := 0
		for i := 0; i < n; i++ {
			sf, err := c.Font(i)
			if err != nil {
				continue
			}
			if first == nil {
				first, firstIdx = sf, i
			}
			if hasJPName(sf) {
				return newFace(sf, data, ttcOffset(data, i)), nil
			}
		}
		if first == nil {
			return nil, fmt.Errorf("text: no parsable face in collection")
		}
		return newFace(first, data, ttcOffset(data, firstIdx)), nil
	}
	sf, err := sfnt.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("text: parse font: %w", err)
	}
	return newFace(sf, data, 0), nil
}

// ttcOffset returns the table directory offset of face i in a TTC, or -1.
func ttcOffset(data []byte, i int) int {
	p := 12 + 4*i
	if p+4 > len(data) {
		return -1
	}
	return int(uint32(data[p])<<24 | uint32(data[p+1])<<16 | uint32(data[p+2])<<8 | uint32(data[p+3]))
}

func hasJPName(sf *sfnt.Font) bool {
	var b sfnt.Buffer
	for _, id := range []sfnt.NameID{sfnt.NameIDFamily, sfnt.NameIDFull} {
		if name, err := sf.Name(&b, id); err == nil && strings.Contains(name, " JP") {
			return true
		}
	}
	return false
}
