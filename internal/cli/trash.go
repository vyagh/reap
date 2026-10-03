package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/vyagh/reap/internal/chat"
)

func cmdTrash(env Env, agents chat.Agents, args []string) int {
	empty := popFlag(&args, "--empty")
	entries := chat.ListTrash(env.Homes, env.CwdOf)
	if empty {
		for _, e := range entries {
			purge(agents, e)
		}
		fmt.Fprintf(env.Stdout, "emptied trash, purged %d chats\n", len(entries))
		return 0
	}
	if len(entries) == 0 {
		fmt.Fprintln(env.Stdout, "trash is empty")
		return 0
	}
	now := env.Now()
	var total int64
	for _, e := range entries {
		c := e.Chat()
		fmt.Fprintf(env.Stdout, "  %4s %6s  %.8s  %dd left  %.48s\n",
			chat.Reltime(c.Mod, now), chat.Human(e.Size), e.Record.UUID, chat.DaysLeft(c.Mod, now), e.Record.Label)
		total += e.Size
	}
	fmt.Fprintf(env.Stdout, "\n%s · %s still on disk (frees after %dd or --empty)\n",
		chat.Plural(len(entries)), chat.Human(total), chat.TrashDays)
	fmt.Fprintln(env.Stdout, "restore: reap restore <id|name>")
	return 0
}

// purge removes a trash entry for good. It never fails: the entry is dropped
// even when the agent could not clean up after it (reap:681-688).
func purge(agents chat.Agents, e chat.Entry) {
	if a := agents.For(chat.Source(e.Record.Src)); a != nil {
		a.Purge(e)
		return
	}
	os.RemoveAll(e.Dir)
}

func cmdRestore(env Env, agents chat.Agents, args []string) int {
	ids := operands(args)
	entries := chat.ListTrash(env.Homes, env.CwdOf)
	if len(ids) == 0 {
		if len(entries) == 0 {
			fmt.Fprintln(env.Stdout, "trash is empty")
			return 0
		}
		fmt.Fprintln(env.Stdout, "usage: reap restore <id-prefix | name>...   in trash:")
		for _, e := range entries {
			fmt.Fprintf(env.Stdout, "  %.8s  %.50s\n", e.Record.UUID, e.Record.Label)
		}
		return 0
	}
	for _, tok := range ids {
		hits := trashed(entries, tok)
		if len(hits) == 0 {
			fmt.Fprintf(env.Stdout, "!! no trashed chat matches '%s'\n", tok)
			continue
		}
		if len(hits) > 1 {
			shorts := make([]string, len(hits))
			for i, e := range hits {
				shorts[i] = fmt.Sprintf("%.8s", e.Record.UUID)
			}
			fmt.Fprintf(env.Stdout, "!! '%s' is ambiguous: %s\n", tok, pyList(shorts))
			continue
		}
		label := "(failed)"
		if fresh, err := chat.ReadEntry(hits[0].Dir); err == nil && restore(agents, fresh) {
			label = fresh.Record.Label
		}
		fmt.Fprintf(env.Stdout, "restored %.8s  %s\n", hits[0].Record.UUID, label)
	}
	return 0
}

// trashed are the entries whose id starts with tok, else those whose label
// holds tok in any letter case (reap:1902-1904).
func trashed(entries []chat.Entry, tok string) []chat.Entry {
	var hits []chat.Entry
	for _, e := range entries {
		if strings.HasPrefix(e.Record.UUID, tok) {
			hits = append(hits, e)
		}
	}
	if len(hits) > 0 {
		return hits
	}
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Record.Label), strings.ToLower(tok)) {
			hits = append(hits, e)
		}
	}
	return hits
}

// restore puts an entry back and reports whether it worked. The caller reads
// the entry again just before, so one that an earlier restore in the same
// command removed counts as failed (reap:697-698).
func restore(agents chat.Agents, e chat.Entry) bool {
	a := agents.For(chat.Source(e.Record.Src))
	return a != nil && a.Restore(e) == nil
}
