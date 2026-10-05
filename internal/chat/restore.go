package chat

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MoveBack puts the files of a trash entry back. A file goes back only if its
// stored name is a plain name inside the entry, its original path lies under one
// of roots (the folders reap owns, such as projects/), the stored copy is still
// there and nothing exists at the original path.
//
// It tries every file and returns the original paths of those that did not go
// back, with an error for each. Those stay in the entry. The caller removes the
// entry once left is empty.
func MoveBack(e Entry, roots []string) (left []string, err error) {
	resolved := make([]string, len(roots))
	for i, r := range roots {
		resolved[i] = realPath(r)
	}
	var failures []error
	for _, m := range e.Record.Moved {
		if err := moveOneBack(e.Dir, m[0], m[1], resolved); err != nil {
			left = append(left, m[0])
			failures = append(failures, err)
		}
	}
	return left, errors.Join(failures...)
}

// moveOneBack moves dir/base to src. 0.5.0 lets "", "." and ".." pass the
// plain-name test, which would move the entry folder or the whole trash. They are
// refused here.
func moveOneBack(dir, src, base string, roots []string) error {
	if base == "" || base == "." || base == ".." || base != filepath.Base(base) {
		return fmt.Errorf("%s: the stored name %q is not a plain file name", src, base)
	}
	if !under(realPath(src), roots) {
		return fmt.Errorf("%s: outside the folders reap restores into", src)
	}
	stored := filepath.Join(dir, base)
	if !exists(stored) {
		return fmt.Errorf("%s: the saved copy %s is missing", src, stored)
	}
	if exists(src) {
		return fmt.Errorf("%s: something is already there", src)
	}
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		return fmt.Errorf("%s: %w", src, err)
	}
	if err := rename(stored, src); err != nil {
		return fmt.Errorf("%s: %w", src, err)
	}
	return nil
}

func under(path string, roots []string) bool {
	for _, r := range roots {
		if path == r || strings.HasPrefix(path, r+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// realPath is the path with symlinks resolved as far as it exists. A restore
// target usually does not exist yet, which filepath.EvalSymlinks refuses.
func realPath(path string) string {
	path, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}
	return filepath.Join(realPath(parent), filepath.Base(path))
}
