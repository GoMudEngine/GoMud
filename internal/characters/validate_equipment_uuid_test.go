package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// TestValidateEquipmentItemsCoversEveryWornSlot pins the guarantee that
// mobs.NewMobById relies on: Character.Validate mints a UUID for an item in
// ANY worn slot, not a hand-listed subset.
//
// Why this matters: items.Item.Equals keys on UUID
// (`i.ItemId == b.ItemId && i.UUID == b.UUID`), and a nil UUID stringifies to
// a constant — so two items that both skipped validation compare EQUAL, and
// @<handle> targeting hits the wrong one. The exotic slots (Shoulders, Back,
// Wrist1/2, Ring2, ExtraArm1-4, ExtraWrist1-4, Tail, ComponentBag) are the
// ones a hand-written list forgets, so they are exactly what this covers.
func TestValidateEquipmentItemsCoversEveryWornSlot(t *testing.T) {
	c := &Character{}

	// Put a distinct item in every slot AllSlots knows about.
	slots := c.Equipment.AllSlots()
	if len(slots) == 0 {
		t.Fatal("AllSlots() returned nothing — Worn slot registry is empty")
	}
	for _, s := range slots {
		*s.Item = items.Item{ItemId: 1}
		if !s.Item.UUID.IsNil() {
			t.Fatalf("slot %q: fresh item should start with a nil UUID", s.Key)
		}
	}

	c.validateEquipmentItems()

	var missed []string
	for _, s := range c.Equipment.AllSlots() {
		if s.Item.UUID.IsNil() {
			missed = append(missed, s.Key)
		}
	}
	if len(missed) > 0 {
		t.Errorf("these worn slots kept a nil UUID after validation: %v\n"+
			"Item.Equals keys on UUID, so items in these slots collide. "+
			"validateEquipmentItems must iterate AllSlots(), never a hand-listed subset.",
			missed)
	}
}

// TestNilUUIDItemsCollide documents WHY the test above matters: two unvalidated
// items of the same ItemId are indistinguishable. If this ever stops holding,
// the severity of a missed slot changes and the comments above need revisiting.
func TestNilUUIDItemsCollide(t *testing.T) {
	a := items.Item{ItemId: 1}
	b := items.Item{ItemId: 1}

	if !a.Equals(b) {
		t.Skip("Item.Equals no longer treats two nil-UUID items as equal — " +
			"the collision hazard has changed; re-read validateEquipmentItems' rationale")
	}

	a.Validate()
	b.Validate()
	if a.Equals(b) {
		t.Error("after Validate, two items still compare equal — UUIDs are not unique")
	}
}
