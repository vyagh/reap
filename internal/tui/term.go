// Package tui is reap's full-screen view, drawn on a tcell screen.
package tui

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// colours is the colour count of the terminal's own entry, which is what
// curses goes by (tput reads the same entry). It is 0 when the entry cannot
// be read, and tcell then guesses from the name of TERM.
func colours() int {
	out, err := exec.Command("tput", "colors").Output()
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return n
}

// openScreen starts tcell. Without it tcell reads no terminfo and takes any
// TERM it does not know for a 256 colour terminal.
func openScreen() (tcell.Screen, error) {
	var opts []tcell.TerminfoScreenOption
	if n := colours(); n > 0 {
		opts = append(opts, tcell.OptColors(n))
	}
	s, err := tcell.NewTerminfoScreen(opts...)
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	s.SetStyle(tcell.StyleDefault.Foreground(color.Reset).Background(color.Reset))
	s.HideCursor()
	s.EnableMouse(tcell.MouseButtonEvents)
	return s, nil
}

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

// parseBackground weighs the colour in an OSC 11 reply as perceived
// brightness, scaling each channel to 0..255 whatever its digit count.
func parseBackground(reply string) (light, known bool) {
	_, rgb, ok := strings.Cut(strings.ToLower(reply), "rgb:")
	parts := strings.SplitN(rgb, "/", 3)
	if !ok || len(parts) != 3 {
		return false, false
	}
	var c [3]int64
	for i, part := range parts {
		hex := hexPrefix(part)
		n, err := strconv.ParseInt(hex, 16, 64)
		if err != nil {
			return false, false
		}
		c[i] = n * 255 / (1<<(4*len(hex)) - 1)
	}
	return c[0]*299+c[1]*587+c[2]*114 >= 128000, true
}

// hexPrefix is the run of hex digits s starts with.
func hexPrefix(s string) string {
	for i, r := range s {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return s[:i]
		}
	}
	return s
}

// add writes text at row y, column x, and stops one column short of the right
// edge the way curses' addstr does there. What falls outside is dropped.
func (u *ui) add(y, x int, text string, style tcell.Style) {
	w, h := u.s.Size()
	if y < 0 || y >= h || x < 0 {
		return
	}
	r := []rune(text)
	if n := max(0, w-1-x); len(r) > n {
		r = r[:n]
	}
	u.s.PutStrStyled(x, y, string(r), style)
}

// put writes one frame cell, the last column included.
func (u *ui) put(y, x int, text string, style tcell.Style) {
	u.s.PutStrStyled(x, y, text, style)
}

// keyName is how the dispatch spells a key: the character typed, or Up, Down,
// Left, Right, PgUp, PgDn, Enter, Esc, Tab, BTab or BSpace. Anything else,
// and any key held with Alt, is "".
func keyName(ev *tcell.EventKey) string {
	if ev.Modifiers()&tcell.ModAlt != 0 {
		return ""
	}
	switch ev.Key() {
	case tcell.KeyRune:
		return ev.Str()
	case tcell.KeyUp:
		return "Up"
	case tcell.KeyDown:
		return "Down"
	case tcell.KeyLeft:
		return "Left"
	case tcell.KeyRight:
		return "Right"
	case tcell.KeyPgUp:
		return "PgUp"
	case tcell.KeyPgDn:
		return "PgDn"
	case tcell.KeyEnter:
		return "Enter"
	case tcell.KeyEsc:
		return "Esc"
	case tcell.KeyTab:
		return "Tab"
	case tcell.KeyBacktab:
		return "BTab"
	case tcell.KeyBackspace:
		return "BSpace"
	}
	return ""
}

// mouseReported are the presses Python's curses asks for: the left button
// and the wheel. Releases and the other buttons never reach reap (reap:932).
const mouseReported = tcell.ButtonPrimary | tcell.WheelUp | tcell.WheelDown

// wait returns the next key, mouse or resize event, or nil once d has passed
// or the screen is gone. A d of 0 waits as long as it takes.
func (u *ui) wait(d time.Duration) tcell.Event {
	var timeout <-chan time.Time
	if d > 0 {
		timeout = time.After(d)
	}
	for {
		select {
		case ev, ok := <-u.s.EventQ():
			if !ok {
				u.quit = true
				return nil
			}
			switch ev := ev.(type) {
			case *tcell.EventKey, *tcell.EventResize:
				return ev
			case *tcell.EventMouse:
				if ev.Buttons()&mouseReported != 0 {
					return ev
				}
			}
		case <-timeout:
			return nil
		}
	}
}
