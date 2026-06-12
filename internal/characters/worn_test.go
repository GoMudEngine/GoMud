package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

func TestWorn_EnableAll(t *testing.T) {
	tests := []struct {
		name     string
		input    Worn
		expected Worn
	}{
		{
			name: "All slots have negative ItemId, should reset to zero value",
			input: Worn{
				Weapon:  items.Item{ItemId: -1},
				Offhand: items.Item{ItemId: -2},
				Head:    items.Item{ItemId: -3},
				Neck:    items.Item{ItemId: -4},
				Body:    items.Item{ItemId: -5},
				Belt:    items.Item{ItemId: -6},
				Gloves:  items.Item{ItemId: -7},
				Ring:    items.Item{ItemId: -8},
				Legs:    items.Item{ItemId: -9},
				Feet:    items.Item{ItemId: -10},
			},
			expected: Worn{
				Weapon:  items.Item{},
				Offhand: items.Item{},
				Head:    items.Item{},
				Neck:    items.Item{},
				Body:    items.Item{},
				Belt:    items.Item{},
				Gloves:  items.Item{},
				Ring:    items.Item{},
				Legs:    items.Item{},
				Feet:    items.Item{},
			},
		},
		{
			name: "All slots have positive ItemId, should remain unchanged",
			input: Worn{
				Weapon:  items.Item{ItemId: 1},
				Offhand: items.Item{ItemId: 2},
				Head:    items.Item{ItemId: 3},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{ItemId: 5},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{ItemId: 7},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{ItemId: 9},
				Feet:    items.Item{ItemId: 10},
			},
			expected: Worn{
				Weapon:  items.Item{ItemId: 1},
				Offhand: items.Item{ItemId: 2},
				Head:    items.Item{ItemId: 3},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{ItemId: 5},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{ItemId: 7},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{ItemId: 9},
				Feet:    items.Item{ItemId: 10},
			},
		},
		{
			name: "Mixed ItemIds, only negative should reset",
			input: Worn{
				Weapon:  items.Item{ItemId: -1},
				Offhand: items.Item{ItemId: 2},
				Head:    items.Item{ItemId: -3},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{ItemId: -5},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{ItemId: -7},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{ItemId: -9},
				Feet:    items.Item{ItemId: 10},
			},
			expected: Worn{
				Weapon:  items.Item{},
				Offhand: items.Item{ItemId: 2},
				Head:    items.Item{},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{},
				Feet:    items.Item{ItemId: 10},
			},
		},
		{
			name: "All slots zero ItemId, should remain unchanged",
			input: Worn{
				Weapon:  items.Item{ItemId: 0},
				Offhand: items.Item{ItemId: 0},
				Head:    items.Item{ItemId: 0},
				Neck:    items.Item{ItemId: 0},
				Body:    items.Item{ItemId: 0},
				Belt:    items.Item{ItemId: 0},
				Gloves:  items.Item{ItemId: 0},
				Ring:    items.Item{ItemId: 0},
				Legs:    items.Item{ItemId: 0},
				Feet:    items.Item{ItemId: 0},
			},
			expected: Worn{
				Weapon:  items.Item{ItemId: 0},
				Offhand: items.Item{ItemId: 0},
				Head:    items.Item{ItemId: 0},
				Neck:    items.Item{ItemId: 0},
				Body:    items.Item{ItemId: 0},
				Belt:    items.Item{ItemId: 0},
				Gloves:  items.Item{ItemId: 0},
				Ring:    items.Item{ItemId: 0},
				Legs:    items.Item{ItemId: 0},
				Feet:    items.Item{ItemId: 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := tt.input
			w.EnableAll()
			assert.Equal(t, tt.expected, w)
		})
	}
}
func TestWorn_GetAllItems(t *testing.T) {
	tests := []struct {
		name     string
		worn     Worn
		expected []items.Item
	}{
		{
			name:     "All slots empty",
			worn:     Worn{},
			expected: []items.Item{},
		},
		{
			name: "All slots have positive ItemId",
			worn: Worn{
				Weapon:  items.Item{ItemId: 1},
				Offhand: items.Item{ItemId: 2},
				Head:    items.Item{ItemId: 3},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{ItemId: 5},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{ItemId: 7},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{ItemId: 9},
				Feet:    items.Item{ItemId: 10},
			},
			expected: []items.Item{
				{ItemId: 1},
				{ItemId: 2},
				{ItemId: 3},
				{ItemId: 4},
				{ItemId: 5},
				{ItemId: 6},
				{ItemId: 7},
				{ItemId: 8},
				{ItemId: 9},
				{ItemId: 10},
			},
		},
		{
			name: "Some slots have positive ItemId",
			worn: Worn{
				Weapon:  items.Item{ItemId: 1},
				Offhand: items.Item{ItemId: 0},
				Head:    items.Item{ItemId: 3},
				Neck:    items.Item{ItemId: -1},
				Body:    items.Item{ItemId: 0},
				Belt:    items.Item{ItemId: 6},
				Gloves:  items.Item{ItemId: 0},
				Ring:    items.Item{ItemId: 8},
				Legs:    items.Item{ItemId: 0},
				Feet:    items.Item{ItemId: 10},
			},
			expected: []items.Item{
				{ItemId: 1},
				{ItemId: 3},
				{ItemId: 6},
				{ItemId: 8},
				{ItemId: 10},
			},
		},
		{
			name: "All slots zero or negative ItemId",
			worn: Worn{
				Weapon:  items.Item{ItemId: 0},
				Offhand: items.Item{ItemId: -2},
				Head:    items.Item{ItemId: 0},
				Neck:    items.Item{ItemId: -4},
				Body:    items.Item{ItemId: 0},
				Belt:    items.Item{ItemId: -6},
				Gloves:  items.Item{ItemId: 0},
				Ring:    items.Item{ItemId: -8},
				Legs:    items.Item{ItemId: 0},
				Feet:    items.Item{ItemId: -10},
			},
			expected: []items.Item{},
		},
		{
			name: "Mixed positive, zero, and negative ItemIds",
			worn: Worn{
				Weapon:  items.Item{ItemId: 1},
				Offhand: items.Item{ItemId: 0},
				Head:    items.Item{ItemId: -3},
				Neck:    items.Item{ItemId: 4},
				Body:    items.Item{ItemId: 0},
				Belt:    items.Item{ItemId: -6},
				Gloves:  items.Item{ItemId: 7},
				Ring:    items.Item{ItemId: 0},
				Legs:    items.Item{ItemId: 9},
				Feet:    items.Item{ItemId: -10},
			},
			expected: []items.Item{
				{ItemId: 1},
				{ItemId: 4},
				{ItemId: 7},
				{ItemId: 9},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.worn.GetAllItems()
			assert.Equal(t, tt.expected, got)
		})
	}
}
func TestGetAllSlotTypes(t *testing.T) {
	expected := []string{
		string(items.Weapon),
		string(items.Offhand),
		string(items.Head),
		string(items.Neck),
		string(items.Shoulders),
		string(items.Body),
		string(items.Back),
		string(items.Belt),
		string(items.Wrist),
		string(items.Gloves),
		string(items.Ring),
		string(items.Legs),
		string(items.Feet),
		string(items.Tail),
		string(items.ComponentBag),
	}

	got := GetAllSlotTypes()
	assert.Equal(t, expected, got, "GetAllSlotTypes should return all slot types in correct order")
}

// ---------------------------------------------------------------------------
// Wear min-strength gate tests
// ---------------------------------------------------------------------------

// newRangedWeapon builds a minimal shooting weapon with an optional MinStrength.
// Uses Subtype=Shooting so HandsRequired() returns early without loading species data.
func newRangedWeapon(id int, minStr int) items.Item {
	return items.Item{
		ItemId: id,
		Spec: &items.ItemSpec{
			ItemId:      id,
			Name:        "heavy arbalest",
			Type:        items.Weapon,
			Subtype:     items.Shooting,
			Hands:       2,
			MinStrength: minStr,
		},
	}
}

// TestLoadEquippedRangedWeapons_LoadsMainAndOffhand verifies that an equipped
// ranged weapon (either hand) is marked Loaded, while a non-ranged item is left
// untouched.
func TestLoadEquippedRangedWeapons_LoadsMainAndOffhand(t *testing.T) {
	c := New()
	c.Equipment.Weapon = newRangedWeapon(9910, 0)
	c.Equipment.Offhand = newRangedWeapon(9911, 0)

	assert.False(t, c.Equipment.Weapon.Loaded, "precondition: starts unloaded")
	assert.False(t, c.Equipment.Offhand.Loaded, "precondition: starts unloaded")

	c.Equipment.LoadEquippedRangedWeapons()

	assert.True(t, c.Equipment.Weapon.Loaded, "main-hand ranged weapon must be loaded")
	assert.True(t, c.Equipment.Offhand.Loaded, "offhand ranged weapon must be loaded")
}

// TestLoadEquippedRangedWeapons_IgnoresMelee verifies a non-ranged weapon is
// not given a (meaningless) Loaded flag and empty slots don't panic.
func TestLoadEquippedRangedWeapons_IgnoresMelee(t *testing.T) {
	c := New()
	c.Equipment.Weapon = items.Item{
		ItemId: 9920,
		Spec: &items.ItemSpec{
			ItemId: 9920, Name: "iron sword", Type: items.Weapon, Subtype: items.Slashing,
		},
	}

	c.Equipment.LoadEquippedRangedWeapons() // empty offhand must not panic

	assert.False(t, c.Equipment.Weapon.Loaded, "melee weapon must not be marked Loaded")
}

// TestWear_MinStrength_Reject verifies that Wear() refuses a weapon whose
// MinStrength exceeds the character's current Strength.ValueAdj.
func TestWear_MinStrength_Reject(t *testing.T) {
	c := New()
	c.Stats.Strength.ValueAdj = 50 // well below the 100 requirement

	bow := newRangedWeapon(9901, 100)
	_, worn, reason := c.Wear(bow)

	assert.False(t, worn, "should be rejected due to low strength")
	assert.Contains(t, reason, "aren't strong enough", "failure reason should mention strength")
	assert.Equal(t, items.Item{}, c.Equipment.Weapon, "weapon slot must stay empty")
}

// TestWear_MinStrength_Accept verifies that Wear() succeeds when the character
// meets the MinStrength requirement exactly.
func TestWear_MinStrength_Accept(t *testing.T) {
	c := New()
	c.Stats.Strength.ValueAdj = 100 // exactly meets the requirement

	bow := newRangedWeapon(9902, 100)
	_, worn, reason := c.Wear(bow)

	assert.True(t, worn, "should succeed: strength meets minimum")
	assert.Empty(t, reason, "no failure reason expected")
	assert.Equal(t, bow.ItemId, c.Equipment.Weapon.ItemId, "weapon should be in weapon slot")
}

// TestWear_MinStrength_Zero verifies that a weapon with no MinStrength set (0)
// is always equippable regardless of the character's strength.
func TestWear_MinStrength_Zero(t *testing.T) {
	c := New()
	c.Stats.Strength.ValueAdj = 1 // extremely weak

	bow := newRangedWeapon(9903, 0) // no minimum
	_, worn, reason := c.Wear(bow)

	assert.True(t, worn, "no MinStrength set — should always equip")
	assert.Empty(t, reason, "no failure reason expected")
}
