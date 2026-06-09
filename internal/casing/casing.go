// Package casing is the single source of truth for player-visible display
// casing. Title applies smart title case to a name or title string.
//
// It MUST only be used on player-visible name/title strings — never on lookup
// keys, noun tags, filenames, ids, or anything used for parsing/matching.
package casing

import (
	"strings"
	"unicode"
)

// minorWords are lower-cased when they appear as an interior word (not the
// first or last word). Title-case convention for articles, coordinating
// conjunctions, and short prepositions.
var minorWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "as": {}, "at": {}, "but": {}, "by": {},
	"for": {}, "from": {}, "in": {}, "nor": {}, "of": {}, "on": {}, "or": {},
	"the": {}, "to": {}, "with": {},
}

// Title returns s in smart title case. Each whitespace-separated word's first
// rune is upper-cased, except interior minor words which are lower-cased. The
// first and last words are always capitalized. Characters after the first rune
// of a word are left untouched, so existing internal capitals (e.g. "McGregor",
// "Olen") are preserved and the function is idempotent on canonical input.
func Title(s string) string {
	words := strings.Fields(s) // trims + collapses whitespace
	last := len(words) - 1
	for i, w := range words {
		lower := strings.ToLower(w)
		if i != 0 && i != last {
			if _, isMinor := minorWords[lower]; isMinor {
				words[i] = lower
				continue
			}
		}
		words[i] = upperFirst(w)
	}
	return strings.Join(words, " ")
}

// upperFirst upper-cases the first rune of w, leaving the remainder unchanged.
func upperFirst(w string) string {
	if w == "" {
		return w
	}
	r := []rune(w)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
