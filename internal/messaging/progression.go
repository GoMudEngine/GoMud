package messaging

import (
	"strings"
)

// ProgressionKind discriminates skill vs stat advancement.
type ProgressionKind int

const (
	ProgSkill ProgressionKind = iota
	ProgStat
)

// TierChange marks a tier crossing (e.g., apprentice→journeyman).
// nil = no tier change; banner omits the third line.
type TierChange struct {
	From string
	To   string
}

const progressionRule = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

// FormatProgression returns the banner string (without trailing
// newline). SendProgression wraps this for the user's connection.
func FormatProgression(kind ProgressionKind, name string, tier *TierChange) string {
	var title string
	switch kind {
	case ProgSkill:
		title = "SKILL ADVANCEMENT"
	case ProgStat:
		title = "STATISTIC INCREASED"
	default:
		title = "PROGRESSION"
	}

	lines := []string{
		progressionRule,
		center(title),
		center(name),
	}
	if tier != nil {
		lines = append(lines, center(tier.From+" → "+tier.To))
	}
	lines = append(lines, progressionRule)
	return strings.Join(lines, "\n")
}

// center pads s to the rule width with leading spaces.
func center(s string) string {
	width := len(progressionRule)
	if len(s) >= width {
		return s
	}
	pad := (width - len(s)) / 2
	return strings.Repeat(" ", pad) + s
}

// SendProgression emits the banner to the user via SendText
// (audio channel — banners are not sight-gated). Uses
// CategorySkillProgress. The user-facing literal numbers in the
// banner (none currently — tier names only) are an exception to the
// "no hard numbers" rule because this IS the mechanical display.
//
// UserSender is the minimal interface SendProgression needs; the
// real *users.UserRecord satisfies it. Decoupled so internal/messaging/
// does not import internal/users/ (the audit's import direction).
//
// Note: post-T9 the UserSender.SendText signature will gain a leading
// Category param. T9 updates this signature in the same commit it
// updates everyone else's. For now the interface matches the existing
// pre-T9 UserRecord.SendText shape.
func SendProgression(user UserSender, kind ProgressionKind, name string, tier *TierChange) {
	if user == nil {
		return
	}
	user.SendText(FormatProgression(kind, name, tier))
}

// UserSender is the minimal interface SendProgression needs. The
// real *users.UserRecord satisfies it. Decoupled so messaging/ does
// not import users/ (the audit's import direction).
type UserSender interface {
	SendText(text string)
}
