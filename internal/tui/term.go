// Package tui is reap's full-screen view, drawn on a tcell screen.
package tui

import (
	"cmp"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
)

// colours is the colour count of the terminal's own compiled terminfo entry,
// which is what curses goes by. It is 0 when no entry is found or readable,
// and tcell then guesses from the name of TERM.
func colours() int {
	name := os.Getenv("TERM")
	if name == "" {
		return 0
	}
	for _, dir := range terminfoDirs() {
		for _, sub := range []string{name[:1], fmt.Sprintf("%x", name[0])} {
			if data, err := os.ReadFile(filepath.Join(dir, sub, name)); err == nil {
				return coloursIn(data)
			}
		}
	}
	return 0
}

// terminfoDirs are the folders curses reads entries from, in its order.
func terminfoDirs() []string {
	var dirs []string
	if d := os.Getenv("TERMINFO"); d != "" {
		dirs = append(dirs, d)
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".terminfo"))
	}
	if v, ok := os.LookupEnv("TERMINFO_DIRS"); ok {
		for _, d := range strings.Split(v, ":") {
			dirs = append(dirs, cmp.Or(d, "/usr/share/terminfo"))
		}
	}
	return append(dirs, "/etc/terminfo", "/lib/terminfo", "/usr/share/terminfo")
}

// coloursIn reads the colors number, the 14th, from a compiled entry: six
// 16 bit counts (magic, names, booleans, numbers, strings, string table), the
// names, the booleans padded to an even length, then the numbers. They are 16
// bits wide, or 32 in the extended format (magic 0x021e). 0 if it has none.
func coloursIn(data []byte) int {
	const colorsIndex = 13
	if len(data) < 12 {
		return 0
	}
	count := func(i int) int { return int(binary.LittleEndian.Uint16(data[2*i:])) }
	width := 2
	switch count(0) {
	case 0x011a:
	case 0x021e:
		width = 4
	default:
		return 0
	}
	at := 12 + count(1) + count(2)
	at += at % 2
	at += colorsIndex * width
	if count(3) <= colorsIndex || len(data) < at+width {
		return 0
	}
	if width == 2 {
		return max(0, int(int16(binary.LittleEndian.Uint16(data[at:]))))
	}
	return max(0, int(int32(binary.LittleEndian.Uint32(data[at:]))))
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
// Left, Right, PgUp, PgDn, Enter (Ctrl-J too, reap:1686), Esc, Tab, BTab or
// BSpace. Anything else, and any key held with Alt, is "".
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
	case tcell.KeyEnter, tcell.KeyCtrlJ, tcell.KeyLF:
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
