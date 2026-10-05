//go:build linux || darwin

package codex

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// lockLive reports whether Codex has the chat open. Codex holds a lock on
// thread-writer-locks/<id>.lock while it writes, so a lock someone else holds
// means the chat is running. Only a missing lock file means it is not: a file
// that cannot be opened, or a probe that fails, counts as running.
func lockLive(codexHome, id string) bool {
	f, err := os.OpenFile(filepath.Join(codexHome, "thread-writer-locks", id+".lock"), os.O_RDWR, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	if err != nil {
		return true
	}
	defer f.Close()
	return flockHeld(f) || fcntlHeld(f)
}

func flockHeld(f *os.File) bool {
	fd := int(f.Fd())
	if syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return true
	}
	return syscall.Flock(fd, syscall.LOCK_UN) != nil
}

// fcntlHeld probes a POSIX record lock, the kind Python calls lockf. It and
// flock do not see each other, so both are probed.
func fcntlHeld(f *os.File) bool {
	fd := f.Fd()
	lk := syscall.Flock_t{Type: syscall.F_WRLCK}
	if syscall.FcntlFlock(fd, syscall.F_SETLK, &lk) != nil {
		return true
	}
	lk.Type = syscall.F_UNLCK
	return syscall.FcntlFlock(fd, syscall.F_SETLK, &lk) != nil
}
