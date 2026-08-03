package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// FindInBackpackWhere backs the type-aware wear/eat/drink dispatch: two items
// share a noun ("stillwater"), only one is wearable — the filtered pass must
// land on the wearable pendant that the plain matcher used to skip in favor
// of the raw pearl (2026-04-25 quester9 smoke finding).

const (
	iwPearlId   = 91301
	iwPendantId = 91302
)

func seedInventoryWhereFixture(t *testing.T) (*Character, func()) {
	t.Helper()
	cleanup := items.SeedItemsForTest(map[int]*items.ItemSpec{
		iwPearlId: {
			ItemId: iwPearlId, Name: "Stillwater black pearl",
			Type: items.Object, Subtype: items.Mundane,
		},
		iwPendantId: {
			ItemId: iwPendantId, Name: "Stillwater pearl pendant",
			Type: items.Neck, Subtype: items.Wearable,
		},
	})
	c := New()
	// Pearl first so the unfiltered matcher hits it before the pendant.
	c.Items = []items.Item{
		{ItemId: iwPearlId},
		{ItemId: iwPendantId},
	}
	return c, cleanup
}

func TestFindInBackpackWhere_PrefersFilteredMatch(t *testing.T) {
	c, cleanup := seedInventoryWhereFixture(t)
	defer cleanup()

	// Unfiltered baseline: the pearl wins the noun.
	plain, found := c.FindInBackpack("stillwater")
	if !found || plain.ItemId != iwPearlId {
		t.Fatalf("fixture expectation: unfiltered match should be the pearl, got %d", plain.ItemId)
	}

	// Wearable-filtered: the pendant wins.
	wearable, found := c.FindInBackpackWhere("stillwater", func(it items.Item) bool {
		spec := it.GetSpec()
		return spec.Type == items.Weapon || spec.Subtype == items.Wearable
	})
	if !found || wearable.ItemId != iwPendantId {
		t.Fatalf("filtered match should be the pendant, got found=%v id=%d", found, wearable.ItemId)
	}
}

func TestFindInBackpackWhere_MissAllowsFallback(t *testing.T) {
	c, cleanup := seedInventoryWhereFixture(t)
	defer cleanup()

	// Nothing edible matches "stillwater" — filtered pass must miss so the
	// caller's unfiltered fallback (and its flavor rejection) can fire.
	_, found := c.FindInBackpackWhere("stillwater", func(it items.Item) bool {
		return it.GetSpec().Subtype == items.Edible
	})
	if found {
		t.Fatal("edible filter must not match the pearl or pendant")
	}
}

func TestFindInBackpackWhere_NilFilterMatchesPlain(t *testing.T) {
	c, cleanup := seedInventoryWhereFixture(t)
	defer cleanup()

	a, af := c.FindInBackpack("stillwater")
	b, bf := c.FindInBackpackWhere("stillwater", nil)
	if af != bf || a.ItemId != b.ItemId {
		t.Fatalf("nil filter must behave exactly like FindInBackpack: %v/%d vs %v/%d", af, a.ItemId, bf, b.ItemId)
	}
}
