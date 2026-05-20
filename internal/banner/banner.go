// Package banner formats the SKILL ADVANCEMENT / STATISTIC INCREASED
// banner used by the progression system. Lives as a leaf package
// (no imports of characters, users, or messaging) so that
// internal/characters/ can import it without creating the
// messaging→characters cycle.
//
// internal/messaging/progression.go re-exports a SendProgression
// helper that delegates to Format() and ships to a UserSender for
// callers that already have one. characters/progression.go (where
// the legacy `*** sharpening ***` literals lived) calls Format()
// and queues the banner directly via events.AddToQueue.
package banner

import (
	"strings"
	"unicode/utf8"
)

// Kind discriminates skill vs stat advancement banners.
type Kind int

const (
	Skill Kind = iota
	Stat
)

// TierChange marks a tier crossing (e.g., apprentice → journeyman).
// nil = no tier change; banner omits the transition line.
type TierChange struct {
	From string
	To   string
}

const rule = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

// Format returns the banner string (no trailing newline). Three lines
// when tier is nil (rule / title / name / rule); four when a tier
// change is supplied.
func Format(kind Kind, name string, tier *TierChange) string {
	var title string
	switch kind {
	case Skill:
		title = "SKILL ADVANCEMENT"
	case Stat:
		title = "STATISTIC INCREASED"
	default:
		title = "PROGRESSION"
	}

	lines := []string{
		rule,
		center(title),
		center(name),
	}
	if tier != nil {
		lines = append(lines, center(tier.From+" → "+tier.To))
	}
	lines = append(lines, rule)
	return strings.Join(lines, "\n")
}

// center pads s with leading spaces to roughly center it inside the
// banner rule width.
//
// Uses rune count (not byte count) for both s and rule because the
// rule is composed of `━` (3-byte UTF-8 chars) — len() would return
// 192 instead of 64 and the centered text would be pushed far to
// the right of the rule.
func center(s string) string {
	width := utf8.RuneCountInString(rule)
	sWidth := utf8.RuneCountInString(s)
	if sWidth >= width {
		return s
	}
	pad := (width - sWidth) / 2
	return strings.Repeat(" ", pad) + s
}
