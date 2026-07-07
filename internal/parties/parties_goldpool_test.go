package parties

import "testing"

// TestParty_GoldPoolAccrue verifies AddGold accumulates into the shared
// party gold pool and is readable back through the registry.
func TestParty_GoldPoolAccrue(t *testing.T) {
	p := New(1)
	if p == nil {
		t.Fatal("New(1) returned nil")
	}
	defer p.Disband()

	p.AddGold(30)
	if got := Get(1).GoldPool; got != 30 {
		t.Fatalf("after AddGold(30): GoldPool = %d, want 30", got)
	}

	p.AddGold(12)
	if got := Get(1).GoldPool; got != 42 {
		t.Fatalf("after AddGold(12): GoldPool = %d, want 42", got)
	}
}
