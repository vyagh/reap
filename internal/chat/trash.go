package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// exists reports whether path is there. A dangling link counts as missing, as in
// 0.5.0.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// rename moves from to to. A move across disks is refused: a half-copied chat is
// worse than one left in place.
func rename(from, to string) error {
	err := os.Rename(from, to)
	if errors.Is(err, syscall.EXDEV) {
		return fmt.Errorf("cannot move %s to %s: they are on different disks and reap only renames", from, to)
	}
	return err
}

// storedName is the name src gets inside the entry. A name already taken
// there gets its parent folder's name in front, so the sidecar folder <uuid>
// and file-history/<uuid> sit side by side.
func storedName(dir, src string) string {
	base := filepath.Base(src)
	if exists(filepath.Join(dir, base)) {
		base = filepath.Base(filepath.Dir(src)) + "." + base
	}
	return base
}

func writeRecord(dir string, rec Record) error {
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "reap-meta.json"), data, 0o644)
}

// Move makes the trash entry <ms>-<key>, moves each path that exists into it and
// writes rec with Moved and At filled in. An empty rec.Proj becomes the pretty
// name of rec.Dir, or ~ when there is none.
//
// Move is all or nothing. If a step fails, the files already moved go back and
// the error says the chat was not trashed. Any that cannot go back stay in the
// entry, the error names it, and a record is written so restore can find them.
func Move(h Homes, key string, paths []string, rec Record) (Entry, error) {
	if err := os.MkdirAll(h.Trash, 0o755); err != nil {
		return Entry{}, fmt.Errorf("not trashed: %w", err)
	}
	dir := filepath.Join(h.Trash, fmt.Sprintf("%d-%s", time.Now().UnixMilli(), key))
	if err := os.Mkdir(dir, 0o755); err != nil {
		return Entry{}, fmt.Errorf("not trashed: %w", err)
	}
	if rec.Proj == "" {
		rec.Proj = "~"
		if rec.Dir != "" {
			rec.Proj = Pretty(h, rec.Dir)
		}
	}
	rec.Moved = [][]string{}
	for _, src := range paths {
		if !exists(src) {
			continue
		}
		base := storedName(dir, src)
		if exists(filepath.Join(dir, base)) {
			return Entry{}, putBack(dir, rec, fmt.Errorf("%s would overwrite a file already in %s", src, dir))
		}
		if err := rename(src, filepath.Join(dir, base)); err != nil {
			return Entry{}, putBack(dir, rec, err)
		}
		rec.Moved = append(rec.Moved, []string{src, base})
	}
	rec.At = float64(time.Now().UnixMicro()) / 1e6
	if err := writeRecord(dir, rec); err != nil {
		return Entry{}, putBack(dir, rec, err)
	}
	return ReadEntry(dir)
}

// putBack undoes the moves in rec.Moved after cause stopped a trash, newest
// first, and removes the empty entry folder. It returns the error Move gives.
func putBack(dir string, rec Record, cause error) error {
	os.Remove(filepath.Join(dir, "reap-meta.json"))
	var stuck [][]string
	var failures []error
	for i := len(rec.Moved) - 1; i >= 0; i-- {
		src, base := rec.Moved[i][0], rec.Moved[i][1]
		if err := rename(filepath.Join(dir, base), src); err != nil {
			stuck = append(stuck, rec.Moved[i])
			failures = append(failures, err)
		}
	}
	if len(stuck) == 0 {
		os.Remove(dir)
		return fmt.Errorf("not trashed, everything is back where it was: %w", cause)
	}
	rec.Moved = stuck
	msg := fmt.Sprintf("not trashed, but %d item(s) could not go back and stay in %s", len(stuck), dir)
	if err := writeRecord(dir, rec); err != nil {
		msg += " (no record could be written there, so restore will not see them)"
		failures = append(failures, err)
	}
	return fmt.Errorf("%s: %w", msg, errors.Join(append([]error{cause}, failures...)...))
}

// trashPaths lists what sits directly in the trash. Names that start with a dot
// are left out, as in 0.5.0.
func trashPaths(h Homes) []string {
	names, err := os.ReadDir(h.Trash)
	if err != nil {
		return nil
	}
	var paths []string
	for _, n := range names {
		if !strings.HasPrefix(n.Name(), ".") {
			paths = append(paths, filepath.Join(h.Trash, n.Name()))
		}
	}
	return paths
}

// entries reads every trash folder with a usable record, newest first. An old
// record without a project gets one from its folder name.
func entries(h Homes) []Entry {
	var out []Entry
	for _, p := range trashPaths(h) {
		if fi, err := os.Stat(p); err != nil || !fi.IsDir() {
			continue
		}
		e, err := ReadEntry(p)
		if err != nil {
			continue
		}
		if e.Proj == "" {
			e.Proj = Pretty(h, e.Record.Dir)
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Record.At > out[j].Record.At })
	return out
}

// ListTrash is every trash entry with a usable record, newest first. An entry
// whose record has no working folder gets one from its trashed transcript, else
// from a transcript still in its project folder. cwdOf reads the folder out of a
// Claude transcript and finds nothing for a Codex entry.
func ListTrash(h Homes, cwdOf func(transcript string) string) []Entry {
	out := entries(h)
	for i := range out {
		if out[i].Cwd == "" {
			out[i].Cwd = findCwd(out[i], cwdOf)
		}
	}
	return out
}

func findCwd(e Entry, cwdOf func(string) string) string {
	if fi, err := os.Stat(e.Path); err == nil && fi.Mode().IsRegular() {
		if cwd := cwdOf(e.Path); cwd != "" {
			return cwd
		}
	}
	if e.Record.Dir == "" {
		return ""
	}
	return dirCwd(e.Record.Dir, cwdOf)
}

// dirCwd is the working folder of the first transcript in dir that has one.
func dirCwd(dir string, cwdOf func(string) string) string {
	names, _ := os.ReadDir(dir)
	for _, n := range names {
		if strings.HasPrefix(n.Name(), ".") || !strings.HasSuffix(n.Name(), ".jsonl") {
			continue
		}
		if cwd := cwdOf(filepath.Join(dir, n.Name())); cwd != "" {
			return cwd
		}
	}
	return ""
}

// TrashStat is how many entries the trash lists and how many bytes they hold.
// It walks the trash, so not on every redraw.
func TrashStat(h Homes) (count int, size int64) {
	list := entries(h)
	for _, e := range list {
		size += e.Size
	}
	return len(list), size
}

// Prune purges every entry older than TrashDays through its agent. A folder with
// no usable record goes by its modified time and is removed whole. Loose files and
// links in the trash are left alone. An entry that cannot be purged stays for the
// next launch.
func Prune(h Homes, agents Agents, now time.Time) {
	cutoff := now.Add(-TrashDays * 24 * time.Hour)
	for _, p := range trashPaths(h) {
		if fi, err := os.Lstat(p); err != nil || !fi.IsDir() {
			continue
		}
		e, err := ReadEntry(p)
		if err != nil {
			if fi, err := os.Stat(p); err == nil && fi.ModTime().Before(cutoff) {
				os.RemoveAll(p)
			}
			continue
		}
		if e.Record.At >= float64(cutoff.UnixMicro())/1e6 {
			continue
		}
		if agent := agents.For(Source(e.Record.Src)); agent != nil {
			agent.Purge(e)
		} else {
			os.RemoveAll(e.Dir)
		}
	}
}
