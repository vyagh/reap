package cli

import (
	"fmt"
	"strings"
	"unicode"
)

// pyStr writes s the way Python's repr does: single quotes unless s holds a
// single quote and no double quote, backslash escapes, and \x, \u or \U for
// what is not printable. The self-check prints its values this way.
func pyStr(s string) string {
	quote := '\''
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = '"'
	}
	var b strings.Builder
	b.WriteRune(quote)
	for _, r := range s {
		switch {
		case r == quote || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case unicode.IsPrint(r):
			b.WriteRune(r)
		case r < 0x100:
			fmt.Fprintf(&b, `\x%02x`, r)
		case r < 0x10000:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			fmt.Fprintf(&b, `\U%08x`, r)
		}
	}
	b.WriteRune(quote)
	return b.String()
}

// pyList is a Python list of strings: ['a', 'b'].
func pyList(items []string) string {
	reprs := make([]string, len(items))
	for i, s := range items {
		reprs[i] = pyStr(s)
	}
	return "[" + strings.Join(reprs, ", ") + "]"
}

// pyCounts is a Python dict of counts, keys in the order they first appeared:
// {'a': 2, 'b': 1}.
func pyCounts(keys []string) string {
	var order []string
	count := map[string]int{}
	for _, k := range keys {
		if count[k] == 0 {
			order = append(order, k)
		}
		count[k]++
	}
	pairs := make([]string, len(order))
	for i, k := range order {
		pairs[i] = fmt.Sprintf("%s: %d", pyStr(k), count[k])
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}
