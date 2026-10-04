package tui

import (
	"os"
	"slices"
	"time"

	"github.com/gdamore/tcell/v3"
	"golang.org/x/term"

	"github.com/vyagh/reap/internal/chat"
	"github.com/vyagh/reap/internal/cli"
)

const (
	rule, bar = "─", "▌" // frame and mark glyphs (reap:62)
	labelMax  = 72       // longest title drawn; a wide terminal must not turn titles into paragraphs
	projMax   = 46       // longest project name in a group header
	listW     = 78       // the list keeps this width; the rest of a wide terminal is the side panel
	paneMin   = 42       // the side panel needs at least this much room
	tabTip    = "P opens reap here every time"
)

// sorts is the order s cycles through, and sortMark what the top row says.
var (
	sorts    = []string{"newest", "oldest", "biggest", "smallest"}
	sortMark = map[string]string{"newest": "date ↓", "oldest": "date ↑", "biggest": "size ↓", "smallest": "size ↑"}
)

// A feature file fills its slot from an init function in that file.
//
//	func init() {
//		keys["d"] = (*ui).askDelete
//	}
//
// A handler for a key that also does something in the trash view checks
// u.trashing itself. Enter and the keys in inTrashIgnored never reach one
// there.
var (
	// keys is what each key does when the user is not typing a filter.
	keys = map[string]func(*ui){
		"j": (*ui).down, "Down": (*ui).down,
		"k": (*ui).up, "Up": (*ui).up,
		"PgDn": (*ui).pageDown, "PgUp": (*ui).pageUp,
		"g": (*ui).first, "G": (*ui).last,
		"s":   (*ui).nextSort,
		"Tab": (*ui).nextTab, "BTab": (*ui).prevTab,
		"P": (*ui).pin,
		"]": (*ui).nextGroup, "[": (*ui).prevGroup,
		"z": (*ui).fold, "Left": (*ui).fold, "Right": (*ui).fold,
		"Z": (*ui).foldAll,
		"a": (*ui).pickAll, " ": (*ui).pick,
		"h": (*ui).hide, ".": (*ui).toggleHidden,
		"?": (*ui).help,
	}
	// filterKey takes every key while u.filtering is set.
	filterKey func(u *ui, k string)
	// mouseEvent takes every mouse event.
	mouseEvent func(u *ui, ev *tcell.EventMouse)
	// detail draws the side panel for the item under the cursor, when the
	// terminal is wide enough, and returns the title that goes on the
	// panel's top edge ("" for none).
	detail func(u *ui, it item) (title string, style tcell.Style)
	// afterRun runs once the screen is closed, to start the chat that
	// u.launch names.
	afterRun func(u *ui) error
)

// inTrashIgnored are the keys that do nothing in the trash view (reap:1517).
var inTrashIgnored = []string{"d", "u", "h", ".", "s"}

// message is the line on the bottom edge: green when ok, yellow when not.
type message struct {
	text string
	ok   bool
}

// launch is the chat Enter opened: the command and the folder to run it in.
type launch struct {
	argv []string
	dir  string
}

// ui is the whole state of the screen: what Python's run_ui keeps in its
// locals (reap:926-1015), one field each. The trash view is a second ui, as
// Python runs main again for it.
type ui struct {
	s     tcell.Screen
	pal   palette
	start cli.Start

	load     func(full bool) ([]chat.Chat, error) // loads the chats the view lists
	grouped  bool                                 // grouped by project, else one flat list
	trashing bool                                 // this view is the trash browser
	home     string                               // project folder the cursor lands on in the first frame, then ""

	chats    []chat.Chat           // everything load returned
	by       map[string]*chat.Chat // id -> the chat in chats
	loadedAt time.Time             // when chats was loaded; ages are counted from here
	keep     map[string]float64    // hidden chat id -> epoch hidden, refreshed on reload
	showKept bool                  // hidden chats are shown (starts false on every launch)

	picked    map[string]bool // ids picked with space
	cur, top  int             // cursor item and first drawn item
	flt       string          // filter text
	filtering bool            // the user is typing the filter
	msg       message         // shown on the bottom edge until the next key
	sortMode  string          // one of sorts
	undo      [][]chat.Entry  // this session's deletes, one list of trash entries each
	folded    map[string]bool // collapsed project groups
	focus     string          // id the cursor lands on after a view change
	changed   bool            // a trash pass altered the trash
	restored  []chat.Chat     // trash chats put back, in order
	tip       bool            // show the pin tip until the next key that is not a tab switch

	trashCount int            // trash entries, for the "trash" label
	trashSize  int64          // bytes they hold
	left       map[string]int // trash view: days left per chat id, for the "Nd left" column

	previews map[string][]chat.Turn // id -> side panel turns, read once
	counts   map[string]int         // id -> prompt count, once the cursor rested there

	tabs        []string       // "all" first, then the installed agents; none with one agent
	agentCounts map[string]int // chats per agent
	tab         string         // current tab, "" when there are no tabs
	pinned      string         // the tab reap opens on, "" for none

	launch *launch // set by Enter; afterRun starts it
	quit   bool    // leave the loop

	f frame // what the last plan worked out from the state above
}

// Run shows the full-screen view until the user quits.
func Run(start cli.Start) error {
	var light, known bool
	if isTerminal(start.Stdin) && isTerminal(start.Stdout) {
		light, known = lightBackground()
	}
	s, err := openScreen()
	if err != nil {
		return err
	}
	defer s.Fini()
	u, err := newUI(s, newPalette(s.Colors(), light, known), start, start.Load, start.Grouped, false, start.Home)
	if err != nil {
		return err
	}
	u.run()
	s.Fini()
	if afterRun != nil {
		return afterRun(u)
	}
	return nil
}

func isTerminal(x any) bool {
	f, ok := x.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// newUI is the start of Python's main (reap:928-1015): the state with its
// chats loaded and the tab chosen.
func newUI(s tcell.Screen, pal palette, start cli.Start, load func(bool) ([]chat.Chat, error), grouped, trashing bool, home string) (*ui, error) {
	u := &ui{
		s: s, pal: pal, start: start,
		load: load, grouped: grouped, trashing: trashing, home: home,
		picked:   map[string]bool{},
		sortMode: sorts[0],
		folded:   map[string]bool{},
		left:     map[string]int{},
		previews: map[string][]chat.Turn{},
		counts:   map[string]int{},
	}
	if err := u.reload(); err != nil {
		return nil, err
	}
	u.pinned = chat.LoadState(start.Homes).Pin
	if len(u.tabs) > 1 {
		u.tab = "all"
		if slices.Contains(u.tabs, u.pinned) {
			u.tab = u.pinned
		}
	}
	return u, nil
}

// reload loads the chats again, after anything that changed the trash or a
// hide (reap:1022-1029).
func (u *ui) reload() error {
	chats, err := u.load(false)
	if err != nil {
		return err
	}
	u.keep = chat.LoadState(u.start.Homes).Keep
	u.chats = chats
	u.loadedAt = time.Now()
	u.by = make(map[string]*chat.Chat, len(chats))
	u.agentCounts = map[string]int{}
	for i := range u.chats {
		c := &u.chats[i]
		u.by[c.ID] = c
		u.agentCounts[string(c.Source)]++
		if n, ok := u.counts[c.ID]; ok {
			c.Msgs = n
		}
	}
	u.trashCount, u.trashSize = chat.TrashStat(u.start.Homes)
	u.tabs = u.tabList()
	if !slices.Contains(u.tabs, u.tab) {
		u.tab = ""
		if len(u.tabs) > 1 {
			u.tab = "all"
		}
	}
	return nil
}

// tabList is the installed agents in tab order, with "all" in front once
// there is more than one. An agent with a data folder gets a tab even at
// 0 chats. One agent gets no tabs at all (reap:1005-1011).
func (u *ui) tabList() []string {
	var tabs []string
	for _, a := range u.start.Agents {
		name := string(a.Name())
		if a.Installed() || u.agentCounts[name] > 0 {
			tabs = append(tabs, name)
		}
	}
	if len(tabs) > 1 {
		return append([]string{"all"}, tabs...)
	}
	return tabs
}

// run draws and reads keys until the user leaves.
func (u *ui) run() {
	for !u.quit {
		u.plan()
		u.draw()
		u.s.Show()
		u.step()
	}
}

// step waits for one event and acts on it. While the cursor rests on a chat
// whose prompt count is not known yet it waits only 250 ms, then reads the
// count and draws again (reap:1490-1497).
func (u *ui) step() {
	var rest *chat.Chat
	if c := u.f.at(u.cur); c != nil && c.Msgs < 0 {
		rest = c
	}
	var wait time.Duration
	if rest != nil {
		wait = 250 * time.Millisecond
	}
	ev := u.wait(wait)
	if ev == nil {
		if rest != nil {
			u.count(rest)
		}
		return
	}
	u.msg = message{}
	u.tip = false
	switch ev := ev.(type) {
	case *tcell.EventKey:
		u.press(keyName(ev))
	case *tcell.EventMouse:
		if mouseEvent != nil {
			mouseEvent(u, ev)
		}
	}
}

// count reads a whole chat for its prompt count, which the list only
// estimates. A chat that cannot be read counts as empty.
func (u *ui) count(c *chat.Chat) {
	n, err := u.start.Agents.For(c.Source).Count(*c)
	if err != nil {
		n = 0
	}
	c.Msgs = n
	u.counts[c.ID] = n
}

// press is the one key dispatch.
func (u *ui) press(k string) {
	if u.filtering {
		filterKey(u, k)
		return
	}
	if k == "q" || k == "Esc" || u.trashing && k == "t" {
		u.quit = true
		return
	}
	if u.trashing && slices.Contains(inTrashIgnored, k) {
		return
	}
	if u.trashing && k == "Enter" {
		u.msg = message{text: "restore it first (r)"}
		return
	}
	if do := keys[k]; do != nil {
		do(u)
	}
}
