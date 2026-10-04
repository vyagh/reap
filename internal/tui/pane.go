package tui

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v3"

	"github.com/vyagh/reap/internal/chat"
)

func init() {
	detail = (*ui).drawPane
}

const (
	paneTurns = 16 // messages read for the panel (reap:1374)
	paneLines = 5  // lines drawn of one message (reap:1381)
)

// drawPane draws the side panel for the item under the cursor (reap:1326-1391):
// a folded group's totals, or a chat's folder, state, size and the start of
// its messages. It returns the title for the panel's top edge.
func (u *ui) drawPane(it item) (string, tcell.Style) {
	f := &u.f
	pw := f.w - f.px - 2
	if pw < 20 {
		return "", tcell.Style{}
	}
	lw := min(pw-2, 28) // the label above the pane stays short
	if it.kind == hdr {
		leaf, parent := chat.SplitPath(u.start.Homes, it.cwd, it.proj)
		y := u.paneField(2, "path", joinPath(parent, leaf))
		y = u.paneField(y, "chats", fmt.Sprint(it.n))
		y = u.paneField(y, "size", chat.Human(it.size))
		u.add(y+1, f.px, "folded · z to open", u.pal.dim)
		return chat.Fit(leaf, lw), u.pal.bold
	}
	c := it.chat
	title, tstyle := chat.Fit("untitled", lw), u.pal.dim
	if c.Titled {
		title, tstyle = chat.Fit(c.Label, lw), u.pal.bold
	}
	leaf, parent := chat.SplitPath(u.start.Homes, c.Cwd, "")
	if full := joinPath(parent, leaf); full != "" {
		u.add(2, f.px, chat.Fit(full, pw), u.pal.dim)
	}
	src := string(c.Source)
	u.add(3, f.px, src, u.pal.dim)
	var state string
	var stateStyle tcell.Style
	switch {
	case c.Hidden:
		state, stateStyle = "hidden "+dayOf(epoch(u.keep[c.ID]))+" · h brings it back", u.pal.dim
	case c.Live:
		state, stateStyle = "running now", u.pal.live
	case u.trashing:
		state, stateStyle = "deleted "+dayOf(c.Mod), u.pal.dim
	}
	if state != "" {
		u.add(3, f.px+rlen(src), " · ", u.pal.dim)
		u.add(3, f.px+rlen(src)+3, chat.Fit(state, max(0, pw-rlen(src)-3)), stateStyle)
	}
	if u.trashing {
		x := f.px + rlen(src) + 3 + rlen(state)
		left := u.left[c.ID]
		style := u.pal.dim
		if left <= 1 {
			style = u.pal.yel
		}
		u.add(3, x, " · ", u.pal.dim)
		u.add(3, x+3, fmt.Sprintf("%dd left", left), style)
	}
	if u.trashing && c.Msgs < 0 && c.Source == chat.Codex { // once per entry
		u.count(c)
	}
	bits := []string{chat.Human(c.Size)}
	if c.Msgs >= 0 {
		bits = append(bits, chat.PluralWord(c.Msgs, "prompt"))
	}
	if !u.trashing {
		bits = append(bits, c.Mod.Format("Jan 02, 15:04"))
	}
	u.add(4, f.px, chat.Fit(strings.Join(bits, " · "), pw), u.pal.dim)
	u.paneTurns(c, pw)
	return title, tstyle
}

// paneTurns draws the first messages of a chat from row 6, each as a name and
// up to five lines under it. The read happens once per chat, so holding j
// stays smooth.
func (u *ui) paneTurns(c *chat.Chat, pw int) {
	f, p := &u.f, &u.pal
	turns, ok := u.previews[c.ID]
	if !ok {
		turns, _ = u.start.Agents.For(c.Source).Peek(*c, paneTurns, true)
		u.previews[c.ID] = turns
	}
	y := 6
	if len(turns) == 0 {
		u.add(y, f.px, "(no readable messages)", p.dim)
	}
	for _, t := range turns {
		if y >= f.h-2 { // room for the name and a line under it
			break
		}
		who := p.acc.Bold(true)
		if t.Who == "you" {
			who = p.ok
		}
		u.add(y, f.px, bar+" "+t.Who, who)
		y++
		lines := wrapText(strings.ReplaceAll(t.Text, "**", ""), pw-2)
		if len(lines) == 0 {
			lines = []string{""}
		}
		for _, ln := range lines[:min(len(lines), paneLines)] {
			if y >= f.h-1 {
				break
			}
			u.add(y, f.px+2, ln, p.dim)
			y++
		}
		y++
	}
}

// paneField draws a dim name with its value beside it, wrapped under itself,
// and returns the row after the value.
func (u *ui) paneField(y int, name, value string) int {
	f := &u.f
	u.add(y, f.px, name, u.pal.dim)
	lines := wrapText(value, max(4, f.w-f.px-2-11))
	if len(lines) == 0 {
		lines = []string{""}
	}
	for i, ln := range lines {
		u.add(y+i, f.px+11, ln, tcell.Style{})
	}
	return y + len(lines)
}

func joinPath(parent, leaf string) string {
	if parent == "" {
		return leaf
	}
	return parent + "/" + leaf
}

// epoch is a time kept as seconds since 1970, the way the state file has it.
func epoch(sec float64) time.Time {
	whole := int64(sec)
	return time.Unix(whole, int64((sec-float64(whole))*1e9))
}

// dayOf is a date as the panel prints it, in the local zone.
func dayOf(t time.Time) string { return t.Format("Jan 02") }

// wrapText breaks text into lines of at most width characters, as Python's
// textwrap.wrap does with its defaults: it breaks at spaces and after a
// hyphen inside a word, and cuts a word that is longer than a line.
func wrapText(text string, width int) []string {
	chunks := wrapChunks([]rune(strings.NewReplacer("\t", " ", "\n", " ", "\v", " ", "\f", " ", "\r", " ").Replace(text)))
	var lines []string
	for len(chunks) > 0 {
		var line []string
		n := 0
		if len(lines) > 0 && isBlank(chunks[0]) {
			chunks = chunks[1:]
		}
		for len(chunks) > 0 && n+rlen(chunks[0]) <= width {
			line = append(line, chunks[0])
			n += rlen(chunks[0])
			chunks = chunks[1:]
		}
		if len(chunks) > 0 && rlen(chunks[0]) > width {
			room := width - n
			r := []rune(chunks[0])
			end := room
			if len(r) > room {
				h := room - 1
				for h >= 0 && r[h] != '-' {
					h--
				}
				if h > 0 && strings.Trim(string(r[:h]), "-") != "" {
					end = h + 1
				}
			}
			line = append(line, string(r[:end]))
			chunks[0] = string(r[end:])
		}
		if len(line) > 0 && isBlank(line[len(line)-1]) {
			line = line[:len(line)-1]
		}
		if len(line) > 0 {
			lines = append(lines, strings.Join(line, ""))
		}
	}
	return lines
}

func isBlank(s string) bool { return strings.TrimSpace(s) == "" }

// wrapChunks cuts text into runs of spaces and pieces of words. A word breaks
// after a hyphen that has letters on both sides, and before a run of two or
// more dashes that follows a word character.
func wrapChunks(r []rune) []string {
	at := func(i int) rune {
		if i < 0 || i >= len(r) {
			return 0
		}
		return r[i]
	}
	letter := func(i int) bool { return unicode.IsLetter(at(i)) || at(i) == '_' }
	word := func(i int) bool { return letter(i) || unicode.IsDigit(at(i)) }
	punct := func(i int) bool { return word(i) || strings.ContainsRune(`!"'&.,?`, at(i)) }
	dashes := func(i int) int { // a run of dashes at i that a word character follows
		n := 0
		for at(i+n) == '-' {
			n++
		}
		if n >= 2 && word(i+n) {
			return n
		}
		return 0
	}
	var out []string
	for i := 0; i < len(r); {
		end := i + 1
		switch {
		case r[i] == ' ':
			for at(end) == ' ' {
				end++
			}
		case punct(i-1) && dashes(i) > 0:
			end = i + dashes(i)
		default:
			for ; end < len(r); end++ {
				hyphen := at(end) == '-' &&
					(letter(end-2) && letter(end-1) || letter(end-3) && at(end-2) == '-' && letter(end-1)) &&
					letter(end+1) && (letter(end+2) || at(end+2) == '-' && letter(end+3))
				if hyphen {
					end++
					break
				}
				if r[end] == ' ' || punct(end-1) && dashes(end) > 0 {
					break
				}
			}
		}
		out = append(out, string(r[i:end]))
		i = end
	}
	return out
}
