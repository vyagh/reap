//go:build !linux && !darwin

package codex

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// lockLive reports whether Codex has the chat open. There is no lock probe here,
// so a lock file that exists counts as running.
func lockLive(codexHome, id string) bool {
	_, err := os.Stat(filepath.Join(codexHome, "thread-writer-locks", id+".lock"))
	return !errors.Is(err, fs.ErrNotExist)
}
