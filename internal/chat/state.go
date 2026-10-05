package chat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

// State is reap's own memory between runs, kept in state.json. The key names match
// 0.5.0, which shares the file.
type State struct {
	Keep map[string]float64 // hidden chat id -> when it was hidden
	Pin  string             // pinned tab, "" if none
}

type stateFile struct {
	Keep map[string]float64 `json:"keep"`
	Pin  *string            `json:"pin,omitempty"`
}

// LoadState reads the state file. It never creates it, and a file that is
// missing, unreadable or not a JSON object reads as empty. A hidden entry whose
// value is not a number is skipped, and so is a pin that is not a string.
func LoadState(h Homes) State {
	s := State{Keep: map[string]float64{}}
	data, err := os.ReadFile(filepath.Join(h.State, "state.json"))
	if err != nil {
		return s
	}
	var raw struct {
		Keep map[string]json.RawMessage `json:"keep"`
		Pin  json.RawMessage            `json:"pin"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return s
	}
	for id, v := range raw.Keep {
		var at *float64
		if json.Unmarshal(v, &at) == nil && at != nil {
			s.Keep[id] = *at
		}
	}
	json.Unmarshal(raw.Pin, &s.Pin)
	return s
}

// SaveState drops hidden ids whose chat is gone, then writes the file through a
// temp file and a rename so a reader never sees half of it. It is for a hide, a
// pin or a delete, never for a read-only run.
func SaveState(h Homes, s State, agents Agents) error {
	f := stateFile{Keep: map[string]float64{}}
	for id, at := range s.Keep {
		if onDisk(h, id, agents) {
			f.Keep[id] = at
		}
	}
	if s.Pin != "" {
		f.Pin = &s.Pin
	}
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(h.State, 0o777); err != nil {
		return err
	}
	path := filepath.Join(h.State, "state.json")
	tmp := path + ".tmp" + strconv.Itoa(os.Getpid())
	if err := os.WriteFile(tmp, data, 0o666); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// onDisk reports whether a hidden id is still worth remembering: some agent
// has the chat, or a trash entry holds it.
func onDisk(h Homes, id string, agents Agents) bool {
	for _, a := range agents {
		if a.Exists(id) {
			return true
		}
	}
	var dirs []string
	for _, t := range h.trashDirs() {
		found, _ := filepath.Glob(filepath.Join(t, "*"))
		dirs = append(dirs, found...)
	}
	for _, d := range dirs {
		data, err := os.ReadFile(filepath.Join(d, "reap-meta.json"))
		if err != nil {
			continue
		}
		var rec struct {
			UUID string `json:"uuid"`
		}
		if json.Unmarshal(data, &rec) == nil && rec.UUID == id {
			return true
		}
	}
	return false
}
