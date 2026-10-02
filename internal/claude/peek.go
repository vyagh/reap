package claude

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/vyagh/reap/internal/chat"
)

// peek reads up to limit turns of a transcript, in file order. With textOnly,
// tool calls and results are left out; otherwise they show as [tool: NAME]
// and [tool result]. A transcript that cannot be opened gives the error; one
// that fails part way gives the turns read so far and the error (reap:325-352).
func peek(path string, limit int, textOnly bool) ([]chat.Turn, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var turns []chat.Turn
	br := bufio.NewReader(f)
	for {
		line, err := br.ReadBytes('\n')
		if t, ok := turnOf(line, textOnly); ok {
			turns = append(turns, t)
			if len(turns) >= limit {
				return turns, nil
			}
		}
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return turns, err
		}
	}
}

// turnOf is the turn one transcript line holds, if any. Meta records and the
// command, caveat and reminder text Claude Code writes in the user's name are
// not turns.
func turnOf(line []byte, textOnly bool) (chat.Turn, bool) {
	var o map[string]any
	if json.Unmarshal(line, &o) != nil {
		return chat.Turn{}, false
	}
	who := "claude"
	switch o["type"] {
	case "user":
		who = "you"
	case "assistant":
	default:
		return chat.Turn{}, false
	}
	if truthy(o["isMeta"]) {
		return chat.Turn{}, false
	}
	msg, _ := o["message"].(map[string]any)
	var parts []string
	switch c := msg["content"].(type) {
	case string:
		parts = []string{c}
	case []any:
		parts = blockParts(c, textOnly)
	}
	text := strings.TrimFunc(strings.Join(nonEmpty(parts), " "), isSpace)
	if text == "" || hasAnyPrefix(text, "<local-command", "<command-", "<task-notification", "<system-reminder", "Caveat:") {
		return chat.Turn{}, false
	}
	return chat.Turn{Who: who, Text: chat.Clean(squeeze(text))}, true
}

// blockParts is the text of each block of a content list, and a marker for
// each tool block unless textOnly.
func blockParts(blocks []any, textOnly bool) []string {
	var parts []string
	for _, b := range blocks {
		m, ok := b.(map[string]any)
		if !ok {
			continue
		}
		switch m["type"] {
		case "text":
			t, _ := m["text"].(string)
			parts = append(parts, t)
		case "tool_use":
			if textOnly {
				continue
			}
			name, ok := m["name"].(string)
			if !ok {
				name = "?"
			}
			parts = append(parts, "[tool: "+name+"]")
		case "tool_result":
			if !textOnly {
				parts = append(parts, "[tool result]")
			}
		}
	}
	return parts
}

func nonEmpty(parts []string) []string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
