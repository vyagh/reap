//go:build !windows

package tui

import (
	"os"
	"os/exec"
	"syscall"
)

// start replaces reap with the agent, so the agent owns the terminal
// (reap:628-637).
func start(l launch) error {
	path, err := exec.LookPath(l.argv[0])
	if err != nil {
		return err
	}
	if err := os.Chdir(l.dir); err != nil {
		return err
	}
	return syscall.Exec(path, l.argv, l.env())
}
