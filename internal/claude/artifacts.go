package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/vyagh/reap/internal/chat"
)

// resumerJobDirs are the job folders whose state.json names this chat as the one
// to resume, which would dangle once it is deleted. A state.json that is not
// valid UTF-8 is an error naming the file, where 0.5.0 stops with a traceback.
func resumerJobDirs(h chat.Homes, id string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(h.Claude, "jobs"))
	if err != nil {
		return nil, nil
	}
	var dirs []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dir := filepath.Join(h.Claude, "jobs", e.Name())
		path := filepath.Join(dir, "state.json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if !utf8.Valid(data) {
			return dirs, fmt.Errorf("%s is not valid UTF-8", path)
		}
		if !bytes.Contains(data, []byte(id)) {
			continue
		}
		var state map[string]any
		if json.Unmarshal(data, &state) == nil && state["resumeSessionId"] == id {
			dirs = append(dirs, dir)
		}
	}
	return dirs, nil
}

// artifacts are the paths that make up a chat, whether or not they exist: the
// transcript, its sidecar folder, the per-session state Claude Code keeps by uuid,
// its job folder and every job that resumes it. On an error from resumerJobDirs
// it returns the paths it has with it.
func artifacts(h chat.Homes, dir, id string) ([]string, error) {
	arts := []string{filepath.Join(dir, id+".jsonl"), filepath.Join(dir, id)}
	for _, sub := range []string{"file-history", "session-env", "tasks"} {
		arts = append(arts, filepath.Join(h.Claude, sub, id))
	}
	if short, _, _ := strings.Cut(id, "-"); short != "" {
		arts = append(arts, filepath.Join(h.Claude, "jobs", short))
	}
	resumers, err := resumerJobDirs(h, id)
	return append(arts, resumers...), err
}

// artifactKinds names what a delete would move, once each and only what is on
// disk.
func artifactKinds(h chat.Homes, dir, id string) []string {
	paths, _ := artifacts(h, dir, id)
	kinds := []string{}
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		kind := filepath.Base(filepath.Dir(p))
		if strings.HasSuffix(p, ".jsonl") {
			kind = "transcript"
		} else if p == filepath.Join(dir, id) {
			kind = "sidecar"
		}
		if !slices.Contains(kinds, kind) {
			kinds = append(kinds, kind)
		}
	}
	return kinds
}
