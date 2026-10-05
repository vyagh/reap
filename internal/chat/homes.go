package chat

import "path/filepath"

// Homes are the folders reap reads and writes, passed in so no path lives in a
// package variable.
type Homes struct {
	Home   string
	Claude string // Claude Code's folder, holding projects/
	Codex  string // Codex's folder, holding sessions/ and archived_sessions/
	State  string
	Trash  string // where soft-deleted chats wait, inside the Claude folder
}

// NewHomes reads the folders as 0.5.0 does: CLAUDE_CONFIG_DIR or ~/.claude,
// CODEX_HOME or ~/.codex, ~/.local/state/reap, and the trash at
// <claude home>/.reap-trash. An empty variable counts as unset.
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
		Home:   home,
		Claude: claude,
		Codex:  codex,
		State:  filepath.Join(home, ".local", "state", "reap"),
		Trash:  filepath.Join(claude, ".reap-trash"),
	}
}
