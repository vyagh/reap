package chat

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TrashDays is how long a trashed chat stays recoverable before the prune removes it.
const TrashDays = 7

// Entry is one trashed chat: a folder under the trash with its record file.
type Entry struct {
	Dir    string // the entry folder
	Record Record // reap-meta.json
	Path   string // the trashed transcript, "" if there is none (reap:749-754)
	Size   int64  // bytes the entry holds; for a Codex CLI entry, the archived file
	Proj   string // from the record, "~" if it names no folder; "" for an old record the caller must resolve
	Cwd    string // from the record, "" if not known; an old record's cwd is for the caller to find
}

// codexArchived reports a Codex entry that moved no files, so its chat lives
// on in Codex's own archive (reap:735, 750).
func (r Record) codexArchived() bool {
	return r.Src == string(Codex) && len(r.Moved) == 0
}

// ReadEntry reads the trash entry in dir. It returns ErrNoRecord when dir has
// no usable reap-meta.json. It does not guess a project or cwd that an old
// record lacks, since that needs Claude's transcript reader (reap:713-718, 732-734).
func ReadEntry(dir string) (Entry, error) {
	data, err := os.ReadFile(filepath.Join(dir, "reap-meta.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return Entry{}, ErrNoRecord
	}
	if err != nil {
		return Entry{}, err
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return Entry{}, ErrNoRecord
	}
	e := Entry{Dir: dir, Record: rec, Proj: rec.Proj}
	if rec.Cwd != nil {
		e.Cwd = *rec.Cwd
	}
	if e.Proj == "" && rec.Dir == "" {
		e.Proj = "~"
	}
	if rec.codexArchived() {
		if rec.Archived != nil {
			e.Path = *rec.Archived
			if fi, err := os.Stat(e.Path); err == nil {
				e.Size = fi.Size()
			}
		}
		return e, nil
	}
	for _, m := range rec.Moved {
		if strings.HasSuffix(m[1], ".jsonl") {
			e.Path = filepath.Join(dir, filepath.Base(m[1]))
			break
		}
	}
	e.Size = DirSize(dir)
	return e, nil
}

// DirSize is the bytes of every file under dir, 0 if dir is not a folder.
// Links to folders are not followed, links to files count what they point at
// (reap:242-248).
func DirSize(dir string) int64 {
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return 0
	}
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return 0
	}
	var total int64
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			total += fi.Size()
		}
		return nil
	})
	return total
}

// Chat shows the entry as a chat, so the list's renderer can draw it. The age
// counts from the deletion, the prompt count is not known, and the chat is
// never live (reap:756-765).
func (e Entry) Chat() Chat {
	src := Source(e.Record.Src)
	if src == "" {
		src = Claude
	}
	label := e.Record.Label
	if label == "" {
		label = "(empty)"
	}
	sec := int64(e.Record.At)
	nsec := int64((e.Record.At - float64(sec)) * 1e9)
	return Chat{
		ID:     e.Record.UUID,
		Source: src,
		Dir:    e.Record.Dir,
		Proj:   e.Proj,
		Path:   e.Path,
		Label:  label,
		Titled: e.Record.Label != "",
		Cwd:    e.Cwd,
		Msgs:   -1,
		Size:   e.Size,
		Mod:    time.Unix(sec, nsec),
	}
}

// DaysLeft is how many days an entry stays recoverable: 7 minus the whole days
// since the deletion, never below 0. The elapsed days truncate toward zero, so
// an at in the future gives 7 for up to a day ahead and 8 from a day ahead
// (reap:201-202).
func DaysLeft(at, now time.Time) int {
	elapsed := int(now.Sub(at).Seconds() / 86400)
	return max(0, TrashDays-elapsed)
}
