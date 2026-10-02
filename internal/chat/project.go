package chat

import (
	"path/filepath"
	"strings"
)

// DirnameFor is the folder name Claude Code gives the project at cwd: every
// character that is not an ASCII letter or digit becomes a dash, so a dot or
// an underscore changes too (reap:109-112).
func DirnameFor(cwd string) string {
	var b strings.Builder
	for _, r := range cwd {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// Pretty is the name the list shows for a project folder: the folder name
// without the home prefix and leading dashes, and ~ for the home folder itself.
// A name that would come out empty is shown whole (reap:147-152).
func Pretty(h Homes, dir string) string {
	base := lastPart(dir)
	prefix := DirnameFor(h.Home)
	if base == prefix {
		return "~"
	}
	name := strings.TrimLeft(strings.TrimPrefix(base, prefix), "-")
	if name == "" {
		return base
	}
	return name
}

// lastPart is what follows the last separator. Unlike filepath.Base it is
// empty for a path that ends in one, as Python's basename is.
func lastPart(path string) string {
	return path[strings.LastIndexByte(path, filepath.Separator)+1:]
}
