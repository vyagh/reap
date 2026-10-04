//go:build windows

package tui

import (
	"os"
	"os/exec"
)

// start runs the agent and waits for it: Windows cannot replace a process.
func start(l launch) error {
	cmd := exec.Command(l.argv[0], l.argv[1:]...)
	cmd.Dir = l.dir
	cmd.Env = l.env()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
