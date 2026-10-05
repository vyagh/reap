package tui

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/gdamore/tcell/v3"

	"github.com/vyagh/reap/internal/chat"
)

func rlen(s string) int { return utf8.RuneCountInString(s) }

func (u *ui) draw() {
	f := &u.f
	u.s.Clear()
	if len(f.vis) == 0 {
		switch {
		case u.trashing:
			u.add(2, 2, "trash is empty, nothing deleted in the last 7 days", u.pal.dim)
		case u.flt != "":
			u.add(2, 2, "no chats matching /"+u.flt, u.pal.dim)
		default:
			u.add(2, 2, "no chats", u.pal.dim)
		}
	}
	base := 2
	if f.pin >= 0 {
		u.drawHdr(base, f.items[f.pin], false)
		f.rowAt[base] = f.pin
		base++
	}
	for k := 0; k < f.body && u.top+k < len(f.items); k++ {
		idx, y := u.top+k, base+k
		f.rowAt[y] = idx
		switch it := f.items[idx]; it.kind {
		case hdr:
			u.drawHdr(y, it, idx == u.cur)
		case label:
			u.drawDivider(y)
		case row:
			u.drawRow(y, idx == u.cur, it.chat)
		}
	}
	var title string
	var tstyle tcell.Style
	if f.pane && detail != nil && u.cur < len(f.items) && (f.items[u.cur].kind == row || f.items[u.cur].kind == hdr) {
		title, tstyle = detail(u, f.items[u.cur])
	}
	opens := u.drawLabels()
	div := -1
	if f.pane {
		div = f.cw
	}
	u.drawFrame(1, f.h-1, 0, f.w-1, div, opens)
	if title != "" {
		u.add(1, f.px, " "+title+" ", tstyle)
	}
	u.drawBottom()
}

// drawHdr: a folded header shows its size after the name, an open one at the right.
func (u *ui) drawHdr(y int, it item, sel bool) {
	f, p := &u.f, &u.pal
	folded := u.folded[it.proj]
	leaf, _ := chat.SplitPath(u.start.Homes, it.cwd, chat.ProjLabel(it.proj, projMax))
	info := fmt.Sprintf("%s · %s", chat.Plural(it.n), chat.Human(it.size))
	var fill, mark, name, inf tcell.Style
	filled := true
	switch {
	case p.band && sel:
		fill, mark, name, inf = p.tint, p.tintAcc, p.tint, p.tint
	case p.band:
		fill, mark, name, inf = p.bandT, p.bandAcc, p.bandT, p.bandD
	case sel:
		fill, mark, name, inf = p.sel, p.sel, p.sel, p.sel
	default:
		filled = false
		mark, inf = p.acc.Bold(true), p.dim
	}
	if filled {
		u.add(y, 2, strings.Repeat(" ", f.cw-3), fill)
	}
	glyph := bar
	if folded {
		glyph = "▸"
	}
	u.add(y, 2, glyph, mark)
	u.add(y, 4, leaf, name.Bold(true))
	if folded {
		u.add(y, 4+rlen(leaf)+2, info, inf)
		return
	}
	n := f.hidden[it.proj]
	size := chat.Human(it.size)
	text := size
	if n > 0 {
		text = fmt.Sprintf("%d hidden · %s", n, size)
	}
	x := f.cw - 2 - rlen(text)
	u.add(y, x, text, inf)
	if n > 0 {
		u.add(y, x, fmt.Sprintf("%d hidden", n), name)
	}
	if !p.band {
		if r := x - 2 - (4 + rlen(leaf) + 2); r > 0 {
			u.add(y, 4+rlen(leaf)+2, strings.Repeat(rule, r), p.bandRule)
		}
	}
}

func (u *ui) drawDivider(y int) {
	lx, tx := 2, 16 // the title column, after mark, age and size
	if u.grouped {
		lx, tx = 4, 17
	}
	text := strings.Repeat(rule, tx-lx-1) + " hidden "
	if pad := u.f.cw - 2 - lx - rlen(text); pad > 0 {
		text += strings.Repeat(rule, pad)
	}
	u.add(y, lx, text, u.pal.divider)
}

func (u *ui) drawRow(y int, cursor bool, c *chat.Chat) {
	f, p := &u.f, &u.pal
	picked := u.picked[key(c)]
	mark, markStyle := "·", p.dim
	switch {
	case c.Live:
		mark, markStyle = "●", p.live
	case picked:
		mark, markStyle = "◉", p.acc.Bold(true)
	}
	indent := ""
	if u.grouped {
		indent = " "
	}
	var tail string
	daysLeft := u.left[key(c)]
	if u.trashing {
		tail = fmt.Sprintf("%dd left", daysLeft)
	}
	tailX := f.cw - 2 - rlen(tail)
	end := f.cw - 2 // titles stop two columns before the tail
	if tail != "" {
		end = tailX - 2
	}
	age, size := chat.Reltime(c.Mod, u.loadedAt), chat.Human(c.Size)

	if cursor && p.band {
		x := 2 + len(indent)
		u.add(y, 2, strings.Repeat(" ", f.cw-3), p.tint)
		style := p.tint
		switch {
		case c.Live:
			style = p.tintLive
		case picked:
			style = p.tintAcc
		}
		u.add(y, x, mark, style)
		x++
		seg := fmt.Sprintf(" %3s %6s  ", age, size)
		u.add(y, x, seg, p.tint)
		x += len(seg)
		u.add(y, x, chat.Fit(c.Label, max(0, min(labelMax, end-x))), p.tint.Bold(true))
		if tail != "" {
			u.add(y, tailX, tail, p.tint)
		}
		return
	}
	if cursor {
		pre := fmt.Sprintf("%s%s %3s %6s  ", indent, mark, age, size)
		title := chat.Fit(c.Label, max(0, min(labelMax, end-2-rlen(pre))))
		u.add(y, 2, fmt.Sprintf("%-*s", f.cw-3, pre+title), p.sel)
		if tail != "" {
			u.add(y, tailX, tail, p.sel)
		}
		return
	}
	x := 2 + len(indent)
	dim := p.dim
	if p.tray && c.Hidden {
		u.add(y, 2, strings.Repeat(" ", f.cw-3), p.trayD)
		dim = p.trayD
		switch {
		case c.Live:
			markStyle = p.trayLive
		case picked:
			markStyle = p.trayAcc
		default:
			markStyle = p.trayD
		}
	}
	u.add(y, x, mark, markStyle)
	x++
	seg := fmt.Sprintf(" %3s ", age)
	u.add(y, x, seg, dim)
	x += len(seg)
	sz := fmt.Sprintf("%6s", size)
	u.add(y, x, sz, dim)
	x += len(sz) + 2
	var titleStyle tcell.Style
	if c.Hidden {
		titleStyle = dim
	}
	title := chat.Fit(c.Label, max(0, min(labelMax, end-x)))
	match := strings.ToLower(u.flt)
	if i := strings.Index(strings.ToLower(title), match); u.flt != "" && i >= 0 {
		pre := rlen(strings.ToLower(title)[:i])
		runes := []rune(title)
		n := rlen(match)
		u.add(y, x, string(runes[:pre]), titleStyle)
		u.add(y, x+pre, string(runes[pre:pre+n]), titleStyle.Bold(true).Underline(true))
		u.add(y, x+pre+n, string(runes[pre+n:]), titleStyle)
	} else {
		u.add(y, x, title, titleStyle)
	}
	if tail != "" {
		tailStyle := dim
		if daysLeft <= 1 {
			tailStyle = p.yel
		}
		u.add(y, tailX, tail, tailStyle)
	}
}

func (u *ui) tabText(name string, bare bool) string {
	star := ""
	if name == u.pinned {
		star = "*"
	}
	if bare {
		return fmt.Sprintf(" %s%s ", star, name)
	}
	n := len(u.chats)
	if name != "all" {
		n = u.agentCounts[name]
	}
	return fmt.Sprintf(" %s%s %d ", star, name, n)
}

// stat is a piece of list state on the top row. A negative drop never goes, the others go lowest first when the row is short.
type stat struct {
	text  string
	style tcell.Style
	drop  int
}

const stays = -1

func statWidth(stats []stat) int {
	n := 0
	for _, s := range stats {
		n += rlen(s.text) + 3
	}
	return n
}

func dropOne(stats []stat) ([]stat, bool) {
	best := -1
	for i, s := range stats {
		if s.drop != stays && (best < 0 || s.drop < stats[best].drop) {
			best = i
		}
	}
	if best < 0 {
		return stats, false
	}
	return slices.Delete(stats, best, best+1), true
}

// drawLabels draws the top row and returns the column spans the frame's top edge opens under.
func (u *ui) drawLabels() [][2]int {
	f, p := &u.f, &u.pal
	var total, picked int64
	hidden := 0
	for _, c := range f.tabChats {
		total += c.Size
		if c.Hidden {
			hidden++
		}
	}
	for id := range u.picked {
		if c, ok := u.by[id]; ok {
			picked += c.Size
		}
	}
	filtered := u.flt != "" && !u.filtering
	var stats []stat
	if len(u.picked) > 0 {
		stats = append(stats, stat{fmt.Sprintf("%d picked · %s", len(u.picked), chat.Human(picked)), p.acc, stays})
	}
	switch {
	case filtered:
		shown := 0
		for _, c := range f.tabChats {
			if u.showKept || !c.Hidden {
				shown++
			}
		}
		stats = append(stats, stat{fmt.Sprintf("%d of %s", len(f.vis), chat.Plural(shown)), p.dim, 1})
	case hidden > 0 && !u.grouped && !u.trashing:
		stats = append(stats, stat{fmt.Sprintf("%d hidden", hidden), p.dim, 1})
	}
	stats = append(stats, stat{sortMark[u.sortMode], p.acc, stays})
	if u.trashing {
		stats = append(stats, stat{fmt.Sprintf("%s · %s", chat.Plural(len(f.tabChats)), chat.Human(total)), p.dim, 0})
	} else {
		if u.trashCount > 0 {
			stats = append(stats, stat{"trash " + chat.Human(u.trashSize), p.yel, stays})
		}
		stats = append(stats, stat{"total " + chat.Human(total), tcell.Style{}, 0})
	}

	var opens [][2]int // first and last column of each label the top edge opens under
	x := 2
	if u.trashing {
		u.add(0, x, " trash ", p.trashOn)
		x += 8
	}
	if len(u.tabs) > 1 {
		bare := false // tabs other than the current one lose their count
		tabsWidth := func() int {
			n := 0
			for _, a := range u.tabs {
				n += rlen(u.tabText(a, bare && a != u.tab)) + 1
			}
			return n
		}
		fltWidth := 0
		if filtered {
			fltWidth = rlen(u.flt) + 4
		}
		for x+tabsWidth()+fltWidth+1+statWidth(stats) > f.cw {
			var more bool
			if stats, more = dropOne(stats); more {
				continue
			}
			if !bare {
				bare = true
			} else if len(stats) > 0 {
				stats = stats[1:]
			} else {
				break
			}
		}
		for _, a := range u.tabs {
			text := u.tabText(a, bare && a != u.tab)
			if a == u.tab {
				u.add(0, x, text, p.tabOn)
			} else {
				u.add(0, x, text, p.tabOff)
				k := strings.Index(text[1:], " ") + 1
				u.add(0, x+k, text[k:], p.tabCnt)
			}
			if a == u.pinned {
				style := p.tabPin
				if a == u.tab {
					style = p.tabOn
				}
				u.add(0, x+1, "*", style)
			}
			f.tabSpans = append(f.tabSpans, span{x, x + rlen(text), a})
			if a == u.tab {
				opens = append(opens, [2]int{x, x + rlen(text) - 1})
			}
			x += rlen(text) + 1
		}
	} else if !u.trashing {
		for len(stats) > 1 && x+12+statWidth(stats) > f.cw {
			if next, more := dropOne(stats); more {
				stats = next
			} else {
				stats = stats[1:]
			}
		}
		name := " " + chat.ProjLabel(u.start.Title, max(8, f.cw-x-rlen(u.flt)-6-statWidth(stats))) + " "
		u.add(0, x, name, p.label)
		opens = append(opens, [2]int{x, x + rlen(name) - 1})
		x += rlen(name) + 1
	}
	if filtered {
		u.add(0, x, " /"+u.flt+" ", p.acc.Bold(true))
		x += rlen(u.flt) + 4
	}
	sx := f.cw - 1 - statWidth(stats) + 3
	if len(u.tabs) > 1 && !u.grouped && !u.trashing && sx-x > 12 { // one folder: say which
		name := " " + chat.ProjLabel(u.start.Title, sx-x-4) + " "
		u.add(0, x, name, p.dim)
		x += rlen(name)
	}
	if u.tip && u.pinned == "" && x+rlen(tabTip) < sx-2 {
		u.add(0, x, tabTip, p.dim)
	}
	for _, s := range stats {
		u.add(0, sx, s.text, s.style)
		if strings.HasPrefix(s.text, "trash ") || strings.HasPrefix(s.text, "total ") {
			u.add(0, sx, s.text[:5], p.dim) // the word steps back, the number reads
		}
		sx += rlen(s.text) + 3
	}
	return opens
}

// drawFrame draws the rounded frame. A div at or above 0 is the column of a divider.
func (u *ui) drawFrame(y0, y1, x0, x1, div int, gaps [][2]int) {
	n := x1 - x0 + 1
	top, bot := make([]string, n), make([]string, n)
	for i := range top {
		top[i], bot[i] = rule, rule
	}
	top[0], top[n-1], bot[0], bot[n-1] = "╭", "╮", "╰", "╯"
	if div >= 0 {
		top[div-x0], bot[div-x0] = "┬", "┴"
	}
	for _, g := range gaps {
		if g[1]-x0 >= n {
			continue
		}
		for i := g[0] - x0 + 1; i < g[1]-x0; i++ {
			top[i] = " "
		}
		top[g[0]-x0], top[g[1]-x0] = "╯", "╰"
	}
	line := u.pal.line
	u.put(y0, x0, strings.Join(top, ""), line)
	for y := y0 + 1; y < y1; y++ {
		u.put(y, x0, "│", line)
		u.put(y, x1, "│", line)
		if div >= 0 {
			u.put(y, div, "│", line)
		}
	}
	u.put(y1, x0, strings.Join(bot, ""), line)
}

func (u *ui) box(y0, x0, hh, ww int, title string, style tcell.Style) {
	for y := y0 + 1; y < y0+hh-1; y++ {
		u.add(y, x0, strings.Repeat(" ", ww), tcell.Style{})
	}
	u.drawFrame(y0, y0+hh-1, x0, x0+ww-1, -1, nil)
	if title != "" {
		u.add(y0, x0+2, " "+title+" ", style)
	}
}

type hint struct{ key, act string }

func hintSpan(pairs []hint) int {
	n := -2
	for _, h := range pairs {
		n += rlen(h.key) + rlen(h.act) + 3
	}
	return n
}

// hints draws key and action pairs from column x and returns the column after them.
func (u *ui) hints(y, x int, pairs []hint) int {
	u.add(y, x-1, strings.Repeat(" ", hintSpan(pairs)+2), tcell.Style{})
	for _, h := range pairs {
		u.add(y, x, h.key, u.pal.bold)
		x += rlen(h.key) + 1
		u.add(y, x, h.act, u.pal.dim)
		x += rlen(h.act) + 2
	}
	return x
}

// drawBottom draws the filter being typed, else a message, else the keys. Keys shed a few at a time on a narrow terminal.
func (u *ui) drawBottom() {
	f, p := &u.f, &u.pal
	y := f.h - 1
	switch {
	case u.filtering:
		matches := fmt.Sprintf("%d match", len(f.vis))
		if len(f.vis) != 1 {
			matches += "es"
		}
		matches += " · enter apply · esc clear"
		u.add(y, 2, " /"+u.flt+"  ", p.bold)
		u.add(y, 3, "/", p.acc.Bold(true))
		u.add(y, 4+rlen(u.flt), "▏", p.dim)
		u.add(y, f.w-4-rlen(matches), " "+matches+" ", p.dim)
		return
	case u.msg.text != "":
		style := p.warn
		if u.msg.ok {
			style = p.ok
		}
		u.add(y, 2, " "+chat.Fit(u.msg.text, f.w-8)+" ", style)
		return
	}
	quit := "quit"
	if u.trashing {
		quit = "back"
	}
	ends := []hint{{"?", "keys"}, {"q", quit}}
	var groups [][]hint
	if u.trashing {
		var pick, peek []hint
		if len(f.vis) > 0 {
			pick = []hint{{"space", "pick"}, {"r", "restore"}, {"x", "purge"}, {"E", "empty"}}
			peek = []hint{{"p", "peek"}}
		}
		groups = [][]hint{pick, peek, {{"/", "filter"}}}
	} else {
		act := []hint{{"space", "pick"}, {"d", "delete"}}
		if len(u.undo) > 0 {
			act = append(act, hint{"u", "undo"})
		}
		act = append(act, hint{"h", "hide"})
		groups = [][]hint{{{"enter", "open"}, {"p", "peek"}}, act, {{"/", "filter"}, {"s", "sort"}, {"t", "trash"}}}
	}
	room := f.w - 9 - hintSpan(ends)
	for _, shed := range []string{"", "s", "h", "p", "space", "u", "/", "t", "E", "x", "r", "d"} {
		var left [][]hint
		for _, g := range groups {
			g = slices.DeleteFunc(slices.Clone(g), func(h hint) bool { return h.key == shed })
			if len(g) > 0 {
				left = append(left, g)
			}
		}
		groups = left
		span := -3
		for _, g := range groups {
			span += hintSpan(g) + 3
		}
		if span <= room {
			break
		}
	}
	x := 3
	for i, g := range groups {
		if i > 0 {
			u.add(y, x-2, " · ", p.line)
			x++
		}
		x = u.hints(y, x, g)
	}
	u.hints(y, f.w-3-hintSpan(ends), ends)
}
