package css

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/dom"
)

var urlScheme = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)

// IsURL reports whether ref is a URL rather than a file path: it starts with
// "//" or with a scheme such as https:, data:, about: or blob:.
func IsURL(ref string) bool {
	return strings.HasPrefix(ref, "//") || urlScheme.MatchString(ref)
}

// ResolveSheets returns the text of doc's stylesheets in document order,
// ready for ComputeStyles. Linked sheets are read from the local file system,
// relative paths resolved against baseDir. Links that are URLs rather than
// file paths (see IsURL) are ignored. A local file that cannot be read is an
// error.
func ResolveSheets(doc *dom.Document, baseDir string) ([]string, error) {
	var out []string
	for _, s := range doc.Sheets {
		if s.Href == "" {
			out = append(out, s.Text)
			continue
		}
		if IsURL(s.Href) {
			continue
		}
		p := s.Href
		if !filepath.IsAbs(p) {
			p = filepath.Join(baseDir, p)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, fmt.Errorf("css: read stylesheet: %w", err)
		}
		out = append(out, string(data))
	}
	return out, nil
}
