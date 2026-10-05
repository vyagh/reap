package chat

import (
	"fmt"
	"time"
	"unicode"
)

// Clean drops control characters (everything below 0x20, plus 0x7f) so text from
// a transcript cannot send escape sequences to the terminal.
func Clean(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if c := s[i]; c >= 0x20 && c != 0x7f {
			b = append(b, c)
		}
	}
	return string(b)
}

// IsSpace is Python's str.isspace: unicode.IsSpace plus the separator
// controls 0x1c to 0x1f.
func IsSpace(r rune) bool {
	return unicode.IsSpace(r) || '\x1c' <= r && r <= '\x1f'
}

// Human is a byte count as the list prints it: whole bytes and kilobytes, one
// decimal for megabytes and gigabytes below 10, whole above.
func Human(n int64) string {
	f := float64(n)
	for _, u := range "BKMG" {
		if f < 1024 {
			if (u == 'M' || u == 'G') && f < 10 {
				return fmt.Sprintf("%.1f%c", f, u)
			}
			return fmt.Sprintf("%.0f%c", f, u)
		}
		f /= 1024
	}
	return fmt.Sprintf("%.0fT", f)
}

// Reltime is how long ago t was, in whole minutes, hours, days or 30-day
// months. A time in the future counts as now.
func Reltime(t, now time.Time) string {
	d := max(0, now.Sub(t))
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", d/time.Minute)
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", d/time.Hour)
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd", d/(24*time.Hour))
	}
	return fmt.Sprintf("%dmo", d/(30*24*time.Hour))
}

// Plural is "1 chat" or "3 chats".
func Plural(n int) string {
	return PluralWord(n, "chat")
}

func PluralWord(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

// ProjLabel shortens a project name to w characters by cutting its start and
// putting … there. A w of 1 or less gives … plus the whole name or its tail from
// the second character on, as 0.5.0 does with a negative slice.
func ProjLabel(p string, w int) string {
	r := []rune(p)
	if len(r) <= w {
		return p
	}
	start := len(r) - (w - 1)
	if w <= 1 {
		start = min(1-w, len(r))
	}
	return "…" + string(r[start:])
}

func Fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}
