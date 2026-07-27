package dialogue

import "testing"

func TestPickGreeting(t *testing.T) {
	gs := []Greeting{
		{Text: "grumpy welcome", Moods: []string{"grumpy"}},
		{Text: "friendly welcome", Moods: []string{"friendly", "cheerful"}},
		{Text: "plain welcome"}, // untagged — the unconditional fallback
	}

	// Mood match wins.
	if text, ok := PickGreeting(gs, "friendly"); !ok || text != "friendly welcome" {
		t.Errorf("friendly: got %q ok=%v", text, ok)
	}
	// Unknown mood falls back to the untagged line.
	if text, ok := PickGreeting(gs, "melancholy"); !ok || text != "plain welcome" {
		t.Errorf("fallback: got %q ok=%v", text, ok)
	}
	// All tagged, none matching, no untagged -> say nothing rather than
	// deliver a line written for a mood the NPC is not in.
	tagged := []Greeting{{Text: "x", Moods: []string{"grumpy"}}}
	if _, ok := PickGreeting(tagged, "friendly"); ok {
		t.Error("no matching mood and no untagged line must yield no greeting")
	}
	// Empty list.
	if _, ok := PickGreeting(nil, "friendly"); ok {
		t.Error("nil greetings must yield no greeting")
	}
}

func TestGreetedOncePerInstance(t *testing.T) {
	const userId = 777001
	if HasGreeted(555001, userId) {
		t.Fatal("fresh memory must not read as greeted")
	}
	MarkGreeted(555001, userId)
	if !HasGreeted(555001, userId) {
		t.Error("marked instance must read as greeted")
	}
	// A respawned mob has a NEW instance id — it greets again by design
	// (spec §3): the in-process memory is keyed per instance.
	if HasGreeted(555002, userId) {
		t.Error("a different instance id must greet afresh")
	}
	// A different player is greeted independently.
	if HasGreeted(555001, 777002) {
		t.Error("another player must be greeted independently")
	}
}
