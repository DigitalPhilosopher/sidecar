package gitstatus

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// NormalizeCommitMessage standardizes a commit message by trimming whitespace,
// capitalizing the subject line, removing trailing periods, stripping trailing
// blank lines, and truncating subjects over 72 characters.
func NormalizeCommitMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return msg
	}

	lines := strings.Split(msg, "\n")

	// Normalize subject line (first line)
	subject := strings.TrimSpace(lines[0])
	subject = capitalizeFirst(subject)
	subject = strings.TrimRight(subject, ".")
	subject = truncateSubject(subject, 72)
	lines[0] = subject

	// Strip trailing blank lines
	for len(lines) > 1 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

func truncateSubject(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max-1]) + "…"
}
