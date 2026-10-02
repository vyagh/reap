package chat

// Clean drops control characters (escape, bell, newline, tab and the rest
// below 0x20, plus 0x7f), so text from a transcript cannot send escape
// sequences to the terminal when printed (reap:154-158).
func Clean(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if c := s[i]; c >= 0x20 && c != 0x7f {
			b = append(b, c)
		}
	}
	return string(b)
}
