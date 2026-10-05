package cli

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/vyagh/reap/internal/chat"
)

// selftest reads the chats in full and checks that the quick read at launch
// agrees. A failed check is an error on stderr and exit 1, where 0.5.0 dies with
// an AssertionError.
func selftest(env Env, agents chat.Agents, load func(bool) ([]chat.Chat, error), where string) int {
	chats, err := load(true)
	if err != nil {
		return die(env, err)
	}
	out := env.Stdout
	fmt.Fprintf(out, "loaded %d chats from %s\n", len(chats), where)
	var live, projs, srcs []string
	for _, c := range chats {
		if c.Live {
			live = append(live, c.Short())
		}
		projs = append(projs, c.Proj)
		srcs = append(srcs, string(c.Source))
	}
	if len(live) > 0 {
		fmt.Fprintln(out, "live    :", pyList(live))
	} else {
		fmt.Fprintln(out, "live    : none")
	}
	fmt.Fprintln(out, "projects:", pyCounts(projs))
	fmt.Fprintln(out, "src     :", pyCounts(srcs))
	if len(chats) > 0 {
		first := "none"
		var turns []chat.Turn
		if a := agents.For(chats[0].Source); a != nil {
			turns, _ = a.Peek(chats[0], 250, false)
		}
		if len(turns) > 0 {
			first = "(" + pyStr(turns[0].Who) + ", " + pyStr(turns[0].Text) + ")"
		}
		fmt.Fprintf(out, "peek(%s): %d turns, first = %s\n", chats[0].Short(), len(turns), first)
	}
	printStatic(env, chats[:min(len(chats), 6)], true)

	quick, err := load(false)
	if err != nil {
		return die(env, err)
	}
	bad, claudeChats := endsOnlyDiffers(chats, quick)
	if len(bad) > 0 {
		return die(env, "selftest failed: ends-only read differs from a full read: "+pyList(bad))
	}
	fmt.Fprintf(out, "meta    : ends-only read matches a full read on %d chats\n", claudeChats)
	if err := selftestTrash(env); err != nil {
		return die(env, "selftest failed: "+err.Error())
	}
	return 0
}

// endsOnlyDiffers compares Claude chats read in full with the same chats read from
// their ends only. It returns "<short> <field>" for each difference in title,
// prompt or cwd, and the number of chats compared.
func endsOnlyDiffers(full, quick []chat.Chat) (bad []string, n int) {
	byID := map[string]chat.Chat{}
	for _, c := range quick {
		byID[c.ID] = c
	}
	for _, c := range full {
		if c.Source != chat.Claude {
			continue
		}
		n++
		q := byID[c.ID]
		if q.Titled != c.Titled || q.Titled && q.Label != c.Label {
			bad = append(bad, c.Short()+" title")
		}
		if !c.Titled && q.Label != c.Label {
			bad = append(bad, c.Short()+" prompt")
		}
		if q.Cwd != c.Cwd {
			bad = append(bad, c.Short()+" cwd")
		}
	}
	return bad, n
}

// selftestTrash makes four trash entries in a scratch folder, one per record
// layout, and checks each lists under the right project. An entry written before
// records carried a project and a cwd must still list.
func selftestTrash(env Env) error {
	scratch, err := os.MkdirTemp("", "reap")
	if err != nil {
		return err
	}
	defer os.RemoveAll(scratch)
	h := env.Homes
	h.Trash, h.CodexTrash = scratch, ""
	proj := func(cwd string) string { return filepath.Join(h.Claude, "projects", chat.DirnameFor(cwd)) }
	app := filepath.Join(h.Home, "app")
	deepCwd := filepath.Join(h.Home, "work", "deep-app")
	old := chat.Record{UUID: "0a", Dir: proj(app), Label: "old", At: 1, Moved: [][]string{{"/gone/0a.jsonl", "0a.jsonl"}}}
	newer := old
	newer.UUID, newer.Label, newer.Proj, newer.Cwd = "0b", "new", "app", &app
	home := old
	home.UUID, home.Dir = "0c", proj(h.Home)
	deep := old
	deep.UUID, deep.Label, deep.Dir = "0d", "deep", proj(deepCwd)
	deep.Moved = [][]string{{"/gone/0d.jsonl", "0d.jsonl"}}
	for _, r := range []chat.Record{old, newer, home, deep} {
		dir := filepath.Join(scratch, r.UUID)
		if err := os.Mkdir(dir, 0o755); err != nil {
			return err
		}
		data, err := json.Marshal(r)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "reap-meta.json"), data, 0o644); err != nil {
			return err
		}
	}
	transcript, err := json.Marshal(map[string]string{"type": "user", "cwd": deepCwd})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(scratch, "0d", "0d.jsonl"), append(transcript, '\n'), 0o644); err != nil {
		return err
	}

	got := map[string]string{}
	var shaped []chat.Chat
	for _, e := range chat.ListTrash(h, env.CwdOf) {
		got[e.Record.UUID] = e.Proj
		shaped = append(shaped, e.Chat())
	}
	want := map[string]string{"0a": "app", "0b": "app", "0c": "~", "0d": "work-deep-app"}
	if !maps.Equal(got, want) {
		return fmt.Errorf("trash lists %v, want %v", got, want)
	}
	labels, projs := map[string]bool{}, map[string]bool{}
	for _, c := range shaped {
		labels[c.Label], projs[c.Proj] = true, true
		if c.Label == "deep" {
			if leaf, _ := chat.SplitPath(h, c.Cwd, c.Proj); leaf != "deep-app" {
				return fmt.Errorf("the deep entry has the folder name %q, want deep-app", leaf)
			}
		}
		if c.Label == "old" && filepath.Base(c.Path) != "0a.jsonl" {
			return fmt.Errorf("an old entry has the transcript %q, want 0a.jsonl", c.Path)
		}
	}
	if !maps.Equal(labels, map[string]bool{"old": true, "new": true, "deep": true}) {
		return fmt.Errorf("the trash shows the labels %v", slices.Sorted(maps.Keys(labels)))
	}
	if !maps.Equal(projs, map[string]bool{"app": true, "~": true, "work-deep-app": true}) {
		return fmt.Errorf("the trash shows the projects %v", slices.Sorted(maps.Keys(projs)))
	}
	fmt.Fprintln(env.Stdout, "trash   : old and new manifests list under", got["0a"], "and", got["0c"])
	return nil
}
