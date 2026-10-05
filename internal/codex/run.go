package codex

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

// codexTimeout is how long one codex call may run before it is stopped and fails.
var codexTimeout = 30 * time.Second

// runCodex runs `codex args...`. A failure is reported as the first line codex
// wrote to stderr, else to stdout, as 0.5.0 shows it.
func runCodex(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), codexTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "codex", args...)
	cmd.WaitDelay = time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	switch {
	case err == nil:
		return nil
	case errors.Is(err, exec.ErrNotFound):
		return errors.New("codex not found on PATH")
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		return errors.New("timed out")
	}
	out := strings.TrimSpace(stderr.String())
	if out == "" {
		out = strings.TrimSpace(stdout.String())
	}
	if line, _, _ := strings.Cut(out, "\n"); line != "" {
		return errors.New(strings.TrimSpace(line))
	}
	return err
}
