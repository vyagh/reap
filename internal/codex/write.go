package codex

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vyagh/reap/internal/chat"
)

var _ chat.Agent = (*Agent)(nil)

// rolloutPaths lists the rollout files of one chat under sessions/ (reap:441-442).
func (a *Agent) rolloutPaths(id string) []string {
	all, _ := filepath.Glob(filepath.Join(a.h.Codex, "sessions", "*", "*", "*", "*"+id+"*.jsonl"))
	var out []string
	for _, p := range all {
		if !dotted(p) {
			out = append(out, p)
		}
	}
	return out
}

// archivedPath is the rollout file `codex archive` left in archived_sessions/
// for a chat, "" if there is none (reap:660).
func (a *Agent) archivedPath(id string) string {
	all, _ := filepath.Glob(filepath.Join(a.h.Codex, "archived_sessions", "*"+id+"*.jsonl"))
	for _, p := range all {
		if !strings.HasPrefix(filepath.Base(p), ".") {
			return p
		}
	}
	return ""
}

// Trash soft-deletes the chat. Normally codex archives it and the entry only
// records where the archived file is, since moving a rollout by hand would
// leave Codex's database pointing at nothing (reap:639-663). With noCLI the
// rollout file moves into reap's trash instead.
func (a *Agent) Trash(c chat.Chat) (chat.Entry, error) {
	rec := chat.Record{UUID: c.ID, Label: c.Label, Proj: c.Proj, Src: string(chat.Codex)}
	if c.Cwd != "" {
		rec.Cwd = &c.Cwd
	}
	if a.noCLI {
		return a.trashFiles(c, rec)
	}
	return a.archive(c, rec)
}

func (a *Agent) trashFiles(c chat.Chat, rec chat.Record) (chat.Entry, error) {
	paths := a.rolloutPaths(c.ID)
	if len(paths) == 0 {
		return chat.Entry{}, fmt.Errorf("not trashed: no rollout file for %s under %s", c.ID, filepath.Join(a.h.Codex, "sessions"))
	}
	return chat.Move(a.h, c.ID, paths, rec)
}

// archive runs codex archive and writes an entry for it. If codex archived the
// chat but no entry could be written, the error says where the chat is and how
// to get it back, since no entry will show it (reap:657-663).
func (a *Agent) archive(c chat.Chat, rec chat.Record) (chat.Entry, error) {
	if err := runCodex("archive", c.ID); err != nil {
		return chat.Entry{}, fmt.Errorf("codex archive failed: %w", err)
	}
	if p := a.archivedPath(c.ID); p != "" {
		rec.Archived = &p
	}
	e, err := chat.Move(a.h, c.ID, nil, rec)
	if err != nil {
		return chat.Entry{}, fmt.Errorf("codex archived %s but reap has no trash entry for it (codex unarchive %s brings it back): %w", c.ID, c.ID, err)
	}
	return e, nil
}

// Delete trashes the chat, then purges the entry (reap:1863).
func (a *Agent) Delete(c chat.Chat) error {
	e, err := a.Trash(c)
	if err != nil {
		return err
	}
	return a.Purge(e)
}

// Restore puts a trashed chat back. An entry that holds files (made with
// noCLI) moves them back to sessions/ and is removed only when every file is
// back. An entry that holds none is a codex archive, so codex unarchive runs.
// That is also so under noCLI (reap:666-680).
func (a *Agent) Restore(e chat.Entry) error {
	if len(e.Record.Moved) > 0 {
		return a.restoreFiles(e)
	}
	if err := runCodex("unarchive", e.Record.UUID); err != nil {
		return fmt.Errorf("codex unarchive failed, the entry stays in %s: %w", e.Dir, err)
	}
	return os.RemoveAll(e.Dir)
}

func (a *Agent) restoreFiles(e chat.Entry) error {
	left, err := chat.MoveBack(e, []string{filepath.Join(a.h.Codex, "sessions")})
	if len(left) > 0 || err != nil {
		return fmt.Errorf("%d file(s) stay in %s: %w", len(left), e.Dir, err)
	}
	return os.RemoveAll(e.Dir)
}

// Purge removes the entry for good and, for an archive entry, asks codex to
// delete the archived chat. It skips that call if the chat is back under
// sessions/ without its archived file, because the user unarchived it by hand
// (reap:683-686). The entry goes even if codex fails, since a purge is
// permanent; the error then says the chat is still in Codex's archive
// (reap:687).
func (a *Agent) Purge(e chat.Entry) error {
	var cliErr error
	if !a.noCLI && len(e.Record.Moved) == 0 && !a.unarchivedByHand(e.Record) {
		if err := runCodex("delete", e.Record.UUID, "--force"); err != nil {
			cliErr = fmt.Errorf("codex delete failed, the chat may still be in Codex's archive: %w", err)
		}
	}
	if err := os.RemoveAll(e.Dir); err != nil {
		return errors.Join(cliErr, err)
	}
	return cliErr
}

func (a *Agent) unarchivedByHand(rec chat.Record) bool {
	if rec.Archived == nil || *rec.Archived == "" {
		return false
	}
	if _, err := os.Stat(*rec.Archived); err == nil {
		return false
	}
	return len(a.rolloutPaths(rec.UUID)) > 0
}
