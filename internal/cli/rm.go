package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vyagh/reap/internal/chat"
)

type removal struct {
	chat   chat.Chat
	orphan string // path of the orphan folder, "" for a chat
	label  string // what the trash record calls it
}

func (r removal) id() string {
	if r.orphan != "" {
		return filepath.Base(r.orphan)
	}
	return r.chat.ID
}

func cmdRm(env Env, agents chat.Agents, args []string) int {
	apply := popFlag(&args, "--apply")
	hard := popFlag(&args, "--hard")
	orphans := popFlag(&args, "--orphans")
	every := popFlag(&args, "--all")
	proj := popOpt(&args, "-p")
	sweep := orphans && every
	dir, err := chat.ResolveDir(env.Homes, proj, !sweep, env.CwdOf)
	if err != nil {
		return die(env, err)
	}
	var chats []chat.Chat
	if !orphans {
		if chats, err = chat.Load(agents, nil, chat.ListOpts{Dir: dir}); err != nil {
			return die(env, err)
		}
	}
	if !sweep && !isDir(dir) && (orphans || len(chats) == 0) {
		return die(env, "no chats found for: "+dir)
	}

	var plan []removal
	if orphans {
		dirs := []string{dir}
		if every {
			dirs = chat.ProjectDirs(env.Homes)
		}
		for _, d := range dirs {
			for _, p := range orphanDirs(d) {
				plan = append(plan, removal{orphan: p, label: "orphan sidecar dir"})
			}
		}
	} else {
		ids := operands(args)
		if len(ids) == 0 {
			return die(env, "usage: reap rm <id-prefix>... [--apply] [--orphans] [-p <project>]")
		}
		for _, tok := range ids {
			hits := matches(chats, tok)
			if len(hits) == 0 {
				fmt.Fprintf(env.Stdout, "!! no chat matches '%s'\n", tok)
				continue
			}
			if len(hits) > 1 {
				fmt.Fprintf(env.Stdout, "!! '%s' is ambiguous: %s\n", tok, pyList(chatIDs(hits)))
				continue
			}
			c := hits[0]
			if c.Live {
				fmt.Fprintf(env.Stdout, "!! SKIP %.8s: session is live\n", c.ID)
				continue
			}
			label := c.Label
			if c.Source == chat.Claude && !c.Titled && label == "(empty)" {
				label = ""
				c.Label = ""
			}
			plan = append(plan, removal{chat: c, label: label})
		}
	}
	if len(plan) == 0 {
		fmt.Fprintln(env.Stdout, "nothing to do")
		return 0
	}

	verb := "trash"
	if hard {
		verb = "purge"
	}
	for _, r := range plan {
		prefix := "would " + verb + " "
		if apply {
			prefix = strings.ToUpper(verb) + " "
		}
		fmt.Fprintln(env.Stdout, prefix+describe(env, agents, r))
	}
	if !apply {
		tail := " (--hard to skip trash)"
		if hard {
			tail = ""
		}
		fmt.Fprintf(env.Stdout, "dry-run, add --apply to %s%s\n", verb, tail)
		return 0
	}

	st := chat.LoadState(env.Homes)
	failed := 0
	for _, r := range plan {
		if err := remove(env, agents, r, hard); err != nil {
			fmt.Fprintf(env.Stdout, "!! %.8s: %v\n", r.id(), err)
			failed++
			continue
		}
		if r.orphan == "" {
			delete(st.Keep, r.chat.ID)
		}
	}
	if err := chat.SaveState(env.Homes, st, agents); err != nil {
		return die(env, err)
	}
	if failed < len(plan) {
		done := fmt.Sprintf("done, in trash for %dd (reap restore to undo)", chat.TrashDays)
		if hard {
			done = "done."
		}
		if failed > 0 {
			done = fmt.Sprintf("%d failed, the rest ", failed) + done
		}
		fmt.Fprintln(env.Stdout, done)
	}
	if failed > 0 {
		return 1
	}
	return 0
}

// matches are the chats whose id starts with tok, Claude chats first. A Claude chat
// matches on its file name, so tok may end in .jsonl.
func matches(chats []chat.Chat, tok string) []chat.Chat {
	var hits []chat.Chat
	for _, c := range chats {
		if c.Source == chat.Claude && strings.HasPrefix(c.ID+".jsonl", tok) {
			hits = append(hits, c)
		}
	}
	for _, c := range chats {
		if c.Source != chat.Claude && strings.HasPrefix(c.ID, tok) {
			hits = append(hits, c)
		}
	}
	return hits
}

func chatIDs(chats []chat.Chat) []string {
	ids := make([]string, len(chats))
	for i, c := range chats {
		ids[i] = c.ID
	}
	return ids
}

func describe(env Env, agents chat.Agents, r removal) string {
	if r.orphan != "" {
		return fmt.Sprintf("orphan dir %s/%.8s", chat.Pretty(env.Homes, filepath.Dir(r.orphan)), filepath.Base(r.orphan))
	}
	kinds := []string{string(r.chat.Source)}
	if a := agents.For(r.chat.Source); a != nil {
		if k := a.Kinds(r.chat); k != nil {
			kinds = k
		}
	}
	return fmt.Sprintf("%.8s (%s)", r.chat.ID, strings.Join(kinds, ", "))
}

func remove(env Env, agents chat.Agents, r removal, hard bool) error {
	if r.orphan != "" {
		if hard {
			removeFolder(r.orphan)
			return nil
		}
		_, err := chat.Move(env.Homes, filepath.Base(r.orphan), []string{r.orphan},
			chat.Record{UUID: filepath.Base(r.orphan), Dir: filepath.Dir(r.orphan), Label: r.label})
		return err
	}
	a := agents.For(r.chat.Source)
	if a == nil {
		return fmt.Errorf("no %s agent", r.chat.Source)
	}
	if hard {
		return a.Delete(r.chat)
	}
	_, err := a.Trash(r.chat)
	return err
}

// removeFolder ignores failures and leaves a link alone, as 0.5.0's rmtree does.
func removeFolder(path string) {
	if fi, err := os.Lstat(path); err == nil && fi.Mode()&os.ModeSymlink == 0 {
		os.RemoveAll(path)
	}
}

// orphanDirs are the folders in a project with no transcript of the same name.
// memory is Claude's own, never an orphan.
func orphanDirs(dir string) []string {
	entries, _ := os.ReadDir(dir)
	transcripts := map[string]bool{}
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".jsonl"); ok && !strings.HasPrefix(e.Name(), ".") {
			transcripts[name] = true
		}
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || name == "memory" || transcripts[name] || !isDir(filepath.Join(dir, name)) {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	return out
}
