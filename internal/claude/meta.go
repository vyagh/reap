package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/vyagh/reap/internal/chat"
)

const (
	headMax  = 4 << 20   // most of the start meta reads
	tailMax  = 64 << 10  // and the end, where late titles land
	wholeMax = 256 << 10 // at or under this a transcript is read whole
)

// info is what the header of a transcript says about its chat.
type info struct {
	title  string // custom title, else agent name, else generated title
	prompt string // the first prompt you typed
	cwd    string // working folder
	n      int    // prompts counted, -1 unless the whole file was read
}

// CwdOf reads the working folder out of one transcript, "" if it records none.
func CwdOf(transcript string) string {
	i, _ := meta(transcript, false)
	return i.cwd
}

// meta reads the title, first prompt and working folder of a transcript. At
// launch it reads the start (until a prompt and a folder are in) and the
// last 64 KB, so a title written only in the middle of a long transcript is
// missed. With full it reads the whole file and counts the prompts by text
// match: a user record that is neither a tool result nor a meta line. A file
// that cannot be opened gives nothing and the error (reap:269-323).
func meta(path string, full bool) (info, error) {
	s := scan{info: info{n: -1}, full: full}
	if full {
		s.n = 0
	}
	f, err := os.Open(path)
	if err != nil {
		return s.info, err
	}
	defer f.Close()
	if full {
		s.readWhole(f)
	} else {
		s.readEnds(f)
	}
	return s.info, nil
}

// scan is a transcript being read: what has been found and whether a prompt
// has been taken, which an empty prompt after cleaning still counts as.
type scan struct {
	info
	full      bool
	hasPrompt bool
}

func (s *scan) readWhole(f *os.File) {
	br := bufio.NewReader(f)
	for {
		line, err := br.ReadString('\n')
		if line != "" {
			s.feed(line)
		}
		if err != nil {
			return
		}
	}
}

// readEnds reads lines from the start while the file is small, or until a
// prompt and a folder are found, then the last tailMax bytes that begin a line.
func (s *scan) readEnds(f *os.File) {
	fi, err := f.Stat()
	if err != nil {
		return
	}
	size := fi.Size()
	br := bufio.NewReader(f)
	var read int64
	for size <= wholeMax || (read < headMax && (s.prompt == "" || s.cwd == "")) {
		line, err := br.ReadBytes('\n')
		read += int64(len(line))
		if len(line) > 0 {
			s.feed(string(line))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
	}
	if read >= size {
		return
	}
	start := max(read, size-tailMax)
	if _, err := f.Seek(start-1, io.SeekStart); err != nil {
		return
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return
	}
	if start > read {
		_, data, _ = bytes.Cut(data, []byte("\n"))
	}
	for _, line := range strings.Split(string(data), "\n") {
		s.feed(line)
	}
}

// feed takes one line. Most lines cannot carry a title, a prompt or a folder,
// and are rejected on a text match before the slow parse.
func (s *scan) feed(line string) {
	isUser := strings.Contains(line, `"type":"user"`)
	if s.full && isUser && !strings.Contains(line, `"tool_result"`) && !strings.Contains(line, `"isMeta":true`) {
		s.n++
	}
	isTitle := strings.Contains(line, `"custom-title"`) || strings.Contains(line, `"agent-name"`) ||
		strings.Contains(line, `"ai-title"`)
	wantCwd := s.cwd == "" && strings.Contains(line, `"cwd"`)
	if !isTitle && !wantCwd && (s.hasPrompt || !isUser) {
		return
	}
	var o map[string]any
	if json.Unmarshal([]byte(line), &o) != nil {
		return
	}
	if s.cwd == "" {
		s.cwd, _ = o["cwd"].(string)
	}
	switch o["type"] {
	case "custom-title":
		if t, _ := o["customTitle"].(string); t != "" {
			s.title = chat.Clean(t)
		}
	case "agent-name":
		if t, _ := o["agentName"].(string); t != "" && s.title == "" {
			s.title = chat.Clean(t)
		}
	case "ai-title":
		if t, _ := o["aiTitle"].(string); t != "" && s.title == "" {
			s.title = chat.Clean(t)
		}
	case "user":
		if !s.hasPrompt && !truthy(o["isMeta"]) {
			s.takePrompt(o)
		}
	}
}

// takePrompt keeps the text of a user record as the prompt unless it is empty
// or one of the command and caveat lines Claude Code writes in the user's name.
func (s *scan) takePrompt(o map[string]any) {
	msg, _ := o["message"].(map[string]any)
	var text string
	switch c := msg["content"].(type) {
	case string:
		text = c
	case []any:
		text = textBlocks(c)
	default:
		return
	}
	text = strings.TrimFunc(text, isSpace)
	if text == "" || hasAnyPrefix(text, "<local-command", "<command-", "Caveat:") {
		return
	}
	s.prompt = chat.Clean(squeeze(text))
	s.hasPrompt = true
}

// textBlocks joins the text blocks of a content list with one space.
func textBlocks(blocks []any) string {
	var texts []string
	for _, b := range blocks {
		if m, ok := b.(map[string]any); ok && m["type"] == "text" {
			t, _ := m["text"].(string)
			texts = append(texts, t)
		}
	}
	return strings.Join(texts, " ")
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// isSpace is Python's str.isspace: unicode.IsSpace plus the separator
// controls 0x1c to 0x1f.
func isSpace(r rune) bool {
	return unicode.IsSpace(r) || '\x1c' <= r && r <= '\x1f'
}

// squeeze turns every run of whitespace into one space and trims the ends.
func squeeze(s string) string {
	return strings.Join(strings.FieldsFunc(s, isSpace), " ")
}
