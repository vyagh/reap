package tui

import (
	"cmp"
	"slices"
	"strings"

	"github.com/vyagh/reap/internal/chat"
)

type kind int

const (
	gap   kind = iota
	hdr        // project group header
	row        // one chat
	label      // the "hidden" divider
)

type item struct {
	kind kind
	chat *chat.Chat // row
	// hdr: the project, how many chats and bytes it holds, and its folder.
	proj string
	n    int
	size int64
	cwd  string
}

type span struct {
	x0, x1 int
	tab    string
}

// frame is what one pass of the loop works out from the state before it draws. A
// key handler reads it as the screen the key was pressed on.
type frame struct {
	w, h     int
	tabChats []*chat.Chat   // the current tab's chats, in loader order
	vis      []*chat.Chat   // the tab's chats after sorting, filter and hidden
	allVis   []*chat.Chat   // vis with the hidden chats, a project can be all hidden
	items    []item         // gap, header, chat and divider lines
	ci       []int          // items the cursor can stand on: chats and folded headers
	pos      int            // where the cursor is in ci
	pin      int            // header drawn above a scrolled list, -1 for none
	body     int            // rows the list has
	hidden   map[string]int // hidden chats per project
	pane     bool           // the terminal is wide enough for the side panel
	cw       int            // list content width
	px       int            // side panel left column
	tabSpans []span         // drawn tabs
	rowAt    map[int]int    // screen row -> item index
}

func (f frame) at(i int) *chat.Chat {
	if i < 0 || i >= len(f.items) || f.items[i].kind != row {
		return nil
	}
	return f.items[i].chat
}

// plan builds the frame and moves the cursor onto a line it can stand on.
func (u *ui) plan() {
	f := &u.f
	f.w, f.h = u.s.Size()
	f.tabChats = f.tabChats[:0]
	for i := range u.chats {
		c := &u.chats[i]
		if u.tab == "" || u.tab == "all" || string(c.Source) == u.tab {
			f.tabChats = append(f.tabChats, c)
		}
	}
	f.allVis = order(f.tabChats, u.sortMode, u.grouped)
	if u.flt != "" {
		match := strings.ToLower(u.flt)
		f.allVis = slices.DeleteFunc(slices.Clone(f.allVis), func(c *chat.Chat) bool {
			return !strings.Contains(strings.ToLower(c.Label), match) && !strings.Contains(c.ID, match) &&
				!strings.Contains(strings.ToLower(c.Proj), match)
		})
	}
	f.vis = f.allVis
	if !u.showKept {
		f.vis = slices.DeleteFunc(slices.Clone(f.vis), func(c *chat.Chat) bool { return c.Hidden })
	}
	f.items, f.hidden = u.lines()

	if u.home != "" {
		u.focus = ""
		for _, it := range f.items {
			if it.kind == row && it.chat.Dir == u.home {
				u.focus = key(it.chat)
				break
			}
		}
		u.home = ""
	}
	if u.focus != "" {
		for i, it := range f.items {
			if it.kind == row && key(it.chat) == u.focus {
				u.cur = i
				break
			}
		}
		u.focus = ""
	}

	f.ci = f.ci[:0]
	for i, it := range f.items {
		if it.kind == row || it.kind == hdr && u.folded[it.proj] {
			f.ci = append(f.ci, i)
		}
	}
	switch {
	case len(f.ci) == 0:
		u.cur = 0
	case u.cur >= len(f.items) || f.items[u.cur].kind != row:
		at := slices.IndexFunc(f.ci, func(i int) bool { return i >= u.cur })
		if at < 0 {
			at = len(f.ci) - 1
		}
		u.cur = f.ci[at]
	}
	f.pos = max(0, slices.Index(f.ci, u.cur))
	u.top, f.pin, f.body = planView(f.items, u.cur, u.top, max(1, f.h-3), u.grouped)

	f.pane = f.w >= listW+paneMin
	f.cw = f.w - 1
	if f.pane {
		f.cw = listW
	}
	f.px = f.cw + 2
	f.tabSpans = f.tabSpans[:0]
	f.rowAt = map[int]int{}
}

// lines lays vis out as gap, header, chat and divider lines. Hidden chats sit
// under their project behind a "hidden" divider. The map counts hidden chats per
// project.
func (u *ui) lines() ([]item, map[string]int) {
	f := &u.f
	var items []item
	hidden := map[string]int{}
	hiddenRows := func(chats []*chat.Chat) {
		var shown, gone []*chat.Chat
		for _, c := range chats {
			if c.Hidden {
				gone = append(gone, c)
			} else {
				shown = append(shown, c)
			}
		}
		for _, c := range shown {
			items = append(items, item{kind: row, chat: c})
		}
		if len(gone) > 0 {
			items = append(items, item{kind: label})
			for _, c := range gone {
				items = append(items, item{kind: row, chat: c})
			}
		}
	}
	if !u.grouped {
		hiddenRows(f.vis)
		return items, hidden
	}
	type stat struct {
		n    int
		size int64
		cwd  string
	}
	stats := map[string]*stat{}
	for _, c := range f.allVis {
		s := stats[c.Proj]
		if s == nil {
			s = &stat{}
			stats[c.Proj] = s
		}
		s.n++
		s.size += c.Size
		if c.Cwd != "" && s.cwd == "" {
			s.cwd = c.Cwd
		}
		if c.Hidden {
			hidden[c.Proj]++
		}
	}
	last, started := "", false
	for _, c := range f.allVis {
		if started && c.Proj == last {
			continue
		}
		if started {
			items = append(items, item{kind: gap})
		}
		last, started = c.Proj, true
		s := stats[c.Proj]
		items = append(items, item{kind: hdr, proj: c.Proj, n: s.n, size: s.size, cwd: s.cwd})
		if u.folded[c.Proj] {
			continue
		}
		var group []*chat.Chat
		for _, x := range f.vis {
			if x.Proj == c.Proj {
				group = append(group, x)
			}
		}
		hiddenRows(group)
	}
	return items, hidden
}

// order sorts the chats by the sort mode. Newest keeps the loader's order. When
// grouped the projects follow the mode too, by their oldest chat or total size.
func order(chats []*chat.Chat, mode string, grouped bool) []*chat.Chat {
	if mode == "newest" {
		return chats
	}
	key := func(c *chat.Chat) int64 { return c.Size }
	if mode == "oldest" {
		key = func(c *chat.Chat) int64 { return c.Mod.UnixNano() }
	}
	sign := int64(1)
	if mode == "biggest" {
		sign = -1
	}
	out := slices.Clone(chats)
	if !grouped {
		slices.SortStableFunc(out, func(a, b *chat.Chat) int { return cmp.Compare(sign*key(a), sign*key(b)) })
		return out
	}
	agg := map[string]int64{}
	for _, c := range chats {
		v, seen := agg[c.Proj]
		switch {
		case !seen, mode == "oldest" && key(c) < v:
			agg[c.Proj] = key(c)
		case mode != "oldest":
			agg[c.Proj] += key(c)
		}
	}
	slices.SortStableFunc(out, func(a, b *chat.Chat) int {
		return cmp.Or(
			cmp.Compare(sign*agg[a.Proj], sign*agg[b.Proj]),
			cmp.Compare(a.Proj, b.Proj),
			cmp.Compare(sign*key(a), sign*key(b)),
		)
	})
	return out
}

func groupOf(items []item, i int) int {
	for j := min(i, len(items)-1); j >= 0; j-- {
		if items[j].kind == hdr {
			return j
		}
	}
	return -1
}

// planView is the scroll position, the header pinned above a scrolled list (or -1) and the rows left.
func planView(items []item, cur, top, avail int, grouped bool) (newTop, pin, body int) {
	if len(items) == 0 {
		return 0, -1, avail
	}
	cur = max(0, min(cur, len(items)-1))
	top = min(top, cur)
	if cur >= top+avail {
		top = cur - avail + 1
	}
	top = max(0, top)
	if grouped {
		if gh := groupOf(items, cur); gh >= 0 && gh < top {
			body = max(1, avail-1)
			if cur >= top+body {
				top = cur - body + 1
			}
			return max(0, top), gh, body
		}
	}
	return top, -1, avail
}
