package shops_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/shops"
)

func TestEffectiveMaxStock_UntieredItem(t *testing.T) {
	// Item with no RarityTier (most items currently) returns 0.
	// Passing 40031 (spirit fetish, quest item) — no rarity_tier set.
	if got := shops.EffectiveMaxStock(40031, 1.0); got != 0 {
		t.Errorf("untiered quest item: got %d, want 0", got)
	}
}

func TestEffectiveMaxStock_UnknownItem(t *testing.T) {
	if got := shops.EffectiveMaxStock(99999, 1.0); got != 0 {
		t.Errorf("unknown item: got %d, want 0", got)
	}
}

func TestEffectiveMaxStock_ZeroMultiplier(t *testing.T) {
	// stockMultiplier == 0 (unset mob field) treated as 1.0 — should
	// still return 0 since no items have rarity_tier set yet.
	if got := shops.EffectiveMaxStock(40001, 0); got != 0 {
		t.Errorf("zero multiplier: got %d, want 0", got)
	}
}

// Note: tests that exercise actual rarity_tier x multiplier math
// would require items on disk to have rarity_tier set. Those are
// added in Task 12. For now, this task verifies the contract:
// untiered/unknown/zero-mult all return 0, the helper behaves
// correctly without exploding.
