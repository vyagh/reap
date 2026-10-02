//go:build !linux && !darwin

package claude

// pidGone is never true here, so every chat with a session file counts as
// running.
func pidGone(pid int) bool {
	return false
}
