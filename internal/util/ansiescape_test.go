package util

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/ansitags"
)

// escapeCount counts ANSI escape sequences produced by the real parser. If a
// player-supplied string contributes any, the injection worked.
func escapeCount(s string) int {
	return strings.Count(ansitags.Parse(s), "\x1b")
}

func TestEscapeAnsiTagsNeutralisesTags(t *testing.T) {
	payloads := []string{
		`</ansi><ansi fg="red">FAKE ADMIN`,
		`<ansi fg="red">red</ansi>`,
		`</ansi>`,
		`<ansi>`,
		`<ansi fg="red"><ansi fg="blue"><ansi fg="green">flood`,
		`</ANSI><ANSI fg="red">upper case`,
		`</AnSi><aNsI fg="red">mixed case`,
		`<<ansi fg="red">doubled bracket`,
		`text <ansi fg="red"> in the middle </ansi> of a line`,
	}

	for _, p := range payloads {
		t.Run(p, func(t *testing.T) {
			escaped := EscapeAnsiTags(p)

			if got := escapeCount(escaped); got != 0 {
				t.Fatalf("escaped payload still produced %d ANSI escape sequence(s): %q -> %q", got, p, ansitags.Parse(escaped))
			}
			if strings.Contains(strings.ToLower(escaped), `<ansi`) {
				t.Fatalf("escaped payload still contains a tag opener: %q", escaped)
			}
			if strings.Contains(strings.ToLower(escaped), `</ansi`) {
				t.Fatalf("escaped payload still contains a tag closer: %q", escaped)
			}
		})
	}
}

// The control: without escaping these payloads really do emit ANSI. Without
// this the test above could pass against a parser that never emits anything.
func TestEscapeAnsiTagsControlUnescapedPayloadDoesInject(t *testing.T) {
	if got := escapeCount(`<ansi fg="red">red</ansi>`); got == 0 {
		t.Fatalf("control failed: an unescaped tag produced no ANSI output, so the test above proves nothing")
	}
}

// Escaping must be safe to apply repeatedly: several of these strings are
// persisted and escaped both on write and on render.
func TestEscapeAnsiTagsIsIdempotent(t *testing.T) {
	inputs := []string{
		`</ansi><ansi fg="red">x`,
		`plain text`,
		`a < b`,
		`<ansi`,
		`<`,
	}

	for _, in := range inputs {
		once := EscapeAnsiTags(in)
		twice := EscapeAnsiTags(once)
		if once != twice {
			t.Fatalf("not idempotent for %q: once=%q twice=%q", in, once, twice)
		}
	}
}

// Players legitimately type '<'. Only the two sequences the parser actually
// recognises may be altered.
func TestEscapeAnsiTagsLeavesInnocentTextAlone(t *testing.T) {
	unchanged := []string{
		``,
		`hello world`,
		`<3 you`,
		`a < b > c`,
		`<-- look over there`,
		`x</b>y`,
		`<answer>`, // starts with "ans" but not "ansi"
		`<an`,
		`</an`,
		`</ansX>`,
	}

	for _, in := range unchanged {
		if got := EscapeAnsiTags(in); got != in {
			t.Fatalf("EscapeAnsiTags(%q) = %q, want it unchanged", in, got)
		}
	}
}

// Escaping must not destroy the message — a moderator reading a log needs to
// see what the player actually typed.
func TestEscapeAnsiTagsPreservesVisibleText(t *testing.T) {
	escaped := EscapeAnsiTags(`</ansi><ansi fg="red">SESSION EXPIRED, RE-ENTER PASSWORD`)

	if !strings.Contains(escaped, `SESSION EXPIRED, RE-ENTER PASSWORD`) {
		t.Fatalf("visible text was lost: %q", escaped)
	}
	// The tag text is still readable, just inert.
	if !strings.Contains(ansitags.Parse(escaped), `fg="red"`) {
		t.Fatalf("tag text should survive as literal characters: %q", ansitags.Parse(escaped))
	}
}
