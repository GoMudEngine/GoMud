package playtestprofiles

import (
	"fmt"
	"strings"
	"unicode"
)

// ForbiddenIdentity reports whether name matches a checked-in prod-identity
// stem or a close variant (spec 0.3d v1 algorithm). Empty/whitespace names
// are allowed (caller validates required fields separately).
func ForbiddenIdentity(name string) error {
	cand := normalizeIdentity(name)
	if cand == "" {
		return nil
	}
	for _, stem := range prodIdentityDenylistStems {
		if stem == "" {
			continue
		}
		if identityMatchesStem(cand, stem) {
			return fmt.Errorf("playtestprofiles: forbidden prod identity %q", strings.TrimSpace(name))
		}
	}
	return nil
}

func identityMatchesStem(cand, stem string) bool {
	if cand == stem {
		return true
	}
	if cand == "pt"+stem {
		return true
	}
	if stripLeadingTrailingDigits(cand) == stem {
		return true
	}
	if len(stem) >= 4 && levenshteinDistance(cand, stem) <= 1 {
		return true
	}
	if len(stem) >= 5 && strings.Contains(cand, stem) {
		return true
	}
	return false
}

func normalizeIdentity(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r == '_' || r == '-' || r == ' ':
			continue
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

func stripLeadingTrailingDigits(s string) string {
	i, j := 0, len(s)
	for i < j && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	for j > i && s[j-1] >= '0' && s[j-1] <= '9' {
		j--
	}
	return s[i:j]
}

func levenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	// Two-row DP.
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := cur[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost
			cur[j] = min(ins, del, sub)
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}
