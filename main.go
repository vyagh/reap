// Command reap views, opens and deletes local Claude Code and Codex chats.
package main

import (
	"fmt"
	"os"
	"os/user"
	"time"

	"github.com/vyagh/reap/internal/chat"
	"github.com/vyagh/reap/internal/claude"
	"github.com/vyagh/reap/internal/cli"
	"github.com/vyagh/reap/internal/codex"
	"github.com/vyagh/reap/internal/tui"
)

func main() {
	home := homeDir()
	if home == "" {
		fmt.Fprintln(os.Stderr, "reap: cannot find your home folder")
		os.Exit(1)
	}
	h := chat.NewHomes(os.Getenv, home)
	fi, _ := os.Stdout.Stat()
	os.Exit(cli.Run(os.Args[1:], cli.Env{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
		Getenv: os.Getenv,
		Homes:  h,
		Agents: func(noCLI bool) chat.Agents {
			return chat.Agents{claude.New(h), codex.New(h, noCLI)}
		},
		CwdOf:    claude.CwdOf,
		Now:      time.Now,
		Terminal: fi != nil && fi.Mode()&os.ModeCharDevice != 0,
		Screen:   tui.Run,
	}))
}

// homeDir is the user's home folder: $HOME, else the entry in the system's user
// list, as expanduser finds it in 0.5.0. "" when neither knows.
func homeDir() string {
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	if u, err := user.Current(); err == nil {
		return u.HomeDir
	}
	return ""
}
