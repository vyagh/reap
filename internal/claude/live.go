package claude

import (
	"encoding/json"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/vyagh/reap/internal/chat"
)

// liveSet is the chats Claude has open. all is set when a file that should say
// which are open could not be read, so no chat can be called closed.
type liveSet struct {
	ids map[string]bool
	all bool
}

func (l liveSet) has(id string) bool {
	return l.all || l.ids[id]
}

// liveSessions reads the session files and the daemon roster once. A session or
// worker whose process is confirmed gone is stale and protects nothing. A session
// file that cannot be read or parsed counts as every chat open, since it cannot
// say which one it is for.
func liveSessions(h chat.Homes) liveSet {
	live := liveSet{ids: map[string]bool{}}
	live.readSessions(filepath.Join(h.Claude, "sessions"))
	live.readRoster(filepath.Join(h.Claude, "daemon", "roster.json"))
	return live
}

func (l *liveSet) readSessions(dir string) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		l.all = true
		return
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".json") {
			continue
		}
		v, err := readJSON(filepath.Join(dir, name))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			l.all = true
			continue
		}
		if o, ok := v.(map[string]any); ok {
			l.add(o)
		}
	}
}

// readRoster adds the daemon's workers. A roster that is missing, unreadable or
// not an object adds nothing. A workers value that is not an object counts as
// every chat open, where 0.5.0 stops with an error.
func (l *liveSet) readRoster(path string) {
	v, _ := readJSON(path)
	r, ok := v.(map[string]any)
	if !ok {
		return
	}
	workers, ok := r["workers"]
	if !ok {
		return
	}
	m, ok := workers.(map[string]any)
	if !ok {
		l.all = true
		return
	}
	for _, w := range m {
		if o, ok := w.(map[string]any); ok {
			l.add(o)
		}
	}
}

func (l *liveSet) add(o map[string]any) {
	if procDead(o["pid"], o["pidDomain"]) {
		return
	}
	if id, ok := o["sessionId"].(string); ok && id != "" {
		l.ids[id] = true
	}
}

// ownPidDomain is Claude Code's pidDomain for this host. Every platform but
// Linux writes the bare string "linux".
func ownPidDomain() string {
	if runtime.GOOS != "linux" {
		return "linux"
	}
	return linuxPidDomain("/etc/machine-id", "/proc/self/ns/pid")
}

// linuxPidDomain is linux:<machine id>:<pid namespace>, with an empty part
// where one cannot be read.
func linuxPidDomain(machineID, pidNS string) string {
	id, _ := os.ReadFile(machineID)
	ns, _ := os.Readlink(pidNS)
	return "linux:" + strings.TrimSpace(string(id)) + ":" + ns
}

// procDead is true only when the pid is confirmed gone. A pid that is not an
// integer, does not fit a C int or belongs to another machine or pid namespace
// (pidDomain) is never judged dead, nor is one the system will not answer for.
func procDead(pid, domain any) bool {
	n, ok := pid.(json.Number)
	if !ok {
		return false
	}
	id, err := strconv.ParseInt(string(n), 10, 64)
	if err != nil || id < math.MinInt32 || id > math.MaxInt32 {
		return false
	}
	if truthy(domain) {
		s, ok := domain.(string)
		if !ok || strings.TrimSpace(s) != ownPidDomain() {
			return false
		}
	}
	return pidGone(int(id))
}
