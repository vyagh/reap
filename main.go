// Command reap views, opens and deletes local Claude Code and Codex chats.
package main

import (
	"os"
	"time"

	"github.com/vyagh/reap/internal/chat"
	"github.com/vyagh/reap/internal/claude"
	"github.com/vyagh/reap/internal/cli"
	"github.com/vyagh/reap/internal/codex"
)

func main() {
	home, _ := os.UserHomeDir()
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
	}))
}
