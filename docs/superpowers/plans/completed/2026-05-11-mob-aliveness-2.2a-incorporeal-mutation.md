# Mob Aliveness 2.2a — Incorporeal Mutation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a new rarest mutation (`incorporeal`) that models ethereal beings — wraiths, spectres, fire and air elementals, the elemental queen. Adds two new mutation effect types (`gear_effectiveness_loss` and `physical_defense_bonus`), wires them into five consumer sites (character stat aggregation, three mitigation getters, weapon damage, defense roll resolution, itemvalue scoring), and tags the five mob template categories that should be ethereal today.

**Architecture:** Single uniform `gear_effectiveness_loss` effect (raw-level multiplied: 0.25/0.50/0.75/1.00 across ranks 1-4) reduces every gear-derived value (stat mods, weapon damage, mitigation, itemvalue score) at every consumer site. A new `physical_defense_bonus` effect (standard `LevelMultiplier` scaling) adds points to defender's roll margin for physical-channel attacks. Pre-existing fallbacks (e.g., unarmed-damage substitution when weapon multiplier is ≤0) correctly handle the rank-4 "no gear" case without explicit zero-out logic. Five mob YAML templates get `mutations: { incorporeal: 4 }` added.

**Tech Stack:** Go 1.21+, existing `internal/mutations` effect plumbing, existing `internal/combat` damage pipeline + best-of-all defense resolution, existing `internal/itemvalue` scoring (chunk 2.2).

**Spec:** `docs/superpowers/specs/2026-05-11-mob-aliveness-2.2a-incorporeal-mutation-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP for the aliveness effort.

---

## File structure

| File | Status | Responsibility |
|------|--------|----------------|
| `_datafiles/world/dogmud/mutations/incorporeal.yaml` | NEW | The mutation definition |
| `internal/mutations/mutations.go` | MODIFY | `GetGearEffectivenessLoss`, `GearEffectivenessMultiplier`, `GetPhysicalDefenseBonus` helpers |
| `internal/mutations/mutations_test.go` | MODIFY | Tests for the three new helpers + conflict verification |
| `internal/itemvalue/delta.go` | MODIFY | `ItemValueDelta` multiplies scores by gear-effectiveness multiplier |
| `internal/itemvalue/delta_test.go` | MODIFY | Test rank-4 incorporeal scores 0 |
| `internal/characters/character.go` | MODIFY | `Recalculate()` scales gear-derived stats; three `Get*Mitigation()` methods scale gear-derived mit |
| `internal/characters/character_test.go` | MODIFY | Tests for scaled stat + mit aggregation |
| `internal/combat/combat_helpers.go` | MODIFY | `buildWeaponSetup` applies gear multiplier to `weaponDmgMult`; defense-margin resolution adds `physical_defense_bonus` for physical attacks |
| `internal/combat/combat_helpers_test.go` (or sibling) | MODIFY | Tests for weapon-damage scaling + defense-margin bonus |
| `internal/combat/calculations.go` | MODIFY | Spell damage in `magAtk` line 90 scales by gear-effectiveness multiplier |
| Mob YAML templates (5 categories) | MODIFY | Add `mutations: { incorporeal: 4 }` |
| `_datafiles/world/dogmud/templates/help/mutations.template` | MODIFY | Append Incorporeal entry |
| `internal/mutations/context.md` | MODIFY | Document new effect types + raw-level carve-out |
| `internal/itemvalue/context.md` | MODIFY | Note gear-effectiveness application in `ItemValueDelta` |
| `internal/characters/context.md` | MODIFY | Note `Recalculate` and `Get*Mitigation` apply gear multiplier |
| `MOB_ALIVENESS_ROADMAP.md` | MODIFY | Insert chunk 2.2a; bump tracker total to 41; roll-up 10/41 |

---

## Task 1: Mutation helpers + tests

**Files:**
- Modify: `internal/mutations/mutations.go`
- Modify: `internal/mutations/mutations_test.go`

Add the three new helper functions. Use TDD — write the tests first against the in-memory mutation registry (no YAML needed yet; Task 2 ships the YAML, but Task 1's tests use a synthetic registry entry).

- [ ] **Step 1: Append the three helpers to `mutations.go`**

Find the existing block of helper functions starting around line 350 (after `GetStatFlat`, `GetNaturalArmor`, etc.). Append:

```go
// GetGearEffectivenessLoss returns the total fraction (0.0–1.0)
// by which all equipment effects (stat mods, weapon damage,
// mitigation values) should be reduced for this character.
// Computed with RAW level multiplication — does NOT call
// LevelMultiplier. This produces a linear progression across
// ranks 1-4 of 0.25/0.50/0.75/1.00, matching the design intent
// for percentage-loss effects. Clamped to [0.0, 1.0].
func GetGearEffectivenessLoss(owned map[string]int) float64 {
	loss := 0.0
	for id, level := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		// Either side of the pros/cons split can declare this
		// effect; the type name implies a downside but we sum
		// both for completeness.
		for _, c := range spec.Cons {
			if c.Type == "gear_effectiveness_loss" {
				loss += c.Value * float64(level)
			}
		}
		for _, p := range spec.Pros {
			if p.Type == "gear_effectiveness_loss" {
				loss += p.Value * float64(level)
			}
		}
	}
	if loss < 0 {
		loss = 0
	} else if loss > 1 {
		loss = 1
	}
	return loss
}

// GearEffectivenessMultiplier returns the multiplier consumers
// apply to gear-derived values (1.0 = full effectiveness, 0.0 = none).
// Convenience wrapper over GetGearEffectivenessLoss.
func GearEffectivenessMultiplier(owned map[string]int) float64 {
	return 1.0 - GetGearEffectivenessLoss(owned)
}

// GetPhysicalDefenseBonus returns the total bonus added to the
// defender's roll margin for physical-channel attacks. Uses
// standard LevelMultiplier scaling.
func GetPhysicalDefenseBonus(owned map[string]int) float64 {
	return sumEffects(owned, "physical_defense_bonus", "")
}
```

- [ ] **Step 2: Add tests to `mutations_test.go`**

The tests need synthetic mutations injected into the in-memory registry so they don't depend on the YAML file. Find where existing tests construct test mutations (or the test setup helpers) and follow the same pattern. If no helper exists, manipulate `allMutations` directly via a test-only helper:

```go
func TestGetGearEffectivenessLoss_PerRank(t *testing.T) {
	// Inject a synthetic mutation for the test.
	prev := allMutations
	defer func() { allMutations = prev }()
	allMutations = map[string]*MutationSpec{
		"test-incorporeal": {
			MutationId: "test-incorporeal",
			Name:       "Test Incorporeal",
			Rarity:     10,
			Cons: []MutationEffect{
				{Type: "gear_effectiveness_loss", Target: "", Value: 0.25},
			},
		},
	}

	cases := []struct {
		level int
		want  float64
	}{
		{1, 0.25},
		{2, 0.50},
		{3, 0.75},
		{4, 1.00},
	}
	for _, c := range cases {
		owned := map[string]int{"test-incorporeal": c.level}
		got := GetGearEffectivenessLoss(owned)
		if got != c.want {
			t.Errorf("level %d: got %.2f, want %.2f", c.level, got, c.want)
		}
	}
}

func TestGetGearEffectivenessLoss_Clamping(t *testing.T) {
	prev := allMutations
	defer func() { allMutations = prev }()
	allMutations = map[string]*MutationSpec{
		"a": {
			MutationId: "a", Name: "A", Rarity: 1,
			Cons: []MutationEffect{
				{Type: "gear_effectiveness_loss", Value: 0.50},
			},
		},
		"b": {
			MutationId: "b", Name: "B", Rarity: 1,
			Cons: []MutationEffect{
				{Type: "gear_effectiveness_loss", Value: 0.50},
			},
		},
	}
	// Two sources each at level 2 → 0.50×2 + 0.50×2 = 2.0 → clamped to 1.0
	owned := map[string]int{"a": 2, "b": 2}
	got := GetGearEffectivenessLoss(owned)
	if got != 1.0 {
		t.Errorf("expected clamp to 1.0, got %.2f", got)
	}
}

func TestGetGearEffectivenessLoss_NoEffect(t *testing.T) {
	got := GetGearEffectivenessLoss(map[string]int{})
	if got != 0.0 {
		t.Errorf("expected 0.0 for empty owned, got %.2f", got)
	}
}

func TestGearEffectivenessMultiplier_Inverts(t *testing.T) {
	prev := allMutations
	defer func() { allMutations = prev }()
	allMutations = map[string]*MutationSpec{
		"x": {
			MutationId: "x", Name: "X", Rarity: 1,
			Cons: []MutationEffect{
				{Type: "gear_effectiveness_loss", Value: 0.25},
			},
		},
	}
	for level := 0; level <= 4; level++ {
		owned := map[string]int{}
		if level > 0 {
			owned["x"] = level
		}
		mul := GearEffectivenessMultiplier(owned)
		loss := GetGearEffectivenessLoss(owned)
		if mul+loss != 1.0 {
			t.Errorf("level %d: mul %.2f + loss %.2f != 1.0", level, mul, loss)
		}
	}
}

func TestGetPhysicalDefenseBonus_PerRank(t *testing.T) {
	prev := allMutations
	defer func() { allMutations = prev }()
	allMutations = map[string]*MutationSpec{
		"y": {
			MutationId: "y", Name: "Y", Rarity: 1,
			Pros: []MutationEffect{
				{Type: "physical_defense_bonus", Value: 15},
			},
		},
	}
	// LevelMultiplier curve: 1.0/1.5/2.0/2.5
	cases := []struct {
		level int
		want  float64
	}{
		{1, 15.0},
		{2, 22.5},
		{3, 30.0},
		{4, 37.5},
	}
	for _, c := range cases {
		owned := map[string]int{"y": c.level}
		got := GetPhysicalDefenseBonus(owned)
		if got != c.want {
			t.Errorf("level %d: got %.2f, want %.2f", c.level, got, c.want)
		}
	}
}
```

If the existing test helpers in `mutations_test.go` use a different injection pattern, conform to that pattern (e.g., a `setupTestMutations(t, ...)` helper, or YAML loading from `testdata/`).

- [ ] **Step 3: Run the tests**

```
go test ./internal/mutations/ -run 'TestGetGearEffectivenessLoss|TestGearEffectivenessMultiplier|TestGetPhysicalDefenseBonus' -v
```

Expected: all PASS.

- [ ] **Step 4: Run the full mutations test suite to confirm no regressions**

```
go test ./internal/mutations/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/mutations.go internal/mutations/mutations_test.go
git commit -m "$(cat <<'EOF'
feat(mutations): add gear_effectiveness_loss + physical_defense_bonus helpers

GetGearEffectivenessLoss uses raw level multiplication (skips
LevelMultiplier curve) to produce a linear 0.25/0.50/0.75/1.00
progression across ranks 1-4 — the right shape for
percentage-loss effects. GearEffectivenessMultiplier is the
1.0-loss convenience inverter. GetPhysicalDefenseBonus uses the
standard LevelMultiplier curve (diminishing returns fit scaling
defense semantics).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Mutation YAML file

**Files:**
- Create: `_datafiles/world/dogmud/mutations/incorporeal.yaml`

Ship the actual mutation data. No code change in this task.

- [ ] **Step 1: Create the YAML file**

```yaml
mutationid: incorporeal
name: Incorporeal
description: |
  Your form has slipped between planes — flesh, bone, and sinew
  giving way to something less tangible. Physical attacks struggle
  to find purchase against your shifting form, but the same
  ethereality that protects you weakens your constitution. Worn
  armor and wielded weapons grow less effective the further you
  drift from the physical world. At the deepest rank, gear becomes
  ornamental — passing through your form as if it weren't there.
rarity: 10
visual: Their form shimmers and grows translucent, edges blurring as if not entirely present.
conflicts:
  - extra-arms
  - extra-legs
  - clawed-hands
  - tail
  - dense-muscles
  - hollow-bones
  - healing-gel
pros:
  - type: stat_flat
    target: willpower
    value: 5
  - type: physical_defense_bonus
    target: ""
    value: 15
cons:
  - type: stat_flat
    target: vitality
    value: -10
  - type: gear_effectiveness_loss
    target: ""
    value: 0.25
```

- [ ] **Step 2: Build to confirm no startup panic**

```
go build ./...
```

Expected: clean compile.

- [ ] **Step 3: Boot the server briefly and watch loader output**

```
./dogmud.exe (background)
```

Wait ~10 seconds, read the background output. Confirm:
- `mutations.LoadMutationFiles()` line shows `loadedCount=` incremented by 1 vs the pre-change count
- No panic / fatal error

Kill the server. Remove `dogmud.exe`.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mutations/incorporeal.yaml
git commit -m "$(cat <<'EOF'
feat(mutations): add incorporeal mutation YAML

New rarest mutation (rarity 10) for ethereal beings. Four ranks
scale gear loss (0.25/level, raw multiplication), vitality
penalty, willpower bonus, and physical defense bonus. Conflict
list: extra-arms, extra-legs, clawed-hands, tail, dense-muscles,
hollow-bones, healing-gel — all body-dependent mutations that
don't make sense on an ethereal form.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: itemvalue gear-effectiveness integration

**Files:**
- Modify: `internal/itemvalue/delta.go`
- Modify: `internal/itemvalue/delta_test.go`

Smallest consumer site. Apply the multiplier to candidate raw and displaced totals inside `ItemValueDelta`. `ItemValue` (pure score) stays char-agnostic.

- [ ] **Step 1: Read the current ItemValueDelta function**

```
Read internal/itemvalue/delta.go offset 276 limit 60
```

Locate the function body. Find the lines where `candidateRaw := ItemValue(...)` is computed and where `displacedTotal` is summed inside the per-slot loop.

- [ ] **Step 2: Add the import and apply the multiplier**

Add the mutations import:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/mutations"
)
```

Modify the body of `ItemValueDelta`:

```go
func ItemValueDelta(char *characters.Character, profile WeightProfile, candidate items.Item) SwapDelta {
	candidateSpec := candidate.GetSpec()
	candidateRaw := ItemValue(candidateSpec, profile)

	// Apply gear-effectiveness multiplier from char's mutations.
	// Incorporeal characters see all gear values scaled toward
	// zero, so equip-if-better naturally returns false at high
	// incorporeal ranks.
	gearMul := mutations.GearEffectivenessMultiplier(char.Mutations)
	candidateRaw *= gearMul

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
			displacedTotal += (ItemValue(dSpec, profile) +
				placementBonus(profile, dSpec, currentSlot, char)) * gearMul
		}

		netScore := candidateAt - displacedTotal
		netScore -= encumbranceTierPenalty(char, displaced, candidate, profile)

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

**Note on `placementBonus`:** the multiplier applies to the COMBINED (ItemValue + placementBonus) for displaced items, since both portions are gear-derived. For the candidate, the bonus is added AFTER multiplication — meaning placement bonuses are NOT scaled by gear-effectiveness. This is a tiny asymmetry. If you prefer fully symmetric scaling (bonuses scale too), multiply at the very end: `candidateAt = (candidateRaw + placementBonus(...)) * gearMul`. Either choice is defensible; the above matches the spec's "apply to gear-derived" framing (the offhand-strategy bonuses are about playstyle synergy, not gear itself). Pick the asymmetric version above for consistency with the spec.

- [ ] **Step 3: Add the test**

Append to `internal/itemvalue/delta_test.go`:

```go
func TestItemValueDelta_IncorporealRank4Scores0(t *testing.T) {
	// Inject a synthetic incorporeal mutation into the registry.
	// (If a test helper exists for this, use it.)
	prev := mutations_allMutationsAccessor()
	defer mutations_restoreAllMutations(prev)
	mutations_setTestMutations(map[string]*mutations.MutationSpec{
		"incorporeal": {
			MutationId: "incorporeal",
			Name:       "Incorporeal",
			Rarity:     10,
			Cons: []mutations.MutationEffect{
				{Type: "gear_effectiveness_loss", Value: 0.25},
			},
		},
	})

	char := &characters.Character{
		Mutations: map[string]int{"incorporeal": 4},
	}

	// Any item that would normally score positive.
	itemSpec := items.ItemSpec{
		Type:             items.Weapon,
		Hands:            items.OneHanded,
		DamageMultiplier: 1.5,
	}
	candidate := items.Item{} // GetSpec stub returns this in test envs
	// If you can't construct a real items.Item that returns itemSpec
	// from GetSpec(), skip this test with a clear note (fixture limit).

	delta := ItemValueDelta(char, PhysicalBruiser, candidate)
	if delta.Score != 0 {
		t.Errorf("rank-4 incorporeal should score 0, got %.2f", delta.Score)
	}
}
```

**Important:** the test uses `mutations_allMutationsAccessor` / `mutations_setTestMutations` / `mutations_restoreAllMutations` as placeholder names for whatever the actual `internal/mutations` test-injection pattern is. If the existing tests in `internal/mutations/mutations_test.go` show a real pattern (e.g., direct `mutations.allMutations` access through a test helper, or a `mutations.TestSetRegistry(...)` exported test function), use that. If injection isn't supported, this test should `t.Skip` with a documented reason ("requires mutation registry test-injection helper not present").

Simpler fallback if injection is too involved: skip this test entirely and rely on Task 1's `mutations_test.go` tests (which already cover the multiplier math) plus the integration tests in later tasks. Note the skip with a clear comment.

- [ ] **Step 4: Build and run tests**

```
go build ./...
go test ./internal/itemvalue/ -v
```

Expected: build clean, all earlier tests still PASS, new test PASSes or SKIPs cleanly.

- [ ] **Step 5: Commit**

```bash
git add internal/itemvalue/delta.go internal/itemvalue/delta_test.go
git commit -m "$(cat <<'EOF'
feat(itemvalue): apply gear-effectiveness multiplier in ItemValueDelta

Multiplies candidate and displaced item values by
mutations.GearEffectivenessMultiplier(char.Mutations). Pure
ItemValue stays char-agnostic. At rank-4 incorporeal, all gear
scores 0 — so chunk 2.3's equip-if-better behavior naturally
returns false for any candidate equipment, with no special
hardcoded skip path.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Character.Recalculate gear-derived stat scaling

**Files:**
- Modify: `internal/characters/character.go`
- Modify: `internal/characters/character_test.go` (or `validate_test.go` per existing test layout)

The hardest task in this chunk — the current shape of `Recalculate` may or may not cleanly separate gear-derived contributions from non-gear ones. The implementer must investigate first.

- [ ] **Step 1: Investigate Recalculate's current shape**

Read `internal/characters/character.go` looking for the `Recalculate` method. Also read `internal/characters/validate.go` (which the earlier exploration showed has Mutations-aware code at lines 435+).

Identify:
- Where equipment stat contributions are summed into the character's final stats.
- Whether non-gear contributions (mutation `stat_flat`, buff stat mods, base/racial values) are summed separately or mixed in.

If they're cleanly separated → proceed to step 2 with a simple multiplier application.
If they're mixed → a small refactor is needed to separate them so the multiplier applies to gear only. Do the refactor as part of this task.

- [ ] **Step 2: Apply the gear-effectiveness multiplier to gear-derived stat contributions**

Wherever `c.Equipment.StatMod(...)` is called inside `Recalculate` (or wherever equipment stats roll up):

```go
gearStat := c.Equipment.StatMod(statName)
gearStat = int(float64(gearStat) * mutations.GearEffectivenessMultiplier(c.Mutations))
// ... then continue summing with non-gear contributions ...
```

Add the import: `"github.com/GoMudEngine/GoMud/internal/mutations"`.

The integration shape depends on the exact code organization in `Recalculate`. Aim for: gear contributions are computed, multiplied by the gear-effectiveness multiplier, then summed with non-gear (mutations.stat_flat, buffs, base values).

- [ ] **Step 3: Add a test**

```go
func TestRecalculate_GearStatsScaledByIncorporeal(t *testing.T) {
	// Inject synthetic incorporeal mutation.
	// (Use whatever the existing test-injection pattern is.)
	prev := mutations.SetTestRegistry(map[string]*mutations.MutationSpec{
		"incorporeal": {
			MutationId: "incorporeal", Name: "Incorporeal", Rarity: 10,
			Cons: []mutations.MutationEffect{
				{Type: "gear_effectiveness_loss", Value: 0.25},
			},
		},
	})
	defer mutations.RestoreRegistry(prev)

	char := &Character{
		Mutations: map[string]int{"incorporeal": 2}, // 50% gear loss
	}
	// Equip an item with +10 strength.
	char.Equipment.Body = items.Item{ /* ... synthetic spec with +10 strength ... */ }
	// Set base strength to 100.
	char.Stats.Strength.Base = 100
	// Set a non-gear strength source (mutation stat_flat) of +20.
	// (Use the synthetic registry to add a mutation that grants +20 str.)

	char.Recalculate()

	// Expected: strength = 100 (base) + 5 (10 gear × 0.5 multiplier) + 20 (mutation flat) = 125
	got := char.Stats.Strength.ValueAdj
	want := 125
	if got != want {
		t.Errorf("strength = %d, want %d", got, want)
	}
}
```

This test depends on `mutations.SetTestRegistry` / `mutations.RestoreRegistry` being either real exported helpers or test-only functions. If neither exists, either add them as test-only helpers in Task 1's commit (lightweight `_test.go` file) or skip this test with a documented note.

- [ ] **Step 4: Run all character tests for regressions**

```
go test ./internal/characters/ -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/characters/character.go internal/characters/character_test.go
git commit -m "$(cat <<'EOF'
feat(characters): scale gear-derived stat contributions by Incorporeal

Character.Recalculate now applies
mutations.GearEffectivenessMultiplier(c.Mutations) to the
equipment-summed stat contributions. Non-gear contributions
(mutation stat_flat, buff stat mods, base values) pass through
unchanged. If the previous body mixed gear and non-gear in a
single pass, this commit also includes the small refactor to
separate them.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Three mitigation getters

**Files:**
- Modify: `internal/characters/character.go`
- Modify: `internal/characters/character_test.go` (or wherever mitigation tests live)

Apply the same multiplier to gear-derived contributions in `GetPhysicalMitigation`, `GetMagicalMitigation`, `GetConvictionMitigation`.

- [ ] **Step 1: Investigate the current shape of the three getters**

Read `internal/characters/character.go`, locate `GetPhysicalMitigation`, `GetMagicalMitigation`, `GetConvictionMitigation`. Note where gear vs non-gear contributions are aggregated. If they're cleanly separated, modify in place. If mixed, refactor first.

- [ ] **Step 2: Apply the multiplier**

For each of the three methods, the pattern is:

```go
func (c *Character) GetPhysicalMitigation() int {
	gearMit := c.Equipment.PhysicalMitigation() // or whatever the existing call is
	gearMit = int(float64(gearMit) * mutations.GearEffectivenessMultiplier(c.Mutations))

	// Non-gear contributions (mutation natural_armor, buffs, etc.)
	nonGearMit := /* existing aggregation for non-gear sources */

	return gearMit + nonGearMit
}
```

Apply identically to `GetMagicalMitigation` (with whichever `Equipment.MagicalMitigation()` call exists) and `GetConvictionMitigation`.

If `c.Equipment` doesn't expose per-channel mitigation getters, the refactor includes adding them. Aim for: per-channel getter on `Worn` that sums equipment-mit only; character-level getter scales by gear multiplier and adds non-gear sources.

- [ ] **Step 3: Add tests**

```go
func TestGetPhysicalMitigation_ScalesByIncorporeal(t *testing.T) {
	// Inject synthetic mutation (per Task 4 pattern).
	// ...

	char := &Character{
		Mutations: map[string]int{"incorporeal": 4}, // 100% gear loss
	}
	// Equip body armor with 30% physical mitigation.
	char.Equipment.Body = items.Item{ /* spec with PhysicalMitigation=30 */ }
	// Add a buff giving +10 mit (or use mutation natural_armor 10).
	// ...

	got := char.GetPhysicalMitigation()
	// Expected: 30 gear × 0 + 10 non-gear = 10
	want := 10
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}
```

Add analogous tests for `GetMagicalMitigation` and `GetConvictionMitigation`.

- [ ] **Step 4: Run tests, commit**

```
go test ./internal/characters/ -v
```

Expected: all PASS.

```bash
git add internal/characters/character.go internal/characters/character_test.go
git commit -m "$(cat <<'EOF'
feat(characters): scale gear-derived mitigation by Incorporeal

GetPhysicalMitigation / GetMagicalMitigation /
GetConvictionMitigation each multiply equipment-derived
contributions by mutations.GearEffectivenessMultiplier.
Non-gear sources (mutation natural_armor, buffs, etc.) pass
through unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Weapon damage scaling in buildWeaponSetup

**Files:**
- Modify: `internal/combat/combat_helpers.go`
- Modify: `internal/combat/combat_helpers_test.go` (or sibling test file)

Apply gear-effectiveness multiplier to `weaponDmgMult` in `buildWeaponSetup`.

- [ ] **Step 1: Locate the modification site**

`internal/combat/combat_helpers.go` around line 257:

```go
ws.weaponDmgMult = itemSpec.DamageMultiplier
if ws.weaponDmgMult <= 0 {
    ws.weaponDmgMult = float64(bal.UnarmedDamageMultiplier)
}
```

Modify to:

```go
gearMul := mutations.GearEffectivenessMultiplier(sourceChar.Mutations)
ws.weaponDmgMult = itemSpec.DamageMultiplier * gearMul
if ws.weaponDmgMult <= 0 {
    ws.weaponDmgMult = float64(bal.UnarmedDamageMultiplier)
}
```

Add the import: `"github.com/GoMudEngine/GoMud/internal/mutations"`.

**Important behavior note:** the existing fallback at the `if ws.weaponDmgMult <= 0` branch catches the rank-4 incorporeal case (where the multiplier produces 0) and substitutes unarmed-level damage. This is correct: a wraith with a sword falls back to unarmed-level swings (their ethereal "claws" still hit), not literally zero damage. The fallback is the right semantics — don't try to suppress it.

- [ ] **Step 2: Add a test**

```go
func TestBuildWeaponSetup_IncorporealRank4FallsBackToUnarmed(t *testing.T) {
	// Inject synthetic incorporeal mutation.
	// (Per the existing patterns.)

	char := &characters.Character{
		Mutations: map[string]int{"incorporeal": 4},
	}
	weapon := items.Item{ /* spec with DamageMultiplier=1.5 */ }
	target := &characters.Character{}

	// Construct a weaponSetup via buildWeaponSetup with the
	// incorporeal char + the 1.5x sword. Expected: weaponDmgMult
	// equals the unarmed damage multiplier (NOT 1.5x and NOT
	// literally 0).
	ws := buildWeaponSetup(char, target, weapon, 0, 1)
	bal := configs.GetBalanceConfig()
	expected := float64(bal.UnarmedDamageMultiplier)
	if ws.weaponDmgMult != expected {
		t.Errorf("rank-4 incorporeal should fall back to unarmed %.2f, got %.2f",
			expected, ws.weaponDmgMult)
	}
}

func TestBuildWeaponSetup_IncorporealRank2HalvesWeaponDmg(t *testing.T) {
	// Mid-rank case: 50% gear effectiveness.
	// ... similar setup, level 2 ...

	ws := buildWeaponSetup(char, target, weapon, 0, 1)
	expected := 1.5 * 0.5 // weapon mult × gear mul
	if math.Abs(ws.weaponDmgMult - expected) > 1e-9 {
		t.Errorf("rank-2 incorporeal: expected %.2f, got %.2f", expected, ws.weaponDmgMult)
	}
}
```

If `buildWeaponSetup` is unexported and not directly callable from another package, write the test in the same `internal/combat/` package.

- [ ] **Step 3: Run tests**

```
go test ./internal/combat/ -v
```

Expected: all PASS, including any prior combat tests (no regressions).

- [ ] **Step 4: Commit**

```bash
git add internal/combat/combat_helpers.go internal/combat/combat_helpers_test.go
git commit -m "$(cat <<'EOF'
feat(combat): scale weapon damage by Incorporeal gear effectiveness

buildWeaponSetup multiplies the weapon's DamageMultiplier by
mutations.GearEffectivenessMultiplier(sourceChar.Mutations).
The existing fallback (if weaponDmgMult <= 0, substitute
unarmed) correctly catches the rank-4 incorporeal case —
ethereal beings with weapons fall back to unarmed-level swings
(their natural claws/bite/slam still hit).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Spell damage scaling in calculations.go

**Files:**
- Modify: `internal/combat/calculations.go`
- Modify: `internal/combat/calculations_test.go` (or sibling)

`internal/combat/calculations.go:90` reads `spellMult` and feeds it into `CalcRawDamage` for the magical channel. Identify where `spellMult` comes from (likely the equipped caster weapon's `SpellDamageMultiplier`), and apply the gear multiplier there.

- [ ] **Step 1: Read calculations.go around line 90**

```
Read internal/combat/calculations.go offset 70 limit 40
```

Find where `spellMult` is computed (probably from `char.Equipment.Weapon.GetSpec().SpellDamageMultiplier` or similar).

- [ ] **Step 2: Apply the multiplier at the spellMult derivation**

Wherever `spellMult` is read from a weapon's `SpellDamageMultiplier`:

```go
gearMul := mutations.GearEffectivenessMultiplier(char.Mutations)
spellMult := weaponSpec.SpellDamageMultiplier * gearMul
```

Add the import if not present.

If `spellMult` is hardcoded (e.g., always 1.0 when no caster weapon) and not derived from a gear field, no change needed here for the magical channel. Same goes for the conviction channel at line 93.

The point is: ANY contribution from equipped gear to the magical/conviction damage formulas should scale by the gear multiplier. If no such contribution exists in this file, skip the task and update the spec/plan to note the discovery.

- [ ] **Step 3: Add a test (if a code change was made in step 2)**

```go
func TestSpellDamage_ScalesByIncorporeal(t *testing.T) {
	// Setup synthetic incorporeal mutation + caster-weapon char.
	// ...

	// Expected: rank-4 incorporeal caster wielding a staff
	// (SpellDamageMultiplier=1.6) deals magical damage as if
	// no spell-damage gear was equipped.
}
```

- [ ] **Step 4: Run tests, commit**

```
go test ./internal/combat/ -v
```

Expected: all PASS.

```bash
git add internal/combat/calculations.go internal/combat/calculations_test.go
git commit -m "$(cat <<'EOF'
feat(combat): scale spell-damage gear contributions by Incorporeal

Where calculations.go reads SpellDamageMultiplier from an
equipped caster weapon, multiply by
mutations.GearEffectivenessMultiplier so incorporeal casters
correctly lose their staff/wand bonuses at higher ranks. Non-
gear magical sources (Willpower stat, mutation effects) are
unaffected — incorporeal casters still gain the +willpower
bonus from the Incorporeal mutation itself.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

If step 2 found no gear contribution to suppress, skip this task and commit nothing. Document the finding in the implementation summary.

---

## Task 8: Defense margin bonus in combat resolution

**Files:**
- Modify: `internal/combat/combat_helpers.go`
- Modify: `internal/combat/combat_helpers_test.go` (or sibling)

Add `GetPhysicalDefenseBonus(defender.Mutations)` to the defender's margin in the best-of-all defense resolution. Channel-scoped to physical attacks only.

- [ ] **Step 1: Read the best-of-all defense resolution**

`internal/combat/combat_helpers.go` around line 714, where `best.margin` is computed for the comparison.

Identify:
- The variable holding the defender's character (`targetChar` likely).
- The variable indicating attack channel — does the function know if the attack is physical / magical / conviction?

If channel info isn't plumbed to this site, a small threading change is needed. Trace upward from `resolveAttack` (or whichever the entry function is) and add a `channel DamageChannel` parameter.

- [ ] **Step 2: Add the defense bonus for physical-channel attacks**

In the resolution function:

```go
if channel == ChannelPhysical {
	defBonus := mutations.GetPhysicalDefenseBonus(targetChar.Mutations)
	best.margin += defBonus
}
```

Apply BEFORE the margin comparison (around the `if best.margin > 0` block at line 714).

Add the imports: `"github.com/GoMudEngine/GoMud/internal/mutations"` and the combat package's `DamageChannel` constants if not already imported in scope.

- [ ] **Step 3: Add tests**

```go
func TestDefenseMargin_IncorporealAddsPhysicalDefenseBonus(t *testing.T) {
	// Setup: defender with incorporeal rank 4.
	// Attacker with weapon, attack channel physical.
	// Expected: defender's effective margin includes +37.5
	// (15 base × 2.5 LevelMultiplier).
	// Verify via the bestMargin / hit-result of a single
	// resolution.
}

func TestDefenseMargin_IncorporealDoesNotApplyToMagical(t *testing.T) {
	// Setup: same defender, attack channel magical.
	// Expected: no defense bonus added — incorporeal beings
	// remain vulnerable to magic per spec.
}
```

- [ ] **Step 4: Run tests, commit**

```
go test ./internal/combat/ -v
```

Expected: all PASS.

```bash
git add internal/combat/combat_helpers.go internal/combat/combat_helpers_test.go
git commit -m "$(cat <<'EOF'
feat(combat): incorporeal physical_defense_bonus in best-of-all defense

When resolving physical-channel attacks, the defender's roll
margin includes mutations.GetPhysicalDefenseBonus
(defender.Mutations). Magical and conviction-channel attacks
unaffected — incorporeal beings remain vulnerable to magic per
spec. Channel routing threaded through the resolution function
if not already present.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Mob YAML audit + tagging

**Files:**
- Modify: 5 categories of mob YAML templates (varies per audit results)

Audit existing mob templates for the five categories that should be ethereal, then add `mutations: { incorporeal: 4 }` to each.

- [ ] **Step 1: Audit by category**

Run these greps to identify candidate templates. Each command lists relative paths for review:

```bash
grep -rln '^name:.*[Ww]raith' _datafiles/world/dogmud/mobs/ --include='*.yaml'
grep -rln '^name:.*[Ss]pec[tr]re' _datafiles/world/dogmud/mobs/ --include='*.yaml'
grep -rln '^name:.*[Ff]ire [Ee]lemental' _datafiles/world/dogmud/mobs/ --include='*.yaml'
grep -rln '^name:.*[Aa]ir [Ee]lemental' _datafiles/world/dogmud/mobs/ --include='*.yaml'
```

And the known one:
- `_datafiles/world/dogmud/mobs/instance_planar_oasis/321-elemental_queen.yaml` (Elemental Queen, confirmed)

For each matched file, read the YAML to confirm the mob is actually what its name implies (sometimes mob names are flavor-overlap with other species). Skip any that are clearly tagged differently in their description.

- [ ] **Step 2: Add the mutation tag to each confirmed YAML**

For each confirmed mob template, append (or insert in the natural location):

```yaml
mutations:
  incorporeal: 4
```

If the YAML already has a `mutations:` block (some mob templates do), append the new key to it:

```yaml
mutations:
  incorporeal: 4
  # ... existing mutations ...
```

- [ ] **Step 3: Verify the server boots cleanly past data load**

```
go build ./...
./dogmud.exe (background)
```

Wait ~10 seconds, read background output. Confirm:
- `mobs.LoadDataFiles() loadedCount=` no decrement (all mobs still load)
- No panic / fatal error

Kill the server, remove `dogmud.exe`.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/
git commit -m "$(cat <<'EOF'
feat(mobs): tag ethereal mob templates with incorporeal rank 4

Wraiths, spectres, fire elementals, air elementals, and the
Elemental Queen now carry mutations: { incorporeal: 4 } in
their templates. Per the chunk 2.2a spec, this gives them
100% gear-effectiveness loss (any equipped gear in their
template effectively becomes flavor only — weapons fall back
to unarmed damage), full physical defense bonus (+37.5 margin
on best-of-all defense), and the stat shifts (-25 vit, +12.5
wil at rank 4).

Earth and water elementals stay corporeal in this pass.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Helpfile update

**Files:**
- Modify: `_datafiles/world/dogmud/templates/help/mutations.template`

Append a new entry for the Incorporeal mutation.

- [ ] **Step 1: Read the existing helpfile format**

```
Read _datafiles/world/dogmud/templates/help/mutations.template
```

Identify the format — likely Markdown-style entries with name, rarity, effects, conflicts.

- [ ] **Step 2: Append the Incorporeal entry**

Format the entry to match existing style. Sample content:

```
### Incorporeal (very rare)

Your form has slipped between planes, growing translucent and
ethereal. Physical attacks struggle to find purchase against
you, but your gear loses its grip on reality as well.

Effects scale per rank (1-4):
  - Vitality penalty: -10 / -15 / -20 / -25
  - Willpower bonus: +5 / +7.5 / +10 / +12.5
  - Physical defense bonus: +15 / +22.5 / +30 / +37.5 to defense rolls
  - Equipment effectiveness: 75% / 50% / 25% / 0%

At rank 4, gear becomes ornamental — weapons effectively fail
to strike (falling back to unarmed swings), armor stops
mitigating physical attacks, and stat-mod gear no longer
contributes. Natural-weapon damage (claws, bite, slam) and
willpower-driven magic remain potent.

Conflicts: extra-arms, extra-legs, clawed-hands, tail,
dense-muscles, hollow-bones, healing-gel.
```

Adjust phrasing to match the existing entries' voice. Player-facing text per the CLAUDE.md "no hard numbers" SOP would normally avoid exact magnitudes — but mutation help is conventionally a deliberate exception (the existing entries are explicit about rank effects). Match what's already there.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/mutations.template
git commit -m "$(cat <<'EOF'
docs(help): add Incorporeal mutation to mutations help

New rarest mutation. Documents the four-rank progression for
the gear-effectiveness loss + stat changes + physical defense
bonus, plus the conflict list.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: context.md updates

**Files:**
- Modify: `internal/mutations/context.md`
- Modify: `internal/itemvalue/context.md`
- Modify: `internal/characters/context.md`

Update the three package docs to reflect the new behavior. Each is a small append/insert.

- [ ] **Step 1: Update `internal/mutations/context.md`**

Append entries for:
- `gear_effectiveness_loss` effect type (note the raw-level carve-out — does NOT use LevelMultiplier; the value is multiplied directly by the level integer)
- `physical_defense_bonus` effect type (uses standard `LevelMultiplier` scaling)
- The three new helper functions (`GetGearEffectivenessLoss`, `GearEffectivenessMultiplier`, `GetPhysicalDefenseBonus`)

Keep the entries concise — match the existing context.md style.

- [ ] **Step 2: Update `internal/itemvalue/context.md`**

Add a brief note in the "ItemValueDelta algorithm" section explaining that the candidate and displaced totals are multiplied by `mutations.GearEffectivenessMultiplier(char.Mutations)`. So incorporeal characters naturally see all gear score 0 at rank 4 — chunk 2.3's equip-if-better doesn't need a special skip path.

- [ ] **Step 3: Update `internal/characters/context.md`**

Add notes in the relevant sections that `Recalculate()`, `GetPhysicalMitigation()`, `GetMagicalMitigation()`, and `GetConvictionMitigation()` apply the gear-effectiveness multiplier to gear-derived contributions; non-gear contributions are unaffected.

- [ ] **Step 4: Commit**

```bash
git add internal/mutations/context.md internal/itemvalue/context.md internal/characters/context.md
git commit -m "$(cat <<'EOF'
docs(context): document Incorporeal integration across packages

mutations/context.md: new effect types + helpers + the raw-level
carve-out rationale for gear_effectiveness_loss.
itemvalue/context.md: gear-effectiveness multiplier in
ItemValueDelta.
characters/context.md: Recalculate + three mitigation methods
apply the multiplier to gear-derived contributions.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Smoke + roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- (No source changes; verification only)

Final verification + roadmap closeout.

- [ ] **Step 1: Build + tests**

```
go build -o dogmud.exe .
go test ./... 2>&1 | tail -50
```

Expected: clean build; all package tests PASS (or have only fixture-dependent SKIPs that were present before this chunk).

- [ ] **Step 2: Server boot smoke**

```
./dogmud.exe (background)
```

Wait ~10 seconds. Read background output. Confirm:
- `mutations.LoadMutationFiles() loadedCount=` incremented by 1 from pre-chunk count
- `mobs.LoadDataFiles() loadedCount=` unchanged from pre-chunk count (no mob templates lost)
- No panic / fatal error
- No species/mutation validation errors

Kill the server, remove `dogmud.exe`.

- [ ] **Step 3: Update the progress tracker**

In `MOB_ALIVENESS_ROADMAP.md`, find the Progress tracker table. Insert a new row for chunk 2.2a between the 2.2 and 2.3 rows:

```
| 2.2a | Tactical | Incorporeal mutation | M | — | Done |
```

Also bump the total row count from 40 to 41 wherever it's referenced.

- [ ] **Step 4: Add the chunk 2.2a mini-brief**

In `MOB_ALIVENESS_ROADMAP.md`, find the section for Phase 2 chunks. Insert a new section between `### 2.2 Item-comparison primitive` and `### 2.3 Equip-if-better behavior`:

```markdown
### 2.2a Incorporeal mutation
**Status:** Done (2026-05-11) • **Size:** M

- **Goal:** Model ethereal beings (wraiths, spectres, fire and
  air elementals, elemental queen) as a new rarest mutation
  (`incorporeal`) with four ranks scaling gear effectiveness
  loss + physical defense bonus + stat shifts.
- **In:** Mutation YAML, two new effect types
  (`gear_effectiveness_loss`, `physical_defense_bonus`), five
  consumer-site integrations (stat aggregation, three
  mitigation getters, weapon damage, defense resolution,
  itemvalue scoring), mob YAML tagging on five categories,
  helpfile + context.md updates.
- **Out:** Per-rank tuning beyond starting values, player
  acquisition trigger beyond rarity weighting, earth/water
  elemental tagging.
- **Depends on:** —
- **Why:** Chunk 2.3 (equip-if-better) needs a gate to skip
  ethereal mobs/players. Soft-scaling via itemvalue scoring
  is cleaner than a hardcoded skip path. Also unblocks future
  "incorporeal player" progression goals.
- **Shipped:** New `_datafiles/world/dogmud/mutations/incorporeal.yaml`
  with rarity 10 + conflict list (seven body-dependent
  mutations). New `GetGearEffectivenessLoss`,
  `GearEffectivenessMultiplier`, `GetPhysicalDefenseBonus`
  helpers in `internal/mutations/mutations.go` —
  `gear_effectiveness_loss` uses raw level multiplication
  (linear 0.25/0.50/0.75/1.00 across ranks), the carve-out
  documented in `internal/mutations/context.md`. Integration
  in five sites: `character.go` (Recalculate gear-stat
  scaling + three Get*Mitigation methods), `combat_helpers.go`
  (buildWeaponSetup applies multiplier to weaponDmgMult;
  best-of-all defense resolution adds physical_defense_bonus
  for physical-channel attacks), `calculations.go` (spell
  damage gear contributions scaled), `itemvalue/delta.go`
  (ItemValueDelta applies multiplier to candidate + displaced
  totals). Five mob categories tagged with
  `mutations: { incorporeal: 4 }`. Helpfile + three context.md
  files updated. Spec at
  `docs/superpowers/specs/2026-05-11-mob-aliveness-2.2a-incorporeal-mutation-design.md`,
  plan at
  `docs/superpowers/plans/2026-05-11-mob-aliveness-2.2a-incorporeal-mutation.md`.
```

- [ ] **Step 5: Bump the roll-up**

Find the line that reads `**Roll-up:** 9 / 40 done • ...` and update to:

```
**Roll-up:** 10 / 41 done • 0 in progress • 31 not started.
```

- [ ] **Step 6: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): mark chunk 2.2a (Incorporeal mutation) as Done

Inserted between 2.2 and 2.3. Ships the Incorporeal mutation +
two new effect types + five consumer-site integrations + five
mob-category taggings. Unblocks chunk 2.3 (equip-if-better)
by providing a soft-scale gate via itemvalue scoring rather
than requiring a hardcoded ethereal-mob skip path. Roll-up
moves to 10/41.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-review checklist (run before declaring done)

- [ ] `go build ./...` passes clean.
- [ ] `go test ./internal/mutations/ ./internal/characters/ ./internal/combat/ ./internal/itemvalue/ -v` all green.
- [ ] `_datafiles/world/dogmud/mutations/incorporeal.yaml` exists and loads cleanly at server boot.
- [ ] Three new helpers (`GetGearEffectivenessLoss`, `GearEffectivenessMultiplier`, `GetPhysicalDefenseBonus`) present in `internal/mutations/mutations.go`.
- [ ] Five consumer sites updated:
  - `internal/itemvalue/delta.go` (Task 3)
  - `internal/characters/character.go` Recalculate (Task 4)
  - `internal/characters/character.go` three mitigation getters (Task 5)
  - `internal/combat/combat_helpers.go` buildWeaponSetup (Task 6)
  - `internal/combat/calculations.go` spell damage (Task 7) — OR documented finding that no gear contribution exists in that file
  - `internal/combat/combat_helpers.go` defense margin (Task 8)
- [ ] Five mob categories tagged (Task 9).
- [ ] Helpfile updated (Task 10).
- [ ] Three context.md files updated (Task 11).
- [ ] Server boots cleanly past data load.
- [ ] Roadmap roll-up updated to 10/41.
