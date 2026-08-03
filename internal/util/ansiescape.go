package util

import "strings"

// EscapeAnsiTags neutralises `<ansi>` markup in untrusted text.
//
// Player-supplied strings are interpolated into `<ansi fg="...">%s</ansi>`
// templates all over the comms commands, and internal/hooks runs
// templates.AnsiParse over the assembled message for every recipient. The input
// cleanser (internal/inputhandlers/cleanser.go) strips only non-printable
// runes, so a raw ESC is blocked but every character in `</ansi><ansi fg="red">`
// is printable and passes through untouched. Without escaping, a player can
// close the surrounding tag and open their own, producing output in other
// players' clients that is indistinguishable from a server or admin message —
// a credible fake "session expired, re-enter your password" prompt — or flood
// colour and padding to make the feed unreadable.
//
// The parser (GoMudEngine/ansitags) recognises exactly two constructs, both
// introduced by '<': an opening `<ansi ...>` and a closing `</ansi>`. Nothing
// else in the input is markup. So the whole attack surface is closed by making
// sure no '<' in untrusted text is ever directly followed by `ansi` or
// `/ansi`: a single space is inserted after such a '<', which makes the
// matcher fail and emit the text literally. The player sees `< ansi fg="red">`
// — visibly inert, and obvious to a moderator reading a log.
//
// Everything else is left alone. Players legitimately type '<' ("<3", "a < b",
// "<--"), and stripping or entity-encoding all of them would be lossy for no
// security gain.
//
// The function is idempotent: after one pass every '<' is followed by a space,
// so a second pass changes nothing. That matters because some of these strings
// are persisted (guild MOTD, character description, mail bodies) and are
// escaped on write as well as on render.
//
// Escape the player-supplied SUBSTRING, never the assembled message —
// server-authored markup (dialogue YAML, emote aliases, merchant lines) is
// legitimate and must keep working.
func EscapeAnsiTags(s string) string {
	if !strings.ContainsRune(s, '<') {
		return s
	}

	var b strings.Builder
	b.Grow(len(s) + 8)

	for i := 0; i < len(s); i++ {
		c := s[i]
		b.WriteByte(c)
		if c != '<' {
			continue
		}
		// Case-insensitive on purpose. The parser is currently byte-exact and
		// lowercase, so `</ANSI>` is already inert, but relying on that would
		// make this fix silently regress if the parser ever gained
		// case-insensitive matching.
		if hasFoldPrefix(s[i+1:], "ansi") || hasFoldPrefix(s[i+1:], "/ansi") {
			b.WriteByte(' ')
		}
	}

	return b.String()
}

// hasFoldPrefix is an ASCII-only case-insensitive strings.HasPrefix. The tag
// vocabulary is ASCII, so no Unicode folding is needed and this avoids
// allocating a lowercased copy of every message.
func hasFoldPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != prefix[i] {
			return false
		}
	}
	return true
}
