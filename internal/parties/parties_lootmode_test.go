package parties

import "testing"

// TestParty_LootModeDefaultAndSet verifies that a freshly-created party has
// a default (empty or "ffa") LootMode, and that setting LootMode persists on
// the registry party retrieved via Get().
func TestParty_LootModeDefaultAndSet(t *testing.T) {
	p := New(1)
	if p == nil {
		t.Fatalf("New(1) returned nil")
	}
	defer p.Disband()

	// Default: empty ("" = ffa) or explicitly "ffa".
	if p.LootMode != "" && p.LootMode != "ffa" {
		t.Fatalf("expected default LootMode \"\" or \"ffa\", got %q", p.LootMode)
	}

	p.LootMode = "roundrobin"

	got := Get(1)
	if got == nil {
		t.Fatalf("Get(1) returned nil after New(1)")
	}
	if got.LootMode != "roundrobin" {
		t.Fatalf("expected LootMode \"roundrobin\" on registry party, got %q", got.LootMode)
	}
}
