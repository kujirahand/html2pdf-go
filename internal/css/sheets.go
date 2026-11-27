package css

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kujirahand/html2pdf-go/internal/dom"
)

// ResolveSheets returns the text of doc's stylesheets in document order,
// ready for ComputeStyles. Linked sheets are read from the local file system,
// relative paths resolved against baseDir. Remote (contains "://") and data:
// links are ignored. A local file that cannot be read is an error.
func ResolveSheets(doc *dom.Document, baseDir string) ([]string, error) {
	var out []string
	for _, s := range doc.Sheets {
		if s.Href == "" {
			out = append(out, s.Text)
			continue
		}
		if strings.Contains(s.Href, "://") || strings.HasPrefix(strings.ToLower(s.Href), "data:") {
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
