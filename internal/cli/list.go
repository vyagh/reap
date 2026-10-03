package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vyagh/reap/internal/chat"
)

// printStatic is the plain list: one line per chat, with a "# project" line
// before each group when grouped, then the totals (reap:839-847).
func printStatic(env Env, chats []chat.Chat, grouped bool) {
	now := env.Now()
	var total int64
	last, started := "", false
	for _, c := range chats {
		if grouped && (!started || c.Proj != last) {
			last, started = c.Proj, true
			fmt.Fprintf(env.Stdout, "\n# %s\n", c.Proj)
		}
		src := ""
		if c.Source == chat.Codex {
			src = "  codex"
		}
		live := ""
		if c.Live {
			live = "  * live"
		}
		fmt.Fprintf(env.Stdout, "  %4s %6s %5d  %-44.44s %s%s%s\n",
			chat.Reltime(c.Mod, now), chat.Human(c.Size), c.Msgs, c.Label, c.Short(), src, live)
		total += c.Size
	}
	fmt.Fprintf(env.Stdout, "\n%d chats - %s total\n", len(chats), chat.Human(total))
}

// projectArg is the folder argument as resolve_dir takes it: "." means the
// current folder, which it spells "".
func projectArg(proj string) string {
	if proj == "." {
		return ""
	}
	return proj
}

func cmdLs(env Env, agents chat.Agents, args []string) int {
	tmp := popFlag(&args, "--tmp")
	subagents := popFlag(&args, "--subagents")
	hidden := popFlag(&args, "--hidden")
	if popFlag(&args, "--all") {
		return die(env, allGone)
	}
	keep := chat.LoadState(env.Homes).Keep
	opts := chat.ListOpts{Tmp: tmp, Subagents: subagents, Full: true}
	grouped := true
	if names := operands(args); len(names) > 0 {
		dir, err := chat.ResolveDir(env.Homes, projectArg(names[0]), true, env.CwdOf)
		if err != nil {
			return die(env, err)
		}
		if !isDir(dir) {
			return die(env, "no chats found for: "+dir)
		}
		opts = chat.ListOpts{Dir: dir, Subagents: subagents, Full: true}
		grouped = false
	}
	chats, err := chat.Load(agents, keep, opts)
	if err != nil {
		return die(env, err)
	}
	shown := chats[:0]
	for _, c := range chats {
		if c.Hidden == hidden {
			shown = append(shown, c)
		}
	}
	printStatic(env, shown, grouped)
	return 0
}

// cmdDefault is plain reap: the full-screen view on a terminal, the plain
// list when piped, and the self-check with --selftest.
func cmdDefault(env Env, agents chat.Agents, args []string) int {
	selftestRun := popFlag(&args, "--selftest")
	tmp := popFlag(&args, "--tmp")
	subagents := popFlag(&args, "--subagents")
	if popFlag(&args, "--all") {
		return die(env, allGone)
	}
	opts := chat.ListOpts{Tmp: tmp, Subagents: subagents}
	where, title, grouped := "all projects", "all projects", true
	if names := operands(args); len(names) > 0 {
		dir, err := chat.ResolveDir(env.Homes, projectArg(names[0]), true, env.CwdOf)
		if err != nil {
			return die(env, err)
		}
		if !isDir(dir) {
			return die(env, "no chats found for: "+dir)
		}
		opts = chat.ListOpts{Dir: dir, Subagents: subagents}
		grouped = false
		title = chat.Pretty(env.Homes, dir)
		if names[0] == "." {
			wd, err := workingDir()
			if err != nil {
				return die(env, err)
			}
			title = wd
		}
		where = title
	}
	load := func(full bool) ([]chat.Chat, error) {
		o := opts
		o.Full = full
		return chat.Load(agents, chat.LoadState(env.Homes).Keep, o)
	}
	switch {
	case selftestRun:
		return selftest(env, agents, load, where)
	case !env.Terminal:
		chats, err := load(true)
		if err != nil {
			return die(env, err)
		}
		printStatic(env, chats, grouped)
		return 0
	}
	return openScreen(env, agents, load, title, grouped)
}

// openScreen starts the full-screen view. With every project shown it also
// tells the view which project folder the cursor starts on.
func openScreen(env Env, agents chat.Agents, load func(bool) ([]chat.Chat, error), title string, grouped bool) int {
	if env.Screen == nil {
		return die(env, "reap: no full-screen view in this build, pipe the output or use reap ls")
	}
	start := Start{Agents: agents, Homes: env.Homes, Stdin: env.Stdin, Stdout: env.Stdout, Load: load, Title: title, Grouped: grouped}
	if grouped {
		home, err := chat.ResolveDir(env.Homes, "", false, env.CwdOf)
		if err != nil {
			return die(env, err)
		}
		start.Home = home
	}
	if err := env.Screen(start); err != nil {
		return die(env, "reap: "+err.Error())
	}
	return 0
}

// workingDir is the current folder with links resolved, as Python's
// os.getcwd gives it. Go's os.Getwd would trust $PWD instead.
func workingDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(wd)
}
