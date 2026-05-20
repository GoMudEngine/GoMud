package messaging

import "regexp"

// nameTagPattern matches <ansi fg="username|mobname|petname">…</ansi>
// for the anonymizer pass.
var nameTagPattern = regexp.MustCompile(
	`<ansi fg="(username|mobname|petname)">[^<]+</ansi>`,
)

// Anonymize strips player/mob/pet name ANSI tags and replaces them
// with a `combat-anon`-colored "a figure" placeholder. Used by the
// pipeline for infrared-only observers in dark rooms.
//
// v1 limitation: bare-name occurrences (names embedded in prose
// without an ANSI tag) leak through. The 228-site audit gets most
// names properly tagged; remaining leaks are tracked as followups.
func Anonymize(text string) string {
	if text == "" {
		return text
	}
	return nameTagPattern.ReplaceAllString(text,
		`<ansi fg="combat-anon">a figure</ansi>`)
}
