package cli

import (
	"fmt"

	"github.com/vyagh/reap/internal/chat"
)

// printHelp colours only on a terminal and not when NO_COLOR is set.
func printHelp(env Env) {
	on := env.Terminal && env.Getenv("NO_COLOR") == ""
	paint := func(s, code string) string {
		if !on {
			return s
		}
		return "\033[" + code + "m" + s + "\033[0m"
	}
	bold := func(s string) string { return paint(s, "1") }
	dim := func(s string) string { return paint(s, "2") }
	cmd := func(s string) string { return paint(s, "36") }
	head := func(s string) string { return paint(s, "1;4") }
	out := env.Stdout

	fmt.Fprintf(out, "%s %s\n", bold("reap"), dim(": Claude Code and Codex chat manager"))
	fmt.Fprintln(out, dim("view, peek, open and delete your local chats"))
	fmt.Fprintln(out)
	fmt.Fprintln(out, dim("a claude chat = transcript + sidecar dir + job record + file-history, session-env, tasks."))
	fmt.Fprintln(out, dim("reap trashes all of it, so it's gone from /resume and the agents view."))
	fmt.Fprintln(out, dim("a codex chat = its rollout file, archived through the codex CLI."))
	fmt.Fprintln(out)
	fmt.Fprintln(out, head("usage"))
	usage := [][2]string{
		{"reap", "picker across all projects, cursor on the current folder's"},
		{"reap .", "picker for the current folder's project only"},
		{"reap <name|path>", "picker for a project (name = folder substring)"},
		{"reap ls [.|<name>]", "plain list (no UI, pipe-friendly)"},
		{"reap rm <id>... [opts]", "delete by uuid prefix (for scripts)"},
		{"reap trash [--empty]", "list recoverable deletes (or purge all)"},
		{"reap restore <id|name>", "restore a chat from the trash"},
		{"reap --help", "this help"},
		{"reap --version", "print version"},
		{"reap --selftest", "sanity check against your own ~/.claude, prints what it saw"},
	}
	w := 0
	for _, u := range usage {
		w = max(w, len(u[0]))
	}
	for _, u := range usage {
		fmt.Fprintf(out, "  %s  %s\n", cmd(fmt.Sprintf("%-*s", w, u[0])), u[1])
	}
	fmt.Fprintln(out)
	fmt.Fprintf(out, "  %s\n", dim("rm options"))
	rmOpts := [][2]string{
		{"--apply", "actually delete (default is a dry-run)"},
		{"--hard", "skip the trash: delete irreversibly"},
		{"--orphans", "remove leftover sidecar dirs (no transcript)"},
		{"--all", "with --orphans: sweep every project"},
		{"-p <name>", "target another project (default: current dir)"},
	}
	for _, o := range rmOpts {
		fmt.Fprintf(out, "    %s  %s\n", cmd(fmt.Sprintf("%-11s", o[0])), o[1])
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, dim("  across all projects, /tmp scratchpad sessions are hidden. pass --tmp to include them."))
	fmt.Fprintln(out, dim("  reap ls --hidden lists the hidden chats. --no-codex-cli (any command) moves codex files into reap's trash instead of calling codex."))
	fmt.Fprintln(out)
	fmt.Fprintln(out, head("picker keys")+dim("   (press ? inside the picker for these)"))
	keys := [][2]string{
		{"move", "up/down or j/k   PgUp/PgDn   g/G   [ ] prev/next workspace"},
		{"select", "space pick   a toggle select-all (skips live)   h hide"},
		{"act", "enter open   p peek   d delete (to trash)   u undo   t trash browser"},
		{"view", "s sort (newest/oldest/biggest/smallest)   tab/shift-tab agent   P pin tab   . hidden   / filter   q quit"},
	}
	for _, k := range keys {
		fmt.Fprintf(out, "  %s  %s\n", cmd(fmt.Sprintf("%-7s", k[0])), k[1])
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, dim("  ● running  ◉ picked"))
	fmt.Fprintln(out)
	fmt.Fprintln(out, dim(fmt.Sprintf("the picker confirms before deleting. deletes go to a trash for %d days unless you pass --hard. live sessions are refused.", chat.TrashDays)))
}
