// Package cli is everything a user types and reads when the full-screen view
// is not involved: ls, rm, trash, restore, --selftest, help and version.
package cli

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/vyagh/reap/internal/chat"
)

// Version is what reap --version prints.
const Version = "0.5.0"

const allGone = "--all is gone: plain reap shows every project"

// Env is everything Run needs from the outside, so a test can run reap on a
// fake home with fake agents.
type Env struct {
	Stdout, Stderr io.Writer
	Stdin          io.Reader
	Getenv         func(string) string
	Homes          chat.Homes
	// Agents builds the agents in tab order. noCodexCLI is the
	// --no-codex-cli flag: Codex files go into reap's trash instead of
	// through the codex command.
	Agents func(noCodexCLI bool) chat.Agents
	// CwdOf reads the working folder out of one Claude transcript.
	CwdOf func(transcript string) string
	Now   func() time.Time
	// Terminal is true when Stdout is a terminal.
	Terminal bool
	// Screen opens the full-screen view. Nil means there is none.
	Screen func(Start) error
}

// Start is what the full-screen view is given to open.
type Start struct {
	Agents  chat.Agents
	Homes   chat.Homes
	Stdin   io.Reader
	Stdout  io.Writer
	Load    func(full bool) ([]chat.Chat, error)
	Title   string
	Grouped bool
	// CwdOf reads the working folder out of one Claude transcript.
	CwdOf func(transcript string) string
	// NoCodexCLI is the --no-codex-cli flag: a Codex purge does not call codex.
	NoCodexCLI bool
	// Home is the project folder the cursor starts on. It is only set when
	// the view shows every project.
	Home string
}

// Run runs reap with the arguments after the program name and returns the exit
// code. Everything it prints goes to env.Stdout and env.Stderr.
func Run(args []string, env Env) int {
	if slices.Contains(args, "--version") || slices.Contains(args, "-V") {
		fmt.Fprintf(env.Stdout, "reap %s\n", Version)
		return 0
	}
	if slices.Contains(args, "-h") || slices.Contains(args, "--help") {
		printHelp(env)
		return 0
	}
	noCLI := popFlag(&args, "--no-codex-cli")
	agents := env.Agents(noCLI)
	chat.Prune(env.Homes, agents, env.Now())
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "ls":
			return cmdLs(env, agents, args[1:])
		case "rm":
			return cmdRm(env, agents, args[1:])
		case "trash":
			return cmdTrash(env, agents, args[1:])
		case "restore":
			return cmdRestore(env, agents, args[1:])
		}
	}
	return cmdDefault(env, agents, noCLI, args)
}

// die prints msg to stderr and returns 1, where Python calls sys.exit with a string.
func die(env Env, msg any) int {
	fmt.Fprintln(env.Stderr, msg)
	return 1
}

// popFlag removes the first occurrence of flag from args and reports whether
// there was one.
func popFlag(args *[]string, flag string) bool {
	i := slices.Index(*args, flag)
	if i < 0 {
		return false
	}
	*args = slices.Delete(*args, i, i+1)
	return true
}

// popOpt removes the first occurrence of opt and the argument after it, which
// is the value even when it looks like a flag. It returns "" when opt is absent
// or last.
func popOpt(args *[]string, opt string) string {
	i := slices.Index(*args, opt)
	if i < 0 {
		return ""
	}
	val := ""
	if i+1 < len(*args) {
		val = (*args)[i+1]
	}
	*args = slices.Delete(*args, i, min(i+2, len(*args)))
	return val
}

// operands are the arguments that do not start with a dash.
func operands(args []string) []string {
	var out []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			out = append(out, a)
		}
	}
	return out
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
