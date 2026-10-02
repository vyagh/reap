//go:build linux || darwin

package claude

import (
	"errors"
	"syscall"
)

// pidGone is true only when signal 0 says there is no such process. A process
// we may not signal, or a zombie, is still there.
func pidGone(pid int) bool {
	return errors.Is(syscall.Kill(pid, 0), syscall.ESRCH)
}
