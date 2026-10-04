//go:build windows

package tui

// lightBackground does not ask: the OSC 11 query needs a Unix terminal, so the
// background is never known here.
func lightBackground() (light, known bool) {
	return false, false
}
