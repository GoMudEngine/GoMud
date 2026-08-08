package messaging

import "testing"

// Finding 32: WrapAnsi promised in its doc comment that a parser panic returns
// the original text, but it used an unnamed result with a bare recover(), so
// the caller silently got "" and the message was erased. These lock the
// contract: WrapAnsi must never return empty for non-empty input.

func TestWrapAnsi_NeverReturnsEmptyForNonEmptyInput(t *testing.T) {
	// Malformed / adversarial ANSI that exercises the orphan-tag and
	// unmatched-closer paths the doc comment calls out.
	inputs := []string{
		`<ansi fg="red">unclosed tag runs off the end of the line`,
		`</ansi>orphan closer with no opener at all`,
		`<ansi fg="red"><ansi fg="blue">nested without closing</ansi>`,
		`<ansi`,
		`<ansi fg=">">weird quoting</ansi>`,
		`plain text with no tags whatsoever`,
		`<ansi fg="red"></ansi>`,
	}

	for _, in := range inputs {
		got := WrapAnsi(in, 20)
		if got == "" {
			t.Errorf("WrapAnsi(%q, 20) returned empty string; contract is to fall back to the original text", in)
		}
	}
}

func TestWrapAnsi_ZeroWidthReturnsInputUnchanged(t *testing.T) {
	const in = `<ansi fg="red">hello</ansi>`
	if got := WrapAnsi(in, 0); got != in {
		t.Errorf("WrapAnsi(%q, 0) = %q, want the input unchanged", in, got)
	}
}

func TestWrapAnsi_EmptyInputStaysEmpty(t *testing.T) {
	if got := WrapAnsi("", 20); got != "" {
		t.Errorf(`WrapAnsi("", 20) = %q, want ""`, got)
	}
}
