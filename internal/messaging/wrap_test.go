package messaging

import "testing"

func TestWrapAnsiShortLineUnchanged(t *testing.T) {
	got := WrapAnsi("short", 80)
	if got != "short" {
		t.Fatalf("short line should be unchanged, got %q", got)
	}
}

func TestWrapAnsiWrapsAtMaxWidthWordBoundary(t *testing.T) {
	// 20-col wrap, ~40 chars of content
	got := WrapAnsi("the quick brown fox jumps over the lazy dog", 20)
	// Expect at least one newline, and no line longer than 20 visible chars.
	lines := splitLines(got)
	if len(lines) < 2 {
		t.Fatalf("expected wrap to produce >=2 lines, got: %q", got)
	}
	for i, line := range lines {
		if displayWidth(line) > 20 {
			t.Fatalf("line %d exceeds 20 cols (%d): %q", i, displayWidth(line), line)
		}
	}
}

func TestWrapAnsiIgnoresAnsiTagsInWidth(t *testing.T) {
	// 12 visible chars wrapped inside a long ANSI tag.
	input := `<ansi fg="hit-melee">strikes hard</ansi>`
	got := WrapAnsi(input, 80)
	if got != input {
		t.Fatalf("12-visible-char line wrapped at 80 must be unchanged, got %q", got)
	}
}

func TestWrapAnsiCarriesOpenTagAcrossLineBreak(t *testing.T) {
	// 10-col wrap forces a break inside an open tag.
	input := `<ansi fg="hit-melee">strikes deeply at the heart</ansi>`
	got := WrapAnsi(input, 10)
	// First line should END with </ansi>; second line should START
	// with <ansi fg="hit-melee">.
	lines := splitLines(got)
	if len(lines) < 2 {
		t.Fatalf("expected break, got 1 line: %q", got)
	}
	if !endsWith(lines[0], `</ansi>`) {
		t.Fatalf("first line must close the open tag, got %q", lines[0])
	}
	if !startsWith(lines[1], `<ansi fg="hit-melee">`) {
		t.Fatalf("second line must reopen the tag, got %q", lines[1])
	}
}

func TestWrapAnsiMalformedTagFallback(t *testing.T) {
	// Orphan opening tag — wrapper must not panic; should fall back
	// to byte-count wrap.
	input := `<ansi fg="bad" missing close ` +
		`this is fifty-plus characters of unwrapped text after`
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("WrapAnsi panicked on malformed input: %v", r)
		}
	}()
	_ = WrapAnsi(input, 20)
}

func TestWrapAnsiZeroWidthIsPassthrough(t *testing.T) {
	// LineWidth=0 (unset) must not wrap or hang.
	got := WrapAnsi("a long line that would otherwise wrap", 0)
	if got != "a long line that would otherwise wrap" {
		t.Fatalf("LineWidth=0 must pass through, got %q", got)
	}
}

// Test helpers — kept here, not exported.

func splitLines(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, cur)
	return out
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func displayWidth(s string) int {
	// Inline scan to count visible chars (skip <ansi …> and </ansi>).
	w := 0
	inTag := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '<' {
			inTag = true
			continue
		}
		if c == '>' {
			inTag = false
			continue
		}
		if !inTag {
			w++
		}
	}
	return w
}
