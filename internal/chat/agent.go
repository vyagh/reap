package chat

// Agent is everything reap needs from one coding agent. Agents return errors
// and never print or exit.
type Agent interface {
	Name() Source
	// Installed reports whether the agent's folder exists.
	Installed() bool
	// List returns the agent's chats with Live set from one read of its
	// session files. Hidden and the sort order are left to the caller.
	List(ListOpts) ([]Chat, error)
	// Live re-reads the session files and reports whether the chat is still
	// open. It runs right before a delete.
	Live(Chat) bool
	// Exists reports whether a chat with this id is still on disk. The state
	// file uses it to drop hidden ids that are gone.
	Exists(id string) bool
	// Kinds names what a delete would move, for the dry-run line. An agent
	// that cannot say returns nil and the command prints its name instead.
	Kinds(Chat) []string
	// Peek returns up to max turns of the chat from c.Path alone. The side
	// panel sets textOnly to leave out tool markers, the reader does not.
	Peek(c Chat, max int, textOnly bool) ([]Turn, error)
	// Count reads the whole chat for its prompt count. It reads c.Path alone,
	// as the trash view calls it on trashed chats.
	Count(Chat) (int, error)
	// ResumeCmd is the command that reopens the chat and the folder to run it in.
	ResumeCmd(Chat) (argv []string, dir string, err error)
	// Trash moves the chat to the trash and returns the entry it made.
	Trash(Chat) (Entry, error)
	// Delete removes the chat for good, with no trash entry.
	Delete(Chat) error
	// Restore puts a trashed chat back and removes its entry.
	Restore(Entry) error
	// Purge removes a trash entry and whatever it still holds for good.
	Purge(Entry) error
}

// Agents is the installed agents in tab order; the first is Claude.
type Agents []Agent

// For returns the agent for a source, or nil if there is none. An empty
// source means the first agent, since records written before Codex support
// carry no src and belong to Claude.
func (a Agents) For(s Source) Agent {
	if s == "" {
		if len(a) == 0 {
			return nil
		}
		return a[0]
	}
	for _, ag := range a {
		if ag.Name() == s {
			return ag
		}
	}
	return nil
}
