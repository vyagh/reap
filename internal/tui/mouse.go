package tui

import (
	"slices"

	"github.com/gdamore/tcell/v3"
)

func init() {
	mouseEvent = (*ui).mouse
}

// wheelStep is how many lines one turn of the wheel moves the cursor (reap:1582).
const wheelStep = 3

// mouse is one press of the left button or one turn of the wheel (reap:1571-1583).
// A press on a tab opens it, a press on a chat puts the cursor there and a
// press on a group header folds or opens the group. The wheel moves the
// cursor three chats. A press in the side panel, on a divider or on the
// gap does nothing. The trash view acts the same. While the filter is being
// typed any mouse event only sends the cursor back to the top, as any key
// does there (reap:1500-1505).
func (u *ui) mouse(ev *tcell.EventMouse) {
	if u.filtering {
		u.cur, u.top = 0, 0
		return
	}
	x, y := ev.Position()
	switch b := ev.Buttons(); {
	case b&tcell.ButtonPrimary != 0:
		u.click(x, y)
	case b&tcell.WheelUp != 0:
		u.goTo(u.f.pos - wheelStep)
	case b&tcell.WheelDown != 0:
		u.goTo(u.f.pos + wheelStep)
	}
}

// click acts on a press at column x, row y of the list screen.
func (u *ui) click(x, y int) {
	if y == 0 {
		for _, s := range u.f.tabSpans {
			if s.x0 <= x && x < s.x1 {
				u.switchTab(slices.Index(u.tabs, s.tab))
				return
			}
		}
	}
	i, ok := u.f.rowAt[y]
	if !ok || x >= u.f.cw {
		return
	}
	switch it := u.f.items[i]; it.kind {
	case row:
		u.cur = i
	case hdr:
		u.folded[it.proj] = !u.folded[it.proj]
	}
}
