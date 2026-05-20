package messaging

import "testing"

// RenderForRecipient is the per-recipient pipeline entry point used
// internally by the Room/UserRecord Send helpers. Returns the final
// text to deliver. An empty return string means "don't deliver to
// this recipient" (used by the sight gate).
func TestRenderForRecipientStubReturnsTextUnchanged(t *testing.T) {
	got := RenderForRecipient(RenderInput{
		Category:  CategoryDefault,
		Text:      "Hello, world.",
		Channel:   ChannelAudio,
		LineWidth: 80,
	})
	if got != "Hello, world." {
		t.Fatalf("stub pipeline mutated text: got %q", got)
	}
}

func TestChannelConstants(t *testing.T) {
	if ChannelAudio == ChannelVisual {
		t.Fatal("ChannelAudio and ChannelVisual must differ")
	}
}

func TestApplyCategoryColorWrapsTagForKnownCategory(t *testing.T) {
	got := applyCategoryColor(CategoryHitMelee, "strikes deeply")
	want := `<ansi fg="hit-melee">strikes deeply</ansi>`
	if got != want {
		t.Fatalf("color wrap: got %q want %q", got, want)
	}
}

func TestApplyCategoryColorDefaultPassesThrough(t *testing.T) {
	got := applyCategoryColor(CategoryDefault, "plain text")
	if got != "plain text" {
		t.Fatalf("CategoryDefault must pass text through unchanged, got %q", got)
	}
}

func TestApplyCategoryColorEmptyTextPassesThrough(t *testing.T) {
	got := applyCategoryColor(CategoryHitMelee, "")
	if got != "" {
		t.Fatalf("empty text must pass through unchanged, got %q", got)
	}
}
