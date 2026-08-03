package items

import "testing"

// Backpack/ground item matching is apostrophe-insensitive on both sides
// (see util.NormalizeForMatch) — `get healers root` finds a Healer's Root.
func TestNameMatch_ApostropheInsensitive(t *testing.T) {
	cleanup := SeedItemsForTest(map[int]*ItemSpec{
		91401: {ItemId: 91401, Name: "Healer's Root", Type: Object, Subtype: Mundane},
	})
	defer cleanup()

	itm := Item{ItemId: 91401}
	for _, q := range []string{"healers root", "healer's root", "healers"} {
		part, _ := itm.NameMatch(q, false)
		if !part {
			t.Errorf("NameMatch(%q) should match Healer's Root", q)
		}
	}
	if part, _ := itm.NameMatch("bitter", false); part {
		t.Error("unrelated query must not match")
	}
}
