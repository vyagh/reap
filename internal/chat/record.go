package chat

import (
	"encoding/json"
	"errors"
)

// ErrNoRecord means a trash folder has no usable reap-meta.json: the file is
// missing or is not a JSON object of the expected shape. Such an entry is not
// listed, and the prune goes by the folder's modified time (reap:729, 780-782).
var ErrNoRecord = errors.New("no trash record")

// Record is reap-meta.json, the file that describes one trash entry. Its keys
// match what Python reap 0.5.0 writes, so either version can read the other's.
// Go never rewrites an existing record.
type Record struct {
	UUID  string
	Dir   string // the Claude project folder, "" for a Codex chat
	Label string // title or first prompt, "" if there was none
	At    float64
	// Moved holds [original path, file name inside the entry] pairs. It is
	// written as [] when empty, never null: Python 0.5.0 crashes restoring a
	// record with null (reap:701).
	Moved [][]string
	Proj  string  // project display name, "~" when there is no folder
	Cwd   *string // nil when the working folder was not known
	// Src is "codex" on a Codex record and "" on a Claude one. Only Codex
	// records carry the src and archived keys (reap:655, 663).
	Src string
	// Archived is where `codex archive` put the rollout file, nil when not
	// known. It is written as null in that case, but only when Src is set.
	Archived *string
}

type claudeRecord struct {
	UUID  string     `json:"uuid"`
	Dir   string     `json:"dir"`
	Label string     `json:"label"`
	At    float64    `json:"at"`
	Moved [][]string `json:"moved"`
	Proj  string     `json:"proj"`
	Cwd   *string    `json:"cwd"`
}

type codexRecord struct {
	claudeRecord
	Src      string  `json:"src"`
	Archived *string `json:"archived"`
}

// MarshalJSON writes the record the way Python does: every key for Codex, and
// for Claude every key except src and archived.
func (r Record) MarshalJSON() ([]byte, error) {
	c := claudeRecord{r.UUID, r.Dir, r.Label, r.At, r.Moved, r.Proj, r.Cwd}
	if c.Moved == nil {
		c.Moved = [][]string{}
	}
	if r.Src == "" {
		return json.Marshal(c)
	}
	return json.Marshal(codexRecord{c, r.Src, r.Archived})
}

// UnmarshalJSON reads a record leniently. Missing keys keep their zero value,
// unknown keys are ignored, and a moved pair that is not two strings is
// skipped while the good pairs stay (reap:702). A missing or null cwd means
// "not known"; so does an empty one (reap:732).
func (r *Record) UnmarshalJSON(data []byte) error {
	var raw struct {
		UUID     string            `json:"uuid"`
		Dir      string            `json:"dir"`
		Label    string            `json:"label"`
		At       float64           `json:"at"`
		Moved    []json.RawMessage `json:"moved"`
		Proj     string            `json:"proj"`
		Cwd      *string           `json:"cwd"`
		Src      string            `json:"src"`
		Archived *string           `json:"archived"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	moved := [][]string{}
	for _, m := range raw.Moved {
		var pair []string
		if json.Unmarshal(m, &pair) == nil && len(pair) == 2 {
			moved = append(moved, pair)
		}
	}
	if raw.Cwd != nil && *raw.Cwd == "" {
		raw.Cwd = nil
	}
	*r = Record{raw.UUID, raw.Dir, raw.Label, raw.At, moved, raw.Proj, raw.Cwd, raw.Src, raw.Archived}
	return nil
}
