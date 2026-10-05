package tui

import (
	"slices"

	"github.com/gdamore/tcell/v3"
)

func init() {
	mouseEvent = (*ui).mouse
}

const wheelStep = 3

// mouse handles one press of the left button or one turn of the wheel. The wheel
// moves the cursor three chats. A press in the side panel, on a divider or on a
// gap does nothing. While the filter is typed any mouse event only sends the
// cursor to the top, as any key does there.
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
