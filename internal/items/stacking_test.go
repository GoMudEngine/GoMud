package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func init() {
	// Register a few minimal item specs for stacking tests.
	RegisterTestItemSpec(&ItemSpec{ItemId: 501, Name: "test-component", Type: Object, IsComponent: true, Value: 5})
	RegisterTestItemSpec(&ItemSpec{ItemId: 502, Name: "test-potion", Type: Potion, Value: 20})
	RegisterTestItemSpec(&ItemSpec{ItemId: 503, Name: "test-sword", Type: Weapon, Value: 50})
	RegisterTestItemSpec(&ItemSpec{ItemId: 504, Name: "test-food", Type: Food, Value: 3})
	RegisterTestItemSpec(&ItemSpec{ItemId: 505, Name: "test-grenade", Type: Grenade, Value: 15})
	RegisterTestItemSpec(&ItemSpec{ItemId: 506, Name: "test-ammo", Type: Ammo, Value: 1})
	RegisterTestItemSpec(&ItemSpec{ItemId: 507, Name: "test-botanical", Type: Botanical, Value: 4})
}

// ─── SameStack ───────────────────────────────────────────────────────────────

func TestSameStack_IdenticalItems(t *testing.T) {
	a := Item{ItemId: 501, Uses: 0}
	b := Item{ItemId: 501, Uses: 0}
	assert.True(t, SameStack(a, b))
}

func TestSameStack_DifferentItemId(t *testing.T) {
	a := Item{ItemId: 501}
	b := Item{ItemId: 502}
	assert.False(t, SameStack(a, b))
}

func TestSameStack_DifferentUses(t *testing.T) {
	a := Item{ItemId: 502, Uses: 3}
	b := Item{ItemId: 502, Uses: 2}
	assert.False(t, SameStack(a, b))
}

func TestSameStack_DifferentEnchantType(t *testing.T) {
	a := Item{ItemId: 501, EnchantType: "fire"}
	b := Item{ItemId: 501, EnchantType: "ice"}
	assert.False(t, SameStack(a, b))
}

func TestSameStack_DifferentEnchantTier(t *testing.T) {
	a := Item{ItemId: 501, EnchantTier: 1}
	b := Item{ItemId: 501, EnchantTier: 2}
	assert.False(t, SameStack(a, b))
}

func TestSameStack_DifferentBottleMultiplier(t *testing.T) {
	a := Item{ItemId: 502, BottleMultiplier: 1.0}
	b := Item{ItemId: 502, BottleMultiplier: 0.5}
	assert.False(t, SameStack(a, b), "different bottle multipliers mean different aging profiles")
}

func TestSameStack_SameBottleMultiplier(t *testing.T) {
	a := Item{ItemId: 502, BottleMultiplier: 0.5}
	b := Item{ItemId: 502, BottleMultiplier: 0.5}
	assert.True(t, SameStack(a, b))
}

func TestSameStack_CustomSpecPreventsStack(t *testing.T) {
	spec := &ItemSpec{ItemId: 501, Name: "custom"}
	a := Item{ItemId: 501, Spec: spec}
	b := Item{ItemId: 501} // no custom Spec
	assert.False(t, SameStack(a, b), "item with Spec override should not stack with plain item")
}

func TestSameStack_BothNilSpec(t *testing.T) {
	a := Item{ItemId: 501}
	b := Item{ItemId: 501}
	assert.True(t, SameStack(a, b), "two plain items with no Spec override should stack")
}

// ─── IsStackable ─────────────────────────────────────────────────────────────

func TestIsStackable_Component(t *testing.T) {
	spec := GetItemSpec(501)
	assert.True(t, spec.IsStackable(), "is_component items should be stackable")
}

func TestIsStackable_Potion(t *testing.T) {
	spec := GetItemSpec(502)
	assert.True(t, spec.IsStackable())
}

func TestIsStackable_Weapon(t *testing.T) {
	spec := GetItemSpec(503)
	assert.False(t, spec.IsStackable())
}

func TestIsStackable_Food(t *testing.T) {
	spec := GetItemSpec(504)
	assert.True(t, spec.IsStackable())
}

func TestIsStackable_Grenade(t *testing.T) {
	spec := GetItemSpec(505)
	assert.True(t, spec.IsStackable())
}

func TestIsStackable_Ammo(t *testing.T) {
	spec := GetItemSpec(506)
	assert.True(t, spec.IsStackable(), "Ammo type should be stackable (forward-declared for arrows/bolts)")
}

func TestIsStackable_Botanical(t *testing.T) {
	spec := GetItemSpec(507)
	assert.True(t, spec.IsStackable())
}
