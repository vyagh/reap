// Package chat holds what every part of reap shares: the chat model, the
// agent interface, the folders reap reads and writes, and the trash record.
package chat

import (
	"strings"
	"time"
)

// Source names the coding agent a chat belongs to. It is also the value of
// the src key in a trash record.
type Source string

const (
	Claude Source = "claude"
	Codex  Source = "codex"
)

// Chat is one session of one agent, as the list shows it.
type Chat struct {
	ID     string
	Source Source
	Dir    string // the Claude project folder it groups under, "" if none
	Proj   string // display name of that project, "~" for the home folder
	Path   string // transcript or rollout file
	Label  string // title, else first prompt, else a placeholder
	Titled bool   // Label is a real title
	Cwd    string // working folder recorded in the chat, "" if unknown
	Msgs   int    // prompt count, -1 if not known
	Size   int64  // bytes of everything that belongs to the chat
	Mod    time.Time
	Live   bool // the agent still has it open; never deleted
	Hidden bool
}

// Short is the first segment of the id, the form the list prints.
func (c Chat) Short() string {
	short, _, _ := strings.Cut(c.ID, "-")
	return short
}

// Turn is one message in the reader or the side panel.
type Turn struct {
	Who  string // "you", "claude", "codex"
	Text string
}

// ListOpts narrows a listing.
type ListOpts struct {
	Dir       string // one project folder, "" for every project
	Tmp       bool   // include projects under the system temp folder
	Subagents bool   // include an agent's own helper sessions
	Full      bool   // read each transcript in full for an exact Msgs
}
