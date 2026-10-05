package tui

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/gdamore/tcell/v3"

	"github.com/vyagh/reap/internal/chat"
	"github.com/vyagh/reap/internal/cli"
)

func init() {
	keys["t"] = (*ui).openTrash
	keys["r"] = (*ui).restore
	keys["x"] = (*ui).purge
	keys["E"] = (*ui).empty
}

// trashLoader lists the trash entries as chats, projects with the newest deletion first.
func trashLoader(start cli.Start, left map[string]int, entries map[string]chat.Entry) func(bool) ([]chat.Chat, error) {
	return func(bool) ([]chat.Chat, error) {
		now := time.Now()
		clear(left)
		clear(entries)
		var chats []chat.Chat
		recent := map[string]time.Time{}
		for _, e := range chat.ListTrash(start.Homes, start.CwdOf) {
			c := e.Chat()
			chats = append(chats, c)
			left[key(&c)] = chat.DaysLeft(c.Mod, now)
			entries[key(&c)] = e
			if c.Mod.After(recent[c.Proj]) {
				recent[c.Proj] = c.Mod
			}
		}
		slices.SortStableFunc(chats, func(a, b chat.Chat) int {
			return cmp.Or(
				recent[b.Proj].Compare(recent[a.Proj]),
				cmp.Compare(a.Proj, b.Proj),
				b.Mod.Compare(a.Mod),
			)
		})
		return chats, nil
	}
}

func (u *ui) openTrash() {
	left := map[string]int{}
	entries := map[string]chat.Entry{}
	t, err := newUI(u.s, u.pal, u.start, trashLoader(u.start, left, entries), true, true, u.start.Home)
	if err != nil {
		u.msg = message{text: err.Error()}
		return
	}
	t.left, t.entries = left, entries
	t.run()
	u.pinned = chat.LoadState(u.start.Homes).Pin
	if t.changed {
		if err := u.reload(); err != nil {
			u.msg = message{text: err.Error()}
			return
		}
		u.forgetUndone()
		u.cur, u.top = 0, 0
	}
	if len(t.restored) > 0 {
		u.focusRestored(t.restored)
	}
}

// forgetUndone drops entries purged or restored in the trash view from the undo lists.
func (u *ui) forgetUndone() {
	var kept [][]chat.Entry
	for _, group := range u.undo {
		var there []chat.Entry
		for _, e := range group {
			if fi, err := os.Stat(e.Dir); err == nil && fi.IsDir() {
				there = append(there, e)
			}
		}
		if len(there) > 0 {
			kept = append(kept, there)
		}
	}
	u.undo = kept
}

func (u *ui) focusRestored(back []chat.Chat) {
	ids := map[string]bool{}
	for _, c := range back {
		ids[c.ID] = true
	}
	all := make([]*chat.Chat, len(u.chats))
	for i := range u.chats {
		all[i] = &u.chats[i]
	}
	for _, c := range order(all, u.sortMode, u.grouped) {
		if ids[c.ID] && (u.tab == "" || u.tab == "all" || u.tab == string(c.Source)) {
			u.focus = key(c)
			break
		}
	}
	if len(back) == 1 {
		u.msg = message{"restored " + chat.Fit(back[0].Label, 40), true}
	} else {
		u.msg = message{"restored " + chat.Plural(len(back)), true}
	}
}

func (u *ui) pickedChats() []*chat.Chat {
	var sel []*chat.Chat
	for i := range u.chats {
		if u.picked[key(&u.chats[i])] {
			sel = append(sel, &u.chats[i])
		}
	}
	return sel
}

// restore puts back the picked chats, or the one under the cursor. A chat that did not fully come back stays.
func (u *ui) restore() {
	if !u.trashing {
		return
	}
	sel := u.pickedChats()
	if c := u.f.at(u.cur); len(sel) == 0 && c != nil {
		sel = []*chat.Chat{c}
	}
	var failed []*chat.Chat
	for _, c := range sel {
		e, ok := u.entries[key(c)]
		if a := u.start.Agents.For(c.Source); ok && a != nil && a.Restore(e) == nil {
			u.restored = append(u.restored, *c)
		} else {
			failed = append(failed, c)
		}
	}
	switch {
	case len(failed) == 1:
		u.msg = message{text: "could not restore " + chat.Fit(failed[0].Label, 40)}
	case len(failed) > 1:
		u.msg = message{text: "could not restore " + chat.Plural(len(failed))}
	case len(sel) > 0:
		u.msg = message{"restored " + chat.Plural(len(sel)), true}
	}
	u.trashChanged(sel)
}

func (u *ui) purge() {
	if !u.trashing {
		return
	}
	sel := u.pickedChats()
	switch {
	case len(sel) == 0:
		u.msg = message{text: "nothing selected"}
	case u.confirmDelete(sel, true):
		u.msg = u.removeForGood(sel, "purged ")
		u.trashChanged(sel)
	default:
		u.msg = message{text: "cancelled"}
	}
}

// empty purges every chat the view shows, after asking. With a filter on, only the rows it matches.
func (u *ui) empty() {
	if !u.trashing {
		return
	}
	shown := u.f.tabChats
	if u.flt != "" {
		shown = u.f.vis
	}
	if len(shown) == 0 {
		return
	}
	var size int64
	for _, c := range shown {
		size += c.Size
	}
	freed := chat.Human(size)
	prompt := "empty trash"
	if u.flt != "" {
		prompt = fmt.Sprintf("empty %d shown", len(shown))
	}
	if u.tab != "" && u.tab != "all" {
		prompt += " · " + u.tab
	}
	if u.flt == "" {
		prompt += " · " + chat.Plural(len(shown)) + " · " + freed
	} else {
		prompt += " · " + freed
	}
	if !u.confirmAsk(prompt, "frees the disk · gone for good · other cancels") {
		u.msg = message{text: "cancelled"}
		return
	}
	sel := slices.Clone(shown)
	u.msg = u.removeForGood(sel, "emptied ")
	if u.msg.ok {
		u.msg.text += " · " + freed
	}
	u.trashChanged(sel)
}

// removeForGood purges the entries of sel. 0.5.0 says it purged them all, even when a purge failed.
func (u *ui) removeForGood(sel []*chat.Chat, done string) message {
	var failed []*chat.Chat
	for _, c := range sel {
		e, ok := u.entries[key(c)]
		if !ok {
			continue
		}
		if a := u.start.Agents.For(c.Source); a == nil || a.Purge(e) != nil {
			failed = append(failed, c)
		}
	}
	switch {
	case len(failed) == 1:
		return message{text: "could not purge " + chat.Fit(failed[0].Label, 40)}
	case len(failed) > 1:
		return message{text: "could not purge " + chat.Plural(len(failed))}
	}
	return message{done + chat.Plural(len(sel)), true}
}

func (u *ui) trashChanged(sel []*chat.Chat) {
	if len(sel) == 0 {
		return
	}
	u.changed = true
	u.picked = map[string]bool{}
	if err := u.reload(); err != nil {
		u.msg = message{text: err.Error()}
	}
}

// confirmAsk reports whether the next key is y. Any other key, a click or a resize cancels.
func (u *ui) confirmAsk(prompt, hint string) bool {
	w, h := u.s.Size()
	ww := min(max(rlen(prompt)+6, rlen(hint)+6, 44), w-2)
	y0, x0 := max(0, h/2-2), max(0, (w-ww)/2)
	u.box(y0, x0, 5, ww, prompt, u.pal.danger)
	u.add(y0+2, x0+3, "y", u.pal.danger)
	u.add(y0+2, x0+5, hint, u.pal.dim)
	return u.askedYes()
}

func (u *ui) askedYes() bool {
	u.s.Show()
	ev, ok := u.wait(0).(*tcell.EventKey)
	return ok && (keyName(ev) == "y" || keyName(ev) == "Y")
}
