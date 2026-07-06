package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestHasOwnMasterwork(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999970: {ItemId: 999970, Name: "masterwork blade", Type: items.Weapon},
	})()

	// no items at all -> false
	c := New()
	c.Name = "Megalomania"
	if c.HasOwnMasterwork(50) {
		t.Fatal("expected false when character carries no items")
	}

	// own item, crafted at skill 65 -> passes a 50 gate
	c1 := New()
	c1.Name = "Megalomania"
	own := items.New(999970)
	own.MakerName = "Megalomania"
	own.CraftSkill = 65
	c1.Items = append(c1.Items, own)
	if !c1.HasOwnMasterwork(50) {
		t.Fatal("expected true for own item crafted above the skill gate")
	}

	// foreign-made item at skill 65 -> does not count, even though skill qualifies
	c2 := New()
	c2.Name = "Megalomania"
	foreign := items.New(999970)
	foreign.MakerName = "Someone Else"
	foreign.CraftSkill = 65
	c2.Items = append(c2.Items, foreign)
	if c2.HasOwnMasterwork(50) {
		t.Fatal("expected false for an item crafted by someone else")
	}

	// own item, but crafted below the skill gate -> does not count
	c3 := New()
	c3.Name = "Megalomania"
	lowSkill := items.New(999970)
	lowSkill.MakerName = "Megalomania"
	lowSkill.CraftSkill = 40
	c3.Items = append(c3.Items, lowSkill)
	if c3.HasOwnMasterwork(50) {
		t.Fatal("expected false for an own item below the skill gate")
	}
}
