package util

import (
	"fmt"
	"strings"
	"testing"
)

func TestSplitString_AnsiClosingTag(t *testing.T) {
	input := fmt.Sprintf("<ansi fg=\"mobname\">Sylara</ansi> says, \"<ansi fg=\"saytext-mob\">Search the bone pile and bring it back. Then I will know you are sincere.'</ansi>\"")

	lines := SplitString(input, 80)
	joined := strings.Join(lines, "\n")

	// The closing tag must not be split across lines
	if strings.Contains(joined, "<\n") || strings.Contains(joined, "\nansi>") {
		t.Errorf("ANSI closing tag was split across lines:\n%s", joined)
	}

	// Verify no raw "ansi>" appears as visible text (tag was destroyed)
	for i, line := range lines {
		vis := ansiTagRegex.ReplaceAllString(line, "")
		if strings.Contains(vis, "ansi>") {
			t.Errorf("Line %d has broken ANSI tag in visible text: %q", i, vis)
		}
	}
}
