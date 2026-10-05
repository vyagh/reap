package codex

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vyagh/reap/internal/chat"
)

// Agent is the Codex CLI. Its chats are the rollouts under sessions/, and the lock
// files say whether one is still open.
type Agent struct {
	h     chat.Homes
	noCLI bool // move files into reap's trash instead of calling codex
}

// New returns the Codex agent for the folders in h. With noCLI set, deletes move the rollout into reap's trash instead of calling codex.
func New(h chat.Homes, noCLI bool) *Agent {
	return &Agent{h: h, noCLI: noCLI}
}

func (a *Agent) Name() chat.Source { return chat.Codex }

func (a *Agent) Installed() bool {
	for _, d := range []string{"sessions", "archived_sessions"} {
		if st, err := os.Stat(filepath.Join(a.h.Codex, d)); err == nil && st.IsDir() {
			return true
		}
	}
	return false
}

// List returns every rollout under sessions/ that starts with a session_meta
// line. Helper runs (subagent, guardian_review) are left out unless
// opts.Subagents is set. A chat groups under the Claude project folder named
// after the folder it last ran in. opts.Tmp and opts.Full do not apply to Codex.
func (a *Agent) List(opts chat.ListOpts) ([]chat.Chat, error) {
	paths, err := rollouts(a.h.Codex)
	if err != nil {
		return nil, err
	}
	titles := readTitles(a.h.Codex)
	var out []chat.Chat
	for _, path := range paths {
		m, ok := readMeta(path)
		if !ok {
			continue
		}
		if !opts.Subagents && (m.ThreadSource == "subagent" || m.ThreadSource == "guardian_review") {
			continue
		}
		cwd := lastCwd(path, tailBytes)
		if cwd == "" {
			cwd = m.Cwd
		}
		dir, proj := "", "~"
		if cwd != "" {
			dir = filepath.Join(a.h.Claude, "projects", chat.DirnameFor(cwd))
			proj = chat.Pretty(a.h, dir)
		}
		if opts.Dir != "" && dir != opts.Dir {
			continue
		}
		st, err := os.Stat(path)
		if err != nil {
			continue
		}
		label := titles[m.ID]
		titled := label != ""
		scanned, n := scan(path)
		if !titled {
			label = scanned
		}
		if label == "" {
			label = "(untitled)"
		}
		out = append(out, chat.Chat{
			ID:     m.ID,
			Source: chat.Codex,
			Dir:    dir,
			Proj:   proj,
			Path:   path,
			Label:  chat.Clean(label),
			Titled: titled,
			Cwd:    cwd,
			Msgs:   n,
			Size:   st.Size(),
			Mod:    st.ModTime(),
			Live:   lockLive(a.h.Codex, m.ID),
		})
	}
	return out, nil
}

func (a *Agent) Live(c chat.Chat) bool { return lockLive(a.h.Codex, c.ID) }

// Exists reports whether a rollout under sessions/ has this id in its name. If
// the folder cannot be searched it says yes, so no hidden id is dropped on a guess.
func (a *Agent) Exists(id string) bool {
	paths, err := rollouts(a.h.Codex)
	if err != nil {
		return true
	}
	for _, p := range paths {
		if strings.Contains(strings.TrimSuffix(filepath.Base(p), ".jsonl"), id) {
			return true
		}
	}
	return false
}

// Kinds is nil, so the dry-run line names the agent.
func (a *Agent) Kinds(chat.Chat) []string { return nil }

// Peek ignores textOnly, since rollouts hold no tool markers.
func (a *Agent) Peek(c chat.Chat, max int, textOnly bool) ([]chat.Turn, error) {
	return peek(c.Path, max)
}

// Count is the real prompts in the first 400 lines of c.Path, as List shows.
func (a *Agent) Count(c chat.Chat) (int, error) {
	if _, err := os.Stat(c.Path); err != nil {
		return 0, err
	}
	_, n := scan(c.Path)
	return n, nil
}

// ResumeCmd is codex resume, run in the folder of the chat's newest turn. That can
// differ from the folder List shows.
func (a *Agent) ResumeCmd(c chat.Chat) ([]string, string, error) {
	dir := lastCwd(c.Path, wholeFile)
	if dir == "" {
		dir = c.Cwd
	}
	return []string{"codex", "resume", c.ID}, dir, nil
}
