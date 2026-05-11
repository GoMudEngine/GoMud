package itemvalue

import (
	"github.com/GoMudEngine/GoMud/internal/items"
)

// SlotName identifies a specific equipment field on
// characters.Worn. Values match the Worn struct field names
// verbatim. Unlike items.ItemType (which can map to multiple
// slots, e.g., ItemType=Ring maps to both SlotRing and
// SlotRing2), a SlotName is unambiguous.
type SlotName string

const (
	SlotWeapon       SlotName = "Weapon"
	SlotOffhand      SlotName = "Offhand"
	SlotExtraArm1    SlotName = "ExtraArm1"
	SlotExtraArm2    SlotName = "ExtraArm2"
	SlotExtraArm3    SlotName = "ExtraArm3"
	SlotExtraArm4    SlotName = "ExtraArm4"
	SlotHead         SlotName = "Head"
	SlotNeck         SlotName = "Neck"
	SlotShoulders    SlotName = "Shoulders"
	SlotBody         SlotName = "Body"
	SlotBack         SlotName = "Back"
	SlotBelt         SlotName = "Belt"
	SlotWrist1       SlotName = "Wrist1"
	SlotWrist2       SlotName = "Wrist2"
	SlotExtraWrist1  SlotName = "ExtraWrist1"
	SlotExtraWrist2  SlotName = "ExtraWrist2"
	SlotExtraWrist3  SlotName = "ExtraWrist3"
	SlotExtraWrist4  SlotName = "ExtraWrist4"
	SlotGloves       SlotName = "Gloves"
	SlotRing         SlotName = "Ring"
	SlotRing2        SlotName = "Ring2"
	SlotLegs         SlotName = "Legs"
	SlotFeet         SlotName = "Feet"
	SlotTail         SlotName = "Tail"
	SlotComponentBag SlotName = "ComponentBag"
)

// WeightProfile defines per-axis multipliers used to score
// items for a given archetype/role. Constructed via ProfileFor.
type WeightProfile struct {
	Name string

	// Damage axes (applied to DamageMultiplier × 100 and
	// SpellDamageMultiplier × 100).
	PhysicalDamageWeight float64
	SpellDamageWeight    float64

	// Mitigation axes (per percentage point).
	PhysicalMitigationWeight   float64
	MagicalMitigationWeight    float64
	ConvictionMitigationWeight float64

	// StatWeights overrides the default weight of 1.0 per stat
	// point. Keys are lowercase stat names ("strength",
	// "dexterity", "vitality", "perception", "willpower",
	// "charisma"). Stats absent from this map default to 1.0.
	// Negative weights are allowed.
	StatWeights map[string]float64

	// Static weight (in lb) cost — applied in ItemValue.
	WeightPenaltyPerLb float64

	// Contextual penalty applied in ItemValueDelta when the
	// swap pushes the buyer's carry weight past a tier
	// threshold (light → moderate → heavy → overburdened →
	// crushed).
	EncumbranceTierPenalty float64

	// Offhand-strategy bonuses, applied symmetrically to both
	// the candidate (at its prospective slot) and any
	// displaced items (at their current slots).
	DualWieldBonus float64 // Weapon placed in Offhand (CONDITIONAL on pre-swap main hand having a 1H weapon)
	ShieldBonus    float64 // Offhand-type placed in Offhand
	TwoHandedBonus float64 // 2H Weapon candidate
}

// SwapDelta is the result of considering equipping a candidate
// over the character's current loadout.
type SwapDelta struct {
	Score     float64      // net value change (gain - sum of displaced values - encumbrance penalty)
	Slot      SlotName     // chosen target slot ("" if not equippable)
	Displaced []items.Item // items unequipped to make room (0, 1, or 2)
}
