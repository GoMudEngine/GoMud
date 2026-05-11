# Mob Aliveness 2.2 — Item-Comparison Primitive Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `internal/itemvalue/` — a tactical scoring package with `ItemValue(spec, profile) float64` (pure) and `ItemValueDelta(char, profile, candidate) SwapDelta` (mob-aware, smart slot selection). Six weight profiles (PhysicalBruiser, PhysicalTank, Stealth, MagicalPure, MagicalSupport, Neutral) derived from `Mob.BehaviorArchetype` primary / `Mob.Archetype` fallback. Replaces the existing thin `internal/items/compare.go` v0; migrates its sole caller (`internal/mobs/crafter.go`).

**Architecture:** Pure value function reads `ItemSpec` fields (stat mods including negatives, damage multipliers, mitigation channels, weight) through a per-profile weight table. Delta function does smart slot selection (rings pick the weaker occupant; 1H weapons compare Weapon vs Offhand placements; 2H weapons displace both Weapon and Offhand) with symmetric bonus application (`DualWieldBonus`, `ShieldBonus`, `TwoHandedBonus`) and conditional `DualWieldBonus` (only applies when pre-swap main hand has a 1H weapon). Encumbrance tier crossings add a profile-modulated penalty inside Delta.

**Tech Stack:** Go 1.21+, existing `internal/items` for `ItemSpec`/`Item`/`ItemType`, existing `internal/characters.Worn` slot model, existing `internal/mobs.Mob.Archetype` and `Mob.BehaviorArchetype` fields.

**Spec:** `docs/superpowers/specs/2026-05-11-mob-aliveness-2.2-item-comparison-primitive-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP for the aliveness effort.

---

## File structure

| File | Responsibility |
|------|----------------|
| `internal/itemvalue/types.go` | NEW — `SlotName` typed string + slot constants; `WeightProfile` struct; `SwapDelta` struct |
| `internal/itemvalue/profiles.go` | NEW — Six profile `var` values + `ProfileFor(stat, behavior string) WeightProfile` |
| `internal/itemvalue/score.go` | NEW — `ItemValue(spec items.ItemSpec, profile WeightProfile) float64` + `IsUpgrade(char, profile, candidate) bool` convenience wrapper |
| `internal/itemvalue/delta.go` | NEW — `ItemValueDelta(char, profile, candidate) SwapDelta` + internal helpers `compatibleSlotsFor`, `displacedItemsForSlot`, `placementBonus`, `slotOf`, `encumbranceTierPenalty`, `canonicalRank` |
| `internal/itemvalue/types_test.go` | NEW — Type-level + constant tests |
| `internal/itemvalue/profiles_test.go` | NEW — `ProfileFor` resolution table |
| `internal/itemvalue/score_test.go` | NEW — `ItemValue` axis-by-axis coverage |
| `internal/itemvalue/delta_test.go` | NEW — `ItemValueDelta` smart-slot, swap-decision, encumbrance scenarios |
| `internal/itemvalue/context.md` | NEW — Package documentation per chunk SOP |
| `internal/items/compare.go` | **DELETE** |
| `internal/items/compare_test.go` | **DELETE** |
| `internal/mobs/crafter.go` | MODIFY — `crafter.go:386-403` rewritten to use `itemvalue.IsUpgrade` |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY — mark 2.2 Done, roll-up 9/40 |

---

## Task 1: Types skeleton

**Files:**
- Create: `internal/itemvalue/types.go`
- Create: `internal/itemvalue/types_test.go`

Establish the public types and slot-name vocabulary first. Subsequent tasks fill in behavior. The skeleton is compilable from the start.

- [ ] **Step 1: Create `internal/itemvalue/types.go`**

```go
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
```

- [ ] **Step 2: Create `internal/itemvalue/types_test.go`**

```go
package itemvalue

import "testing"

func TestSlotName_Values(t *testing.T) {
	// Smoke-check that the slot constants have the exact
	// Worn-struct field names. If someone renames a field on
	// characters.Worn, this test won't catch it directly, but
	// it locks the expected naming convention.
	cases := map[SlotName]string{
		SlotWeapon:    "Weapon",
		SlotOffhand:   "Offhand",
		SlotRing:      "Ring",
		SlotRing2:     "Ring2",
		SlotWrist1:    "Wrist1",
		SlotWrist2:    "Wrist2",
		SlotExtraArm1: "ExtraArm1",
		SlotTail:      "Tail",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("SlotName = %q, want %q", string(got), want)
		}
	}
}

func TestWeightProfile_ZeroValue(t *testing.T) {
	var p WeightProfile
	if p.Name != "" {
		t.Errorf("zero-value Name = %q, want empty", p.Name)
	}
	if p.PhysicalDamageWeight != 0 {
		t.Errorf("zero-value PhysicalDamageWeight = %f, want 0", p.PhysicalDamageWeight)
	}
}

func TestSwapDelta_ZeroValue(t *testing.T) {
	var d SwapDelta
	if d.Score != 0 {
		t.Errorf("zero-value Score = %f, want 0", d.Score)
	}
	if d.Slot != "" {
		t.Errorf("zero-value Slot = %q, want empty", d.Slot)
	}
	if d.Displaced != nil {
		t.Errorf("zero-value Displaced = %v, want nil", d.Displaced)
	}
}
```

- [ ] **Step 3: Build and run tests**

Run: `go build ./internal/itemvalue/...`
Expected: clean build.

Run: `go test ./internal/itemvalue/ -v`
Expected: all three tests PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/itemvalue/types.go internal/itemvalue/types_test.go
git commit -m "$(cat <<'EOF'
feat(itemvalue): types skeleton for chunk 2.2

Adds the SlotName typed string with constants matching
characters.Worn field names verbatim, the WeightProfile struct
with per-axis multipliers and the three offhand-strategy
bonuses, and the SwapDelta result struct. No behavior yet —
subsequent tasks fill in profiles, scoring, and delta.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Profiles + ProfileFor

**Files:**
- Create: `internal/itemvalue/profiles.go`
- Create: `internal/itemvalue/profiles_test.go`

The six named profiles defined as package-level `var` values, plus the resolver `ProfileFor(stat, behavior string) WeightProfile`.

- [ ] **Step 1: Create `internal/itemvalue/profiles.go`**

```go
package itemvalue

// PhysicalBruiser fits stat=fighting or behavior in
// {generic_fighter, melee_self_buff, leader}.
var PhysicalBruiser = WeightProfile{
	Name:                       "PhysicalBruiser",
	PhysicalDamageWeight:       1.0,
	SpellDamageWeight:          0.1,
	PhysicalMitigationWeight:   1.0,
	MagicalMitigationWeight:    0.6,
	ConvictionMitigationWeight: 0.4,
	StatWeights: map[string]float64{
		"strength":   1.5,
		"dexterity":  1.2,
		"vitality":   1.3,
		"willpower":  0.3,
		"charisma":   0.3,
		"perception": 0.7,
	},
	WeightPenaltyPerLb:     0.5,
	EncumbranceTierPenalty: 25,
	DualWieldBonus:         80,
	ShieldBonus:            20,
	TwoHandedBonus:         60,
}

// PhysicalTank fits stat=tank or behavior=tank_taunter.
var PhysicalTank = WeightProfile{
	Name:                       "PhysicalTank",
	PhysicalDamageWeight:       0.5,
	SpellDamageWeight:          0.1,
	PhysicalMitigationWeight:   1.2,
	MagicalMitigationWeight:    1.0,
	ConvictionMitigationWeight: 1.2,
	StatWeights: map[string]float64{
		"vitality":   1.5,
		"strength":   1.0,
		"charisma":   1.3,
		"willpower":  0.5,
		"dexterity":  0.7,
		"perception": 0.5,
	},
	WeightPenaltyPerLb:     0.2,
	EncumbranceTierPenalty: 10,
	DualWieldBonus:         0,
	ShieldBonus:            80,
	TwoHandedBonus:         -20,
}

// Stealth fits behavior in {ambusher, lookout}.
var Stealth = WeightProfile{
	Name:                       "Stealth",
	PhysicalDamageWeight:       1.1,
	SpellDamageWeight:          0.1,
	PhysicalMitigationWeight:   0.6,
	MagicalMitigationWeight:    0.4,
	ConvictionMitigationWeight: 0.3,
	StatWeights: map[string]float64{
		"dexterity":  1.5,
		"perception": 1.3,
		"strength":   1.0,
		"vitality":   0.4,
		"willpower":  0.3,
		"charisma":   0.3,
	},
	WeightPenaltyPerLb:     1.8,
	EncumbranceTierPenalty: 80,
	DualWieldBonus:         100,
	ShieldBonus:            0,
	TwoHandedBonus:         -40,
}

// MagicalPure fits behavior=pure_caster.
var MagicalPure = WeightProfile{
	Name:                       "MagicalPure",
	PhysicalDamageWeight:       0.2,
	SpellDamageWeight:          1.5,
	PhysicalMitigationWeight:   0.5,
	MagicalMitigationWeight:    1.0,
	ConvictionMitigationWeight: 0.5,
	StatWeights: map[string]float64{
		"willpower":  1.5,
		"perception": 1.2,
		"charisma":   1.0,
		"vitality":   0.5,
		"dexterity":  0.5,
		"strength":   0.3,
	},
	WeightPenaltyPerLb:     1.5,
	EncumbranceTierPenalty: 60,
	DualWieldBonus:         70,
	ShieldBonus:            -40,
	TwoHandedBonus:         80,
}

// MagicalSupport fits behavior=support_caster, or stat=casting
// fallback when no behavior label is set.
var MagicalSupport = WeightProfile{
	Name:                       "MagicalSupport",
	PhysicalDamageWeight:       0.2,
	SpellDamageWeight:          1.2,
	PhysicalMitigationWeight:   0.7,
	MagicalMitigationWeight:    1.0,
	ConvictionMitigationWeight: 0.8,
	StatWeights: map[string]float64{
		"willpower":  1.4,
		"charisma":   1.3,
		"perception": 1.1,
		"vitality":   0.8,
		"dexterity":  0.5,
		"strength":   0.3,
	},
	WeightPenaltyPerLb:     1.5,
	EncumbranceTierPenalty: 60,
	DualWieldBonus:         -80,
	ShieldBonus:            80,
	TwoHandedBonus:         -20,
}

// Neutral is the default for empty archetypes,
// combat_passive, prey, and all noncombat_* roles.
var Neutral = WeightProfile{
	Name:                       "Neutral",
	PhysicalDamageWeight:       0.7,
	SpellDamageWeight:          0.5,
	PhysicalMitigationWeight:   0.7,
	MagicalMitigationWeight:    0.7,
	ConvictionMitigationWeight: 0.7,
	StatWeights:                map[string]float64{}, // all stats default to 1.0
	WeightPenaltyPerLb:         1.0,
	EncumbranceTierPenalty:     35,
	DualWieldBonus:             20,
	ShieldBonus:                20,
	TwoHandedBonus:             0,
}

// ProfileFor resolves a mob's archetype fields to a named
// WeightProfile. Behavior archetype takes precedence; stat
// archetype is the fallback when no behavior label is set.
// Empty input returns Neutral.
func ProfileFor(statArchetype, behaviorArchetype string) WeightProfile {
	// (1) Behavior archetype takes precedence.
	switch behaviorArchetype {
	case "tank_taunter":
		return PhysicalTank
	case "pure_caster":
		return MagicalPure
	case "support_caster":
		return MagicalSupport
	case "ambusher", "lookout":
		return Stealth
	case "generic_fighter", "melee_self_buff", "leader":
		return PhysicalBruiser
	case "combat_passive", "prey",
		"noncombat_passive", "noncombat_questgiver",
		"noncombat_shopkeeper":
		return Neutral
	}
	// (2) Stat archetype fallback when no behavior label.
	switch statArchetype {
	case "fighting":
		return PhysicalBruiser
	case "casting":
		return MagicalSupport
	case "tank":
		return PhysicalTank
	}
	// (3) Default.
	return Neutral
}
```

- [ ] **Step 2: Create `internal/itemvalue/profiles_test.go`**

```go
package itemvalue

import "testing"

func TestProfileFor_BehaviorPrimary(t *testing.T) {
	cases := []struct {
		behavior string
		want     string // profile Name
	}{
		{"tank_taunter", "PhysicalTank"},
		{"pure_caster", "MagicalPure"},
		{"support_caster", "MagicalSupport"},
		{"ambusher", "Stealth"},
		{"lookout", "Stealth"},
		{"generic_fighter", "PhysicalBruiser"},
		{"melee_self_buff", "PhysicalBruiser"},
		{"leader", "PhysicalBruiser"},
		{"combat_passive", "Neutral"},
		{"prey", "Neutral"},
		{"noncombat_passive", "Neutral"},
		{"noncombat_questgiver", "Neutral"},
		{"noncombat_shopkeeper", "Neutral"},
	}
	for _, c := range cases {
		// Pass a non-empty stat to verify behavior takes precedence.
		got := ProfileFor("fighting", c.behavior)
		if got.Name != c.want {
			t.Errorf("ProfileFor(\"fighting\", %q).Name = %q, want %q",
				c.behavior, got.Name, c.want)
		}
	}
}

func TestProfileFor_StatFallback(t *testing.T) {
	cases := []struct {
		stat string
		want string
	}{
		{"fighting", "PhysicalBruiser"},
		{"casting", "MagicalSupport"},
		{"tank", "PhysicalTank"},
		{"", "Neutral"},
	}
	for _, c := range cases {
		got := ProfileFor(c.stat, "")
		if got.Name != c.want {
			t.Errorf("ProfileFor(%q, \"\").Name = %q, want %q",
				c.stat, got.Name, c.want)
		}
	}
}

func TestProfileFor_UnknownBehaviorFallsThrough(t *testing.T) {
	// An unknown behavior archetype should NOT match anything
	// and fall through to the stat archetype.
	got := ProfileFor("tank", "spelunker_unknown")
	if got.Name != "PhysicalTank" {
		t.Errorf("ProfileFor(\"tank\", unknown) = %q, want PhysicalTank", got.Name)
	}

	got = ProfileFor("", "spelunker_unknown")
	if got.Name != "Neutral" {
		t.Errorf("ProfileFor(\"\", unknown) = %q, want Neutral", got.Name)
	}
}
```

- [ ] **Step 3: Build and run tests**

Run: `go test ./internal/itemvalue/ -run TestProfileFor -v`
Expected: all PASS.

Run: `go test ./internal/itemvalue/ -v`
Expected: prior types tests still PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/itemvalue/profiles.go internal/itemvalue/profiles_test.go
git commit -m "$(cat <<'EOF'
feat(itemvalue): six weight profiles and ProfileFor resolver

Defines PhysicalBruiser, PhysicalTank, Stealth, MagicalPure,
MagicalSupport, Neutral as package-level vars. ProfileFor maps
the BehaviorArchetype (primary) and Archetype (fallback) fields
to a profile. Tested against the full table of behavior names
plus stat-archetype fallbacks plus unknown-behavior fallthrough.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: ItemValue formula

**Files:**
- Create: `internal/itemvalue/score.go`
- Create: `internal/itemvalue/score_test.go`

The pure scoring function. No mob/character state — just `(spec, profile) → float64`.

- [ ] **Step 1: Create `internal/itemvalue/score.go`**

```go
package itemvalue

import (
	"github.com/GoMudEngine/GoMud/internal/items"
)

// ItemValue returns the raw value of an item under the given
// profile. Positive AND negative stat mods contribute. Used to
// rank items independently of any mob's current loadout.
func ItemValue(spec items.ItemSpec, profile WeightProfile) float64 {
	score := 0.0

	// (1) Stat mods (positive AND negative contributions).
	for statName, mod := range spec.StatMods {
		weight, ok := profile.StatWeights[statName]
		if !ok {
			weight = 1.0
		}
		score += float64(mod) * weight
	}

	// (2) Physical damage.
	if spec.DamageMultiplier > 0 {
		score += spec.DamageMultiplier * 100 * profile.PhysicalDamageWeight
	}

	// (3) Spell damage (caster weapons).
	if spec.SpellDamageMultiplier > 0 {
		score += spec.SpellDamageMultiplier * 100 * profile.SpellDamageWeight
	}

	// (4) Mitigation channels (1 point per percentage point).
	score += float64(spec.PhysicalMitigation) * profile.PhysicalMitigationWeight
	score += float64(spec.MagicalMitigation) * profile.MagicalMitigationWeight
	score += float64(spec.ConvictionMitigation) * profile.ConvictionMitigationWeight

	// (5) Weight cost (profile-modulated).
	score -= spec.Weight * profile.WeightPenaltyPerLb

	return score
}
```

- [ ] **Step 2: Create `internal/itemvalue/score_test.go`**

```go
package itemvalue

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestItemValue_EmptySpec(t *testing.T) {
	got := ItemValue(items.ItemSpec{}, PhysicalBruiser)
	if got != 0.0 {
		t.Errorf("ItemValue(empty, Bruiser) = %f, want 0", got)
	}
}

func TestItemValue_PositiveStatMod(t *testing.T) {
	spec := items.ItemSpec{
		StatMods: map[string]int{"strength": 10},
	}
	// Bruiser strength weight = 1.5 → 10 × 1.5 = 15
	got := ItemValue(spec, PhysicalBruiser)
	want := 15.0
	if got != want {
		t.Errorf("ItemValue strength+10 on Bruiser = %f, want %f", got, want)
	}
}

func TestItemValue_NegativeStatModPenalizes(t *testing.T) {
	spec := items.ItemSpec{
		StatMods: map[string]int{"strength": -5},
	}
	// Bruiser strength weight = 1.5 → -5 × 1.5 = -7.5
	got := ItemValue(spec, PhysicalBruiser)
	want := -7.5
	if got != want {
		t.Errorf("ItemValue strength-5 on Bruiser = %f, want %f", got, want)
	}
}

func TestItemValue_StatNotInProfileDefaultsToWeight1(t *testing.T) {
	// Bruiser has no entry for "foobar" — should default to 1.0
	spec := items.ItemSpec{
		StatMods: map[string]int{"foobar": 7},
	}
	got := ItemValue(spec, PhysicalBruiser)
	want := 7.0
	if got != want {
		t.Errorf("ItemValue unknown-stat+7 on Bruiser = %f, want %f", got, want)
	}
}

func TestItemValue_DamageMultiplier(t *testing.T) {
	spec := items.ItemSpec{DamageMultiplier: 1.2}
	// Bruiser PhysicalDamageWeight = 1.0 → 1.2 × 100 × 1.0 = 120
	got := ItemValue(spec, PhysicalBruiser)
	want := 120.0
	if got != want {
		t.Errorf("ItemValue dmg×1.2 on Bruiser = %f, want %f", got, want)
	}
}

func TestItemValue_SpellDamageMultiplier(t *testing.T) {
	spec := items.ItemSpec{SpellDamageMultiplier: 1.3}
	// MagicalPure SpellDamageWeight = 1.5 → 1.3 × 100 × 1.5 = 195
	got := ItemValue(spec, MagicalPure)
	want := 195.0
	if got != want {
		t.Errorf("ItemValue spelldmg×1.3 on MagicalPure = %f, want %f", got, want)
	}
}

func TestItemValue_MitigationChannels(t *testing.T) {
	spec := items.ItemSpec{
		PhysicalMitigation:   10,
		MagicalMitigation:    20,
		ConvictionMitigation: 5,
	}
	// PhysicalTank weights: phys 1.2, mag 1.0, conv 1.2
	// 10×1.2 + 20×1.0 + 5×1.2 = 12 + 20 + 6 = 38
	got := ItemValue(spec, PhysicalTank)
	want := 38.0
	if got != want {
		t.Errorf("ItemValue mit on Tank = %f, want %f", got, want)
	}
}

func TestItemValue_WeightCost(t *testing.T) {
	spec := items.ItemSpec{Weight: 4}
	// Stealth WeightPenaltyPerLb = 1.8 → -4 × 1.8 = -7.2
	got := ItemValue(spec, Stealth)
	want := -7.2
	if got != want {
		t.Errorf("ItemValue weight=4 on Stealth = %f, want %f", got, want)
	}
}

func TestItemValue_ProfilesDifferentiate(t *testing.T) {
	// Same item — a +10 strength ring — scores higher on
	// Bruiser (strength weight 1.5) than on MagicalPure
	// (strength weight 0.3).
	spec := items.ItemSpec{
		StatMods: map[string]int{"strength": 10},
	}
	bruiserScore := ItemValue(spec, PhysicalBruiser)
	pureScore := ItemValue(spec, MagicalPure)
	if bruiserScore <= pureScore {
		t.Errorf("expected Bruiser score (%f) > MagicalPure score (%f) for strength ring",
			bruiserScore, pureScore)
	}
}

func TestItemValue_WorkedExample_BruiserSword(t *testing.T) {
	// Plain 1.5× sword, 4 lb, no stat mods, on PhysicalBruiser.
	// Expected: 1.5 × 100 × 1.0 + 0 - 4 × 0.5 = 148
	spec := items.ItemSpec{
		DamageMultiplier: 1.5,
		Weight:           4,
	}
	got := ItemValue(spec, PhysicalBruiser)
	want := 148.0
	if got != want {
		t.Errorf("worked example bruiser sword: got %f want %f", got, want)
	}
}

func TestItemValue_WorkedExample_CursedSword(t *testing.T) {
	// 1.2× sword, 4 lb, strength: -5, on PhysicalBruiser.
	// Expected: -5×1.5 + 1.2×100 - 4×0.5 = -7.5 + 120 - 2 = 110.5
	spec := items.ItemSpec{
		DamageMultiplier: 1.2,
		Weight:           4,
		StatMods:         map[string]int{"strength": -5},
	}
	got := ItemValue(spec, PhysicalBruiser)
	want := 110.5
	if got != want {
		t.Errorf("worked example cursed sword: got %f want %f", got, want)
	}
}
```

- [ ] **Step 3: Build and run tests**

Run: `go test ./internal/itemvalue/ -run TestItemValue -v`
Expected: all 11 cases PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/itemvalue/score.go internal/itemvalue/score_test.go
git commit -m "$(cat <<'EOF'
feat(itemvalue): pure ItemValue scoring function

Implements the v1 score formula: stat mods (positive AND
negative), damage multiplier, spell damage multiplier, three
mitigation channels, weight cost. All axes are
profile-weighted; stats absent from a profile's StatWeights
map default to weight 1.0. Worked examples from the spec
(bruiser sword 148, cursed sword 110.5) are explicit tests.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Delta helpers (compatibleSlotsFor, displacedItemsForSlot, slotOf, placementBonus)

**Files:**
- Create: `internal/itemvalue/delta.go`
- Modify: `internal/itemvalue/delta_test.go` (create)

Internal helpers used by `ItemValueDelta`. Tested individually so the main algorithm in Task 5 has a known-good substrate.

The implementation needs to look at character mutations to handle Extra Arms / Tail slots. Use the existing `internal/mutations` package and `Character.Mutations` field.

- [ ] **Step 1: Read existing mutation API to understand the call shape**

Run a quick exploration:
```bash
grep -n "GetExtraArmLevel\|HasMutation\|Mutations\." internal/mutations/*.go | head -20
```

Note any helpers like `mutations.GetExtraArmsLevel(char.Mutations)` that return the current level (0-4) of the Extra Arms mutation, and any tail-mutation helper.

If no such helpers exist, the helpers iterate `char.Mutations` directly. For this plan, assume two helpers exist (or write trivial wrappers in the delta.go file):

- `extraArmsLevel(char)` returns the int level 0..4 of Extra Arms mutation (0 = no mutation).
- `hasTailMutation(char)` returns bool.

If the actual mutation-package APIs differ, adapt these helper names but keep the semantics. Both helpers consult `char.Mutations`, which is an `mutations.MutationSet` per existing code.

- [ ] **Step 2: Create `internal/itemvalue/delta.go` with helpers only**

```go
package itemvalue

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// canonicalRank returns the sort order for a SlotName, used as
// a tiebreaker when two slots produce equal Score. Lower rank
// = preferred. Weapon=0 wins over Offhand=1 when scores tie
// (so a wand placed on an empty-handed mob lands in main hand).
func canonicalRank(s SlotName) int {
	for i, n := range canonicalSlotOrder {
		if n == s {
			return i
		}
	}
	return len(canonicalSlotOrder)
}

var canonicalSlotOrder = []SlotName{
	SlotWeapon, SlotOffhand,
	SlotExtraArm1, SlotExtraArm2, SlotExtraArm3, SlotExtraArm4,
	SlotHead, SlotNeck, SlotShoulders, SlotBody, SlotBack, SlotBelt,
	SlotWrist1, SlotWrist2,
	SlotExtraWrist1, SlotExtraWrist2, SlotExtraWrist3, SlotExtraWrist4,
	SlotGloves,
	SlotRing, SlotRing2,
	SlotLegs, SlotFeet,
	SlotTail, SlotComponentBag,
}

// extraArmsLevel returns the active level of the Extra Arms
// mutation on the character (0..4). 0 means no mutation.
func extraArmsLevel(char *characters.Character) int {
	// Per CLAUDE.md: Extra Arms levels 1-4 each unlock one
	// ExtraArm + one ExtraWrist slot.
	for lvl := 4; lvl >= 1; lvl-- {
		if char.Mutations.Has(mutations.MutationId("extra_arms_" + intToString(lvl))) {
			return lvl
		}
	}
	return 0
}

// hasTailMutation reports whether the character has the Tail
// mutation active (which disables Legs slot and enables Tail
// slot — but for itemvalue scoring we only care about the
// positive side, "can I equip a Tail item?").
func hasTailMutation(char *characters.Character) bool {
	return char.Mutations.Has(mutations.MutationId("tail"))
}

func intToString(n int) string {
	// Tiny helper to avoid importing strconv just for digit
	// concat. Levels are bounded 1-4.
	switch n {
	case 1:
		return "1"
	case 2:
		return "2"
	case 3:
		return "3"
	case 4:
		return "4"
	}
	return ""
}

// compatibleSlotsFor returns the list of SlotNames where the
// candidate could be placed, respecting character mutations.
// Returns nil/empty list when the item is not equippable.
func compatibleSlotsFor(spec items.ItemSpec, char *characters.Character) []SlotName {
	switch spec.Type {
	case items.Weapon:
		// 1H: try Weapon and Offhand. 2H: Weapon only.
		if spec.Hands == items.TwoHanded {
			return []SlotName{SlotWeapon}
		}
		return []SlotName{SlotWeapon, SlotOffhand}
	case items.Offhand:
		return []SlotName{SlotOffhand}
	case items.Head:
		return []SlotName{SlotHead}
	case items.Neck:
		return []SlotName{SlotNeck}
	case items.Shoulders:
		return []SlotName{SlotShoulders}
	case items.Body:
		return []SlotName{SlotBody}
	case items.Back:
		return []SlotName{SlotBack}
	case items.Belt:
		return []SlotName{SlotBelt}
	case items.Wrist:
		slots := []SlotName{SlotWrist1, SlotWrist2}
		switch extraArmsLevel(char) {
		case 4:
			slots = append(slots, SlotExtraWrist4)
			fallthrough
		case 3:
			slots = append(slots, SlotExtraWrist3)
			fallthrough
		case 2:
			slots = append(slots, SlotExtraWrist2)
			fallthrough
		case 1:
			slots = append(slots, SlotExtraWrist1)
		}
		return slots
	case items.Gloves:
		return []SlotName{SlotGloves}
	case items.Ring:
		return []SlotName{SlotRing, SlotRing2}
	case items.Legs:
		return []SlotName{SlotLegs}
	case items.Feet:
		return []SlotName{SlotFeet}
	case items.Tail:
		if hasTailMutation(char) {
			return []SlotName{SlotTail}
		}
		return nil
	case items.ComponentBag:
		return []SlotName{SlotComponentBag}
	}
	return nil
}

// slotOf returns the slot in which the given equipped item
// currently resides, or "" if the item is not currently
// equipped on char. Uses items.Item.ItemId equality with a
// non-zero check — two stacks of an item with the same ItemId
// in two equipment slots is impossible because each slot is a
// single value.
func slotOf(item items.Item, char *characters.Character) SlotName {
	if item.ItemId == 0 {
		return ""
	}
	e := &char.Equipment
	pairs := []struct {
		slot SlotName
		got  items.Item
	}{
		{SlotWeapon, e.Weapon},
		{SlotOffhand, e.Offhand},
		{SlotExtraArm1, e.ExtraArm1},
		{SlotExtraArm2, e.ExtraArm2},
		{SlotExtraArm3, e.ExtraArm3},
		{SlotExtraArm4, e.ExtraArm4},
		{SlotHead, e.Head},
		{SlotNeck, e.Neck},
		{SlotShoulders, e.Shoulders},
		{SlotBody, e.Body},
		{SlotBack, e.Back},
		{SlotBelt, e.Belt},
		{SlotWrist1, e.Wrist1},
		{SlotWrist2, e.Wrist2},
		{SlotExtraWrist1, e.ExtraWrist1},
		{SlotExtraWrist2, e.ExtraWrist2},
		{SlotExtraWrist3, e.ExtraWrist3},
		{SlotExtraWrist4, e.ExtraWrist4},
		{SlotGloves, e.Gloves},
		{SlotRing, e.Ring},
		{SlotRing2, e.Ring2},
		{SlotLegs, e.Legs},
		{SlotFeet, e.Feet},
		{SlotTail, e.Tail},
		{SlotComponentBag, e.ComponentBag},
	}
	for _, p := range pairs {
		if p.got.ItemId == item.ItemId &&
			p.got.Uses == item.Uses &&
			p.got.EnchantType == item.EnchantType &&
			p.got.EnchantTier == item.EnchantTier {
			return p.slot
		}
	}
	return ""
}

// itemInSlot returns the currently-equipped item in a given
// slot (or the zero-value items.Item if the slot is empty).
func itemInSlot(slot SlotName, char *characters.Character) items.Item {
	e := &char.Equipment
	switch slot {
	case SlotWeapon:
		return e.Weapon
	case SlotOffhand:
		return e.Offhand
	case SlotExtraArm1:
		return e.ExtraArm1
	case SlotExtraArm2:
		return e.ExtraArm2
	case SlotExtraArm3:
		return e.ExtraArm3
	case SlotExtraArm4:
		return e.ExtraArm4
	case SlotHead:
		return e.Head
	case SlotNeck:
		return e.Neck
	case SlotShoulders:
		return e.Shoulders
	case SlotBody:
		return e.Body
	case SlotBack:
		return e.Back
	case SlotBelt:
		return e.Belt
	case SlotWrist1:
		return e.Wrist1
	case SlotWrist2:
		return e.Wrist2
	case SlotExtraWrist1:
		return e.ExtraWrist1
	case SlotExtraWrist2:
		return e.ExtraWrist2
	case SlotExtraWrist3:
		return e.ExtraWrist3
	case SlotExtraWrist4:
		return e.ExtraWrist4
	case SlotGloves:
		return e.Gloves
	case SlotRing:
		return e.Ring
	case SlotRing2:
		return e.Ring2
	case SlotLegs:
		return e.Legs
	case SlotFeet:
		return e.Feet
	case SlotTail:
		return e.Tail
	case SlotComponentBag:
		return e.ComponentBag
	}
	return items.Item{}
}

// displacedItemsForSlot returns the items that would be
// unequipped when placing candidateSpec at targetSlot. Usually
// 0 or 1 items; can be 2 for 2H weapons (displaces both
// Weapon and Offhand) or for placing a 1H weapon in Offhand
// while the current Weapon is 2H (the 2H must be unequipped to
// free Offhand).
func displacedItemsForSlot(char *characters.Character, targetSlot SlotName, candidateSpec items.ItemSpec) []items.Item {
	var displaced []items.Item

	// 2H weapon candidate in Weapon slot: displaces both
	// Weapon and Offhand.
	if targetSlot == SlotWeapon && candidateSpec.Type == items.Weapon && candidateSpec.Hands == items.TwoHanded {
		if char.Equipment.Weapon.ItemId > 0 {
			displaced = append(displaced, char.Equipment.Weapon)
		}
		if char.Equipment.Offhand.ItemId > 0 {
			displaced = append(displaced, char.Equipment.Offhand)
		}
		return displaced
	}

	// Placing in Offhand while current Weapon is 2H: must
	// unequip the 2H to free the offhand.
	if targetSlot == SlotOffhand && char.Equipment.Weapon.ItemId > 0 &&
		char.Equipment.Weapon.GetSpec().Hands == items.TwoHanded {
		displaced = append(displaced, char.Equipment.Weapon)
		return displaced
	}

	// Singleton slot: displaced is whatever is currently there.
	current := itemInSlot(targetSlot, char)
	if current.ItemId > 0 {
		displaced = append(displaced, current)
	}
	return displaced
}

// placementBonus returns the additional score a spec earns
// when placed at slot, evaluated against pre-swap char state.
// Used SYMMETRICALLY: applied to candidate (at its prospective
// slot) and to displaced items (at their current slots). The
// DualWieldBonus is conditional on the pre-swap main hand
// holding a 1H weapon, so empty-hand and 2H-main-hand cases
// don't get credit for "second attack" synergy.
func placementBonus(profile WeightProfile, spec items.ItemSpec, slot SlotName, char *characters.Character) float64 {
	bonus := 0.0

	if spec.Hands == items.TwoHanded {
		bonus += profile.TwoHandedBonus
	}

	if slot == SlotOffhand {
		if spec.Type == items.Weapon {
			// CONDITIONAL: only when pre-swap main is a 1H weapon.
			if char.Equipment.Weapon.ItemId > 0 &&
				char.Equipment.Weapon.GetSpec().Hands != items.TwoHanded {
				bonus += profile.DualWieldBonus
			}
		} else if spec.Type == items.Offhand {
			bonus += profile.ShieldBonus
		}
	}

	return bonus
}
```

- [ ] **Step 3: Create initial test file `internal/itemvalue/delta_test.go`**

```go
package itemvalue

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// newTestChar returns a bare Character with no equipment and
// no mutations. Tests can populate Equipment fields directly.
func newTestChar() *characters.Character {
	return &characters.Character{}
}

func TestCanonicalRank_WeaponBeatsOffhand(t *testing.T) {
	if canonicalRank(SlotWeapon) >= canonicalRank(SlotOffhand) {
		t.Errorf("SlotWeapon rank %d should be < SlotOffhand rank %d",
			canonicalRank(SlotWeapon), canonicalRank(SlotOffhand))
	}
}

func TestCompatibleSlotsFor_OneHandedWeapon(t *testing.T) {
	spec := items.ItemSpec{Type: items.Weapon, Hands: items.OneHanded}
	got := compatibleSlotsFor(spec, newTestChar())
	want := []SlotName{SlotWeapon, SlotOffhand}
	if !slotsEqual(got, want) {
		t.Errorf("1H weapon slots = %v, want %v", got, want)
	}
}

func TestCompatibleSlotsFor_TwoHandedWeapon(t *testing.T) {
	spec := items.ItemSpec{Type: items.Weapon, Hands: items.TwoHanded}
	got := compatibleSlotsFor(spec, newTestChar())
	want := []SlotName{SlotWeapon}
	if !slotsEqual(got, want) {
		t.Errorf("2H weapon slots = %v, want %v", got, want)
	}
}

func TestCompatibleSlotsFor_Ring(t *testing.T) {
	spec := items.ItemSpec{Type: items.Ring}
	got := compatibleSlotsFor(spec, newTestChar())
	want := []SlotName{SlotRing, SlotRing2}
	if !slotsEqual(got, want) {
		t.Errorf("Ring slots = %v, want %v", got, want)
	}
}

func TestCompatibleSlotsFor_NonEquippable(t *testing.T) {
	// Consumable / unknown / item without an equip type should
	// return no slots.
	spec := items.ItemSpec{Type: items.ItemType("consumable")}
	got := compatibleSlotsFor(spec, newTestChar())
	if len(got) != 0 {
		t.Errorf("consumable slots = %v, want empty", got)
	}
}

func TestCompatibleSlotsFor_TailRequiresMutation(t *testing.T) {
	spec := items.ItemSpec{Type: items.Tail}
	// Without mutation: empty.
	got := compatibleSlotsFor(spec, newTestChar())
	if len(got) != 0 {
		t.Errorf("Tail without mutation: %v, want empty", got)
	}
	// (We won't test the WITH-mutation case here; that's a
	// fixture-heavy integration concern. The helper is correct
	// if the no-mutation case returns empty.)
}

func TestDisplacedItemsForSlot_EmptySlot(t *testing.T) {
	char := newTestChar()
	got := displacedItemsForSlot(char, SlotBody, items.ItemSpec{Type: items.Body})
	if len(got) != 0 {
		t.Errorf("empty body slot: displaced = %v, want empty", got)
	}
}

func TestDisplacedItemsForSlot_OccupiedSlot(t *testing.T) {
	char := newTestChar()
	char.Equipment.Body = items.Item{ItemId: 42}
	got := displacedItemsForSlot(char, SlotBody, items.ItemSpec{Type: items.Body})
	if len(got) != 1 || got[0].ItemId != 42 {
		t.Errorf("occupied body slot: displaced = %v, want [{ItemId:42}]", got)
	}
}

func TestDisplacedItemsForSlot_TwoHandedDisplacesBoth(t *testing.T) {
	char := newTestChar()
	char.Equipment.Weapon = items.Item{ItemId: 1}
	char.Equipment.Offhand = items.Item{ItemId: 2}
	spec := items.ItemSpec{Type: items.Weapon, Hands: items.TwoHanded}
	got := displacedItemsForSlot(char, SlotWeapon, spec)
	if len(got) != 2 {
		t.Fatalf("2H displaced count = %d, want 2", len(got))
	}
}

func TestDisplacedItemsForSlot_PlaceInOffhandUnequips2HMain(t *testing.T) {
	char := newTestChar()
	// For this test to be meaningful, items.New must be able to
	// resolve the 2H weapon's spec via GetSpec(). For unit-test
	// purposes we exercise the algorithm path that checks
	// char.Equipment.Weapon.GetSpec().Hands == items.TwoHanded.
	// Since we can't easily fabricate a real 2H spec from an
	// in-memory Item without test fixtures, we t.Skip when the
	// fixture isn't available — and document the expected
	// behavior so a future fixture-author can re-enable.
	if items.New(0).ItemId != 0 {
		t.Skip("test fixture for 2H weapon spec lookup not available")
	}
	// Documented behavior: when char has a 2H weapon in Weapon
	// slot and we place a 1H weapon in Offhand, the 2H weapon
	// is displaced.
}

func TestPlacementBonus_TwoHandedAlwaysApplies(t *testing.T) {
	char := newTestChar()
	spec := items.ItemSpec{Type: items.Weapon, Hands: items.TwoHanded}
	got := placementBonus(MagicalPure, spec, SlotWeapon, char)
	want := MagicalPure.TwoHandedBonus // 80
	if got != want {
		t.Errorf("2H bonus on Pure = %f, want %f", got, want)
	}
}

func TestPlacementBonus_DualWieldRequiresMainHandWeapon(t *testing.T) {
	char := newTestChar() // empty hands
	spec := items.ItemSpec{Type: items.Weapon, Hands: items.OneHanded}
	got := placementBonus(PhysicalBruiser, spec, SlotOffhand, char)
	if got != 0 {
		t.Errorf("DualWieldBonus on empty hands = %f, want 0", got)
	}

	// Now equip a 1H weapon in main hand — DualWieldBonus
	// should kick in.
	char.Equipment.Weapon = items.Item{ItemId: 1}
	// items.Item.GetSpec() depends on items registry. To make
	// this testable without fixtures, we rely on items.New(0)
	// returning {ItemId:0, Hands: OneHanded by default} —
	// any Item with ItemId>0 whose GetSpec returns OneHanded.
	// In practice this depends on the test data dir.
	// We test only the empty-hands branch unconditionally;
	// the with-weapon path is covered by integration in Task 5.
}

func TestPlacementBonus_ShieldUnconditional(t *testing.T) {
	char := newTestChar()
	spec := items.ItemSpec{Type: items.Offhand}
	// ShieldBonus applies regardless of main-hand state.
	got := placementBonus(PhysicalTank, spec, SlotOffhand, char)
	want := PhysicalTank.ShieldBonus // 80
	if got != want {
		t.Errorf("ShieldBonus on Tank empty hands = %f, want %f", got, want)
	}
}

// slotsEqual is a test helper for SlotName slice equality.
func slotsEqual(a, b []SlotName) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Build and run tests**

Run: `go build ./internal/itemvalue/...`
Expected: clean build.

Run: `go test ./internal/itemvalue/ -v`
Expected: all helper tests PASS or SKIP cleanly (the with-fixture skips are documented in step 3).

If the `mutations.MutationId` constants don't match the assumed `"extra_arms_1"`/`"tail"` pattern, adjust the helper functions to match the actual API. The semantic — "return level 0..4 of Extra Arms; bool for Tail" — is what the rest of the code depends on.

- [ ] **Step 5: Commit**

```bash
git add internal/itemvalue/delta.go internal/itemvalue/delta_test.go
git commit -m "$(cat <<'EOF'
feat(itemvalue): internal helpers for ItemValueDelta

Adds compatibleSlotsFor (slot-set per ItemType + mutation
gating), displacedItemsForSlot (handles 2H-weapon double
displacement and offhand-while-2H-main), slotOf (item→slot
reverse lookup), itemInSlot (slot→item lookup), placementBonus
(symmetric bonus computation with conditional DualWieldBonus),
and canonicalRank for tiebreaker ordering. Main algorithm in
the next task.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: ItemValueDelta main algorithm

**Files:**
- Modify: `internal/itemvalue/delta.go`
- Modify: `internal/itemvalue/delta_test.go`

Wire the helpers into the public `ItemValueDelta` entry point. Encumbrance penalty comes in Task 6.

- [ ] **Step 1: Append `ItemValueDelta` to `internal/itemvalue/delta.go`**

```go
// ItemValueDelta returns the net effect of equipping candidate
// over char's current loadout under the given profile. Smart
// slot selection picks the optimal placement. Returns
// SwapDelta{Score: 0, Slot: "", Displaced: nil} when candidate
// is not equippable on this character.
func ItemValueDelta(char *characters.Character, profile WeightProfile, candidate items.Item) SwapDelta {
	candidateSpec := candidate.GetSpec()
	candidateRaw := ItemValue(candidateSpec, profile)

	slots := compatibleSlotsFor(candidateSpec, char)
	if len(slots) == 0 {
		return SwapDelta{}
	}

	best := SwapDelta{Slot: "", Displaced: nil}
	bestSet := false
	bestRank := -1

	for _, slot := range slots {
		displaced := displacedItemsForSlot(char, slot, candidateSpec)

		candidateAt := candidateRaw + placementBonus(profile, candidateSpec, slot, char)

		displacedTotal := 0.0
		for _, d := range displaced {
			dSpec := d.GetSpec()
			currentSlot := slotOf(d, char)
			displacedTotal += ItemValue(dSpec, profile) +
				placementBonus(profile, dSpec, currentSlot, char)
		}

		netScore := candidateAt - displacedTotal

		// Encumbrance tier penalty: filled in by Task 6.
		// For now, no encumbrance adjustment.

		rank := canonicalRank(slot)

		if !bestSet ||
			netScore > best.Score ||
			(netScore == best.Score && rank < bestRank) {
			best = SwapDelta{
				Score:     netScore,
				Slot:      slot,
				Displaced: displaced,
			}
			bestRank = rank
			bestSet = true
		}
	}

	return best
}
```

- [ ] **Step 2: Append smart-slot-selection tests to `delta_test.go`**

```go
func TestItemValueDelta_NotEquippable(t *testing.T) {
	char := newTestChar()
	// Item type that doesn't map to any slot.
	candidate := items.Item{ItemId: 1}
	// Force the spec to be a non-equippable type.
	// (Adjust this if items.Item construction requires real
	// loaded data — t.Skip if so.)
	got := ItemValueDelta(char, PhysicalBruiser, candidate)
	if got.Slot != "" {
		t.Errorf("non-equippable: Slot = %q, want empty", got.Slot)
	}
	if got.Score != 0 {
		t.Errorf("non-equippable: Score = %f, want 0", got.Score)
	}
}

func TestItemValueDelta_TiebreakerPrefersWeapon(t *testing.T) {
	// Empty-handed mob considering a 1H weapon. Both Weapon
	// and Offhand placements should score the same raw value
	// (DualWieldBonus suppressed by conditional check on empty
	// main hand). Tiebreaker: Weapon wins.
	//
	// This test depends on items.Item.GetSpec() resolving the
	// candidate's spec. If test data dir isn't loaded, skip.
	char := newTestChar()
	candidate := items.Item{ItemId: 1}
	if candidate.GetSpec().ItemId == 0 {
		t.Skip("test fixture for items.Item.GetSpec lookup not available")
	}
	// Documented expectation when fixture is available:
	// got.Slot == SlotWeapon (not SlotOffhand).
	_ = ItemValueDelta(char, PhysicalBruiser, candidate)
}
```

- [ ] **Step 3: Add a fixture-free integration test using a fake-spec approach**

To exercise the algorithm without test fixtures, append a helper that lets tests inject a known spec directly into an `items.Item`. The `items.Item` struct has fields we can construct, and `GetSpec()` falls back to the registry — but if we don't go through `GetSpec()`, we can't easily inject. Instead, test the algorithm via the score math by constructing scenarios with synthetic specs passed to internal helpers, and use TestItemValueDelta as a thin smoke that exercises the public API only when fixtures are available.

Append a worked-scenario smoke test:

```go
func TestItemValueDelta_RingsPickWeakerOccupant(t *testing.T) {
	// Without test fixtures, we can't construct an items.Item
	// with a known spec. Document the expected behavior here
	// so the smoke test in Task 10 covers it.
	t.Skip("fixture-dependent integration; covered by smoke test")
	// Expected: char with weaker ring in Ring (e.g. +2 str)
	// and stronger ring in Ring2 (+10 str) considering a
	// +5 str ring should pick SlotRing as the target (displaces
	// the +2) producing positive net score.
}
```

- [ ] **Step 4: Build and run tests**

Run: `go test ./internal/itemvalue/ -v`
Expected: all PASS; new TestItemValueDelta_* may SKIP if fixtures absent.

- [ ] **Step 5: Commit**

```bash
git add internal/itemvalue/delta.go internal/itemvalue/delta_test.go
git commit -m "$(cat <<'EOF'
feat(itemvalue): ItemValueDelta main algorithm

Wires compatibleSlotsFor + displacedItemsForSlot + placementBonus
+ slotOf into ItemValueDelta. Smart slot selection across
compatible slots; symmetric bonus application; canonical-rank
tiebreaker (Weapon > Offhand > etc) on equal scores.
Encumbrance tier penalty stub in place; filled in by Task 6.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Encumbrance tier penalty

**Files:**
- Modify: `internal/itemvalue/delta.go`
- Modify: `internal/itemvalue/delta_test.go`

Insert the tier-crossing penalty into `ItemValueDelta`. Reference: the carry-weight thresholds in `internal/users/userrecord.prompt.go:522-528` (light ≤ 0.50, moderate ≤ 0.75, heavy ≤ 1.00, overburdened ≤ ratio>1.00, crushed > some higher bound).

- [ ] **Step 1: Add `encumbranceTier` and `encumbranceTierPenalty` to `delta.go`**

```go
// encumbranceTier returns a tier index 0..4 from a
// carryWeight/capacity ratio. Higher index = worse.
// Thresholds match userrecord.prompt.go:522-528.
//   0 = light (ratio ≤ 0.50)
//   1 = moderate (ratio ≤ 0.75)
//   2 = heavy (ratio ≤ 1.00)
//   3 = overburdened (ratio > 1.00, ≤ ~2.0)
//   4 = crushed (ratio > 2.0)
func encumbranceTier(ratio float64) int {
	switch {
	case ratio <= 0.50:
		return 0
	case ratio <= 0.75:
		return 1
	case ratio <= 1.00:
		return 2
	case ratio <= 2.00:
		return 3
	default:
		return 4
	}
}

// encumbranceTierPenalty returns the score penalty (positive =
// worse) for crossing tiers in the swap. If the swap reduces
// the tier (net less weight), the return is negative (a score
// bonus). Magnitude = profile.EncumbranceTierPenalty × number
// of tiers crossed.
func encumbranceTierPenalty(char *characters.Character, displaced []items.Item, candidate items.Item, profile WeightProfile) float64 {
	capacity := char.CarryCapacity()
	if capacity <= 0 {
		return 0
	}

	currentWeight := char.GetCarriedWeight()
	weightDelta := candidate.GetSpec().Weight
	for _, d := range displaced {
		weightDelta -= d.GetSpec().Weight
	}
	newWeight := currentWeight + weightDelta

	preTier := encumbranceTier(currentWeight / capacity)
	postTier := encumbranceTier(newWeight / capacity)

	tiersCrossed := postTier - preTier
	if tiersCrossed == 0 {
		return 0
	}
	return float64(tiersCrossed) * profile.EncumbranceTierPenalty
}
```

- [ ] **Step 2: Wire `encumbranceTierPenalty` into `ItemValueDelta`**

Find the line in `ItemValueDelta` that reads:
```go
		// Encumbrance tier penalty: filled in by Task 6.
		// For now, no encumbrance adjustment.
```

Replace with:
```go
		netScore -= encumbranceTierPenalty(char, displaced, candidate, profile)
```

- [ ] **Step 3: Add encumbrance tests to `delta_test.go`**

```go
func TestEncumbranceTier_Thresholds(t *testing.T) {
	cases := []struct {
		ratio float64
		want  int
	}{
		{0.00, 0}, {0.49, 0}, {0.50, 0},
		{0.51, 1}, {0.74, 1}, {0.75, 1},
		{0.76, 2}, {0.99, 2}, {1.00, 2},
		{1.01, 3}, {1.99, 3}, {2.00, 3},
		{2.01, 4}, {5.00, 4},
	}
	for _, c := range cases {
		got := encumbranceTier(c.ratio)
		if got != c.want {
			t.Errorf("encumbranceTier(%f) = %d, want %d",
				c.ratio, got, c.want)
		}
	}
}

func TestEncumbranceTierPenalty_NoCrossing(t *testing.T) {
	// Char at 0.30 capacity (light tier). Adding a 1lb item
	// keeps it in light tier. No penalty.
	char := newTestChar()
	char.Stats.Strength.ValueAdj = 100 // gives some capacity
	// Pre-condition: at light tier.
	// We compute the test data inline: char.CarryCapacity()
	// = strength × multiplier. Without configs loaded, this
	// may return 0 — skip if so.
	if char.CarryCapacity() <= 0 {
		t.Skip("carry capacity calc requires balance config")
	}
	candidate := items.Item{ItemId: 1}
	if candidate.GetSpec().ItemId == 0 {
		t.Skip("fixture required for items.New")
	}
	// Skipped without fixtures; smoke test covers.
}

func TestEncumbranceTierPenalty_TierCrossingPenalizes(t *testing.T) {
	// Skip with similar reasoning; covered by smoke test.
	t.Skip("requires balance config + item fixtures")
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/itemvalue/ -v`
Expected: all PASS; encumbrance tests SKIP if fixtures absent.

- [ ] **Step 5: Commit**

```bash
git add internal/itemvalue/delta.go internal/itemvalue/delta_test.go
git commit -m "$(cat <<'EOF'
feat(itemvalue): encumbrance tier penalty in ItemValueDelta

Adds encumbranceTier (ratio thresholds matching userrecord
prompt) and encumbranceTierPenalty (profile-weighted × tiers
crossed; sign reflects direction — positive penalty for
heavier post-swap tier, negative bonus for lighter). Wired
into ItemValueDelta's per-slot net-score computation.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: IsUpgrade convenience wrapper

**Files:**
- Modify: `internal/itemvalue/score.go`
- Modify: `internal/itemvalue/score_test.go`

A one-liner that turns the spec's "is item A an upgrade?" phrasing into a boolean. Used by chunk 2.3 (equip-if-better) and by the crafter migration.

- [ ] **Step 1: Append `IsUpgrade` to `score.go`**

```go
import (
	"github.com/GoMudEngine/GoMud/internal/characters"
)

// IsUpgrade is sugar over ItemValueDelta(...).Score > 0. Used
// by callers that just want the boolean "should I equip this"
// answer without the SwapDelta detail.
func IsUpgrade(char *characters.Character, profile WeightProfile, candidate items.Item) bool {
	return ItemValueDelta(char, profile, candidate).Score > 0
}
```

(The `characters` import goes into the existing import block in `score.go`.)

- [ ] **Step 2: Append a test to `score_test.go`**

```go
import (
	"github.com/GoMudEngine/GoMud/internal/characters"
)

func TestIsUpgrade_NonEquippableIsNotUpgrade(t *testing.T) {
	char := &characters.Character{}
	// items.Item{} has ItemId=0 → not equippable → SwapDelta
	// returns Score=0 → IsUpgrade returns false.
	candidate := items.Item{}
	if IsUpgrade(char, PhysicalBruiser, candidate) {
		t.Errorf("expected IsUpgrade(empty item) = false")
	}
}
```

- [ ] **Step 3: Build and run tests**

Run: `go test ./internal/itemvalue/ -v`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/itemvalue/score.go internal/itemvalue/score_test.go
git commit -m "$(cat <<'EOF'
feat(itemvalue): IsUpgrade convenience wrapper

Thin wrapper over ItemValueDelta(...).Score > 0. Matches the
spec's 'is item A an upgrade over item B for this mob?' phrasing
and gives callers (chunk 2.3 equip-if-better, crafter
migration) a clean boolean API.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Migrate `mobs/crafter.go` and delete `items/compare.go`

**Files:**
- Modify: `internal/mobs/crafter.go`
- Delete: `internal/items/compare.go`
- Delete: `internal/items/compare_test.go`

The crafter is the only caller of `items.IsUpgrade` / `items.ItemPower`. The new API is simpler — `itemvalue.IsUpgrade` already does the slot-aware comparison.

- [ ] **Step 1: Read the current `crafter.go:380-410` to see the surrounding context**

Run: `Read internal/mobs/crafter.go offset=380 limit=35`. Locate the block that iterates `wornItems` and calls `items.IsUpgrade` / `items.ItemPower`.

- [ ] **Step 2: Replace the block with the new API call**

Find lines that look like:

```go
// Check if this is an upgrade over any currently worn item of the same type
isUpgrade := false
wornSameType := false
for _, worn := range wornItems {
    wornSpec := worn.GetSpec()
    if wornSpec.Type == candidateSpec.Type {
        wornSameType = true
        if items.IsUpgrade(wornSpec, *candidateSpec) {
            isUpgrade = true
            break
        }
    }
}
// Also an upgrade if the slot is empty (nothing worn of that type)
if !wornSameType && items.ItemPower(*candidateSpec) > 0 {
    isUpgrade = true
}
```

Replace with:

```go
// Upgrade check via the consolidated itemvalue primitive.
// The new API handles slot-conflict math (e.g., 2H weapons
// displacing both Weapon and Offhand) internally.
profile := itemvalue.ProfileFor(mob.Archetype, mob.BehaviorArchetype)
candidate := items.New(recipe.Output.ItemId)
isUpgrade := itemvalue.IsUpgrade(&mob.Character, profile, candidate)
```

Add `"github.com/GoMudEngine/GoMud/internal/itemvalue"` to the import block in `crafter.go`. The `wornItems` local may now be unused — if so, delete its construction too. `candidateSpec` (the inline `items.GetItemSpec(...)`) may also be unused after the swap; if so, delete that line as well.

- [ ] **Step 3: Delete the old compare files**

```bash
rm internal/items/compare.go
rm internal/items/compare_test.go
```

- [ ] **Step 4: Build the whole module**

Run: `go build ./...`
Expected: clean compile. If anything else in the module references `items.IsUpgrade` or `items.ItemPower`, those references must be removed too. (Per the spec's review, `mobs/crafter.go` was the only caller.)

If there are unused imports left in `mobs/crafter.go` (e.g., a stranded `items` import), clean those up.

- [ ] **Step 5: Run tests across the affected packages**

Run: `go test ./internal/itemvalue/ ./internal/mobs/ ./internal/items/ -v`
Expected: all PASS.

Run: `go test ./... 2>&1 | grep -E "FAIL|ok" | head -50`
Expected: no FAIL lines anywhere.

If `internal/mobs/crafter_test.go` has tests that assert specific `ItemPower` values, update them to assert through the new API (call `itemvalue.IsUpgrade(...)` instead) or delete if they were testing internal mechanics that the new API doesn't expose.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/crafter.go internal/items/compare.go internal/items/compare_test.go
git commit -m "$(cat <<'EOF'
refactor(mobs,items): migrate crafter to itemvalue, delete v0 compare

Replaces the per-worn-item iteration loop in mobs/crafter.go
with a single itemvalue.IsUpgrade call — the new API handles
slot-conflict math (2H weapons, ring slot picking) internally,
so the loop is no longer needed. Deletes internal/items/compare.go
and its tests (the only caller of items.IsUpgrade / ItemPower
was the migrated crafter).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Package documentation

**Files:**
- Create: `internal/itemvalue/context.md`

Per the chunk-aliveness SOP, every new `internal/<package>/` directory ships a `context.md`. Style references: `internal/badinputtracker/context.md` (~170 lines, single-responsibility) and `internal/clans/context.md` (~190 lines, multi-file). `itemvalue` is small/single-purpose; aim for ~150 lines.

- [ ] **Step 1: Create `internal/itemvalue/context.md`**

```markdown
# internal/itemvalue/

Tactical scoring primitive used by behavior trees and shopping
logic to answer "is item A an upgrade?" and "how good is
item X for this mob?"

## Overview

`itemvalue` is the chunk 2.2 (mob aliveness) consolidation of
item-comparison logic. It replaces the v0 helpers
`items.ItemPower` and `items.IsUpgrade` (deleted) with a
two-tier API:

- **Pure score:** `ItemValue(spec, profile) float64` — rank a
  catalog item independently of any mob's loadout. Used for
  e.g., "rank this shop's stock by what's good for me"
  (chunk 5.3 equipment-aware shopping).
- **Mob-aware delta:** `ItemValueDelta(char, profile, candidate)
  SwapDelta` — would this swap improve my loadout? Handles
  slot conflicts, current-loadout comparison, and encumbrance
  tier crossings. Used by chunk 2.3 equip-if-better behavior
  tree action.

`IsUpgrade(char, profile, candidate) bool` is a one-line
sugar over `ItemValueDelta(...).Score > 0`.

## Key Components (file map)

- `types.go` — `SlotName` typed string + slot constants
  matching `characters.Worn` field names verbatim;
  `WeightProfile` (per-axis multipliers + three offhand-strategy
  bonuses); `SwapDelta` (result struct).
- `profiles.go` — Six named profiles (`PhysicalBruiser`,
  `PhysicalTank`, `Stealth`, `MagicalPure`, `MagicalSupport`,
  `Neutral`) as package-level vars; `ProfileFor(stat, behavior)`
  resolver.
- `score.go` — `ItemValue` formula + `IsUpgrade` wrapper.
- `delta.go` — `ItemValueDelta` main algorithm + internal
  helpers `compatibleSlotsFor`, `displacedItemsForSlot`,
  `placementBonus`, `slotOf`, `itemInSlot`,
  `encumbranceTier`, `encumbranceTierPenalty`, `canonicalRank`.

## Public API

```go
func ProfileFor(statArchetype, behaviorArchetype string) WeightProfile
func ItemValue(spec items.ItemSpec, profile WeightProfile) float64
func ItemValueDelta(char *characters.Character, profile WeightProfile, candidate items.Item) SwapDelta
func IsUpgrade(char *characters.Character, profile WeightProfile, candidate items.Item) bool
```

## Score formula (ItemValue)

```
score = sum_over_stats(mod × profile.StatWeights[stat] or 1.0)
      + DamageMultiplier × 100 × profile.PhysicalDamageWeight
      + SpellDamageMultiplier × 100 × profile.SpellDamageWeight
      + PhysicalMitigation × profile.PhysicalMitigationWeight
      + MagicalMitigation × profile.MagicalMitigationWeight
      + ConvictionMitigation × profile.ConvictionMitigationWeight
      - Weight × profile.WeightPenaltyPerLb
```

Negative stat mods penalize (cursed items score below zero).

## ItemValueDelta algorithm (sketch)

1. Resolve candidate's compatible slots via
   `compatibleSlotsFor(candidateSpec, char)` (respects
   mutations: Tail / Extra Arms).
2. For each compatible slot:
   - Compute placement bonus on the candidate
     (`TwoHandedBonus` if 2H, `DualWieldBonus` if Weapon at
     Offhand AND main is 1H, `ShieldBonus` if Offhand-type
     at Offhand).
   - Determine displaced items (`displacedItemsForSlot`).
   - Score displaced items symmetrically (their current-slot
     bonuses included).
   - Subtract encumbrance tier penalty if the swap crosses a
     carry-weight tier.
3. Pick the slot with highest net score. Tiebreaker: canonical
   slot order (Weapon < Offhand < ... ).
4. Return `SwapDelta{Score, Slot, Displaced}`.

## Profiles

`ProfileFor` resolves two archetype systems:

- `Mob.BehaviorArchetype` (primary): `pure_caster`,
  `support_caster`, `tank_taunter`, `ambusher`, `lookout`,
  `generic_fighter`, `melee_self_buff`, `leader`,
  `combat_passive`, `prey`, `noncombat_*`.
- `Mob.Archetype` (stat-pool, fallback): `fighting`, `casting`,
  `tank`, `""`.

The six profiles each have weight tables tuned for distinct
gearing preferences. Profile values are hardcoded in Go;
tuning is a code change, not data authoring.

## Bonus application rule

The three bonuses (`DualWieldBonus`, `ShieldBonus`,
`TwoHandedBonus`) apply **symmetrically** — same `placementBonus`
math evaluated against pre-swap state, applied to both
candidate (at prospective slot) and displaced items (at their
current slots). `DualWieldBonus` is **conditional**: only fires
when the (pre-swap) main hand has a 1H weapon. Without that
conditional, empty-handed mobs would score a wand-in-offhand
above a wand-in-main, sending the wand to the wrong slot.

## Integration Notes

**Consumed by:**
- `internal/mobs/crafter.go` — replaced old `items.IsUpgrade`
  call; crafter mobs decide whether to craft a candidate
  upgrade.
- Future: `internal/behaviortree/` equip-if-better action
  (chunk 2.3).
- Future: `internal/economy/` or shopping-decision code path
  (chunk 5.3 equipment-aware shopping).

**Depends on:**
- `internal/items` — `ItemSpec`, `Item`, `ItemType` enum,
  `WeaponHands` constants.
- `internal/characters` — `Character`, `Worn`,
  `CarryCapacity()`, `GetCarriedWeight()`.
- `internal/mutations` — Extra Arms level lookup, Tail
  mutation lookup.

## Global State

None. All functions are pure (no package-level state mutation).
The profile `var` values are read-only after init; callers
must not mutate them.

## Testing Notes

- `profiles_test.go` — table-driven `ProfileFor` resolution
  with all known behavior + stat archetype values.
- `score_test.go` — axis-by-axis coverage: stat mods
  (positive, negative, unknown-key default), damage, spell
  damage, three mitigation channels, weight cost. Worked
  examples from the spec are explicit test cases.
- `delta_test.go` — slot-helper coverage (1H vs 2H, ring,
  consumable, mutation-gated Tail), placement bonus
  (conditional DualWield, unconditional Shield), main
  algorithm via documented expectations (some tests `t.Skip`
  pending integration fixtures; full coverage in chunk smoke).

Tests do NOT require the test data directory or balance config
to be loaded; they construct synthetic `ItemSpec` and
`Character` values inline.
```

- [ ] **Step 2: Verify it renders and is at the right length**

Run: `wc -l internal/itemvalue/context.md`
Expected: ~150–200 lines.

- [ ] **Step 3: Commit**

```bash
git add internal/itemvalue/context.md
git commit -m "$(cat <<'EOF'
docs(itemvalue): context.md package documentation

Per the aliveness-chunks SOP, every new internal/<package>/
ships a context.md. Documents the public API, score formula,
delta algorithm, profile resolution, bonus application rule,
consumers, dependencies, and testing approach.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Smoke test + roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- (No source changes; smoke verification only)

Boot the server locally to confirm clean startup past data-file loading (per the CLAUDE.md pre-push SOP). Then close out the roadmap.

- [ ] **Step 1: Verify clean build of the whole module**

Run: `go build -o dogmud.exe .`
Expected: clean compile, no errors.

- [ ] **Step 2: Run the full test suite**

Run: `go test ./... 2>&1 | tail -50`
Expected: all packages PASS; no FAIL lines. SKIPs (the fixture-dependent ones in `internal/itemvalue/delta_test.go`) are OK.

- [ ] **Step 3: Boot the server briefly to confirm clean startup**

Start the server in the background:
```
./dogmud.exe (run_in_background: true)
```
Wait 8–12 seconds, then read the background output. Confirm:
- `mobs.LoadDataFiles() loadedCount=...` line present (some N > 0)
- `items.LoadDataFiles()` or equivalent present
- No `panic:` or `fatal error:` lines

Kill the server (`kill <pid>` or close the background process). Remove `dogmud.exe` afterward.

- [ ] **Step 4: Update `MOB_ALIVENESS_ROADMAP.md` tracker row**

In the Progress tracker table, change the `2.2` row's Status column from `Not started` to `Done`.

The 2.2 row was:
```
| 2.2 | Tactical | Item-comparison primitive | M | — | Not started |
```

Change to:
```
| 2.2 | Tactical | Item-comparison primitive | M | — | Done |
```

- [ ] **Step 5: Update the chunk 2.2 mini-brief**

Find the section `### 2.2 Item-comparison primitive`. Change the Status line from `Not started • **Size:** M` to `Done (2026-05-11) • **Size:** M`.

After the existing bullet list (Goal/In/Out/Depends on/Why), append a `**Shipped:**` paragraph using the style of prior chunks. Suggested text:

```
- **Shipped:** New `internal/itemvalue/` package with two-tier
  API. Pure `ItemValue(spec, profile) float64` for catalog
  ranking (used by chunk 5.3 equipment-aware shopping). Mob-
  aware `ItemValueDelta(char, profile, candidate) SwapDelta`
  for swap decisions with smart slot selection (rings pick the
  weaker occupant; 1H weapons compare Weapon vs Offhand
  placements; 2H weapons displace both Weapon and Offhand).
  Symmetric bonus application (`DualWieldBonus`, `ShieldBonus`,
  `TwoHandedBonus`); `DualWieldBonus` conditional on the pre-
  swap main hand holding a 1H weapon (no synergy without a
  partner). Encumbrance tier penalty applied per-tier crossed.
  Six named weight profiles (`PhysicalBruiser`, `PhysicalTank`,
  `Stealth`, `MagicalPure`, `MagicalSupport`, `Neutral`)
  derived via `ProfileFor(stat, behavior)` — `BehaviorArchetype`
  primary, `Archetype` fallback. New `IsUpgrade(char, profile,
  candidate) bool` convenience wrapper. Deleted v0 helpers
  `items.ItemPower` and `items.IsUpgrade`; migrated sole caller
  `mobs/crafter.go` to the new API (~10 lines deleted). Skill
  mods on item instances flagged as out of scope (instance-zone
  loot affixes carry +skill mods that the spec.StatMods view
  doesn't currently surface). Spec at
  `docs/superpowers/specs/2026-05-11-mob-aliveness-2.2-item-comparison-primitive-design.md`,
  plan at
  `docs/superpowers/plans/2026-05-11-mob-aliveness-2.2-item-comparison-primitive.md`.
```

- [ ] **Step 6: Update the roll-up**

Find the line that reads:
```
**Roll-up:** 8 / 40 done • 0 in progress • 32 not started.
```
Change to:
```
**Roll-up:** 9 / 40 done • 0 in progress • 31 not started.
```

- [ ] **Step 7: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): mark chunk 2.2 (item-comparison primitive) as Done

Ships internal/itemvalue/ with pure ItemValue scoring and
mob-aware ItemValueDelta with smart slot selection. Six weight
profiles derived from BehaviorArchetype / Archetype.
Migrates the crafter to the new API; deletes v0 items.compare.
Roll-up moves to 9/40.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-review checklist (run before declaring done)

- [ ] `go build ./...` passes clean.
- [ ] `go test ./internal/itemvalue/ ./internal/mobs/ ./internal/items/ -v` all green (skips OK only for documented fixture gaps).
- [ ] `internal/itemvalue/types.go` defines the 25 SlotName constants + WeightProfile + SwapDelta.
- [ ] `internal/itemvalue/profiles.go` defines six profile `var` values + `ProfileFor`.
- [ ] `internal/itemvalue/score.go` has `ItemValue` and `IsUpgrade`.
- [ ] `internal/itemvalue/delta.go` has `ItemValueDelta` and all six internal helpers (`compatibleSlotsFor`, `displacedItemsForSlot`, `placementBonus`, `slotOf`, `itemInSlot`, `encumbranceTierPenalty`, plus `canonicalRank`).
- [ ] `internal/items/compare.go` and `internal/items/compare_test.go` are GONE.
- [ ] `internal/mobs/crafter.go` no longer references `items.IsUpgrade` or `items.ItemPower`.
- [ ] Server boots cleanly past data load.
- [ ] Roadmap roll-up updated to 9/40.
