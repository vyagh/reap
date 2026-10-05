//go:build windows

package tui

// lightBackground does not ask: the OSC 11 query needs a Unix terminal.
func lightBackground() (light, known bool) {
	return false, false
}
