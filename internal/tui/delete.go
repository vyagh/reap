package tui

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/gdamore/tcell/v3"

	"github.com/vyagh/reap/internal/chat"
)

func init() {
	keys["d"] = (*ui).askDelete
	keys["u"] = (*ui).undoDelete
}

// askDelete asks before it moves the picked chats to the trash (reap:1690-1721).
// With nothing picked it does nothing.
func (u *ui) askDelete() {
	if len(u.picked) == 0 {
		u.msg = message{text: "nothing selected"}
		return
	}
	var sel []*chat.Chat
	for id := range u.picked {
		if c, ok := u.by[id]; ok {
			sel = append(sel, c)
		}
	}
	slices.SortFunc(sel, func(a, b *chat.Chat) int {
		return cmp.Or(cmp.Compare(b.Size, a.Size), cmp.Compare(a.ID, b.ID))
	})
	if !u.confirmDelete(sel) {
		u.msg = message{text: "cancelled"}
		return
	}
	u.trashPicked(sel)
}

// trashPicked moves each chat to the trash. A chat can start running between
// the pick and the confirm, so each one is asked again right before it moves,
// and one that is running now is left alone. A chat the agent cannot trash is
// left where it is, and the message says why. The entries that were made go on
// the undo list, even when there are none (reap:1697-1721).
func (u *ui) trashPicked(sel []*chat.Chat) {
	st := chat.LoadState(u.start.Homes)
	var done []chat.Entry
	var fail string
	for _, c := range sel {
		name := []rune(c.Label)
		name = name[:min(40, len(name))]
		agent := u.start.Agents.For(c.Source)
		if agent == nil {
			fail = fmt.Sprintf("no %s agent to trash %s", c.Source, string(name))
			continue
		}
		if agent.Live(*c) {
			fail = string(name) + " is running now, left alone"
			continue
		}
		e, err := agent.Trash(*c)
		if err != nil {
			fail = err.Error()
			continue
		}
		done = append(done, e)
		delete(st.Keep, c.ID)
	}
	if len(done) > 0 {
		u.save(st)
	}
	u.undo = append(u.undo, done)
	u.picked = map[string]bool{}
	if err := u.reload(); err != nil {
		u.msg = message{text: "could not reload: " + err.Error()}
		return
	}
	switch {
	case fail != "":
		u.msg = message{text: fail}
	case len(done) > 0:
		u.msg = message{"trashed " + chat.Plural(len(done)) + " · u undo · t trash", true}
	default:
		u.msg = message{text: "nothing trashed"}
	}
}

// undoDelete puts back the chats of the last delete (reap:1627-1634).
func (u *ui) undoDelete() {
	if len(u.undo) == 0 {
		u.msg = message{text: "nothing to undo"}
		return
	}
	last := u.undo[len(u.undo)-1]
	u.undo = u.undo[:len(u.undo)-1]
	var back []chat.Entry
	var failed error
	for _, e := range last {
		agent := u.start.Agents.For(chat.Source(e.Record.Src))
		if agent == nil {
			continue
		}
		if err := agent.Restore(e); err != nil {
			failed = err
			continue
		}
		back = append(back, e)
	}
	if err := u.reload(); err != nil {
		u.msg = message{text: "could not reload: " + err.Error()}
		return
	}
	switch {
	case len(back) == 0 && failed != nil:
		u.msg = message{text: "could not restore: " + failed.Error()}
	case len(back) == 0:
		u.msg = message{text: "could not restore"}
	case len(back) == 1:
		label := back[0].Record.Label
		if label == "" {
			label = "?"
		}
		u.msg = message{"restored " + chat.Fit(label, 40), true}
	default:
		u.msg = message{"restored " + chat.Plural(len(back)), true}
	}
}

// confirmDelete draws the box that lists what would be deleted, the biggest
// chats first, and reports whether the user answered y (reap:1143-1176).
func (u *ui) confirmDelete(sel []*chat.Chat) bool {
	var total int64
	for _, c := range sel {
		total += c.Size
	}
	w, h := u.s.Size()
	shown := min(len(sel), 8, max(1, h-8))
	more := len(sel) - shown
	ww := min(max(50, w*2/3), w-2)
	hh := shown + 5
	if more > 0 {
		hh++
	}
	y0, x0 := max(0, (h-hh)/2), max(0, (w-ww)/2)
	u.box(y0, x0, hh, ww, fmt.Sprintf("delete %s · %s", chat.Plural(len(sel)), chat.Human(total)), u.pal.bold)
	var others []string
	for _, c := range sel {
		src := string(c.Source)
		if src != string(chat.Claude) && !slices.Contains(others, src) {
			others = append(others, src)
		}
	}
	slices.Sort(others)
	for j, c := range sel[:shown] {
		src := string(c.Source)
		extra := 0
		if src != string(chat.Claude) {
			extra = rlen(src) + 2
		}
		u.add(y0+2+j, x0+3, fmt.Sprintf("%6s", chat.Human(c.Size)), u.pal.dim)
		u.add(y0+2+j, x0+11, chat.Fit(c.Label, ww-14-extra), tcell.Style{})
		if extra > 0 {
			u.add(y0+2+j, x0+ww-2-rlen(src), src, u.pal.dim)
		}
	}
	if more > 0 {
		u.add(y0+2+shown, x0+11, fmt.Sprintf("… and %d more", more), u.pal.dim)
	}
	hint := fmt.Sprintf("move to trash · undo for %dd · other key cancels", chat.TrashDays)
	if len(others) > 0 {
		hint = fmt.Sprintf("move to trash · %s chats via codex archive · undo for %dd", strings.Join(others, ", "), chat.TrashDays)
	}
	u.add(y0+hh-2, x0+3, "y", u.pal.ok)
	u.add(y0+hh-2, x0+5, hint, u.pal.dim)
	u.s.Show()
	ev := u.wait(0)
	key, ok := ev.(*tcell.EventKey)
	return ok && (keyName(key) == "y" || keyName(key) == "Y")
}
