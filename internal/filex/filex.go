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

// Read reads the file ref refers to, a reference made by a document.
//
// A ref starting with "//" or `\\` is rejected: on Windows it is a UNC path
// (a network share). With allowExternal false, ref must be relative to
// baseDir and the file is opened through os.OpenRoot, so neither ".." nor a
// symbolic link can leave baseDir; absolute refs are rejected. With
// allowExternal true, ref is used as given (relative refs resolve against
// baseDir, absolute ones are taken as they are). The target must be a
// regular file of at most limit bytes.
func Read(baseDir, ref string, limit int64, allowExternal bool) ([]byte, error) {
	f, err := Open(baseDir, ref, allowExternal)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readLimited(f, ref, limit)
}

// ReadUnder is Read for a reference the caller has already resolved to a
// path (for example a frame document resolving "../x" against its own
// directory). The confinement root is rootDir rather than the directory the
// reference was resolved against, so a nested document can use files
// anywhere inside the outermost document's tree, but never outside it.
// With allowExternal true the path is used as given.
func ReadUnder(rootDir, resolved string, limit int64, allowExternal bool) ([]byte, error) {
	f, err := OpenUnder(rootDir, resolved, allowExternal)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readLimited(f, resolved, limit)
}

func readLimited(f *os.File, name string, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("filex: %s is larger than %d bytes", name, limit)
	}
	return data, nil
}

// Open is Read's streaming twin for readers that only need the start of a
// file, such as image sizing. It applies the same confinement, rejects
// non-regular files and leaves closing the file to the caller.
func Open(baseDir, ref string, allowExternal bool) (*os.File, error) {
	if err := checkRef(ref); err != nil {
		return nil, err
	}
	if !allowExternal && filepath.IsAbs(ref) {
		return nil, fmt.Errorf("filex: absolute path %q is outside the base directory", ref)
	}
	p := ref
	if !filepath.IsAbs(p) {
		p = filepath.Join(baseDir, p)
	}
	return openResolved(baseDir, p, allowExternal)
}

// OpenUnder is Open for an already-resolved path; see ReadUnder.
func OpenUnder(rootDir, resolved string, allowExternal bool) (*os.File, error) {
	if err := checkRef(resolved); err != nil {
		return nil, err
	}
	return openResolved(rootDir, resolved, allowExternal)
}

func checkRef(ref string) error {
	if ref == "" {
		return errors.New("filex: empty file reference")
	}
	if strings.HasPrefix(ref, "//") || strings.HasPrefix(ref, `\\`) {
		return fmt.Errorf("filex: network path %q is not allowed", ref)
	}
	return nil
}

// openResolved opens path, which the caller resolved against some document
// directory, confining it to rootDir unless allowExternal is set. The
// target must be a regular file.
func openResolved(rootDir, path string, allowExternal bool) (*os.File, error) {
	if rootDir == "" {
		rootDir = "."
	}
	var f *os.File
	var err error
	if allowExternal {
		// Check before opening: opening a FIFO would block.
		if err := checkRegular(os.Stat(path)); err != nil {
			return nil, err
		}
		f, err = os.Open(path)
	} else {
		absRoot, err := filepath.Abs(rootDir)
		if err != nil {
			return nil, err
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(absRoot, absPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("filex: path %q is outside the base directory", path)
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
