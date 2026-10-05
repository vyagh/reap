package claude

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vyagh/reap/internal/chat"
)

// Agent is Claude Code. Its chats are the transcripts under <claude home>/projects.
type Agent struct {
	home chat.Homes
}

func New(h chat.Homes) *Agent {
	return &Agent{home: h}
}

func (a *Agent) Name() chat.Source {
	return chat.Claude
}

func (a *Agent) Installed() bool {
	fi, err := os.Stat(a.projects())
	return err == nil && fi.IsDir()
}

// List returns the chats of opts.Dir, or of every project folder when it is empty.
// Projects under the system temp folder are left out unless opts.Tmp is set. A
// folder that cannot be read has no chats, as in 0.5.0, so List does not fail.
func (a *Agent) List(opts chat.ListOpts) ([]chat.Chat, error) {
	dirs := []string{opts.Dir}
	if opts.Dir == "" {
		dirs = a.projectDirs(opts.Tmp)
	}
	live := liveSessions(a.home)
	var chats []chat.Chat
	for _, dir := range dirs {
		chats = append(chats, a.listDir(dir, live, opts.Full)...)
	}
	return chats, nil
}

// Live re-reads the session files. A chat that cannot be told apart from an open
// one counts as open.
func (a *Agent) Live(c chat.Chat) bool {
	return liveSessions(a.home).has(c.ID)
}

// Exists reports whether some project folder holds a transcript with this id.
func (a *Agent) Exists(id string) bool {
	entries, _ := os.ReadDir(a.projects())
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if _, err := os.Lstat(filepath.Join(a.projects(), e.Name(), id+".jsonl")); err == nil {
			return true
		}
	}
	return false
}

func (a *Agent) Kinds(c chat.Chat) []string {
	return artifactKinds(a.home, c.Dir, c.ID)
}

func (a *Agent) Peek(c chat.Chat, limit int, textOnly bool) ([]chat.Turn, error) {
	return peek(c.Path, limit, textOnly)
}

// Count reads the whole transcript at c.Path. A file that cannot be opened is an error.
func (a *Agent) Count(c chat.Chat) (int, error) {
	i, err := meta(c.Path, true)
	return i.n, err
}

func (a *Agent) ResumeCmd(c chat.Chat) ([]string, string, error) {
	return []string{"claude", "--resume", c.ID}, c.Cwd, nil
}

func (a *Agent) projects() string {
	return filepath.Join(a.home.Claude, "projects")
}

// projectDirs are the project folders, by name. A link out of the projects folder
// is left out, so reap never reads or deletes through it.
func (a *Agent) projectDirs(tmp bool) []string {
	entries, _ := os.ReadDir(a.projects())
	var dirs []string
	for _, e := range entries {
		name := e.Name()
		dir := filepath.Join(a.projects(), name)
		if strings.HasPrefix(name, ".") || !chat.UnderProj(a.home, dir) {
			continue
		}
		if !tmp && (strings.HasPrefix(name, "-tmp-") || strings.HasPrefix(name, "-private-tmp-")) {
			continue
		}
		dirs = append(dirs, dir)
	}
	return dirs
}

// listDir is the chats in one project folder. A transcript that vanishes mid-read is left out.
func (a *Agent) listDir(dir string, live liveSet, full bool) []chat.Chat {
	entries, _ := os.ReadDir(dir)
	var chats []chat.Chat
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		path := filepath.Join(dir, name)
		fi, err := os.Stat(path)
		if err != nil {
			continue
		}
		id := strings.TrimSuffix(name, ".jsonl")
		i, _ := meta(path, full)
		c := chat.Chat{
			ID:     id,
			Source: chat.Claude,
			Dir:    dir,
			Proj:   chat.Pretty(a.home, dir),
			Path:   path,
			Label:  "(empty)",
			Titled: i.title != "",
			Cwd:    i.cwd,
			Msgs:   i.n,
			Size: fi.Size() + chat.DirSize(filepath.Join(dir, id)) +
				chat.DirSize(filepath.Join(a.home.Claude, "file-history", id)),
			Mod:  fi.ModTime(),
			Live: live.has(id),
		}
		if i.title != "" {
			c.Label = i.title
		} else if i.prompt != "" {
			c.Label = i.prompt
		}
		chats = append(chats, c)
	}
	return chats
}
