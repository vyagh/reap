//go:build !windows

package tui

import (
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// lightBackground asks the terminal for its background colour with OSC 11.
// known is false when it does not answer within 150 ms (reap:894-919).
func lightBackground() (light, known bool) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false, false
	}
	defer tty.Close()
	fd := int(tty.Fd())
	old, err := term.MakeRaw(fd)
	if err != nil {
		return false, false
	}
	defer term.Restore(fd, old)
	if _, err := tty.WriteString("\x1b]11;?\x1b\\"); err != nil {
		return false, false
	}
	// A local terminal answers in a few ms. The reply is read byte by byte so
	// that typeahead after it stays unread. Fd put the file in blocking mode,
	// so a read deadline would not work; poll waits instead.
	end := time.Now().Add(150 * time.Millisecond)
	var reply []byte
	b := make([]byte, 1)
	for !strings.HasSuffix(string(reply), "\x1b\\") && !strings.HasSuffix(string(reply), "\a") {
		wait := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		if n, err := unix.Poll(wait, max(0, int(time.Until(end).Milliseconds()))); n == 0 || err != nil {
			break
		}
		if _, err := tty.Read(b); err != nil {
			break
		}
		reply = append(reply, b[0])
	}
	return parseBackground(string(reply))
}
