//go:build !linux && !darwin

package codex

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// lockLive reports whether Codex still has the chat open. This system has no
// lock probe, so a lock file that exists counts as running and only a missing
// file means it is not.
func lockLive(codexHome, id string) bool {
	_, err := os.Stat(filepath.Join(codexHome, "thread-writer-locks", id+".lock"))
	return !errors.Is(err, fs.ErrNotExist)
}
