package items

// SameStack returns true if a and b are interchangeable enough to occupy
// the same bank slot or the same inventory display row.
//
// Two items are the same stack when every per-instance field that
// differentiates them visually or mechanically is equal:
//   - ItemId
//   - Uses
//   - EnchantType
//   - EnchantTier
//   - BottleMultiplier (different bottles age potions at different rates)
//   - Spec override: if either item carries a custom Spec (enchant stat
//     mods, renamed items, etc.) the items are treated as distinct unless
//     their overrides are pointer-equal (same source object).
//
// Note: EnchantBonus and StatModsType are not yet separate fields on
// Item; their effect is captured by the Spec override check.
func SameStack(a, b Item) bool {
	if a.ItemId != b.ItemId {
		return false
	}
	if a.Uses != b.Uses {
		return false
	}
	if a.EnchantType != b.EnchantType {
		return false
	}
	if a.EnchantTier != b.EnchantTier {
		return false
	}
	// BottleMultiplier: treat zero and non-zero as distinct aging profiles.
	// Use a small epsilon for float equality.
	if !floatEq(a.BottleMultiplier, b.BottleMultiplier) {
		return false
	}
	// Custom Spec overrides carry enchant stat mods, renames, etc.
	// Items with any Spec override live in their own slot unless the
	// override pointer is identical (same runtime object — won't happen
	// in practice for separately-loaded items, so effectively: any
	// item with a Spec override never stacks with another item).
	if a.Spec != b.Spec {
		// Both nil is fine (handled above). One nil, one non-nil: not same stack.
		// Both non-nil but different pointers: also not same stack.
		return false
	}
	return true
}

// floatEq reports whether a and b are equal within a small epsilon.
func floatEq(a, b float64) bool {
	const eps = 1e-9
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < eps
}

// IsStackable reports whether two instances of this item spec can ever
// occupy the same bank slot. Returns true for types that have no
// meaningful per-instance state beyond what SameStack already checks.
//
// Eligible types:
//   - Items with IsComponent flag (crafting materials — iron ore, leather
//     strips, herbs, etc.)
//   - Food, Drink, Potion, Grenade, Ammo, Botanical
//
// Gear, weapons, quest items, and other types always return false —
// each occupies its own slot with Count == 1.
func (s *ItemSpec) IsStackable() bool {
	if s.IsComponent {
		return true
	}
	switch s.Type {
	case Food, Drink, Potion, Grenade, Ammo, Botanical:
		return true
	default:
		return false
	}
}
