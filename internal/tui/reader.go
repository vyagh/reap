package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/gdamore/tcell/v3"

	"github.com/vyagh/reap/internal/chat"
)

func init() {
	keys["p"] = (*ui).peek
}

// readerTurns is how many messages the reader loads (reap:325).
const readerTurns = 250

// readerLine is one line of the reader. Its tag is "u" for the line that
// names the user, "a" for the one that names the agent, "" for text.
type readerLine struct{ tag, text string }

// peek opens the reader on the chat under the cursor (reap:1612).
func (u *ui) peek() {
	c := u.f.at(u.cur)
	if c == nil {
		return
	}
	turns, _ := u.start.Agents.For(c.Source).Peek(*c, readerTurns, false)
	w, _ := u.s.Size()
	u.read(c, readerLines(turns, w))
}

// readerLines lays the turns out for a terminal w columns wide: a name line,
// the wrapped text and a blank line per turn. The width is taken once, when
// the reader opens (reap:1097-1104).
func readerLines(turns []chat.Turn, w int) []readerLine {
	var lines []readerLine
	for _, t := range turns {
		tag := "a"
		if t.Who == "you" {
			tag = "u"
		}
		lines = append(lines, readerLine{tag, bar + " " + t.Who})
		wrapped := wrapText(strings.ReplaceAll(t.Text, "**", ""), max(10, w-8))
		if len(wrapped) == 0 {
			wrapped = []string{""}
		}
		for _, ln := range wrapped {
			lines = append(lines, readerLine{"", "   " + ln})
		}
		lines = append(lines, readerLine{})
	}
	if len(lines) == 0 {
		lines = []readerLine{{"", "(no readable messages)"}}
	}
	return lines
}

// read shows the lines and scrolls them until q, Esc or p (reap:1105-1132).
func (u *ui) read(c *chat.Chat, lines []readerLine) {
	off := 0
	for {
		_, h := u.s.Size()
		page := max(1, h-2)
		off = max(0, min(off, max(0, len(lines)-page)))
		u.drawReader(c, lines, off, page)
		u.s.Show()
		ev := u.wait(0)
		switch ev := ev.(type) {
		case nil:
			return
		case *tcell.EventKey:
			switch keyName(ev) {
			case "q", "Esc", "p":
				return
			case "Down", "j":
				off++
			case "Up", "k":
				off--
			case "PgDn":
				off += page
			case "PgUp":
				off -= page
			case "g":
				off = 0
			case "G":
				off = len(lines)
			}
		case *tcell.EventMouse:
			switch {
			case ev.Buttons()&tcell.WheelUp != 0:
				off -= 3
			case ev.Buttons()&tcell.WheelDown != 0:
				off += 3
			}
		}
	}
}

// drawReader paints one page of the reader inside a frame titled with the
// chat's label, with the scroll bar on the bottom edge when there is more
// than a page.
func (u *ui) drawReader(c *chat.Chat, lines []readerLine, off, page int) {
	p := &u.pal
	w, h := u.s.Size()
	u.s.Clear()
	for j, ln := range lines[off:min(off+page, len(lines))] {
		style := tcell.Style{}
		switch ln.tag {
		case "u":
			style = p.ok
		case "a":
			style = p.acc.Bold(true)
		}
		u.add(1+j, 2, ln.text, style)
	}
	u.drawFrame(0, h-1, 0, w-1, -1, nil)
	if title := chat.Fit(c.Label, w-8); title != "" {
		u.add(0, 2, " "+title+" ", p.bold)
	}
	back := []hint{{"q", "back"}}
	u.hints(h-1, 3, []hint{{"↑/↓", "scroll"}, {"g/G", "ends"}})
	u.hints(h-1, w-3-hintSpan(back), back)
	cells := min(28, w-50)
	if len(lines) > page && cells >= 6 {
		u.progress(h-1, w-12-hintSpan(back)-cells, float64(off)/float64(len(lines)-page), cells)
	}
}

// progress draws how far down the reader is: a bar of cells and a percent.
func (u *ui) progress(y, x int, frac float64, cells int) {
	full, pct := barSize(frac, cells)
	u.add(y, x, " "+strings.Repeat("█", full), u.pal.acc)
	u.add(y, x+1+full, strings.Repeat("░", cells-full), u.pal.line)
	u.add(y, x+1+cells, fmt.Sprintf(" %02d%% ", pct), u.pal.dim)
}

// barSize is the filled cells and the percent for frac. Python's round goes
// to the even number on a half, so does this.
func barSize(frac float64, cells int) (full, pct int) {
	full = int(math.RoundToEven(max(0, min(1, frac)) * float64(cells)))
	return full, int(math.RoundToEven(frac * 100))
}
