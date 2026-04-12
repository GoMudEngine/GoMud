package textutil

import (
	"regexp"
	"strings"
)

// TokenContext holds actor names for substitution in YAML text fields.
type TokenContext struct {
	SourceName      string // ANSI-tagged display name
	SourcePlainName string // Plain name (for possessives)
	TargetName      string // ANSI-tagged display name (empty if no target)
	TargetPlainName string // Plain name (empty if no target)
}

// SubstituteTokens replaces known tokens in text with values from ctx.
// Unknown tokens are left as-is. Empty string input returns empty string.
func SubstituteTokens(text string, ctx TokenContext) string {
	if text == "" {
		return ""
	}
	r := strings.NewReplacer(
		`{source}`, ctx.SourceName,
		`{target}`, ctx.TargetName,
		`{source_plain}`, ctx.SourcePlainName,
		`{target_plain}`, ctx.TargetPlainName,
	)
	return r.Replace(text)
}

var tokenPattern = regexp.MustCompile(`\{[a-z_]+\}`)

var knownTokens = map[string]bool{
	`{source}`:       true,
	`{target}`:       true,
	`{source_plain}`: true,
	`{target_plain}`: true,
}

// ValidateTokens scans text for {token} patterns and returns warnings
// for any that are not in the known set.
func ValidateTokens(text string) []string {
	if text == "" {
		return nil
	}
	var warnings []string
	matches := tokenPattern.FindAllString(text, -1)
	for _, m := range matches {
		if !knownTokens[m] {
			warnings = append(warnings, "unknown token: "+m)
		}
	}
	return warnings
}
