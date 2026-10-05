package claude

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vyagh/reap/internal/chat"
)

var _ chat.Agent = (*Agent)(nil)

// chatPaths are the files of a chat. A chat with no project folder has none, as
// its paths would be relative to wherever reap runs.
func (a *Agent) chatPaths(c chat.Chat) ([]string, error) {
	if c.Dir == "" {
		return nil, fmt.Errorf("chat %s has no project folder", c.ID)
	}
	return artifacts(a.home, c.Dir, c.ID)
}

// Trash moves everything that belongs to the chat into a trash entry. If a step
// fails the files are put back, or the error says where they stay. A job state
// file that cannot be read stops it before anything moves, where 0.5.0 stops with
// a traceback.
func (a *Agent) Trash(c chat.Chat) (chat.Entry, error) {
	paths, err := a.chatPaths(c)
	if err != nil {
		return chat.Entry{}, fmt.Errorf("not trashed: %w", err)
	}
	rec := chat.Record{UUID: c.ID, Dir: c.Dir, Label: c.Label}
	if c.Cwd != "" {
		rec.Cwd = &c.Cwd
	}
	return chat.Move(a.home, c.ID, paths, rec)
}

// Delete removes everything that belongs to the chat for good, then the project
// folder if that leaves it bare. A path that cannot be removed is an error naming
// it, where 0.5.0 skips it and reports success. Nothing is removed if the job
// folders cannot be read.
func (a *Agent) Delete(c chat.Chat) error {
	paths, err := a.chatPaths(c)
	if err != nil {
		return fmt.Errorf("not deleted: %w", err)
	}
	var failures []error
	for _, p := range paths {
		if err := os.RemoveAll(p); err != nil {
			failures = append(failures, err)
		}
	}
	dropBareProj(a.home, c.Dir)
	return errors.Join(failures...)
}

// Restore puts the files of the entry back and removes the entry. A file that
// cannot go back stays in the entry and the error says which and why.
func (a *Agent) Restore(e chat.Entry) error {
	roots := []string{a.projects()}
	for _, sub := range []string{"jobs", "file-history", "session-env", "tasks"} {
		roots = append(roots, filepath.Join(a.home.Claude, sub))
	}
	left, err := chat.MoveBack(e, roots)
	if err != nil {
		return fmt.Errorf("%d item(s) not restored, they stay in %s: %w", len(left), e.Dir, err)
	}
	return os.RemoveAll(e.Dir)
}

// Purge removes the entry and what it holds for good, then the chat's project
// folder if that leaves it bare.
func (a *Agent) Purge(e chat.Entry) error {
	if err := os.RemoveAll(e.Dir); err != nil {
		return fmt.Errorf("not purged: %w", err)
	}
	dropBareProj(a.home, e.Record.Dir)
	return nil
}

// dropBareProj removes a project folder that holds nothing, or only an empty
// memory folder, which Claude Code makes again.
func dropBareProj(h chat.Homes, dir string) {
	if dir == "" || !chat.UnderProj(h, dir) {
		return
	}
	names, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(names) == 1 && names[0].Name() == "memory" {
		if !rmdir(filepath.Join(dir, "memory")) {
			return
		}
	} else if len(names) > 0 {
		return
	}
	rmdir(dir)
}

// rmdir is os.Remove for folders only, as os.rmdir in 0.5.0 is.
func rmdir(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.IsDir() && os.Remove(path) == nil
}
