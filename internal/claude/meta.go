package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"

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

func CwdOf(transcript string) string {
	i, _ := meta(transcript, false)
	return i.cwd
}

// meta reads the title, first prompt and working folder of a transcript. At
// launch it reads the start (until a prompt and a folder are in) and the last
// 64 KB, so a title written only in the middle of a long transcript is missed.
// With full it reads the whole file and counts the prompts by text match: a user
// record that is neither a tool result nor a meta line.
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

// scan is a transcript being read. A prompt that is empty once cleaned still
// counts as taken.
type scan struct {
	info
	full      bool
	hasPrompt bool
}

// readers pools the 64 KB buffers so a listing does not make one per transcript.
var readers = sync.Pool{New: func() any { return bufio.NewReaderSize(nil, 64<<10) }}

func (s *scan) readWhole(f *os.File) {
	br := readers.Get().(*bufio.Reader)
	defer readers.Put(br)
	br.Reset(f)
	var long []byte
	for {
		line, err := readLine(br, &long)
		if len(line) > 0 {
			s.feed(line)
		}
		if err != nil {
			return
		}
	}
}

// readLine is the next line with its newline, good until the next call. A line
// longer than the reader's buffer is gathered in long, which is reused.
func readLine(br *bufio.Reader, long *[]byte) ([]byte, error) {
	line, err := br.ReadSlice('\n')
	if err != bufio.ErrBufferFull {
		return line, err
	}
	*long = append((*long)[:0], line...)
	for err == bufio.ErrBufferFull {
		line, err = br.ReadSlice('\n')
		*long = append(*long, line...)
	}
	return *long, err
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
	var long []byte
	var read int64
	for size <= wholeMax || (read < headMax && (s.prompt == "" || s.cwd == "")) {
		line, err := readLine(br, &long)
		read += int64(len(line))
		if len(line) > 0 {
			s.feed(line)
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
	br = readers.Get().(*bufio.Reader)
	defer readers.Put(br)
	br.Reset(f)
	if start > read {
		if _, err := readLine(br, &long); err != nil {
			return
		}
	}
	for {
		line, err := readLine(br, &long)
		if len(line) > 0 {
			s.feed(line)
		}
		if err != nil {
			return
		}
	}
}

// feed takes one line. Most lines cannot carry a title, a prompt or a folder and
// are rejected on a text match before the slow parse.
func (s *scan) feed(line []byte) {
	isUser := bytes.Contains(line, []byte(`"type":"user"`))
	if s.full && isUser && !bytes.Contains(line, []byte(`"tool_result"`)) && !bytes.Contains(line, []byte(`"isMeta":true`)) {
		s.n++
	}
	isTitle := bytes.Contains(line, []byte(`"custom-title"`)) || bytes.Contains(line, []byte(`"agent-name"`)) ||
		bytes.Contains(line, []byte(`"ai-title"`))
	wantCwd := s.cwd == "" && bytes.Contains(line, []byte(`"cwd"`))
	if !isTitle && !wantCwd && (s.hasPrompt || !isUser) {
		return
	}
	var o map[string]any
	if json.Unmarshal(line, &o) != nil {
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
	text = strings.TrimFunc(text, chat.IsSpace)
	if text == "" || hasAnyPrefix(text, "<local-command", "<command-", "Caveat:") {
		return
	}
	s.prompt = chat.Clean(squeeze(text))
	s.hasPrompt = true
}

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

func squeeze(s string) string {
	return strings.Join(strings.FieldsFunc(s, chat.IsSpace), " ")
}
