package chat

import (
	"path/filepath"
	"slices"
)

// Homes are the folders reap reads and writes, passed in so no path lives in a
// package variable.
type Homes struct {
	Home       string
	Claude     string // Claude Code's folder, holding projects/
	Codex      string // Codex's folder, holding sessions/ and archived_sessions/
	State      string
	Trash      string // where soft-deleted Claude chats wait, inside the Claude folder
	CodexTrash string // where Codex chats wait, inside the Codex folder
}

// trashDirs are the folders a trash entry can sit in. An entry is handled by
// the agent its record names, so a Codex record in Claude's folder still counts.
func (h Homes) trashDirs() []string {
	var dirs []string
	for _, d := range []string{h.Trash, h.CodexTrash} {
		if d != "" && !slices.Contains(dirs, d) {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// NewHomes reads the folders as 0.5.0 does: CLAUDE_CONFIG_DIR or ~/.claude,
// CODEX_HOME or ~/.codex, ~/.local/state/reap, and the trash at
// <claude home>/.reap-trash for Claude chats and <codex home>/.reap-trash for Codex chats. An empty variable counts as unset.
func NewHomes(getenv func(string) string, home string) Homes {
	claude := getenv("CLAUDE_CONFIG_DIR")
	if claude == "" {
		claude = filepath.Join(home, ".claude")
	}
	codex := getenv("CODEX_HOME")
	if codex == "" {
		codex = filepath.Join(home, ".codex")
	}
	return Homes{
		Home:       home,
		Claude:     claude,
		Codex:      codex,
		State:      filepath.Join(home, ".local", "state", "reap"),
		Trash:      filepath.Join(claude, ".reap-trash"),
		CodexTrash: filepath.Join(codex, ".reap-trash"),
	}
}
