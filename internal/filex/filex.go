// Package filex is the only way documents read local files (images,
// stylesheets, frame documents). It confines references to a root directory
// unless the caller explicitly allows external files, and refuses anything
// but regular files so a document cannot point the converter at a device or
// pipe.
package filex

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// IsNetworkPath reports whether a document reference is a network path
// rather than a local file: it starts with two path separators in any mix
// ("//host/share", `\\host\share`, "/\host\share"), which on Windows is a
// UNC path that would make the process contact a remote host. Loaders
// reject such references before resolving them; the confinement below is
// about the resolved path and deliberately does not repeat this check, so
// a root directory that itself sits on a network share keeps working.
func IsNetworkPath(ref string) bool {
	if len(ref) >= 2 && isSep(ref[0]) && isSep(ref[1]) {
		return true
	}
	// A Windows UNC volume name (\\host\share) is longer than a drive
	// letter volume (C:). Empty on other platforms.
	return len(filepath.VolumeName(ref)) > 2
}

func isSep(c byte) bool { return c == '/' || c == '\\' }

// ReadUnder reads the file an already-resolved document reference points
// to (for example a frame document resolving "../x" against its own
// directory). The confinement root is rootDir rather than the directory
// the reference was resolved against, so a nested document can use files
// anywhere inside the outermost document's tree, but never outside it:
// the path is opened through os.OpenRoot, so neither ".." nor a symbolic
// link can leave rootDir. With allowExternal true the path is used as
// given. With allowExternal false and an empty rootDir, no file is read
// at all. The target must be a regular file of at most limit bytes.
func ReadUnder(rootDir, resolved string, limit int64, allowExternal bool) ([]byte, error) {
	f, err := OpenUnder(rootDir, resolved, allowExternal)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("filex: %s is larger than %d bytes", resolved, limit)
	}
	return data, nil
}

// OpenUnder is ReadUnder's streaming twin for readers that only need the
// start of a file, such as image sizing. It applies the same confinement,
// rejects non-regular files and leaves closing the file to the caller.
func OpenUnder(rootDir, resolved string, allowExternal bool) (*os.File, error) {
	if resolved == "" {
		return nil, errors.New("filex: empty file reference")
	}
	if rootDir == "" {
		if !allowExternal {
			return nil, errors.New("filex: no base directory: documents cannot read local files")
		}
		rootDir = "."
	}
	var f *os.File
	var err error
	if allowExternal {
		// Check before opening: opening a FIFO would block.
		if err := checkRegular(os.Stat(resolved)); err != nil {
			return nil, err
		}
		f, err = os.Open(resolved)
	} else {
		absRoot, err := filepath.Abs(rootDir)
		if err != nil {
			return nil, err
		}
		absPath, err := filepath.Abs(resolved)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(absRoot, absPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("filex: path %q is outside the base directory", resolved)
		}
		var root *os.Root
		if root, err = os.OpenRoot(absRoot); err != nil {
			return nil, err
		}
		defer root.Close()
		if err := checkRegular(root.Stat(rel)); err != nil {
			return nil, err
		}
		f, err = root.Open(rel)
	}
	if err != nil {
		return nil, err
	}
	if err := checkRegular(f.Stat()); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func checkRegular(fi os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("filex: %s is not a regular file", fi.Name())
	}
	return nil
}
