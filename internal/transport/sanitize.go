package transport

import (
	"strings"
	"unicode"
)

// SanitizePlayerName rejects names that could inject CRLF/@event lines into other clients.
func SanitizePlayerName(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 32 {
		return "", false
	}
	for _, r := range name {
		switch {
		case r >= '0' && r <= '9', r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r == '-', r == '_':
		default:
			return "", false
		}
	}
	return name, true
}

// SanitizePlayerText strips control chars and newlines from player-authored prose.
func SanitizePlayerText(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\r' || r == '\n' {
			b.WriteByte(' ')
			continue
		}
		if r >= 0x20 && r <= 0x7e && !unicode.IsControl(r) {
			b.WriteRune(r)
		} else if r == '\t' {
			b.WriteByte(' ')
		}
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	if len(out) > 500 {
		out = out[:500]
	}
	return strings.TrimSpace(out)
}
