package messaging

import "testing"

func TestAnonymizeReplacesUsernameTag(t *testing.T) {
	in := `<ansi fg="username">Calabe</ansi> attacks`
	want := `<ansi fg="combat-anon">a figure</ansi> attacks`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeReplacesMobnameTag(t *testing.T) {
	in := `<ansi fg="mobname">Thornwall Thug</ansi> snarls`
	want := `<ansi fg="combat-anon">a figure</ansi> snarls`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeReplacesPetnameTag(t *testing.T) {
	in := `<ansi fg="petname">Rex</ansi> follows`
	want := `<ansi fg="combat-anon">a figure</ansi> follows`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeReplacesMultipleNamesInOneLine(t *testing.T) {
	in := `<ansi fg="mobname">Thug</ansi> strikes ` +
		`<ansi fg="username">Calabe</ansi> with a longsword`
	want := `<ansi fg="combat-anon">a figure</ansi> strikes ` +
		`<ansi fg="combat-anon">a figure</ansi> with a longsword`
	if got := Anonymize(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnonymizeLeavesOtherTagsAlone(t *testing.T) {
	in := `<ansi fg="hit-melee">strikes deeply</ansi>`
	if got := Anonymize(in); got != in {
		t.Fatalf("non-name tag must pass through, got %q", got)
	}
}

func TestAnonymizeEmpty(t *testing.T) {
	if got := Anonymize(""); got != "" {
		t.Fatalf("empty must pass through, got %q", got)
	}
}
