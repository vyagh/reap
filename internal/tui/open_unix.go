//go:build !windows

package tui

import (
	"os"
	"os/exec"
	"syscall"
)

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
