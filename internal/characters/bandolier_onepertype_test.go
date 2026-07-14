package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// TestStoreItem_AmbientBandolier_OnePerType proves an ambient bandolier (Vitalis)
// accepts one of each potion type but routes a duplicate type to the backpack —
// the anti-immortality cap. A distinct potion type still slots normally.
func TestStoreItem_AmbientBandolier_OnePerType(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999970: {ItemId: 999970, Name: "Vitalis Bandolier", Type: items.Belt,
			IsBandolier: true, BandolierCapacity: 4, AmbientPotions: true},
		999971: {ItemId: 999971, Name: "healing salve", Type: items.Potion, BuffIds: []int{54}, Weight: 0.1},
		999972: {ItemId: 999972, Name: "elixir", Type: items.Potion, BuffIds: []int{55}, Weight: 0.1},
	})()

	c := New()
	c.Stats.Strength.ValueAdj = 100 // ample carry capacity
	c.Equipment.Belt = items.New(999970)

	// First salve routes into the bandolier.
	c.StoreItem(items.New(999971))
	if len(c.PotionItems) != 1 {
		t.Fatalf("first salve should slot in the bandolier; PotionItems=%d", len(c.PotionItems))
	}

	// A DUPLICATE salve must NOT slot — it falls through to the backpack.
	c.StoreItem(items.New(999971))
	if len(c.PotionItems) != 1 {
		t.Fatalf("duplicate salve must not enter the ambient bandolier; PotionItems=%d", len(c.PotionItems))
	}
	if len(c.Items) != 1 {
		t.Fatalf("duplicate salve should land in the backpack; Items=%d", len(c.Items))
	}

	// A DIFFERENT potion type still slots.
	c.StoreItem(items.New(999972))
	if len(c.PotionItems) != 2 {
		t.Fatalf("a distinct potion type should slot; PotionItems=%d", len(c.PotionItems))
	}
}

// TestStoreItem_NonAmbientBandolier_AllowsDuplicates proves an ordinary storage
// bandolier (no ambient ticking) still accepts duplicate potion types — the cap
// only exists to stop passive stacking, which non-ambient belts don't do.
func TestStoreItem_NonAmbientBandolier_AllowsDuplicates(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999973: {ItemId: 999973, Name: "plain bandolier", Type: items.Belt,
			IsBandolier: true, BandolierCapacity: 4, AmbientPotions: false},
		999974: {ItemId: 999974, Name: "healing salve", Type: items.Potion, BuffIds: []int{54}, Weight: 0.1},
	})()

	c := New()
	c.Stats.Strength.ValueAdj = 100
	c.Equipment.Belt = items.New(999973)

	c.StoreItem(items.New(999974))
	c.StoreItem(items.New(999974))
	if len(c.PotionItems) != 2 {
		t.Fatalf("a non-ambient bandolier should allow duplicate potion types; PotionItems=%d", len(c.PotionItems))
	}
}
