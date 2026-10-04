package tui

import (
	"fmt"
	"slices"
	"time"

	"github.com/vyagh/reap/internal/chat"
)

// Cursor moves. All of them stay on lines the cursor can stand on (f.ci).

func (u *ui) goTo(pos int) {
	if len(u.f.ci) > 0 {
		u.cur = u.f.ci[max(0, min(pos, len(u.f.ci)-1))]
	}
}

func (u *ui) down()     { u.goTo(u.f.pos + 1) }
func (u *ui) up()       { u.goTo(u.f.pos - 1) }
func (u *ui) pageDown() { u.goTo(u.f.pos + u.f.body) }
func (u *ui) pageUp()   { u.goTo(u.f.pos - u.f.body) }
func (u *ui) first()    { u.goTo(0) }
func (u *ui) last()     { u.goTo(len(u.f.ci) - 1) }

// headers are the items that open a group.
func (u *ui) headers() []int {
	var hd []int
	for i, it := range u.f.items {
		if it.kind == hdr {
			hd = append(hd, i)
		}
	}
	return hd
}

// nextGroup moves to the first chat of the next group.
func (u *ui) nextGroup() {
	if !u.grouped {
		return
	}
	hd := u.headers()
	gh := -1
	for _, i := range hd {
		if i <= u.cur {
			gh = i
		}
	}
	for _, i := range hd {
		if i > gh {
			u.cur = i + 1
			return
		}
	}
}

// prevGroup moves to the first chat of this group, or of the one before it
// when already there.
func (u *ui) prevGroup() {
	if !u.grouped {
		return
	}
	gh, prev := -1, -1
	for _, i := range u.headers() {
		if i <= u.cur {
			prev, gh = gh, i
		}
	}
	switch {
	case gh >= 0 && u.cur > gh+1:
		u.cur = gh + 1
	case prev >= 0:
		u.cur = prev + 1
	}
}

// fold opens or closes the group the cursor is in.
func (u *ui) fold() {
	if !u.grouped {
		return
	}
	if gi := groupOf(u.f.items, u.cur); gi >= 0 {
		proj := u.f.items[gi].proj
		u.folded[proj] = !u.folded[proj]
	}
}

// foldAll closes every shown group, or opens them all when they are closed.
func (u *ui) foldAll() {
	if !u.grouped {
		return
	}
	all := true
	for _, c := range u.f.vis {
		all = all && u.folded[c.Proj]
	}
	if all {
		clear(u.folded)
	} else {
		for _, c := range u.f.vis {
			u.folded[c.Proj] = true
		}
	}
	u.cur, u.top = 0, 0
}

func (u *ui) nextSort() {
	u.sortMode = sorts[(slices.Index(sorts, u.sortMode)+1)%len(sorts)]
	u.cur, u.top = 0, 0
}

func (u *ui) switchTab(i int) {
	n := len(u.tabs)
	u.tab = u.tabs[(i%n+n)%n]
	u.cur, u.top = 0, 0
	u.tip = true
}

func (u *ui) nextTab() {
	if len(u.tabs) > 1 {
		u.switchTab(slices.Index(u.tabs, u.tab) + 1)
	}
}

func (u *ui) prevTab() {
	if len(u.tabs) > 1 {
		u.switchTab(slices.Index(u.tabs, u.tab) - 1)
	}
}

// save writes the state file and says so when that fails.
func (u *ui) save(st chat.State) {
	if err := chat.SaveState(u.start.Homes, st, u.start.Agents); err != nil {
		u.msg = message{text: "could not save: " + err.Error()}
	}
}

// pin makes the current tab the one reap opens on, or undoes that.
func (u *ui) pin() {
	if len(u.tabs) < 2 {
		return
	}
	st := chat.LoadState(u.start.Homes)
	if u.tab == "all" || u.tab == u.pinned {
		st.Pin, u.pinned = "", ""
		u.msg = message{"reap opens on the all tab again", true}
	} else {
		st.Pin, u.pinned = u.tab, u.tab
		u.msg = message{fmt.Sprintf("reap opens on the %s tab from now on", u.tab), true}
	}
	u.save(st)
}

// pick toggles the chat under the cursor and moves down. A running chat
// cannot be picked.
func (u *ui) pick() {
	c := u.f.at(u.cur)
	if c == nil {
		return
	}
	if c.Live {
		u.msg = message{text: "can't pick a live session"}
	} else if u.picked[c.ID] {
		delete(u.picked, c.ID)
	} else {
		u.picked[c.ID] = true
	}
	u.goTo(u.f.pos + 1)
}

// pickAll picks every chat shown that can be picked, or unpicks them when
// they all are.
func (u *ui) pickAll() {
	var able []string
	for _, c := range u.f.vis {
		if !c.Live && !(u.grouped && u.folded[c.Proj]) {
			able = append(able, c.ID)
		}
	}
	all := len(able) > 0
	for _, id := range able {
		all = all && u.picked[id]
	}
	for _, id := range able {
		if all {
			delete(u.picked, id)
		} else {
			u.picked[id] = true
		}
	}
}

// hide hides the picked chats from the list, or the one under the cursor. It
// brings back what is already hidden. Nothing is deleted.
func (u *ui) hide() {
	st := chat.LoadState(u.start.Homes)
	now := float64(time.Now().UnixNano()) / 1e9
	if len(u.picked) > 0 {
		var hidden, fresh []*chat.Chat
		for id := range u.picked {
			if c, ok := u.by[id]; ok && c.Hidden {
				hidden = append(hidden, c)
			} else if ok {
				fresh = append(fresh, c)
			}
		}
		if len(hidden) == len(u.picked) {
			for _, c := range hidden {
				u.setHidden(&st, c, false, now)
			}
			u.msg = message{"back in the list: " + chat.Plural(len(hidden)), true}
		} else {
			for _, c := range fresh {
				u.setHidden(&st, c, true, now)
			}
			u.msg = message{fmt.Sprintf("hid %s · . shows hidden", chat.Plural(len(fresh))), true}
		}
		u.picked = map[string]bool{}
		u.save(st)
		return
	}
	c := u.f.at(u.cur)
	if c == nil {
		return
	}
	if c.Hidden {
		u.setHidden(&st, c, false, now)
		u.msg = message{"back in the list: " + chat.Fit(c.Label, 40), true}
	} else {
		u.setHidden(&st, c, true, now)
		u.msg = message{fmt.Sprintf("hidden %s · . shows hidden", chat.Fit(c.Label, 40)), true}
	}
	u.save(st)
}

// setHidden marks c hidden or not in the list and in the state to write.
func (u *ui) setHidden(st *chat.State, c *chat.Chat, hidden bool, now float64) {
	c.Hidden = hidden
	if hidden {
		st.Keep[c.ID], u.keep[c.ID] = now, now
	} else {
		delete(st.Keep, c.ID)
		delete(u.keep, c.ID)
	}
}

// toggleHidden shows or hides the hidden chats, and keeps the cursor on its
// chat, or on the nearest one still shown.
func (u *ui) toggleHidden() {
	c := u.f.at(u.cur)
	u.showKept = !u.showKept
	if c == nil {
		u.cur, u.top = 0, 0
		return
	}
	at := slices.Index(u.f.allVis, c)
	var near *chat.Chat
	best := [2]int{}
	for i, x := range u.f.allVis {
		if !u.showKept && x.Hidden {
			continue
		}
		d := [2]int{0, max(i-at, at-i)}
		if x.Proj != c.Proj {
			d[0] = 1
		}
		if near == nil || d[0] < best[0] || d[0] == best[0] && d[1] < best[1] {
			near, best = x, d
		}
	}
	// Python reuses the variable that holds the first frame's folder here,
	// so the cursor only follows when its chat is the first one in the list
	// (reap:1680-1683, 1210-1212).
	if near != nil && at == 0 {
		u.focus = near.ID
	}
}
