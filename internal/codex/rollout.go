// Package codex reads the Codex CLI's chats and decides whether one is still
// open. A chat is one rollout file under sessions/.
package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vyagh/reap/internal/chat"
)

const (
	scanLines = 400   // lines scan reads, so a long chat still opens fast
	tailBytes = 65536 // how far back from the end lastCwd looks
	wholeFile = -1    // lastCwd limit: search back to the first line
)

// skipPrefixes start the boilerplate turns Codex writes before the user's
// first real prompt (reap:60).
var skipPrefixes = []string{"<", "#", "This session is being continued"}

// meta is what the first line of a rollout says about its chat.
type meta struct {
	ID           string
	Cwd          string
	ThreadSource string
}

// block is one piece of a message's content.
type block struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// message is a response_item record: one turn of the conversation.
type message struct {
	Type    string `json:"type"`
	Payload struct {
		Type    string  `json:"type"`
		Role    string  `json:"role"`
		Content []block `json:"content"`
	} `json:"payload"`
}

// collapse turns every run of whitespace into one space and trims the ends.
func collapse(s string) string {
	return strings.Join(strings.FieldsFunc(s, chat.IsSpace), " ")
}

func skipped(text string) bool {
	for _, p := range skipPrefixes {
		if strings.HasPrefix(text, p) {
			return true
		}
	}
	return false
}

// eachLine calls fn with every line of the file until fn returns false. It
// returns the error that stopped the read, nil at the end of the file.
func eachLine(path string, fn func(line []byte) bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 && !fn(line) {
			return nil
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func textOf(blocks []block, kinds ...string) []string {
	var parts []string
	for _, b := range blocks {
		for _, k := range kinds {
			if b.Type == k {
				parts = append(parts, b.Text)
			}
		}
	}
	return parts
}

// peek returns up to maxTurns turns of a rollout: user and assistant messages,
// whitespace collapsed. A user turn that is boilerplate is left out (reap:354-374).
func peek(path string, maxTurns int) ([]chat.Turn, error) {
	var out []chat.Turn
	err := eachLine(path, func(line []byte) bool {
		var m message
		if json.Unmarshal(line, &m) != nil || m.Type != "response_item" {
			return true
		}
		p := m.Payload
		if p.Type != "message" || p.Role != "user" && p.Role != "assistant" {
			return true
		}
		var parts []string
		for _, s := range textOf(p.Content, "input_text", "output_text", "text") {
			if s != "" {
				parts = append(parts, s)
			}
		}
		text := strings.TrimFunc(strings.Join(parts, " "), chat.IsSpace)
		if text == "" || p.Role == "user" && skipped(text) {
			return true
		}
		who := "codex"
		if p.Role == "user" {
			who = "you"
		}
		out = append(out, chat.Turn{Who: who, Text: chat.Clean(collapse(text))})
		return len(out) < maxTurns
	})
	return out, err
}

// readMeta reads the first line of a rollout. ok is false when that line is
// not a session_meta record with an id (reap:376-386).
func readMeta(path string) (m meta, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return meta{}, false
	}
	defer f.Close()
	line, err := bufio.NewReader(f).ReadBytes('\n')
	if err != nil && err != io.EOF {
		return meta{}, false
	}
	var rec struct {
		Type    string `json:"type"`
		Payload struct {
			ID           string `json:"id"`
			Cwd          string `json:"cwd"`
			ThreadSource string `json:"thread_source"`
		} `json:"payload"`
	}
	if json.Unmarshal(line, &rec) != nil || rec.Type != "session_meta" || rec.Payload.ID == "" {
		return meta{}, false
	}
	return meta(rec.Payload), true
}

// lastCwd is the folder of the newest turn_context record. Codex resumes a
// chat where it was last used, which can differ from the folder in the first
// line. It reads backwards from the end, at most limit bytes (wholeFile for no
// limit), and gives "" when it finds no folder (reap:388-406).
//
// reap:396-404: the first line of each chunk is held back as a possible
// fragment, and at the last chunk it is dropped unless the chunk starts at byte
// 0. So a line that starts exactly at the limit is not seen.
func lastCwd(path string, limit int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return ""
	}
	end := st.Size()
	stop := int64(0)
	if limit != wholeFile {
		stop = max(0, end-limit)
	}
	var rest []byte
	for end > stop {
		start := max(stop, end-tailBytes)
		buf := make([]byte, end-start)
		n, err := f.ReadAt(buf, start)
		if err != nil && err != io.EOF {
			return ""
		}
		lines := bytes.Split(append(buf[:n], rest...), []byte("\n"))
		rest = nil
		if start > 0 {
			rest, lines = lines[0], lines[1:]
		}
		for i := len(lines) - 1; i >= 0; i-- {
			if !bytes.Contains(lines[i], []byte(`"type":"turn_context"`)) {
				continue
			}
			var rec struct {
				Payload struct {
					Cwd string `json:"cwd"`
				} `json:"payload"`
			}
			if json.Unmarshal(lines[i], &rec) != nil {
				continue
			}
			return rec.Payload.Cwd
		}
		end = start
	}
	return ""
}

// scan returns the first real user prompt of a rollout ("" if none) and how
// many real prompts the first 400 lines hold. A file that cannot be read gives
// what was read before the failure (reap:408-428).
func scan(path string) (title string, n int) {
	i := 0
	eachLine(path, func(line []byte) bool {
		i++
		if i > scanLines {
			return false
		}
		if !bytes.Contains(line, []byte(`"role":"user"`)) && !bytes.Contains(line, []byte(`"role": "user"`)) {
			return true
		}
		var m message
		if json.Unmarshal(line, &m) != nil || m.Type != "response_item" || m.Payload.Role != "user" {
			return true
		}
		text := chat.Clean(collapse(strings.Join(textOf(m.Payload.Content, "input_text", "text"), " ")))
		if text == "" || skipped(text) {
			return true
		}
		n++
		if title == "" {
			title = text
		}
		return true
	})
	return title, n
}

// readTitles maps a chat id to its name in session_index.jsonl. When an id
// appears more than once the last line wins (reap:430-440).
func readTitles(codexHome string) map[string]string {
	titles := map[string]string{}
	eachLine(filepath.Join(codexHome, "session_index.jsonl"), func(line []byte) bool {
		var rec struct {
			ID         string `json:"id"`
			ThreadName string `json:"thread_name"`
		}
		if json.Unmarshal(line, &rec) == nil && rec.ID != "" && rec.ThreadName != "" {
			titles[rec.ID] = rec.ThreadName
		}
		return true
	})
	return titles
}

// rollouts lists the rollout files under sessions/<year>/<month>/<day>.
// archived_sessions is not looked at, and neither are names that start with a
// dot (reap:442-443, 473).
func rollouts(codexHome string) ([]string, error) {
	all, err := filepath.Glob(filepath.Join(codexHome, "sessions", "*", "*", "*", "*.jsonl"))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, p := range all {
		if !dotted(p) {
			out = append(out, p)
		}
	}
	return out, nil
}

// dotted reports whether the year, month, day or file part of a rollout path
// starts with a dot, which Python's glob does not match.
func dotted(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, p := range parts[len(parts)-4:] {
		if strings.HasPrefix(p, ".") {
			return true
		}
	}
	return false
}
