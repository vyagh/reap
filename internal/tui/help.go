package tui

import (
	"github.com/gdamore/tcell/v3"

	"github.com/vyagh/reap/internal/chat"
)

// helpLine is one line of the key list: a heading (H), a footnote (D) or
// plain text.
type helpLine struct{ tag, text string }

var helpLines = []helpLine{
	{"H", "move"},
	{"", "  ↑/↓  j/k       line            PgUp/PgDn   page"},
	{"", "  g / G          top / bottom    [ / ]       prev / next workspace"},
	{"", "  z / Z          fold this group / fold all"},
	{"", "  mouse          click a tab, row or group line (folds); wheel scrolls"},
	{"", ""},
	{"H", "select & act"},
	{"", "  space          toggle pick (skips live sessions)"},
	{"", "  a              toggle select-all shown"},
	{"", "  p              peek: read-only preview"},
	{"", "  enter          open the chat under the cursor in its agent (quits reap)"},
	{"", "  d              delete selected → trash (confirms first)"},
	{"", "  h              hide from the list, never deleted (on hidden chats: unhide)"},
	{"", ""},
	{"H", "trash · undo"},
	{"", "  u              undo the last delete"},
	{"", "  t              trash browser: restore / purge"},
	{"", "  space r x E    in the trash: pick, restore, purge, empty"},
	{"", "                 with a filter on, E empties only the rows shown"},
	{"", ""},
	{"H", "view"},
	{"", "  s              sort: newest, oldest, biggest, smallest"},
	{"", "  .              show / hide again hidden chats"},
	{"", "  tab            next agent tab"},
	{"", "  shift-tab      previous agent tab"},
	{"", "  P              make this tab the one reap opens on (again: undo) · shown as *"},
	{"", "  /              filter title · uuid · workspace"},
	{"", "  q              quit"},
	{"", ""},
	{"D", "●  running    ◉  picked    *  the tab reap opens on"},
	{"D", "delete moves the chat to a trash, recoverable for 7 days. hide only hides it."},
}

// help shows the key list over the screen until any key is pressed.
func (u *ui) help() {
	u.s.Clear()
	w, h := u.s.Size()
	longest := 0
	for _, l := range helpLines {
		longest = max(longest, rlen(l.text))
	}
	ww, hh := min(longest+6, w-2), min(len(helpLines)+4, h)
	y0, x0 := max(0, (h-hh)/2), max(0, (w-ww)/2)
	u.box(y0, x0, hh, ww, "reap · keys", u.pal.bold)
	for i, l := range helpLines[:max(0, hh-4)] {
		style := tcell.Style{}
		switch l.tag {
		case "H":
			style = u.pal.acc.Bold(true)
		case "D":
			style = u.pal.dim
		}
		u.add(y0+2+i, x0+3, chat.Fit(l.text, ww-6), style)
	}
	u.add(y0+hh-2, x0+3, "any key to close", u.pal.dim)
	u.s.Show()
	u.wait(0)
}
