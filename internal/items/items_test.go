package items

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/stretchr/testify/assert"
)

// seedRegistry populates the global items map with diverse specs for testing.
// Call the returned cleanup function (or use defer) to restore the original.
func seedRegistry() func() {
	orig := items
	items = map[int]*ItemSpec{
		10001: {
			ItemId:           10001,
			Name:             "Iron Sword",
			NameSimple:       "Sword",
			Description:      "A sturdy iron sword.",
			Type:             Weapon,
			Subtype:          Slashing,
			Hands:            1,
			Value:            100,
			DamageMultiplier: 1.0,
			Damage:           Damage{Attacks: 1, BaseDamage: 25, Variance: 5},
		},
		10002: {
			ItemId:                10002,
			Name:                  "Oak Staff",
			NameSimple:            "Staff",
			Description:           "A gnarled oak staff crackling with energy.",
			Type:                  Weapon,
			Subtype:               Staff,
			Hands:                 2,
			Value:                 150,
			DamageMultiplier:      0.80,
			SpellDamageMultiplier: 1.60,
			Damage:                Damage{Attacks: 1, BaseDamage: 15, Variance: 3},
		},
		20001: {
			ItemId:             20001,
			Name:               "Chain Mail",
			Description:        "A suit of interlocking metal rings.",
			Type:               Body,
			Subtype:            Wearable,
			Value:              200,
			PhysicalMitigation: 15,
			MagicalMitigation:  5,
		},
		20002: {
			ItemId:      20002,
			Name:        "Cursed Ring",
			Description: "A ring that whispers dark secrets.",
			Type:        Ring,
			Subtype:     Wearable,
			Value:       50,
			Cursed:      true,
			StatMods:    statmods.StatMods{"strength": 5, "willpower": -3},
		},
		30001: {
			ItemId:      30001,
			Name:        "Healing Potion",
			Description: "A vial of red liquid.",
			Type:        Potion,
			Subtype:     Usable,
			Uses:        3,
			Value:       25,
			BuffIds:     []int{1},
		},
		5001: {
			ItemId:      5001,
			Name:        "Rusty Key",
			Description: "A key covered in rust.",
			Type:        Key,
			KeyLockId:   "100-north",
			Value:       10,
		},
		10003: {
			ItemId:      10003,
			Name:        "Enchanted Dagger",
			NameSimple:  "Dagger",
			Description: "A dagger shimmering with magic.",
			Type:        Weapon,
			Subtype:     Stabbing,
			Hands:       1,
			Value:       120,
			Damage:      Damage{Attacks: 1, BaseDamage: 18, Variance: 4},
			QuestToken:  "dagger-quest-start",
		},
		1: {
			ItemId:      1,
			Name:        "Wooden Stick",
			Description: "Just a stick.",
			Type:        Junk,
			Value:       1,
		},
	}
	return func() { items = orig }
}

// ─── GetItemSpec ────────────────────────────────────────────────────────────

func TestGetItemSpec(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("valid ID returns spec", func(t *testing.T) {
		spec := GetItemSpec(10001)
		assert.NotNil(t, spec)
		assert.Equal(t, "Iron Sword", spec.Name)
	})

	t.Run("missing ID returns nil", func(t *testing.T) {
		spec := GetItemSpec(99999)
		assert.Nil(t, spec)
	})

	t.Run("zero ID returns nil", func(t *testing.T) {
		spec := GetItemSpec(0)
		assert.Nil(t, spec)
	})

	t.Run("negative ID returns nil", func(t *testing.T) {
		spec := GetItemSpec(-5)
		assert.Nil(t, spec)
	})
}

// ─── GetAllItemSpecs ────────────────────────────────────────────────────────

func TestGetAllItemSpecs(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	specs := GetAllItemSpecs()
	assert.Len(t, specs, 8, "should return all 8 seeded specs")
}

// ─── GetAllItemNames ────────────────────────────────────────────────────────

func TestGetAllItemNames(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	names := GetAllItemNames()
	assert.Len(t, names, 8)
	assert.Contains(t, names, "Iron Sword")
	assert.Contains(t, names, "Healing Potion")
	assert.Contains(t, names, "Wooden Stick")
}

// ─── FindItem ───────────────────────────────────────────────────────────────

func TestFindItem(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("find by name", func(t *testing.T) {
		id := FindItem("Iron Sword")
		assert.Equal(t, 10001, id)
	})

	t.Run("find by ID string", func(t *testing.T) {
		id := FindItem("10001")
		assert.Equal(t, 10001, id)
	})

	t.Run("not found returns 0", func(t *testing.T) {
		id := FindItem("Nonexistent Widget")
		assert.Equal(t, 0, id)
	})
}

// ─── FindItemByName ─────────────────────────────────────────────────────────

func TestFindItemByName(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("exact match", func(t *testing.T) {
		id := FindItemByName("Healing Potion")
		assert.Equal(t, 30001, id)
	})

	t.Run("prefix match", func(t *testing.T) {
		id := FindItemByName("Iron")
		assert.Equal(t, 10001, id)
	})

	t.Run("substring match", func(t *testing.T) {
		id := FindItemByName("Dagger")
		assert.Equal(t, 10003, id)
	})

	t.Run("not found", func(t *testing.T) {
		id := FindItemByName("Zzzzzz")
		assert.Equal(t, 0, id)
	})
}

// ─── FindKeyByLockId ────────────────────────────────────────────────────────

func TestFindKeyByLockId(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("finds key", func(t *testing.T) {
		id := FindKeyByLockId("100-north")
		assert.Equal(t, 5001, id)
	})

	t.Run("not found returns 0", func(t *testing.T) {
		id := FindKeyByLockId("999-south")
		assert.Equal(t, 0, id)
	})
}

// ─── IsEnchanted ────────────────────────────────────────────────────────────

func TestIsEnchanted(t *testing.T) {
	t.Run("enchanted", func(t *testing.T) {
		item := Item{ItemId: 1, Enchantments: 2}
		assert.True(t, item.IsEnchanted())
	})

	t.Run("not enchanted", func(t *testing.T) {
		item := Item{ItemId: 1, Enchantments: 0}
		assert.False(t, item.IsEnchanted())
	})
}

// ─── Enchant / UnEnchant ────────────────────────────────────────────────────

func TestEnchantUnEnchant(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	item := Item{ItemId: 10001}

	// Enchant sets Spec override and increments Enchantments
	item.Enchant(5, 3, map[string]int{"strength": 2}, false)
	assert.True(t, item.IsEnchanted())
	assert.NotNil(t, item.Spec)
	assert.Equal(t, uint8(1), item.Enchantments)
	assert.Equal(t, 30, item.Spec.Damage.BaseDamage) // 25 + 5

	// UnEnchant clears
	item.UnEnchant()
	assert.False(t, item.IsEnchanted())
	assert.Nil(t, item.Spec)
	assert.Equal(t, uint8(0), item.Enchantments)
}

// ─── IsCursed ───────────────────────────────────────────────────────────────

func TestIsCursed(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("cursed item", func(t *testing.T) {
		item := Item{ItemId: 20002}
		assert.True(t, item.IsCursed())
	})

	t.Run("uncursed override", func(t *testing.T) {
		item := Item{ItemId: 20002, Uncursed: true}
		assert.False(t, item.IsCursed())
	})

	t.Run("non-cursed item", func(t *testing.T) {
		item := Item{ItemId: 10001}
		assert.False(t, item.IsCursed())
	})
}

// ─── IsValid ────────────────────────────────────────────────────────────────

func TestIsValid(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("valid item", func(t *testing.T) {
		item := Item{ItemId: 10001}
		assert.True(t, item.IsValid())
	})

	t.Run("invalid item", func(t *testing.T) {
		item := Item{ItemId: 99999}
		assert.False(t, item.IsValid())
	})
}

// ─── IsSpecial ──────────────────────────────────────────────────────────────

func TestIsSpecial(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("has quest token spec override", func(t *testing.T) {
		item := Item{ItemId: 10003, Spec: &ItemSpec{QuestToken: "test"}}
		assert.True(t, item.IsSpecial())
	})

	t.Run("has blob", func(t *testing.T) {
		item := Item{ItemId: 10001, Blob: "data"}
		assert.True(t, item.IsSpecial())
	})

	t.Run("uses differ from spec", func(t *testing.T) {
		item := Item{ItemId: 30001, Uses: 1} // spec has Uses=3
		assert.True(t, item.IsSpecial())
	})

	t.Run("plain item not special", func(t *testing.T) {
		item := Item{ItemId: 1}
		assert.False(t, item.IsSpecial())
	})
}

// ─── StatMod ────────────────────────────────────────────────────────────────

func TestStatMod(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	spec := GetItemSpec(20002)
	assert.NotNil(t, spec)
	assert.Equal(t, 5, spec.StatMods.Get("strength"))
	assert.Equal(t, -3, spec.StatMods.Get("willpower"))
	assert.Equal(t, 0, spec.StatMods.Get("dexterity"))
}

// ─── GetSpec ────────────────────────────────────────────────────────────────

func TestGetSpec(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("falls back to registry", func(t *testing.T) {
		item := Item{ItemId: 10001}
		spec := item.GetSpec()
		assert.Equal(t, "Iron Sword", spec.Name)
	})

	t.Run("returns override when set", func(t *testing.T) {
		override := &ItemSpec{Name: "Custom Sword", Value: 999}
		item := Item{ItemId: 10001, Spec: override}
		spec := item.GetSpec()
		assert.Equal(t, "Custom Sword", spec.Name)
		assert.Equal(t, 999, spec.Value)
	})

	t.Run("unknown item returns empty spec", func(t *testing.T) {
		item := Item{ItemId: 99999}
		spec := item.GetSpec()
		assert.Equal(t, 0, spec.ItemId)
	})
}

// ─── SetAdjective ───────────────────────────────────────────────────────────

func TestSetAdjective(t *testing.T) {
	t.Run("add adjective", func(t *testing.T) {
		item := Item{ItemId: 1}
		item.SetAdjective("shiny", true)
		assert.True(t, item.HasAdjective("shiny"))
	})

	t.Run("add duplicate is no-op", func(t *testing.T) {
		item := Item{ItemId: 1, Adjectives: []string{"shiny"}}
		item.SetAdjective("shiny", true)
		assert.Len(t, item.Adjectives, 1)
	})

	t.Run("remove adjective", func(t *testing.T) {
		item := Item{ItemId: 1, Adjectives: []string{"shiny", "glowing"}}
		item.SetAdjective("shiny", false)
		assert.False(t, item.HasAdjective("shiny"))
		assert.True(t, item.HasAdjective("glowing"))
	})

	t.Run("remove nonexistent is no-op", func(t *testing.T) {
		item := Item{ItemId: 1, Adjectives: []string{"shiny"}}
		item.SetAdjective("rusty", false)
		assert.Len(t, item.Adjectives, 1)
	})
}

// ─── TempData ───────────────────────────────────────────────────────────────

func TestItem_TempData(t *testing.T) {
	item := Item{ItemId: 1}

	t.Run("set and get", func(t *testing.T) {
		item.SetTempData("key", "value")
		assert.Equal(t, "value", item.GetTempData("key"))
	})

	t.Run("missing key returns nil", func(t *testing.T) {
		assert.Nil(t, item.GetTempData("missing"))
	})

	t.Run("delete by nil", func(t *testing.T) {
		item.SetTempData("key", nil)
		assert.Nil(t, item.GetTempData("key"))
	})
}

// ─── IsDisabled ─────────────────────────────────────────────────────────────

func TestIsDisabled(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		assert.True(t, Item{ItemId: -1}.IsDisabled())
	})
	t.Run("not disabled", func(t *testing.T) {
		assert.False(t, Item{ItemId: 1}.IsDisabled())
	})
	t.Run("zero not disabled", func(t *testing.T) {
		assert.False(t, Item{ItemId: 0}.IsDisabled())
	})
}

// ─── Uncurse ────────────────────────────────────────────────────────────────

func TestUncurse(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	item := Item{ItemId: 20002} // cursed ring
	assert.True(t, item.IsCursed())
	item.Uncurse()
	assert.False(t, item.IsCursed())
}

// ─── GetDefense ─────────────────────────────────────────────────────────────

func TestGetDefense(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	item := Item{ItemId: 20001} // chain mail
	assert.Equal(t, 0, item.GetDefense()) // DamageReduction not set on chain mail

	// Test with override spec
	item2 := Item{ItemId: 1, Spec: &ItemSpec{DamageReduction: 15}}
	assert.Equal(t, 15, item2.GetDefense())
}

// ─── CanBackstab ────────────────────────────────────────────────────────────

func TestCanBackstab(t *testing.T) {
	assert.True(t, CanBackstab(Stabbing))
	assert.True(t, CanBackstab(Slashing))
	assert.True(t, CanBackstab(Cleaving))
	assert.True(t, CanBackstab(Claws))
	assert.False(t, CanBackstab(Bludgeoning))
	assert.False(t, CanBackstab(Shooting))
	assert.False(t, CanBackstab(Generic))
}

// ─── GetAttackStaminaCost ───────────────────────────────────────────────────

func TestGetAttackStaminaCost(t *testing.T) {
	t.Run("explicit cost", func(t *testing.T) {
		spec := &ItemSpec{Type: Weapon, StaminaCost: 12}
		assert.Equal(t, 12, spec.GetAttackStaminaCost())
	})

	t.Run("default weapon cost", func(t *testing.T) {
		spec := &ItemSpec{Type: Weapon}
		assert.Equal(t, 8, spec.GetAttackStaminaCost())
	})

	t.Run("non-weapon returns 0", func(t *testing.T) {
		spec := &ItemSpec{Type: Body}
		assert.Equal(t, 0, spec.GetAttackStaminaCost())
	})
}

// ─── GetSpeedMultiplier ─────────────────────────────────────────────────────

func TestGetSpeedMultiplier(t *testing.T) {
	t.Run("explicit speed", func(t *testing.T) {
		spec := ItemSpec{SpeedMultiplier: 0.7}
		assert.InDelta(t, 0.7, spec.GetSpeedMultiplier(), 0.001)
	})

	t.Run("default speed", func(t *testing.T) {
		spec := ItemSpec{}
		assert.InDelta(t, 1.0, spec.GetSpeedMultiplier(), 0.001)
	})
}

// ─── GetWeight ──────────────────────────────────────────────────────────────

func TestGetWeight(t *testing.T) {
	spec := ItemSpec{Weight: 3.5}
	assert.InDelta(t, 3.5, spec.GetWeight(), 0.001)

	spec2 := ItemSpec{}
	assert.InDelta(t, 0.0, spec2.GetWeight(), 0.001)
}

// ─── String methods ─────────────────────────────────────────────────────────

func TestStringMethods(t *testing.T) {
	t.Run("Element", func(t *testing.T) {
		assert.Equal(t, "fire", Fire.String())
	})

	t.Run("ItemType", func(t *testing.T) {
		assert.Equal(t, "weapon", Weapon.String())
	})

	t.Run("ItemSubType", func(t *testing.T) {
		assert.Equal(t, "slashing", Slashing.String())
	})

	t.Run("Damage String base damage", func(t *testing.T) {
		d := Damage{BaseDamage: 30, Variance: 7}
		assert.Equal(t, "~30 ±7", d.String())
	})

	t.Run("Damage String dice roll", func(t *testing.T) {
		d := Damage{DiceRoll: "2d6+1"}
		assert.Equal(t, "2d6+1", d.String())
	})

	t.Run("Damage String N/A", func(t *testing.T) {
		d := Damage{}
		assert.Equal(t, "N/A", d.String())
	})
}

// ─── Id ─────────────────────────────────────────────────────────────────────

func TestItemSpec_Id(t *testing.T) {
	spec := ItemSpec{ItemId: 42}
	assert.Equal(t, 42, spec.Id())
}

// ─── Filename / Filepath / ItemFolder ───────────────────────────────────────

func TestItemSpec_Filename(t *testing.T) {
	spec := ItemSpec{ItemId: 10001, Name: "Iron Sword"}
	assert.Equal(t, "10001-iron_sword.yaml", spec.Filename())
}

func TestItemSpec_Filepath(t *testing.T) {
	spec := ItemSpec{ItemId: 10001, Name: "Iron Sword"}
	assert.Contains(t, spec.Filepath(), "weapons-10000/10001-iron_sword.yaml")
}

func TestItemSpec_ItemFolder(t *testing.T) {
	tests := []struct {
		name     string
		itemId   int
		expected string
	}{
		{"other", 500, "other-0"},
		{"weapons", 10001, "weapons-10000"},
		{"armor", 20001, "armor-20000/body"},
		{"consumables", 30001, "consumables-30000"},
		{"materials", 40001, "materials-40000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := ItemSpec{ItemId: tt.itemId, Type: Body}
			got := spec.ItemFolder()
			assert.Equal(t, tt.expected, got)
		})
	}

	t.Run("armor base only", func(t *testing.T) {
		spec := ItemSpec{ItemId: 20001, Type: Body}
		got := spec.ItemFolder(true)
		assert.Equal(t, "armor-20000", got)
	})
}

// ─── HasChrysalisEnchantment ────────────────────────────────────────────────

func TestHasChrysalisEnchantment(t *testing.T) {
	t.Run("has enchantment", func(t *testing.T) {
		item := Item{ItemId: 1, EnchantType: "fire"}
		assert.True(t, item.HasChrysalisEnchantment())
	})

	t.Run("no enchantment", func(t *testing.T) {
		item := Item{ItemId: 1}
		assert.False(t, item.HasChrysalisEnchantment())
	})
}

// ─── Rename / Redescribe ────────────────────────────────────────────────────

func TestRename(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	item := Item{ItemId: 10001}
	item.Rename("Firebrand", "A flaming sword")
	spec := item.GetSpec()
	assert.Equal(t, "Firebrand", spec.Name)
}

func TestRedescribe(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	item := Item{ItemId: 10001}
	item.Redescribe("A newly forged blade.")
	spec := item.GetSpec()
	assert.Equal(t, "A newly forged blade.", spec.Description)
}

// ─── AddWornBuff ────────────────────────────────────────────────────────────

func TestAddWornBuff(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	item := Item{ItemId: 10001}
	item.AddWornBuff(5)
	assert.Contains(t, item.Spec.WornBuffIds, 5)
}

// ─── ItemTypes / ItemSubtypes ───────────────────────────────────────────────

func TestItemTypes(t *testing.T) {
	types := ItemTypes()
	assert.True(t, len(types) > 0, "should return item types")
}

func TestItemSubtypes(t *testing.T) {
	subtypes := ItemSubtypes()
	assert.True(t, len(subtypes) > 0, "should return item subtypes")
}

// ─── Item.Name ──────────────────────────────────────────────────────────────

func TestItem_Name(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("valid item", func(t *testing.T) {
		item := Item{ItemId: 10001}
		assert.Equal(t, "Iron Sword", item.Name())
	})

	t.Run("zero item", func(t *testing.T) {
		item := Item{ItemId: 0}
		assert.Equal(t, "-nothing-", item.Name())
	})

	t.Run("disabled item", func(t *testing.T) {
		item := Item{ItemId: -1}
		assert.Equal(t, "***disabled***", item.Name())
	})
}

// ─── Item.NameSimple ────────────────────────────────────────────────────────

func TestItem_NameSimple(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("valid item", func(t *testing.T) {
		item := Item{ItemId: 10001}
		assert.Equal(t, "Sword", item.NameSimple())
	})

	t.Run("zero item", func(t *testing.T) {
		item := Item{ItemId: 0}
		assert.Equal(t, "-nothing-", item.NameSimple())
	})

	t.Run("disabled item", func(t *testing.T) {
		item := Item{ItemId: -1}
		assert.Equal(t, "***disabled***", item.NameSimple())
	})
}

// ─── Item.DisplayName ───────────────────────────────────────────────────────

func TestItem_DisplayName(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("zero returns nothing", func(t *testing.T) {
		item := Item{ItemId: 0}
		assert.Contains(t, item.DisplayName(), "-nothing-")
	})

	t.Run("disabled returns disabled", func(t *testing.T) {
		item := Item{ItemId: -1}
		assert.Contains(t, item.DisplayName(), "disabled")
	})

	t.Run("plain item", func(t *testing.T) {
		item := Item{ItemId: 10001}
		assert.Contains(t, item.DisplayName(), "Iron Sword")
	})

	t.Run("quest token item has star prefix", func(t *testing.T) {
		item := Item{ItemId: 10003}
		name := item.DisplayName()
		assert.Contains(t, name, "★")
	})

	t.Run("item with adjectives", func(t *testing.T) {
		item := Item{ItemId: 1, Adjectives: []string{"shiny"}}
		name := item.DisplayName()
		assert.Contains(t, name, "shiny")
	})
}

// ─── Item.NameComplex ───────────────────────────────────────────────────────

func TestItem_NameComplex(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("zero item", func(t *testing.T) {
		item := Item{ItemId: 0}
		assert.Contains(t, item.NameComplex(), "-nothing-")
	})

	t.Run("plain item", func(t *testing.T) {
		item := Item{ItemId: 10001}
		assert.Contains(t, item.NameComplex(), "Iron Sword")
	})

	t.Run("cursed item has flag", func(t *testing.T) {
		item := Item{ItemId: 20002} // cursed ring
		name := item.NameComplex()
		assert.Contains(t, name, "c") // cursed flag
	})
}

// ─── Item.NameMatch ─────────────────────────────────────────────────────────

func TestItem_NameMatch(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	item := Item{ItemId: 10001} // "Iron Sword"

	t.Run("exact match", func(t *testing.T) {
		partial, full := item.NameMatch("Iron Sword", false)
		assert.True(t, partial)
		assert.True(t, full)
	})

	t.Run("prefix match", func(t *testing.T) {
		partial, full := item.NameMatch("Iron", false)
		assert.True(t, partial)
		assert.False(t, full)
	})

	t.Run("contains match", func(t *testing.T) {
		partial, full := item.NameMatch("Sword", true)
		assert.True(t, partial)
		assert.False(t, full)
	})

	t.Run("no match", func(t *testing.T) {
		partial, full := item.NameMatch("Axe", true)
		assert.False(t, partial)
		assert.False(t, full)
	})

	t.Run("zero item", func(t *testing.T) {
		zeroItem := Item{ItemId: 0}
		partial, full := zeroItem.NameMatch("test", true)
		assert.False(t, partial)
		assert.False(t, full)
	})
}

// ─── Item.StatMod (instance method) ─────────────────────────────────────────

func TestItem_StatMod(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("valid item with stat mods", func(t *testing.T) {
		item := Item{ItemId: 20002} // cursed ring: strength +5, willpower -3
		assert.Equal(t, 5, item.StatMod("strength"))
		assert.Equal(t, -3, item.StatMod("willpower"))
		assert.Equal(t, 0, item.StatMod("dexterity"))
	})

	t.Run("zero item", func(t *testing.T) {
		item := Item{ItemId: 0}
		assert.Equal(t, 0, item.StatMod("strength"))
	})
}

// ─── AttrString ─────────────────────────────────────────────────────────────

func TestAttrString(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("no flags", func(t *testing.T) {
		item := Item{ItemId: 10001}
		assert.Equal(t, "", item.AttrString())
	})

	t.Run("cursed flag", func(t *testing.T) {
		item := Item{ItemId: 20002}
		assert.Contains(t, item.AttrString(), "c")
	})

	t.Run("enchanted flag", func(t *testing.T) {
		item := Item{ItemId: 10001, Enchantments: 1}
		assert.Contains(t, item.AttrString(), "e")
	})
}

// ─── ShorthandId ────────────────────────────────────────────────────────────

func TestShorthandId(t *testing.T) {
	t.Run("valid item", func(t *testing.T) {
		item := Item{ItemId: 42, UUID: [16]byte{1, 2, 3}}
		id := item.ShorthandId()
		assert.Contains(t, id, "!42:")
	})

	t.Run("zero item", func(t *testing.T) {
		item := Item{ItemId: 0}
		assert.Equal(t, "", item.ShorthandId())
	})
}

// ─── startsWithVowel ────────────────────────────────────────────────────────

func TestStartsWithVowel(t *testing.T) {
	assert.True(t, startsWithVowel("apple"))
	assert.True(t, startsWithVowel("Umbrella"))
	assert.False(t, startsWithVowel("banana"))
	assert.False(t, startsWithVowel(""))
}

// ─── GetLongDescription ─────────────────────────────────────────────────────

func TestGetLongDescription(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("weapon description", func(t *testing.T) {
		item := Item{ItemId: 10001} // sword
		desc := item.GetLongDescription()
		assert.Contains(t, desc, "A sturdy iron sword.")
		assert.Contains(t, desc, "1-Handed weapon")
	})

	t.Run("key description", func(t *testing.T) {
		item := Item{ItemId: 5001}
		desc := item.GetLongDescription()
		assert.Contains(t, desc, "keyring")
	})

	t.Run("potion description", func(t *testing.T) {
		item := Item{ItemId: 30001}
		desc := item.GetLongDescription()
		assert.Contains(t, desc, "A vial of red liquid.")
	})
}

// ─── FindMatchIn ────────────────────────────────────────────────────────────

func TestFindMatchIn(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	itemList := []Item{
		{ItemId: 10001, UUID: [16]byte{1}}, // Iron Sword
		{ItemId: 30001, UUID: [16]byte{2}}, // Healing Potion
		{ItemId: 1, UUID: [16]byte{3}},     // Wooden Stick
	}

	t.Run("exact match by name", func(t *testing.T) {
		_, full := FindMatchIn("Iron Sword", itemList...)
		assert.Equal(t, 10001, full.ItemId)
	})

	t.Run("prefix match", func(t *testing.T) {
		partial, _ := FindMatchIn("Iron", itemList...)
		assert.Equal(t, 10001, partial.ItemId)
	})

	t.Run("no match", func(t *testing.T) {
		partial, full := FindMatchIn("Nonexistent", itemList...)
		assert.Equal(t, 0, partial.ItemId)
		assert.Equal(t, 0, full.ItemId)
	})

	t.Run("shorthand by ID", func(t *testing.T) {
		_, full := FindMatchIn("!10001", itemList...)
		assert.Equal(t, 10001, full.ItemId)
	})
}

// ─── Validate (ItemSpec) ────────────────────────────────────────────────────

func TestItemSpec_Validate(t *testing.T) {
	t.Run("weapon defaults", func(t *testing.T) {
		spec := &ItemSpec{Type: Weapon, Name: "Test Sword"}
		err := spec.Validate()
		assert.NoError(t, err)
		assert.Equal(t, 1, spec.Hands, "should default to 1 hand")
		assert.Equal(t, 1, spec.Damage.Attacks, "should default to 1 attack")
		assert.Equal(t, "Test Sword", spec.NameSimple, "should default NameSimple to Name")
	})

	t.Run("auto calculate value", func(t *testing.T) {
		spec := &ItemSpec{Type: Weapon, Name: "Test", Value: 0}
		spec.Validate()
		assert.True(t, spec.Value > 0, "should auto calculate value")
	})
}

// ─── HasAdjective ───────────────────────────────────────────────────────────

func TestHasAdjective(t *testing.T) {
	tests := []struct {
		name       string
		adjectives []string
		search     string
		want       bool
	}{
		{"nil adjectives", nil, "exploding", false},
		{"empty adjectives", []string{}, "exploding", false},
		{"found", []string{"shiny", "exploding"}, "exploding", true},
		{"not found", []string{"shiny", "glowing"}, "exploding", false},
		{"exact match only", []string{"exploding"}, "Exploding", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := Item{ItemId: 1, Adjectives: tt.adjectives}
			assert.Equal(t, tt.want, item.HasAdjective(tt.search))
		})
	}
}

// ─── IsBetterThan ───────────────────────────────────────────────────────────

func TestIsBetterThan(t *testing.T) {
	tests := []struct {
		name  string
		item  Item
		other Item
		want  bool
	}{
		{
			name:  "higher value is better",
			item:  Item{ItemId: 1, Spec: &ItemSpec{Value: 200}},
			other: Item{ItemId: 2, Spec: &ItemSpec{Value: 100}},
			want:  true,
		},
		{
			name:  "lower value is not better",
			item:  Item{ItemId: 1, Spec: &ItemSpec{Value: 50}},
			other: Item{ItemId: 2, Spec: &ItemSpec{Value: 100}},
			want:  false,
		},
		{
			name:  "other has zero ItemId",
			item:  Item{ItemId: 1, Spec: &ItemSpec{Value: 10}},
			other: Item{ItemId: 0},
			want:  true,
		},
		{
			name:  "both have zero ItemId",
			item:  Item{ItemId: 0},
			other: Item{ItemId: 0},
			want:  false,
		},
		{
			name:  "self has zero ItemId, other valid",
			item:  Item{ItemId: 0},
			other: Item{ItemId: 1, Spec: &ItemSpec{Value: 10}},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.item.IsBetterThan(tt.other))
		})
	}
}

// ─── Equals ─────────────────────────────────────────────────────────────────

func TestEquals(t *testing.T) {
	uuid1 := [16]byte{1}
	uuid2 := [16]byte{2}
	uuid0 := [16]byte{}

	tests := []struct {
		name string
		a    Item
		b    Item
		want bool
	}{
		{
			name: "same ID and UUID",
			a:    Item{ItemId: 1, UUID: uuid1},
			b:    Item{ItemId: 1, UUID: uuid1},
			want: true,
		},
		{
			name: "different ID",
			a:    Item{ItemId: 1, UUID: uuid1},
			b:    Item{ItemId: 2, UUID: uuid1},
			want: false,
		},
		{
			name: "different UUID",
			a:    Item{ItemId: 1, UUID: uuid1},
			b:    Item{ItemId: 1, UUID: uuid2},
			want: false,
		},
		{
			name: "both zero",
			a:    Item{ItemId: 0, UUID: uuid0},
			b:    Item{ItemId: 0, UUID: uuid0},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.Equals(tt.b))
		})
	}
}

// ─── GetDiceRoll ────────────────────────────────────────────────────────────

func TestGetDiceRoll(t *testing.T) {
	// Zero ItemId → defaults (1, 1, 3, 0, [])
	item := Item{ItemId: 0}
	attacks, dCount, dSides, bonus, critBuffs := item.GetDiceRoll()
	assert.Equal(t, 1, attacks)
	assert.Equal(t, 1, dCount)
	assert.Equal(t, 3, dSides)
	assert.Equal(t, 0, bonus)
	assert.Empty(t, critBuffs)

	// Item with spec
	item2 := Item{ItemId: 1, Spec: &ItemSpec{
		Damage: Damage{
			Attacks:   2,
			DiceCount: 3,
			SideCount: 6,
			BonusDamage: 4,
			CritBuffIds: []int{10, 20},
		},
	}}
	attacks, dCount, dSides, bonus, critBuffs = item2.GetDiceRoll()
	assert.Equal(t, 2, attacks)
	assert.Equal(t, 3, dCount)
	assert.Equal(t, 6, dSides)
	assert.Equal(t, 4, bonus)
	assert.Equal(t, []int{10, 20}, critBuffs)
}

// ─── GetDistributionDamage ──────────────────────────────────────────────────

func TestGetDistributionDamage(t *testing.T) {
	// Zero ItemId → defaults (1, 2.0, 1.0, [])
	item := Item{ItemId: 0}
	attacks, baseDmg, variance, critBuffs := item.GetDistributionDamage()
	assert.Equal(t, 1, attacks)
	assert.InDelta(t, 2.0, baseDmg, 0.01)
	assert.InDelta(t, 1.0, variance, 0.01)
	assert.Empty(t, critBuffs)

	// BaseDamage path (new-style)
	item2 := Item{ItemId: 1, Spec: &ItemSpec{
		Damage: Damage{
			Attacks:    2,
			BaseDamage: 25,
			Variance:   5,
		},
	}}
	attacks, baseDmg, variance, _ = item2.GetDistributionDamage()
	assert.Equal(t, 2, attacks)
	assert.InDelta(t, 25.0, baseDmg, 0.01)
	assert.InDelta(t, 5.0, variance, 0.01)

	// Legacy dice path (DiceCount/SideCount → converted)
	item3 := Item{ItemId: 1, Spec: &ItemSpec{
		Damage: Damage{
			Attacks:   1,
			DiceCount: 2,
			SideCount: 6,
			BonusDamage: 1,
		},
	}}
	attacks, baseDmg, variance, _ = item3.GetDistributionDamage()
	assert.Equal(t, 1, attacks)
	// 2d6+1: mean = 2*3.5+1 = 8, stdDev ≈ 2.42
	assert.InDelta(t, 8.0, baseDmg, 1.0, "legacy dice mean should be roughly 8")
	assert.True(t, variance > 0, "legacy dice variance should be positive")
}

// ─── GetDamage ──────────────────────────────────────────────────────────────

func TestGetDamage(t *testing.T) {
	dmg := Damage{
		Attacks:    1,
		BaseDamage: 30,
		Variance:   7,
	}
	item := Item{ItemId: 1, Spec: &ItemSpec{Damage: dmg}}

	got := item.GetDamage()
	assert.Equal(t, 1, got.Attacks)
	assert.Equal(t, 30, got.BaseDamage)
	assert.Equal(t, 7, got.Variance)
}
