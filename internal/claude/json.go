// Package claude reads Claude Code's chats and decides which of them are
// still open in a running session.
package claude

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"time"
	"unicode/utf8"
)

const (
	jsonTries = 3
	jsonWait  = 30 * time.Millisecond
)

// readJSON reads a JSON file that may be caught mid-write, trying up to three
// times with a short wait between them. A missing file returns fs.ErrNotExist
// at once. Numbers come back as json.Number so an integer and a float stay
// apart (reap:160-169).
func readJSON(path string) (any, error) {
	var err error
	for i := range jsonTries {
		var data []byte
		data, err = os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		if err == nil {
			var v any
			if v, err = parseJSON(data); err == nil {
				return v, nil
			}
		}
		if i+1 < jsonTries {
			time.Sleep(jsonWait)
		}
	}
	return nil, err
}

// parseJSON decodes one JSON value. Bytes that are not UTF-8 are an error, as
// they are for Python, not replaced.
func parseJSON(data []byte) (any, error) {
	if !utf8.Valid(data) {
		return nil, errors.New("not valid UTF-8")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("data after the JSON value")
	}
	return v, nil
}

// truthy is Python's truth test on a decoded JSON value: null, false, 0, ""
// and empty lists and objects are false.
func truthy(v any) bool {
	switch v := v.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case float64:
		return v != 0
	case json.Number:
		f, err := v.Float64()
		return err != nil || f != 0
	case []any:
		return len(v) > 0
	case map[string]any:
		return len(v) > 0
	}
	return true
}
