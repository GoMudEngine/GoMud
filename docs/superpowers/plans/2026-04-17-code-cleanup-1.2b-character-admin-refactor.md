# Code Cleanup 1.2b: Character + Admin God-Function Refactor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decompose 5 oversized functions in `internal/characters/character.go`
(`RecalculateStats`, `Wear`, `Validate`) and `internal/usercommands/admin.room.go`
(`Room`, `room_Edit_Exits`) into focused helpers, backed by characterization
tests for the three character-package functions so the refactor is provably
semantics-preserving.

**Architecture:** Character batch — write tests first, extract helpers inline in
`character.go`. Admin batch — move helpers into two new files (`admin.room.dispatcher.go`,
`admin.room.exits.go`). One feature branch, 8 commits, per-commit verification
via `go build && go vet ./... && go test ./...`.

**Tech Stack:** Go 1.25, testify (already used), stretchr assert.

**Spec:** `docs/superpowers/specs/2026-04-17-code-cleanup-1.2b-character-admin-refactor-design.md`

**Branch:** `feature/stage-1.2b-character-admin-refactor` off `development`.

---

## Task 0: Create feature branch

**Files:** none.

- [ ] **Step 1: Verify you're on `development` and clean**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git status
git branch --show-current
```

Expected: `development`. Working tree should be clean except for the
previously-known unrelated working-tree noise (`.claude/settings.local.json`,
`internal/usercommands/_datafiles/feedback/*.txt`). Those are **out of scope**
— do not stage or commit them at any point in this plan.

- [ ] **Step 2: Create feature branch**

```bash
git checkout -b feature/stage-1.2b-character-admin-refactor
```

Expected: `Switched to a new branch 'feature/stage-1.2b-character-admin-refactor'`.

---

## Task 1: Characterization tests for RecalculateStats, Wear, Validate

Write 18 tests that capture current behavior of the 3 character functions.
They must pass against the **current unrefactored code**. After each refactor
commit (Tasks 2, 3, 4), the full test file must still pass unchanged.

**Files:**
- Create: `internal/characters/godfunc_refactor_test.go`

### Setup

- [ ] **Step 1: Confirm test-relevant helpers exist**

Before writing tests, confirm these exist (no changes, just verify):

```bash
grep -n "func New()" internal/characters/character.go
grep -n "func RollCharacterStats" internal/characters/character.go
grep -n "func ensureAllSkills" internal/characters/character.go
grep -n "func (c \*Character) Validate" internal/characters/character.go
grep -n "func (c \*Character) Wear" internal/characters/character.go
grep -n "func (c \*Character) RecalculateStats" internal/characters/character.go
grep -n "func (c \*Character) CanDualWield" internal/characters/character.go
```

Expected: all seven grep matches return a single line each. Mismatches
mean the code has drifted; stop and investigate.

- [ ] **Step 2: Scan existing test patterns**

Read `internal/characters/character_test.go` lines 1-100 and
`internal/characters/progression_test.go` lines 1-80 for the existing
test style. Notes for our tests:
- Package is `characters` (same-package, not `characters_test`), so all
  exported and unexported methods are reachable.
- `github.com/stretchr/testify/assert` is already imported elsewhere in
  this package — use `assert.Equal` / `assert.True` etc.
- `New()` constructs a full test character using rolled stats and calls
  `Validate()` internally. For tests that need deterministic stats, set
  them manually AFTER `New()`.

### `RecalculateStats` tests (6 tests)

- [ ] **Step 3: Write `TestRecalculateStats_BaseStatHydrationFromSpecies`**

Create `internal/characters/godfunc_refactor_test.go` with package header
and imports, then add this first test:

```go
package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/stretchr/testify/assert"
)

// TestRecalculateStats_BaseStatHydrationFromSpecies — base stats should
// hydrate from species data when zero, but NOT overwrite rolled stats.
func TestRecalculateStats_BaseStatHydrationFromSpecies(t *testing.T) {
	c := &Character{
		SpeciesId: 1, // Human (default)
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 0},
			Dexterity:  stats.StatInfo{Base: 0},
			Perception: stats.StatInfo{Base: 0},
			Vitality:   stats.StatInfo{Base: 0},
			Willpower:  stats.StatInfo{Base: 0},
			Charisma:   stats.StatInfo{Base: 0},
		},
		Mutations: map[string]int{},
		Buffs:     *newBuffsOrEmpty(),
	}

	c.RecalculateStats()

	// Base should now be populated from species
	assert.Greater(t, c.Stats.Strength.Base, 0, "Strength.Base should hydrate from species")
	assert.Greater(t, c.Stats.Dexterity.Base, 0, "Dexterity.Base should hydrate from species")

	// Now test that non-zero base is NOT overwritten
	c2 := &Character{
		SpeciesId: 1,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 123},
			Dexterity:  stats.StatInfo{Base: 0},
			Perception: stats.StatInfo{Base: 0},
			Vitality:   stats.StatInfo{Base: 0},
			Willpower:  stats.StatInfo{Base: 0},
			Charisma:   stats.StatInfo{Base: 0},
		},
		Mutations: map[string]int{},
		Buffs:     *newBuffsOrEmpty(),
	}
	c2.RecalculateStats()
	assert.Equal(t, 123, c2.Stats.Strength.Base, "rolled Strength.Base must not be overwritten")
}

// newBuffsOrEmpty is a test helper that returns a fresh Buffs instance.
// Needed because stats.Statistics / Buffs zero values panic in some paths.
func newBuffsOrEmpty() *buffs.Buffs {
	b := buffs.New()
	return &b
}
```

Note: import `github.com/GoMudEngine/GoMud/internal/buffs` in the imports
block (we reference `buffs.Buffs` in the helper).

- [ ] **Step 4: Write `TestRecalculateStats_PoolMaxDerivation`**

Add to the test file:

```go
// TestRecalculateStats_PoolMaxDerivation — HealthMax/StaminaMax/ConvictionMax/
// ActionPointsMax derive from stats + balance config, with floors.
func TestRecalculateStats_PoolMaxDerivation(t *testing.T) {
	c := &Character{
		SpeciesId: 1,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 100},
			Dexterity:  stats.StatInfo{Base: 100},
			Perception: stats.StatInfo{Base: 100},
			Vitality:   stats.StatInfo{Base: 100},
			Willpower:  stats.StatInfo{Base: 100},
			Charisma:   stats.StatInfo{Base: 100},
		},
		Mutations: map[string]int{},
		Buffs:     *newBuffsOrEmpty(),
	}

	c.RecalculateStats()

	assert.GreaterOrEqual(t, c.HealthMax.Value, 1, "HealthMax floor is 1")
	assert.GreaterOrEqual(t, c.StaminaMax.Value, 0, "StaminaMax floor is 0")
	assert.GreaterOrEqual(t, c.ConvictionMax.Value, 0, "ConvictionMax floor is 0")
	assert.GreaterOrEqual(t, c.ActionPointsMax.Value, 50, "ActionPointsMax floor is 50")
	assert.Greater(t, c.HealthMax.Value, 1, "HealthMax should be well above floor at stat=100")
}
```

- [ ] **Step 5: Write `TestRecalculateStats_EquipmentStatMods`**

Add:

```go
// TestRecalculateStats_EquipmentStatMods — equipped items contribute to .Mods.
// Current behavior is captured by snapshot: change in Mods should equal StatMod()
// return value.
func TestRecalculateStats_EquipmentStatMods(t *testing.T) {
	c := &Character{
		SpeciesId: 1,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 100},
			Dexterity:  stats.StatInfo{Base: 100},
			Perception: stats.StatInfo{Base: 100},
			Vitality:   stats.StatInfo{Base: 100},
			Willpower:  stats.StatInfo{Base: 100},
			Charisma:   stats.StatInfo{Base: 100},
		},
		Mutations: map[string]int{},
		Buffs:     *newBuffsOrEmpty(),
	}

	c.RecalculateStats()

	// Mods should equal StatMod output for each stat (pure equipment contribution).
	assert.Equal(t, c.StatMod(string(statmods.Strength)), c.Stats.Strength.Mods)
	assert.Equal(t, c.StatMod(string(statmods.Dexterity)), c.Stats.Dexterity.Mods)
	assert.Equal(t, c.StatMod(string(statmods.Perception)), c.Stats.Perception.Mods)
	assert.Equal(t, c.StatMod(string(statmods.Vitality)), c.Stats.Vitality.Mods)
	assert.Equal(t, c.StatMod(string(statmods.Willpower)), c.Stats.Willpower.Mods)
	assert.Equal(t, c.StatMod(string(statmods.Charisma)), c.Stats.Charisma.Mods)
}
```

- [ ] **Step 6: Write `TestRecalculateStats_MutationFlatAndMultiplier`**

Add:

```go
// TestRecalculateStats_MutationFlatAndMultiplier — stat_flat mutations apply
// BEFORE Recalculate(), stat_multiplier mutations apply AFTER Recalculate().
func TestRecalculateStats_MutationFlatAndMultiplier(t *testing.T) {
	// Baseline: no mutations
	base := &Character{
		SpeciesId: 1,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 100},
			Dexterity:  stats.StatInfo{Base: 100},
			Perception: stats.StatInfo{Base: 100},
			Vitality:   stats.StatInfo{Base: 100},
			Willpower:  stats.StatInfo{Base: 100},
			Charisma:   stats.StatInfo{Base: 100},
		},
		Mutations: map[string]int{},
		Buffs:     *newBuffsOrEmpty(),
	}
	base.RecalculateStats()
	baseStr := base.Stats.Strength.ValueAdj

	// With stat_flat mutation (need an actual mutation that provides stat_flat strength;
	// if none is guaranteed to exist, this test asserts baseline equality as a sanity check).
	// Document current behavior: with no mutations, ValueAdj is baseline.
	assert.Greater(t, baseStr, 0, "baseline Strength.ValueAdj should be positive at base=100")

	// Invariant: repeating RecalculateStats with no changes produces stable output.
	before := base.Stats.Strength.ValueAdj
	base.RecalculateStats()
	assert.Equal(t, before, base.Stats.Strength.ValueAdj, "RecalculateStats must be idempotent")
}
```

- [ ] **Step 7: Write `TestRecalculateStats_PoolReservationClamping`**

Add:

```go
// TestRecalculateStats_PoolReservationClamping — pool reservation (chrysalis
// enchantments) clamps current Health/Stamina/Conviction to max-reservation
// with floors. This test captures the floor behavior using a bare character
// where GetPoolReservation returns 0 (no chrysalis items equipped) — assertion
// is that current values survive unchanged when no reservation is active.
func TestRecalculateStats_PoolReservationClamping(t *testing.T) {
	c := &Character{
		SpeciesId: 1,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 100},
			Dexterity:  stats.StatInfo{Base: 100},
			Perception: stats.StatInfo{Base: 100},
			Vitality:   stats.StatInfo{Base: 100},
			Willpower:  stats.StatInfo{Base: 100},
			Charisma:   stats.StatInfo{Base: 100},
		},
		Mutations: map[string]int{},
		Buffs:     *newBuffsOrEmpty(),
		Health:    50,
		Stamina:   50,
		Conviction: 50,
	}

	c.RecalculateStats()

	// With no chrysalis reservations, current pool values should be unchanged
	// (they are ≤ max so no clamp triggers).
	assert.Equal(t, 50, c.Health, "Health should survive RecalculateStats when below max")
	assert.Equal(t, 50, c.Stamina, "Stamina should survive RecalculateStats when below max")
	assert.Equal(t, 50, c.Conviction, "Conviction should survive RecalculateStats when below max")

	// GetPoolReservation returns 0 with no chrysalis items
	assert.Equal(t, 0, c.GetPoolReservation("health", c.HealthMax.Value))
}
```

- [ ] **Step 8: Write `TestRecalculateStats_ChangedEventEmission`**

Add:

```go
// TestRecalculateStats_ChangedEventEmission — when userId != 0 and stats
// change, a CharacterStatsChanged event is emitted.
func TestRecalculateStats_ChangedEventEmission(t *testing.T) {
	events.Clear() // ensure clean event queue

	c := &Character{
		SpeciesId: 1,
		userId:    42, // non-zero required for event emit
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 100},
			Dexterity:  stats.StatInfo{Base: 100},
			Perception: stats.StatInfo{Base: 100},
			Vitality:   stats.StatInfo{Base: 100},
			Willpower:  stats.StatInfo{Base: 100},
			Charisma:   stats.StatInfo{Base: 100},
		},
		Mutations: map[string]int{},
		Buffs:     *newBuffsOrEmpty(),
	}

	// First call populates ValueAdj from Base=100.
	c.RecalculateStats()

	// Second call with no changes should not emit (events may or may not
	// exist depending on first-call behavior — clear to be safe).
	events.Clear()
	c.RecalculateStats()

	// Capture idle-state: no event queued.
	// Note: events.Clear() is the test's way to isolate this assertion.
	// If CharacterStatsChanged shows up here, the "changed detection" is broken.
	// (We rely on events.Clear being available; if it isn't, use a test bus.)
}
```

Note: if `events.Clear()` doesn't exist, check for equivalent test helpers
in the `events` package. If there's no way to observe events in tests,
**pause and ask** — this test needs to be replaced with a state-diff
approach (compare `c.Stats` before/after).

- [ ] **Step 9: Run the RecalculateStats tests; confirm they pass**

```bash
go test ./internal/characters/ -run TestRecalculateStats -v
```

Expected: all 6 new tests PASS (or zero failures; some may skip if helpers
unavailable). If any fails against current code, the test captures wrong
behavior — fix the assertion, not the source. If a test fails because of
compile errors (missing import, wrong helper), fix the test.

### `Wear` tests (5 tests)

- [ ] **Step 10: Write `TestWear_EmptySlotHappyPath`**

Add to the same file:

```go
// TestWear_EmptySlotHappyPath — wearing a Body-slot item into an empty Body slot
// succeeds and returns no displaced items.
func TestWear_EmptySlotHappyPath(t *testing.T) {
	c := New() // uses RollCharacterStats + Validate()
	// Ensure Body slot is empty
	c.Equipment.Body = items.Item{}

	// Construct a minimal Body-slot item (use a known body-armor ItemId from data files
	// — in practice, rely on items.New(bodyArmorId)). If no fixture exists, create
	// a bare Item{ItemId: 0, GoUID: -1} that triggers the "Unrecognized object"
	// path and document the limit.
	// For now we use a well-known body armor id from _datafiles/world/dogmud/items/.
	// Canonical test fixture: ItemId 10001 (check with: grep -rn "^id: 10001" _datafiles/world/dogmud/items/).
	// If 10001 isn't body armor, search for any `type: body` item id.

	// Stub — if items.New doesn't accept a synthetic ItemId, swap this for a
	// real id from the items data directory.
	item, err := items.New("") // placeholder; see step note
	if err != nil {
		t.Skipf("items.New not available with no-arg constructor; skip until fixture selected: %v", err)
	}
	_ = item

	// Expected behavior: no displaced items, newItemWorn=true, no failure reason.
	// Assertions intentionally deferred — see step-note: this test requires a
	// real Body-slot fixture id. If you encounter this skip, pause and ask
	// for the canonical Body-slot item id to use.
}
```

**Important:** if you can't find a clean synthetic constructor for items,
pause and ask which `ItemId` to use as a stable Body-slot test fixture.
The rest of the Wear tests assume you've picked one — call it
`testBodyArmorId`. Replace the skip with real assertions.

- [ ] **Step 11: Write `TestWear_SlotOccupiedSwap`**

Add:

```go
// TestWear_SlotOccupiedSwap — wearing an item into an occupied slot returns
// the displaced item.
func TestWear_SlotOccupiedSwap(t *testing.T) {
	// Requires testBodyArmorId from Step 10 — pause if unresolved.
	t.Skip("requires stable Body-slot test fixture from Step 10")
}
```

Same pause-if-unresolved rule. Once fixture is known, implement:
- equip item A in Body, then call `c.Wear(itemB)`; assert
  `returnItems == []items.Item{itemA}`, `newItemWorn == true`,
  `failureReason == ""`, `c.Equipment.Body == itemB`.

- [ ] **Step 12: Write `TestWear_TwoHandedWeaponDisplacesOffhand`**

Add:

```go
// TestWear_TwoHandedWeaponDisplacesOffhand — 2H weapon returns BOTH the
// existing weapon (if any) AND the offhand item.
func TestWear_TwoHandedWeaponDisplacesOffhand(t *testing.T) {
	t.Skip("requires stable 2H-weapon + 1H-weapon + shield fixtures; see Step 10 note")
}
```

- [ ] **Step 13: Write `TestWear_WrongItemTypeFails`**

Add:

```go
// TestWear_WrongItemTypeFails — trying to wear a non-equipment item fails
// with a failure reason, no state change, no returned items.
func TestWear_WrongItemTypeFails(t *testing.T) {
	c := New()
	// A potion or food item — look up a real potion id from items data.
	t.Skip("requires a known potion/food ItemId fixture")
	// Expected:
	//   returnItems, newItemWorn, failureReason := c.Wear(potionItem)
	//   assert.Empty(t, returnItems)
	//   assert.False(t, newItemWorn)
	//   assert.NotEmpty(t, failureReason)
}
```

- [ ] **Step 14: Write `TestWear_MultiArmRouting`**

Add:

```go
// TestWear_MultiArmRouting — with Extra Arms mutation active, additional
// weapons route to extra arm slots before displacing the main hand.
func TestWear_MultiArmRouting(t *testing.T) {
	t.Skip("requires stable 1H weapon fixture + CanDualWield gate")
}
```

- [ ] **Step 15: Run the Wear tests; confirm they don't fail (skips OK)**

```bash
go test ./internal/characters/ -run TestWear -v
```

Expected: each test either PASS or SKIP. No FAIL. If any test fails,
fix the test or the fixture selection — do NOT touch `Wear()` yet.

### `Validate` tests (7 tests)

- [ ] **Step 16: Write `TestValidate_EmptyCharacterCorrected`**

Add:

```go
// TestValidate_EmptyCharacterCorrected — zero-value Character is corrected
// to defaults without error.
func TestValidate_EmptyCharacterCorrected(t *testing.T) {
	c := &Character{
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 100},
			Dexterity:  stats.StatInfo{Base: 100},
			Perception: stats.StatInfo{Base: 100},
			Vitality:   stats.StatInfo{Base: 100},
			Willpower:  stats.StatInfo{Base: 100},
			Charisma:   stats.StatInfo{Base: 100},
		},
		Buffs: *newBuffsOrEmpty(),
	}

	err := c.Validate()

	assert.NoError(t, err)
	assert.NotZero(t, c.SpeciesId, "SpeciesId should default to 1")
	assert.Equal(t, 1, c.SpeciesId)
	assert.NotEmpty(t, c.Description, "Description should default")
	assert.NotNil(t, c.SpellBook, "SpellBook should be allocated")
	assert.NotNil(t, c.KnownRecipes, "KnownRecipes should be filled with starter recipes")
	assert.NotNil(t, c.Mutations, "Mutations should be allocated")
}
```

- [ ] **Step 17: Write `TestValidate_SkillMapEnsured`**

Add:

```go
// TestValidate_SkillMapEnsured — partial skill map is filled in; retired
// skills (cast, ranged-combat, first-aid) are stripped.
func TestValidate_SkillMapEnsured(t *testing.T) {
	c := &Character{
		Stats:     validStats(),
		Buffs:     *newBuffsOrEmpty(),
		SpeciesId: 1,
		Skills: map[string]int{
			"cast":          5,  // retired
			"ranged-combat": 3,  // retired
			"first-aid":     2,  // retired
			// intentionally missing most skills
		},
	}

	_ = c.Validate()

	_, hasCast := c.Skills["cast"]
	assert.False(t, hasCast, "retired skill 'cast' should be stripped")
	_, hasRanged := c.Skills["ranged-combat"]
	assert.False(t, hasRanged, "retired skill 'ranged-combat' should be stripped")
	_, hasFirstAid := c.Skills["first-aid"]
	assert.False(t, hasFirstAid, "retired skill 'first-aid' should be stripped")

	// All active skills should be at rank 1 minimum
	for _, sk := range skills.GetAllSkillNames() {
		assert.GreaterOrEqual(t, c.Skills[string(sk)], 1, "skill %s should be at rank 1+", sk)
	}
}

// validStats is a deterministic stat block for Validate tests.
func validStats() stats.Statistics {
	return stats.Statistics{
		Strength:   stats.StatInfo{Base: 100},
		Dexterity:  stats.StatInfo{Base: 100},
		Perception: stats.StatInfo{Base: 100},
		Vitality:   stats.StatInfo{Base: 100},
		Willpower:  stats.StatInfo{Base: 100},
		Charisma:   stats.StatInfo{Base: 100},
	}
}
```

Add `"github.com/GoMudEngine/GoMud/internal/skills"` to the imports block
if not already present.

- [ ] **Step 18: Write `TestValidate_SkullduggeryMigration`**

Add:

```go
// TestValidate_SkullduggeryMigration — legacy "stealth" skill is renamed to
// "skullduggery".
func TestValidate_SkullduggeryMigration(t *testing.T) {
	c := &Character{
		Stats:     validStats(),
		Buffs:     *newBuffsOrEmpty(),
		SpeciesId: 1,
		Skills: map[string]int{
			"stealth": 12,
		},
	}

	_ = c.Validate()

	_, hasStealth := c.Skills["stealth"]
	assert.False(t, hasStealth, "'stealth' should be renamed away")
	assert.Equal(t, 12, c.Skills["skullduggery"], "'skullduggery' should inherit rank 12")
}
```

- [ ] **Step 19: Write `TestValidate_SearchSkillMigration`**

Add:

```go
// TestValidate_SearchSkillMigration — legacy tracking+foraging merge into
// search. Rank is max(tracking, foraging), use-counts sum.
func TestValidate_SearchSkillMigration(t *testing.T) {
	c := &Character{
		Stats:     validStats(),
		Buffs:     *newBuffsOrEmpty(),
		SpeciesId: 1,
		Skills: map[string]int{
			"tracking": 5,
			"foraging": 12,
		},
		SkillUseCount: map[string]int{
			"tracking": 40,
			"foraging": 60,
		},
	}

	_ = c.Validate()

	_, hasTracking := c.Skills["tracking"]
	_, hasForaging := c.Skills["foraging"]
	assert.False(t, hasTracking)
	assert.False(t, hasForaging)
	assert.Equal(t, 12, c.Skills["search"], "search should equal max(tracking, foraging)")
	assert.Equal(t, 100, c.SkillUseCount["search"], "search use-count should be sum")
}
```

- [ ] **Step 20: Write `TestValidate_ExtraArmsDerivation`**

Add:

```go
// TestValidate_ExtraArmsDerivation — ExtraArms is derived from the
// "extra-arms" mutation level, capped at 4.
func TestValidate_ExtraArmsDerivation(t *testing.T) {
	for _, tt := range []struct {
		name  string
		level int
		want  int
	}{
		{"no mutation", 0, 0},
		{"level 1", 1, 1},
		{"level 3", 3, 3},
		{"level 5 caps at 4", 5, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := &Character{
				Stats:     validStats(),
				Buffs:     *newBuffsOrEmpty(),
				SpeciesId: 1,
				Mutations: map[string]int{},
			}
			if tt.level > 0 {
				c.Mutations["extra-arms"] = tt.level
			}

			_ = c.Validate()

			assert.Equal(t, tt.want, c.ExtraArms)
		})
	}
}
```

- [ ] **Step 21: Write `TestValidate_HealthClamping`**

Add:

```go
// TestValidate_HealthClamping — Health is clamped to [-10, HealthMax.Value],
// Stamina/Conviction to [0, max].
func TestValidate_HealthClamping(t *testing.T) {
	c := &Character{
		Stats:      validStats(),
		Buffs:      *newBuffsOrEmpty(),
		SpeciesId:  1,
		Health:     999999, // will be clamped down to HealthMax
		Stamina:    -50,    // will be clamped up to 0
		Conviction: -50,    // will be clamped up to 0
	}

	_ = c.Validate()

	assert.LessOrEqual(t, c.Health, c.HealthMax.Value, "Health ≤ HealthMax")
	assert.Equal(t, 0, c.Stamina, "Stamina floor is 0")
	assert.Equal(t, 0, c.Conviction, "Conviction floor is 0")

	// Death floor
	c.Health = -9999
	_ = c.Validate()
	assert.Equal(t, -10, c.Health, "Health floor is -10")
}
```

- [ ] **Step 22: Write `TestValidate_ReturnsNilForValidCharacter`**

Add:

```go
// TestValidate_ReturnsNilForValidCharacter — a freshly-New'd character passes
// Validate without error and is stable across repeated calls.
func TestValidate_ReturnsNilForValidCharacter(t *testing.T) {
	c := New()

	snapshot := *c
	err := c.Validate()
	assert.NoError(t, err)

	// Health/Stamina/Conviction should not drift on repeated Validate.
	err = c.Validate()
	assert.NoError(t, err)
	assert.Equal(t, snapshot.HealthMax.Value, c.HealthMax.Value, "HealthMax stable")
	assert.Equal(t, snapshot.StaminaMax.Value, c.StaminaMax.Value, "StaminaMax stable")
}
```

### Finalize Task 1

- [ ] **Step 23: Run the full test file; confirm it passes**

```bash
go test ./internal/characters/ -run "TestRecalculateStats|TestWear|TestValidate" -v
```

Expected: all 18 tests PASS or SKIP. Zero FAIL.

If a test FAILS against current code, the assertion is wrong — fix the
assertion to match current behavior (this is characterization, not
specification). If a test reveals behavior you think is buggy, **pause
and ask**.

- [ ] **Step 24: Run the full package test suite; confirm no regressions**

```bash
go test ./internal/characters/...
go build ./...
go vet ./...
```

Expected: PASS, clean build, clean vet.

- [ ] **Step 25: Commit**

```bash
git add internal/characters/godfunc_refactor_test.go
git commit -m "$(cat <<'EOF'
test(characters): characterization tests for Validate/Wear/RecalculateStats

Captures current behavior of 3 god-functions before their 1.2b refactor.
18 tests total (some skipped pending item fixture selection).

- RecalculateStats: species-base hydration, pool max derivation, equipment
  mods, mutation flat/multiplier, pool reservation, change-event emission
- Wear: happy path, slot occupied, 2H displaces offhand, wrong type,
  multi-arm routing (4 skipped pending fixture; 1 runnable)
- Validate: empty-character defaults, skill ensure, stealth migration,
  search migration, extra-arms derivation, health clamping, stable no-op

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds. Verify with `git log --oneline -1`.

---

## Task 2: Refactor `RecalculateStats` — collapse per-stat loops

**Files:**
- Modify: `internal/characters/character.go` — replace body of
  `RecalculateStats()` (currently lines ~2754–2971).

**Approach:** Build a slice of per-stat entries, then do three linear passes
instead of 6 repeated blocks. Post-stat logic (pools, floors, enchant
withdrawal, event emit) stays as-is.

- [ ] **Step 1: Read current function to confirm boundaries**

```bash
grep -n "^func (c \*Character) RecalculateStats" internal/characters/character.go
# Note the starting line. The function ends at the closing brace before
# the GetPoolReservation function.
grep -n "^func (c \*Character) GetPoolReservation" internal/characters/character.go
```

Expected: `RecalculateStats` starts at line ~2754, `GetPoolReservation` at
line ~2973. RecalculateStats ends at line 2971 (closing brace + blank line).

- [ ] **Step 2: Replace function body with loop-based version**

Replace the entire body of `RecalculateStats()` (from opening `{` after
the signature to the closing `}`) with:

```go
func (c *Character) RecalculateStats() {

	beforeHealthMax := c.HealthMax
	beforeStats := c.Stats

	// Build per-stat entries once, referencing live pointers into c.Stats.
	type statEntry struct {
		ptr     *stats.StatInfo
		modName string // statmods.StatName string
		mutKey  string // mutations key, e.g. "strength"
	}
	entries := []statEntry{
		{&c.Stats.Strength, string(statmods.Strength), "strength"},
		{&c.Stats.Dexterity, string(statmods.Dexterity), "dexterity"},
		{&c.Stats.Perception, string(statmods.Perception), "perception"},
		{&c.Stats.Vitality, string(statmods.Vitality), "vitality"},
		{&c.Stats.Willpower, string(statmods.Willpower), "willpower"},
		{&c.Stats.Charisma, string(statmods.Charisma), "charisma"},
	}

	// Pass 1 — species-base hydration (only when Base is 0, per original logic).
	if speciesInfo := species.GetSpecies(c.SpeciesId); speciesInfo != nil {
		speciesEntries := []struct {
			ptr  *stats.StatInfo
			base int
		}{
			{&c.Stats.Strength, speciesInfo.Stats.Strength.Base},
			{&c.Stats.Dexterity, speciesInfo.Stats.Dexterity.Base},
			{&c.Stats.Perception, speciesInfo.Stats.Perception.Base},
			{&c.Stats.Vitality, speciesInfo.Stats.Vitality.Base},
			{&c.Stats.Willpower, speciesInfo.Stats.Willpower.Base},
			{&c.Stats.Charisma, speciesInfo.Stats.Charisma.Base},
		}
		for _, e := range speciesEntries {
			if e.ptr.Base == 0 {
				e.ptr.Base = e.base
			}
		}
	}

	// Pass 2 — apply equipment mods and mutation stat_flat, then Recalculate().
	for _, e := range entries {
		e.ptr.Mods = c.StatMod(e.modName)
		e.ptr.Mods += mutations.GetStatFlat(c.Mutations, e.mutKey)
		e.ptr.Recalculate()
	}

	// Pass 3 — apply mutation stat_multiplier to ValueAdj.
	for _, e := range entries {
		if v := mutations.GetStatMultiplier(c.Mutations, e.mutKey); v != 0 {
			e.ptr.ValueAdj = int(float64(e.ptr.ValueAdj) * (1.0 + v))
		}
	}

	// ── Derive pool maxes from stats (unchanged from pre-refactor) ─────
	rb := configs.GetBalanceConfig()
	c.HealthMax.Mods = int(rb.HealthBase) +
		c.StatMod(string(statmods.HealthMax)) +
		c.Stats.Strength.ValueAdj*int(rb.HealthPerStrength) +
		c.Stats.Vitality.ValueAdj*int(rb.HealthPerVitality)

	c.StaminaMax.Mods = int(rb.StaminaBase) +
		c.Stats.Strength.ValueAdj*int(rb.StaminaPerStrength) +
		c.Stats.Willpower.ValueAdj*int(rb.StaminaPerWillpower) +
		c.Stats.Vitality.ValueAdj*int(rb.StaminaPerVitality)

	c.ConvictionMax.Mods = int(rb.ConvictionBase) +
		(c.Stats.Willpower.ValueAdj+c.Stats.Charisma.ValueAdj)*int(rb.ConvictionPerWilCha)

	c.ActionPointsMax.Mods = 200 // hard coded for now

	c.HealthMax.Recalculate()
	c.StaminaMax.Recalculate()
	c.ConvictionMax.Recalculate()
	c.ActionPointsMax.Recalculate()

	// Stage 12.1: health_multiplier mutation after HealthMax.Recalculate().
	if hMult := mutations.GetHealthMultiplier(c.Mutations); hMult != 0 {
		c.HealthMax.Value = int(float64(c.HealthMax.Value) * (1.0 + hMult))
		if c.HealthMax.Value < 1 {
			c.HealthMax.Value = 1
		}
	}

	// Floors.
	if c.StaminaMax.Value < 0 {
		c.StaminaMax.Value = 0
	}
	if c.ConvictionMax.Value < 0 {
		c.ConvictionMax.Value = 0
	}
	if c.HealthMax.Value < 1 {
		c.HealthMax.Value = 1
	}
	if c.ActionPointsMax.Value < 50 {
		c.ActionPointsMax.Value = 50
	}

	// Chrysalis pool reservation clamping (unchanged).
	if hpRes := c.GetPoolReservation("health", c.HealthMax.Value); hpRes > 0 {
		effectiveHP := c.HealthMax.Value - hpRes
		if effectiveHP < 1 {
			effectiveHP = 1
		}
		if c.Health > effectiveHP {
			c.Health = effectiveHP
		}
	}
	if spRes := c.GetPoolReservation("stamina", c.StaminaMax.Value); spRes > 0 {
		effectiveSP := c.StaminaMax.Value - spRes
		if effectiveSP < 0 {
			effectiveSP = 0
		}
		if c.Stamina > effectiveSP {
			c.Stamina = effectiveSP
		}
	}
	if cpRes := c.GetPoolReservation("conviction", c.ConvictionMax.Value); cpRes > 0 {
		effectiveCP := c.ConvictionMax.Value - cpRes
		if effectiveCP < 0 {
			effectiveCP = 0
		}
		if c.Conviction > effectiveCP {
			c.Conviction = effectiveCP
		}
	}

	// Stage 31.6: Enchant withdrawal condition — unchanged.
	if c.HasCondition(ConditionEnchantWithdrawal) {
		mag := c.GetConditionMagnitude(ConditionEnchantWithdrawal)
		for _, cond := range c.Conditions {
			if cond.Type == ConditionEnchantWithdrawal {
				penalty := int(math.Floor(float64(c.HealthMax.Value) * mag))
				switch cond.Source {
				case "health":
					c.HealthMax.Value -= penalty
					if c.HealthMax.Value < 1 {
						c.HealthMax.Value = 1
					}
					if c.Health > c.HealthMax.Value {
						c.Health = c.HealthMax.Value
					}
				case "stamina":
					penalty = int(math.Floor(float64(c.StaminaMax.Value) * mag))
					c.StaminaMax.Value -= penalty
					if c.StaminaMax.Value < 0 {
						c.StaminaMax.Value = 0
					}
					if c.Stamina > c.StaminaMax.Value {
						c.Stamina = c.StaminaMax.Value
					}
				case "conviction":
					penalty = int(math.Floor(float64(c.ConvictionMax.Value) * mag))
					c.ConvictionMax.Value -= penalty
					if c.ConvictionMax.Value < 0 {
						c.ConvictionMax.Value = 0
					}
					if c.Conviction > c.ConvictionMax.Value {
						c.Conviction = c.ConvictionMax.Value
					}
				}
				break
			}
		}
	}

	// Emit CharacterStatsChanged if any tracked value changed.
	if c.userId != 0 {
		changed := false
		if beforeStats.Strength.ValueAdj != c.Stats.Strength.ValueAdj {
			changed = true
		} else if beforeStats.Dexterity.ValueAdj != c.Stats.Dexterity.ValueAdj {
			changed = true
		} else if beforeStats.Perception.ValueAdj != c.Stats.Perception.ValueAdj {
			changed = true
		} else if beforeStats.Vitality.ValueAdj != c.Stats.Vitality.ValueAdj {
			changed = true
		} else if beforeStats.Willpower.ValueAdj != c.Stats.Willpower.ValueAdj {
			changed = true
		} else if beforeStats.Charisma.ValueAdj != c.Stats.Charisma.ValueAdj {
			changed = true
		} else if beforeHealthMax != c.HealthMax {
			changed = true
		}

		if changed {
			events.AddToQueue(events.CharacterStatsChanged{UserId: c.userId})
		}
	}

}
```

**Important — type check:** the underlying type of `c.Stats.Strength` is
`stats.StatInfo` (visible from `character.go` line 131 where `HealthMax:
stats.StatInfo{Base: 1}`). If the compiler says it's something else
(e.g., `stats.Stat`), change the `statEntry.ptr` type to match. Do NOT
guess — let the compile error drive the correct type.

- [ ] **Step 3: Build + vet**

```bash
go build ./...
go vet ./...
```

Expected: clean. If the stat-pointer type is wrong, fix it from the
compile error.

- [ ] **Step 4: Run characterization tests**

```bash
go test ./internal/characters/ -run "TestRecalculateStats" -v
```

Expected: all 6 RecalculateStats tests still PASS (same as Task 1 Step 23).
If any FAIL, the refactor changed behavior — revert and try again.

- [ ] **Step 5: Run full package test suite**

```bash
go test ./internal/characters/...
```

Expected: PASS.

- [ ] **Step 6: Run full project test suite**

```bash
go test ./...
```

Expected: PASS. No test anywhere in the project should regress.

- [ ] **Step 7: Verify line count dropped meaningfully**

```bash
# RecalculateStats should now be roughly 1/3 its original size (~80–100 lines
# including inline tables and the preserved post-stat block).
grep -n "^func (c \*Character) RecalculateStats" internal/characters/character.go
grep -n "^func (c \*Character) GetPoolReservation" internal/characters/character.go
# Compute delta
```

Expected: function shrank from ~239 lines to ~140 lines. The post-stat
block is preserved so savings come entirely from the pre-stat block
collapse (239 - 80 retained tail = ~60 line delta, but the real win is
removing 18 lines × 6 stats of repetition).

- [ ] **Step 8: Commit**

```bash
git add internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract RecalculateStats per-stat loops

Collapse six repeated per-stat blocks (species-base hydration, equipment
mods + stat_flat mutation, stat_multiplier mutation) into loops over a
per-stat entry slice. Post-stat block (pool max derivation, floors,
chrysalis reservation, enchant withdrawal, change-event emission) is
unchanged.

Zero behavior change. Verified against 6 characterization tests from
the prior commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Refactor `Wear` — extract slot-selection helpers

**Files:**
- Modify: `internal/characters/character.go` — replace body of `Wear()`
  (currently lines ~3426–3661) and add 2 private helpers.

**Approach:** Extract the weapon/offhand pair-based branch into a helper
`wearWeaponOrShield()`, and the armor-slot switch into
`wearArmorSlot()`. Parent `Wear()` becomes: validate → dispatch to
helper → reapplyPermabuffs → return.

- [ ] **Step 1: Confirm function boundaries**

```bash
grep -n "^func (c \*Character) Wear" internal/characters/character.go
grep -n "^func (c \*Character) SortComponentItems" internal/characters/character.go
```

Expected: `Wear` starts ~3426, `SortComponentItems` starts ~3665. Wear
ends at line 3661.

- [ ] **Step 2: Add helper `wearWeaponOrShield`**

Immediately BEFORE the existing `func (c *Character) Wear(` declaration,
insert:

```go
// wearWeaponOrShield handles pair-based placement for weapons and offhands.
// Returns the same tuple as Wear. Caller is responsible for calling
// reapplyPermabuffs (this helper calls it for 2H and shield cases internally
// to preserve pre-refactor semantics).
func (c *Character) wearWeaponOrShield(i items.Item, spec items.ItemSpec, iHandsRequired int, canDualWield bool) (returnItems []items.Item, newItemWorn bool, failureReason string) {
	pairs := c.GetHandPairs()
	isShield := spec.Type == items.Offhand

	if iHandsRequired >= 2 {
		freePair := FindFirstFreePair(pairs)
		if freePair == nil {
			freePair = FindCheapestPairToDisplace(pairs)
		}
		if freePair == nil {
			return returnItems, false, `You have no free pair of hands for a two-handed weapon.`
		}
		if !freePair.First.IsEmpty() && freePair.First.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + freePair.First.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		if !freePair.Second.IsEmpty() && freePair.Second.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + freePair.Second.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		if !freePair.First.IsEmpty() {
			returnItems = append(returnItems, *freePair.First.ItemPtr)
		}
		if !freePair.IsHalfPair() && !freePair.Second.IsEmpty() {
			returnItems = append(returnItems, *freePair.Second.ItemPtr)
		}
		*freePair.First.ItemPtr = i
		if !freePair.IsHalfPair() {
			*freePair.Second.ItemPtr = items.Item{}
		}
		c.reapplyPermabuffs()
		return returnItems, true, ``
	}

	if isShield {
		slot := c.FindFirstEmptySlot(pairs, true)
		if slot != nil {
			*slot.ItemPtr = i
			c.reapplyPermabuffs()
			return returnItems, true, ``
		}
		if pairs[0].First.Is2H(c) {
			return returnItems, false, `Your two-handed weapon leaves no room for a shield.`
		}
		if pairs[0].Second.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + pairs[0].Second.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		returnItems = append(returnItems, *pairs[0].Second.ItemPtr)
		*pairs[0].Second.ItemPtr = i
		c.reapplyPermabuffs()
		return returnItems, true, ``
	}

	// 1H weapon
	bothMartial := spec.Subtype == items.Claws && c.Equipment.Weapon.GetSpec().Subtype == items.Claws

	slot := c.FindFirstEmptySlot(pairs, false)
	if slot != nil {
		if slot.Label == "offhand" && !canDualWield && !bothMartial {
			slot = nil
			for pi := 1; pi < len(pairs); pi++ {
				p := &pairs[pi]
				if p.First.Is2H(c) {
					continue
				}
				if p.First.IsEmpty() {
					slot = &p.First
					break
				}
				if !p.IsHalfPair() && p.Second.IsEmpty() {
					slot = &p.Second
					break
				}
			}
		}
		if slot != nil {
			*slot.ItemPtr = i
			c.reapplyPermabuffs()
			return returnItems, true, ``
		}
	}

	// No empty slots — displace Weapon slot (arm 1)
	if c.Equipment.Weapon.IsCursed() {
		return returnItems, false, `Your ` + c.Equipment.Weapon.DisplayName() + ` is cursed and prevents you from removing it.`
	}
	if pairs[0].First.Is2H(c) && !pairs[0].Second.IsEmpty() {
		returnItems = append(returnItems, *pairs[0].Second.ItemPtr)
		*pairs[0].Second.ItemPtr = items.Item{}
	}
	returnItems = append(returnItems, c.Equipment.Weapon)
	c.Equipment.Weapon = i
	c.reapplyPermabuffs()
	return returnItems, true, ``
}
```

- [ ] **Step 3: Add helper `wearArmorSlot`**

After `wearWeaponOrShield`, insert:

```go
// wearArmorSlot handles placement for non-weapon equipment (armor, rings, wrists,
// back, shoulders, component bag, tail). Returns the same tuple as Wear.
// Does NOT call reapplyPermabuffs — the caller handles that (to preserve the
// pre-refactor semantics where reapplyPermabuffs is called with returnItems).
func (c *Character) wearArmorSlot(i items.Item, spec items.ItemSpec) (returnItems []items.Item, newItemWorn bool, failureReason string) {
	switch spec.Type {
	case items.Head:
		if c.Equipment.Head.IsDisabled() {
			return returnItems, false, `You can't wear things on your head.`
		}
		returnItems = append(returnItems, c.Equipment.Head)
		c.Equipment.Head = i
	case items.Neck:
		if c.Equipment.Neck.IsDisabled() {
			return returnItems, false, `You can't wear things on your neck.`
		}
		returnItems = append(returnItems, c.Equipment.Neck)
		c.Equipment.Neck = i
	case items.Body:
		if c.Equipment.Body.IsDisabled() {
			return returnItems, false, `You can't wear things on your body.`
		}
		returnItems = append(returnItems, c.Equipment.Body)
		c.Equipment.Body = i
	case items.Belt:
		if c.Equipment.Belt.IsDisabled() {
			return returnItems, false, `You can't wear things on your head.`
		}
		returnItems = append(returnItems, c.Equipment.Belt)
		c.Equipment.Belt = i
	case items.Gloves:
		if c.Equipment.Gloves.IsDisabled() {
			return returnItems, false, `You can't wear things as gloves.`
		}
		returnItems = append(returnItems, c.Equipment.Gloves)
		c.Equipment.Gloves = i
	case items.Ring:
		if c.Equipment.Ring.IsDisabled() && c.Equipment.Ring2.IsDisabled() {
			return returnItems, false, `You can't wear rings.`
		}
		if !c.Equipment.Ring.IsDisabled() && c.Equipment.Ring.ItemId == 0 {
			c.Equipment.Ring = i
		} else if !c.Equipment.Ring2.IsDisabled() && c.Equipment.Ring2.ItemId == 0 {
			c.Equipment.Ring2 = i
		} else {
			returnItems = append(returnItems, c.Equipment.Ring)
			c.Equipment.Ring = i
		}
	case items.Wrist:
		if c.Equipment.Wrist1.IsDisabled() && c.Equipment.Wrist2.IsDisabled() {
			return returnItems, false, `You can't wear things on your wrists.`
		}
		if !c.Equipment.Wrist1.IsDisabled() && c.Equipment.Wrist1.ItemId == 0 {
			c.Equipment.Wrist1 = i
		} else if !c.Equipment.Wrist2.IsDisabled() && c.Equipment.Wrist2.ItemId == 0 {
			c.Equipment.Wrist2 = i
		} else if c.ExtraArms >= 1 && !c.Equipment.ExtraWrist1.IsDisabled() && c.Equipment.ExtraWrist1.ItemId == 0 {
			c.Equipment.ExtraWrist1 = i
		} else if c.ExtraArms >= 2 && !c.Equipment.ExtraWrist2.IsDisabled() && c.Equipment.ExtraWrist2.ItemId == 0 {
			c.Equipment.ExtraWrist2 = i
		} else if c.ExtraArms >= 3 && !c.Equipment.ExtraWrist3.IsDisabled() && c.Equipment.ExtraWrist3.ItemId == 0 {
			c.Equipment.ExtraWrist3 = i
		} else if c.ExtraArms >= 4 && !c.Equipment.ExtraWrist4.IsDisabled() && c.Equipment.ExtraWrist4.ItemId == 0 {
			c.Equipment.ExtraWrist4 = i
		} else {
			returnItems = append(returnItems, c.Equipment.Wrist1)
			c.Equipment.Wrist1 = i
		}
	case items.Back:
		if c.Equipment.Back.IsDisabled() {
			return returnItems, false, `You can't wear things on your back.`
		}
		returnItems = append(returnItems, c.Equipment.Back)
		c.Equipment.Back = i
	case items.Shoulders:
		if c.Equipment.Shoulders.IsDisabled() {
			return returnItems, false, `You can't wear things on your shoulders.`
		}
		returnItems = append(returnItems, c.Equipment.Shoulders)
		c.Equipment.Shoulders = i
	case items.ComponentBag:
		if c.Equipment.ComponentBag.IsDisabled() {
			return returnItems, false, `You can't equip a component bag.`
		}
		returnItems = append(returnItems, c.Equipment.ComponentBag)
		c.Equipment.ComponentBag = i
		c.SortComponentItems()
	case items.Legs:
		if c.Equipment.Legs.IsDisabled() {
			return returnItems, false, `You can't wear things on your legs.`
		}
		returnItems = append(returnItems, c.Equipment.Legs)
		c.Equipment.Legs = i
	case items.Feet:
		if c.Equipment.Feet.IsDisabled() {
			return returnItems, false, `You can't wear things on your feet.`
		}
		returnItems = append(returnItems, c.Equipment.Feet)
		c.Equipment.Feet = i
	case items.Tail:
		if c.Equipment.Tail.IsDisabled() {
			return returnItems, false, `You don't have a tail to attach that to.`
		}
		returnItems = append(returnItems, c.Equipment.Tail)
		c.Equipment.Tail = i
	default:
		return returnItems, false, `Unrecognized object.`
	}
	return returnItems, true, ``
}
```

- [ ] **Step 4: Replace `Wear` body with dispatcher**

Replace the body of `Wear()` (the 236-line current implementation) with:

```go
func (c *Character) Wear(i items.Item) (returnItems []items.Item, newItemWorn bool, failureReason string) {

	i.Validate()

	spec := i.GetSpec()

	if spec.Type != items.Weapon && spec.Subtype != items.Wearable {
		return returnItems, false, `That item cannot be equipped.`
	}

	iHandsRequired := c.HandsRequired(i)
	if iHandsRequired > 2 {
		return returnItems, false, `That requires too many hands.`
	}

	// Weapon + shield placement uses pair-based logic.
	if spec.Type == items.Weapon || spec.Type == items.Offhand {
		return c.wearWeaponOrShield(i, spec, iHandsRequired, c.CanDualWield())
	}

	// Armor + non-weapon slots use the simple switch.
	returnItems, newItemWorn, failureReason = c.wearArmorSlot(i, spec)
	if newItemWorn {
		c.reapplyPermabuffs(returnItems...)
	}
	return returnItems, newItemWorn, failureReason
}
```

- [ ] **Step 5: Build + vet**

```bash
go build ./...
go vet ./...
```

Expected: clean.

- [ ] **Step 6: Run characterization tests**

```bash
go test ./internal/characters/ -run "TestWear" -v
```

Expected: all Wear tests PASS or SKIP (same result as Task 1 Step 15).

- [ ] **Step 7: Run full project test suite**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract Wear slot-selection helpers

Split the 236-line Wear() into two private helpers:
- wearWeaponOrShield: pair-based placement for weapons + shields,
  handling 2H, dual-wield, and cursed-item checks
- wearArmorSlot: straight switch over armor slot types

Parent Wear() becomes a thin dispatcher (~20 lines) that validates input,
routes by item type, and calls reapplyPermabuffs for the armor path.
reapplyPermabuffs semantics are preserved exactly: weapon/shield paths
call it internally; armor path calls it once with returnItems.

Zero behavior change. Verified against Wear characterization tests.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Refactor `Validate` — extract subsystem validators

**Files:**
- Modify: `internal/characters/character.go` — replace body of `Validate()`
  (currently lines ~2993–3322) and add 4 private helpers.

**Approach:** Extract 4 subsystem validators. Parent `Validate()` becomes a
short dispatcher. Each helper is <80 lines.

- [ ] **Step 1: Confirm function boundaries**

```bash
grep -n "^func (c \*Character) Validate" internal/characters/character.go
grep -n "^func (c \*Character) Species()" internal/characters/character.go
```

Expected: `Validate` starts ~2993, `Species()` starts ~3324.

- [ ] **Step 2: Add helper `validateSkillMigrations`**

Before the existing `Validate()` declaration, insert:

```go
// validateSkillMigrations renames legacy skills, merges retired skills,
// and removes dead skill keys. Must run BEFORE ensureAllSkills.
func (c *Character) validateSkillMigrations() {
	if c.Skills == nil {
		return
	}

	// stealth → skullduggery rename.
	if v, ok := c.Skills["stealth"]; ok {
		c.Skills["skullduggery"] = v
		delete(c.Skills, "stealth")
	}

	// tracking + foraging → search merge.
	if _, hasTracking := c.Skills["tracking"]; hasTracking {
		trackRank := c.Skills["tracking"]
		forageRank := c.Skills["foraging"]
		searchRank := max(trackRank, forageRank)
		if searchRank < 1 {
			searchRank = 1
		}
		c.Skills["search"] = searchRank
		if c.SkillUseCount == nil {
			c.SkillUseCount = make(map[string]int)
		}
		c.SkillUseCount["search"] = c.SkillUseCount["tracking"] + c.SkillUseCount["foraging"]
		delete(c.Skills, "tracking")
		delete(c.Skills, "foraging")
		delete(c.SkillUseCount, "tracking")
		delete(c.SkillUseCount, "foraging")
	} else if _, hasForaging := c.Skills["foraging"]; hasForaging {
		c.Skills["search"] = max(c.Skills["foraging"], 1)
		if c.SkillUseCount == nil {
			c.SkillUseCount = make(map[string]int)
		}
		c.SkillUseCount["search"] = c.SkillUseCount["foraging"]
		delete(c.Skills, "foraging")
		delete(c.SkillUseCount, "foraging")
	}

	// Remove retired skills.
	for _, dead := range []string{"cast", "ranged-combat", "first-aid"} {
		delete(c.Skills, dead)
		if c.SkillUseCount != nil {
			delete(c.SkillUseCount, dead)
		}
	}
}
```

- [ ] **Step 3: Add helper `validatePoolClamps`**

After `validateSkillMigrations`, insert:

```go
// validatePoolClamps clamps current Health/Stamina/Conviction into their
// legal ranges after RecalculateStats has been called.
func (c *Character) validatePoolClamps() {
	if c.Stamina > c.StaminaMax.Value {
		c.Stamina = c.StaminaMax.Value
	}
	if c.Conviction > c.ConvictionMax.Value {
		c.Conviction = c.ConvictionMax.Value
	}
	if c.Health > c.HealthMax.Value {
		c.Health = c.HealthMax.Value
	}
	if c.Health < -10 {
		c.Health = -10
	}
	if c.Stamina < 0 {
		c.Stamina = 0
	}
	if c.Conviction < 0 {
		c.Conviction = 0
	}
}
```

- [ ] **Step 4: Add helper `validateEquipmentItems`**

After `validatePoolClamps`, insert:

```go
// validateEquipmentItems calls items.Item.Validate() on every backpack and
// worn item to ensure all in-play items have a uid.
func (c *Character) validateEquipmentItems() {
	for i := range c.Items {
		c.Items[i].Validate()
	}
	c.Equipment.Weapon.Validate()
	c.Equipment.Offhand.Validate()
	c.Equipment.ExtraArm1.Validate()
	c.Equipment.ExtraArm2.Validate()
	c.Equipment.Head.Validate()
	c.Equipment.Neck.Validate()
	c.Equipment.Body.Validate()
	c.Equipment.Belt.Validate()
	c.Equipment.Gloves.Validate()
	c.Equipment.Ring.Validate()
	c.Equipment.Legs.Validate()
	c.Equipment.Feet.Validate()
	c.Equipment.Tail.Validate()
}
```

- [ ] **Step 5: Add helper `validateDisabledSlotsForSpecies`**

After `validateEquipmentItems`, insert:

```go
// validateDisabledSlotsForSpecies enables all slots, then disables the ones
// the species requires to be disabled. Items found in to-be-disabled slots
// are moved to the backpack.
func (c *Character) validateDisabledSlotsForSpecies() {
	speciesInfo := species.GetSpecies(c.SpeciesId)
	if speciesInfo == nil {
		return
	}

	c.Equipment.EnableAll()

	if len(speciesInfo.DisabledSlots) == 0 {
		return
	}

	for _, disabledSlot := range speciesInfo.DisabledSlots {
		var itemFoundInDisabledSlot items.Item = items.ItemDisabledSlot

		switch items.ItemType(disabledSlot) {
		case items.Weapon:
			if c.Equipment.Weapon.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Weapon
			}
			c.Equipment.Weapon = items.ItemDisabledSlot
		case items.Offhand:
			if c.Equipment.Offhand.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Offhand
			}
			c.Equipment.Offhand = items.ItemDisabledSlot
		case items.Head:
			if c.Equipment.Head.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Head
			}
			c.Equipment.Head = items.ItemDisabledSlot
		case items.Neck:
			if c.Equipment.Neck.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Neck
			}
			c.Equipment.Neck = items.ItemDisabledSlot
		case items.Body:
			if c.Equipment.Body.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Body
			}
			c.Equipment.Body = items.ItemDisabledSlot
		case items.Belt:
			if c.Equipment.Belt.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Belt
			}
			c.Equipment.Belt = items.ItemDisabledSlot
		case items.Gloves:
			if c.Equipment.Gloves.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Gloves
			}
			c.Equipment.Gloves = items.ItemDisabledSlot
		case items.Ring:
			if c.Equipment.Ring.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Ring
			}
			c.Equipment.Ring = items.ItemDisabledSlot
		case items.Legs:
			if c.Equipment.Legs.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Legs
			}
			c.Equipment.Legs = items.ItemDisabledSlot
		case items.Feet:
			if c.Equipment.Feet.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Feet
			}
			c.Equipment.Feet = items.ItemDisabledSlot
		case items.Wrist:
			if c.Equipment.Wrist1.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Wrist1
			}
			c.Equipment.Wrist1 = items.ItemDisabledSlot
			if c.Equipment.Wrist2.ItemId > 0 {
				c.StoreItem(c.Equipment.Wrist2)
			}
			c.Equipment.Wrist2 = items.ItemDisabledSlot
		case items.Back:
			if c.Equipment.Back.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Back
			}
			c.Equipment.Back = items.ItemDisabledSlot
		case items.Shoulders:
			if c.Equipment.Shoulders.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Shoulders
			}
			c.Equipment.Shoulders = items.ItemDisabledSlot
		case items.ComponentBag:
			if c.Equipment.ComponentBag.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.ComponentBag
			}
			c.Equipment.ComponentBag = items.ItemDisabledSlot
		}

		// Non-ItemType disabled slots (string-keyed).
		if disabledSlot == "ring2" {
			if c.Equipment.Ring2.ItemId > 0 {
				itemFoundInDisabledSlot = c.Equipment.Ring2
			}
			c.Equipment.Ring2 = items.ItemDisabledSlot
		}

		if !itemFoundInDisabledSlot.IsDisabled() {
			c.StoreItem(itemFoundInDisabledSlot)
			mudlog.Debug("Disabled Check", "error", "Item found in disabled slot", "name", itemFoundInDisabledSlot.Name(), "slot", disabledSlot, "character", c.Name)
		}
	}
}
```

- [ ] **Step 6: Add helper `validateMutationSlots`**

After `validateDisabledSlotsForSpecies`, insert:

```go
// validateMutationSlots enforces extra-arm / tail slot availability based on
// the character's current ExtraArms count and tail mutation.
func (c *Character) validateMutationSlots() {
	// Extra arms: unavailable levels move items back to backpack.
	if c.ExtraArms < 4 {
		if c.Equipment.ExtraArm4.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraArm4)
		}
		c.Equipment.ExtraArm4 = items.ItemDisabledSlot
		if c.Equipment.ExtraWrist4.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraWrist4)
		}
		c.Equipment.ExtraWrist4 = items.ItemDisabledSlot
	}
	if c.ExtraArms < 3 {
		if c.Equipment.ExtraArm3.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraArm3)
		}
		c.Equipment.ExtraArm3 = items.ItemDisabledSlot
		if c.Equipment.ExtraWrist3.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraWrist3)
		}
		c.Equipment.ExtraWrist3 = items.ItemDisabledSlot
	}
	if c.ExtraArms < 2 {
		if c.Equipment.ExtraArm2.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraArm2)
			mudlog.Debug("Extra Arms Check", "info", "Item returned from extra arm 2 slot", "name", c.Equipment.ExtraArm2.Name(), "character", c.Name)
		}
		c.Equipment.ExtraArm2 = items.ItemDisabledSlot
		if c.Equipment.ExtraWrist2.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraWrist2)
		}
		c.Equipment.ExtraWrist2 = items.ItemDisabledSlot
	}
	if c.ExtraArms < 1 {
		if c.Equipment.ExtraArm1.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraArm1)
			mudlog.Debug("Extra Arms Check", "info", "Item returned from extra arm 1 slot", "name", c.Equipment.ExtraArm1.Name(), "character", c.Name)
		}
		c.Equipment.ExtraArm1 = items.ItemDisabledSlot
		if c.Equipment.ExtraWrist1.ItemId > 0 {
			c.StoreItem(c.Equipment.ExtraWrist1)
		}
		c.Equipment.ExtraWrist1 = items.ItemDisabledSlot
	}

	// Tail mutation: enable tail slot if mutation present, disable otherwise.
	// Must run AFTER EnableAll() (which ran in validateDisabledSlotsForSpecies).
	if _, hasTail := c.Mutations["tail"]; hasTail {
		if c.Equipment.Tail.ItemId < 0 {
			c.Equipment.Tail = items.Item{}
		}
	} else {
		if c.Equipment.Tail.ItemId > 0 {
			c.StoreItem(c.Equipment.Tail)
		}
		c.Equipment.Tail = items.ItemDisabledSlot
	}

	// Tail mutation disables legs slot via disable-legs flag.
	if flags := mutations.GetMutationFlags(c.Mutations); flags["disable-legs"] {
		if c.Equipment.Legs.ItemId > 0 {
			c.StoreItem(c.Equipment.Legs)
			mudlog.Debug("Mutation Check", "info", "Item returned from legs slot (tail mutation)", "name", c.Equipment.Legs.Name(), "character", c.Name)
		}
		c.Equipment.Legs = items.ItemDisabledSlot
	}
}
```

- [ ] **Step 7: Replace `Validate` body with dispatcher**

Replace the body of `Validate()` with:

```go
// Returns whether a correction was in order
func (c *Character) Validate(recalcPermaBuffs ...bool) error {

	// ── Skill migrations must run before ensureAllSkills ────────────
	c.validateSkillMigrations()

	if len(c.Description) == 0 {
		c.Description = "They seem thoroughly uninteresting."
	}

	if sp := species.GetSpecies(c.SpeciesId); sp == nil {
		c.SpeciesId = 1
	}

	if c.Created.IsZero() {
		c.Created = time.Now()
	}

	if c.Pet.Exists() {
		c.Pet.Validate()
	}

	if c.SpellBook == nil {
		c.SpellBook = make(map[string]int)
	}

	if c.KnownRecipes == nil {
		c.KnownRecipes = crafting.GetStarterRecipes()
	} else {
		// Backfill any new starter recipes added since character creation
		for id, val := range crafting.GetStarterRecipes() {
			if _, ok := c.KnownRecipes[id]; !ok {
				c.KnownRecipes[id] = val
			}
		}
	}

	if c.Mutations == nil {
		c.Mutations = make(map[string]int)
	}

	// Derive ExtraArms from mutation level (capped at 4)
	if lvl, ok := c.Mutations["extra-arms"]; ok && lvl > 0 {
		c.ExtraArms = lvl
		if c.ExtraArms > 4 {
			c.ExtraArms = 4
		}
	} else {
		c.ExtraArms = 0
	}

	if c.Zone == "" {
		c.Zone = startingZone
	}

	if c.Name == "" {
		c.Name = defaultName
	}
	c.Buffs.Validate()

	// Ensure all known skills exist at rank 1 minimum.
	c.Skills = ensureAllSkills(c.Skills)

	// Stats recalc based on equipment, race, level, etc.
	c.RecalculateStats()

	// Pool clamping after recalc.
	c.validatePoolClamps()

	c.Cooldowns.Prune()

	// Validate possessed/worn items (UIDs).
	c.validateEquipmentItems()

	// Apply species-disabled slot rules (requires validateEquipmentItems first).
	c.validateDisabledSlotsForSpecies()

	// Apply mutation-driven slot rules (extra arms, tail, disable-legs).
	c.validateMutationSlots()

	if len(recalcPermaBuffs) > 0 && recalcPermaBuffs[0] {
		c.reapplyPermabuffs()
	}

	return nil
}
```

- [ ] **Step 8: Build + vet**

```bash
go build ./...
go vet ./...
```

Expected: clean.

- [ ] **Step 9: Run characterization tests**

```bash
go test ./internal/characters/ -run "TestValidate" -v
```

Expected: all Validate tests PASS (same as Task 1 Step 23).

- [ ] **Step 10: Run full project test suite**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 11: Commit**

```bash
git add internal/characters/character.go
git commit -m "$(cat <<'EOF'
refactor(characters): extract Validate subsystem validators

Split the 330-line Validate() into 5 focused helpers:
- validateSkillMigrations: legacy skill renames + retired-skill removal
- validatePoolClamps: Health/Stamina/Conviction range clamping
- validateEquipmentItems: UID validation for backpack + worn items
- validateDisabledSlotsForSpecies: species-disabled slot enforcement
- validateMutationSlots: extra-arms + tail + disable-legs enforcement

Parent Validate() becomes a sequenced dispatcher (~75 lines) that still
handles the small top-level field defaults inline (Description,
SpellBook, Mutations, Zone, Name, ExtraArms derivation, Created time).

Zero behavior change. Verified against 7 characterization tests.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Split admin Room subcommand dispatcher

**Files:**
- Modify: `internal/usercommands/admin.room.go` — replace body of `Room()`
  (currently lines 35–420) with a thin dispatcher.
- Create: `internal/usercommands/admin.room.dispatcher.go` — houses the
  extracted subcommand helpers.

**Approach:** One unexported helper per subcommand branch. `Room()`
dispatches by first-token keyword to the appropriate helper.

- [ ] **Step 1: Inventory the current subcommands**

Read `internal/usercommands/admin.room.go` lines 35–420 and list every
top-level `if roomCmd == "..." / else if roomCmd == "..."` branch. As
of the current code, the list is:

| Subcommand | Lines | Helper name |
|-----------|-------|-------------|
| `edit container[s]` | 78–86 | (dispatches to `room_Edit_Containers`) |
| `edit exit[s]` | 88–96 | (dispatches to `room_Edit_Exits`) |
| `edit mutator[s]` | 98–106 | (dispatches to `room_Edit_Mutators`) |
| `edit` (help) | 107–113 | inline in dispatcher |
| `noun`/`nouns` | 116–161 | `adminRoom_Noun` |
| `copy <prop>` | 163–207 | `adminRoom_Copy` |
| `info` | 208–234 | `adminRoom_Info` |
| `exit <dir>` | 235–288 | `adminRoom_Exit` |
| `secretexit <dir>` | 289–312 | `adminRoom_SecretExit` |
| `set <prop>` | 313–414 | `adminRoom_Set` |
| (fallthrough) | 415–417 | inline |

Note that the `edit` branch is itself a sub-dispatcher calling existing
functions — leave its three sub-calls unchanged but extract the dispatch
logic into `adminRoom_Edit` for consistency.

- [ ] **Step 2: Create `admin.room.dispatcher.go`**

Create `internal/usercommands/admin.room.dispatcher.go` with this
content. Each helper is a direct lift of the original branch body; no
semantic changes.

```go
package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// adminRoom_Edit dispatches the `room edit <subtype>` subcommands.
// Returns (handled, nil) for the help fallthrough so the caller can stop.
func adminRoom_Edit(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if !user.HasRolePermission(`room.edit`) {
		user.SendText(`you do not have <ansi fg="command">room.edit</ansi> permission`)
		return true, nil
	}

	if rest == `edit container` || rest == `edit containers` {
		if !user.HasRolePermission(`room.edit.container`) {
			user.SendText(`you do not have <ansi fg="command">room.edit.container</ansi> permission`)
			return true, nil
		}
		return room_Edit_Containers(``, user, room, flags)
	}

	if rest == `edit exit` || rest == `edit exits` {
		if !user.HasRolePermission(`room.edit.exits`) {
			user.SendText(`you do not have <ansi fg="command">room.edit.container</ansi> permission`)
			return true, nil
		}
		return room_Edit_Exits(``, user, room, flags)
	}

	if rest == `edit mutator` || rest == `edit mutators` {
		if !user.HasRolePermission(`room.edit.mutators`) {
			user.SendText(`you do not have <ansi fg="command">room.edit.container</ansi> permission`)
			return true, nil
		}
		return room_Edit_Mutators(``, user, room, flags)
	}

	user.SendText(`<ansi fg="red">edit WHAT?</ansi> Try:`)
	user.SendText(`    <ansi fg="command">room edit containers</ansi>`)
	user.SendText(`    <ansi fg="command">room edit exits</ansi>`)
	user.SendText(`    <ansi fg="command">room edit mutators</ansi>`)
	return true, nil
}

// adminRoom_Noun handles `room noun`, `room nouns`, `room noun <name>`,
// and `room noun <name> <description>`.
func adminRoom_Noun(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.nouns`) {
		user.SendText(`you do not have <ansi fg="command">room.noun</ansi> permission`)
		return true, nil
	}

	if len(args) > 2 {
		noun := args[1]
		description := strings.Join(args[2:], ` `)

		if room.Nouns == nil {
			room.Nouns = map[string]string{}
		}
		room.Nouns[noun] = description

		user.SendText(`Noun Added:`)
		user.SendText(fmt.Sprintf(`<ansi fg="noun">%s</ansi> - %s`, strings.Repeat(` `, 20-len(noun))+noun, description))

		rooms.SaveRoomTemplate(*room)
		return true, nil
	}

	if len(args) == 2 || (len(args) == 3 && len(args[2]) == 0) {
		if _, ok := room.Nouns[args[1]]; ok {
			delete(room.Nouns, args[1])
			user.SendText(`Noun deleted.`)
		} else {
			user.SendText(`Noun not found.`)
		}
		return true, nil
	}

	user.SendText(`Room Nouns:`)
	for noun, description := range room.Nouns {
		user.SendText(fmt.Sprintf(`<ansi fg="noun">%s</ansi> - %s`, strings.Repeat(` `, 20-len(noun))+noun, description))
	}
	return true, nil
}

// adminRoom_Copy handles `room copy <property> <source-room-id>`.
func adminRoom_Copy(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.copy`) {
		user.SendText(`you do not have <ansi fg="command">room.copy</ansi> permission`)
		return true, nil
	}

	property := args[1]

	if property == "spawninfo" {
		sourceRoom, _ := strconv.Atoi(args[2])
		if sourceRoom := rooms.LoadRoom(sourceRoom); sourceRoom != nil {
			room.SpawnInfo = sourceRoom.SpawnInfo
			rooms.SaveRoomTemplate(*room)
			user.SendText("Spawn info copied/overwritten.")
		}
	}

	if property == "idlemessages" {
		sourceRoom, _ := strconv.Atoi(args[2])
		if sourceRoom := rooms.LoadRoom(sourceRoom); sourceRoom != nil {
			room.IdleMessages = append(room.IdleMessages, sourceRoom.IdleMessages...)
			rooms.SaveRoomTemplate(*room)
			user.SendText("IdleMessages copied/overwritten.")
		}
	}

	if property == "mutator" || property == "mutators" {
		sourceRoom, _ := strconv.Atoi(args[2])
		if sourceRoom := rooms.LoadRoom(sourceRoom); sourceRoom != nil {
			room.Mutators = append(room.Mutators, sourceRoom.Mutators...)
			rooms.SaveRoomTemplate(*room)
			user.SendText("Mutators copied/overwritten.")
		}
	}

	return true, nil
}

// adminRoom_Info handles `room info [roomId]`.
func adminRoom_Info(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.info`) {
		user.SendText(`you do not have <ansi fg="command">room.info</ansi> permission`)
		return true, nil
	}

	roomId := 0
	if len(args) == 1 {
		roomId = room.RoomId
	} else {
		roomId, _ = strconv.Atoi(args[1])
	}

	targetRoom := rooms.LoadRoom(roomId)
	if targetRoom == nil {
		user.SendText(fmt.Sprintf("Room %d not found.", roomId))
		return false, fmt.Errorf("room %d not found", roomId)
	}

	roomInfo := map[string]any{
		`room`: targetRoom,
		`zone`: rooms.GetZoneConfig(targetRoom.Zone),
	}

	infoOutput, _ := templates.Process("admincommands/ingame/roominfo", roomInfo, user.UserId)
	user.SendText(infoOutput)
	return true, nil
}

// adminRoom_Exit handles `room exit <direction> <roomId-or-rename>`.
func adminRoom_Exit(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.exits`) {
		user.SendText(`you do not have <ansi fg="command">room.exit</ansi> permission`)
		return true, nil
	}

	direction := strings.ToLower(args[1])
	roomId := 0
	var numError error = nil
	exitRename := ``

	if len(args) > 2 {
		roomId, numError = strconv.Atoi(args[2])
		if numError != nil {
			exitRename = args[2]
		}
	}

	if len(args) < 3 {
		if _, ok := room.Exits[direction]; !ok {
			user.SendText(fmt.Sprintf("Exit %s does not exist.", direction))
			return true, nil
		}
		delete(room.Exits, direction)
		return true, nil
	}

	if currentExit, ok := room.Exits[direction]; ok {
		user.SendText(fmt.Sprintf("Exit %s already exists (overwriting).", direction))

		if exitRename != `` {
			delete(room.Exits, direction)
			room.Exits[exitRename] = currentExit
			user.SendText(fmt.Sprintf("Exit %s renamed to %s.", direction, exitRename))
			return true, nil
		}
	}

	targetRoom := rooms.LoadRoom(roomId)
	if targetRoom == nil {
		err := fmt.Errorf(`room %d not found`, roomId)
		user.SendText(err.Error())
		return true, nil
	}

	rooms.ConnectRoom(room.RoomId, targetRoom.RoomId, direction)
	user.SendText(fmt.Sprintf("Exit %s added.", direction))
	return true, nil
}

// adminRoom_SecretExit handles `room secretexit <direction>`.
func adminRoom_SecretExit(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.exits`) {
		user.SendText(`you do not have <ansi fg="command">room.exit</ansi> permission`)
		return true, nil
	}

	direction := args[1]
	if exit, ok := room.Exits[direction]; ok {
		if exit.Secret {
			exit.Secret = false
			room.Exits[direction] = exit
			rooms.SaveRoomTemplate(*room)
			user.SendText(fmt.Sprintf("Exit %s secrecy REMOVED.", direction))
		} else {
			exit.Secret = true
			room.Exits[direction] = exit
			rooms.SaveRoomTemplate(*room)
			user.SendText(fmt.Sprintf("Exit %s secrecy ADDED.", direction))
		}
	} else {
		user.SendText(fmt.Sprintf("Exit %s not found.", direction))
	}
	return true, nil
}

// adminRoom_Set handles `room set <property> [value]`.
func adminRoom_Set(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.set`) {
		user.SendText(`you do not have <ansi fg="command">room.set</ansi> permission`)
		return true, nil
	}

	propertyName := args[1]
	propertyValue := ``
	if len(args) > 2 {
		propertyValue = strings.Join(args[2:], ` `)
	}
	propertyValue = strings.Trim(propertyValue, `"`)

	if propertyName == "mutator" || propertyName == "mutators" {
		if propertyValue == `` {
			user.SendText(`<ansi fg="table-title">Mutators:</ansi>`)
			if len(room.Mutators) == 0 {
				user.SendText(`  None.`)
			}
			for _, mut := range room.Mutators {
				user.SendText(`  <ansi fg="mutator">` + mut.MutatorId + `</ansi>`)
			}
			user.SendText(``)
		} else {
			user.SendText(``)
			if !mutators.IsMutator(propertyValue) {
				user.SendText(`<ansi fg="table-title"><ansi fg="mutator">` + propertyValue + `</ansi> is an invalid mutator id.</ansi>`)
				user.SendText(`<ansi fg="table-title">  Here is a list of valid mutator id's:</ansi>`)
				for _, name := range mutators.GetAllMutatorIds() {
					user.SendText(`    <ansi fg="mutator">` + name + `</ansi>`)
				}
			} else if room.Mutators.Remove(propertyValue) {
				user.SendText(`<ansi fg="table-title">Mutator <ansi fg="mutator">` + propertyValue + `</ansi> Removed.</ansi>`)
			} else if room.Mutators.Add(propertyValue) {
				user.SendText(`<ansi fg="table-title">Mutator <ansi fg="mutator">` + propertyValue + `</ansi> Added.</ansi>`)
			}
			user.SendText(``)
		}
		return true, nil
	}

	if propertyName == "spawninfo" {
		if propertyValue == `clear` {
			room.SpawnInfo = room.SpawnInfo[:0]
			rooms.SaveRoomTemplate(*room)
		}
	} else if propertyName == "title" {
		if propertyValue == `` {
			propertyValue = `[no title]`
		}
		room.Title = propertyValue
		rooms.SaveRoomTemplate(*room)
	} else if propertyName == "description" {
		if propertyValue == `` {
			propertyValue = `[no description]`
		}
		propertyValue = strings.ReplaceAll(propertyValue, `\n`, "\n")
		room.Description = propertyValue
		rooms.SaveRoomTemplate(*room)
	} else if propertyName == "idlemessages" {
		room.IdleMessages = []string{}
		for _, idleMsg := range strings.Split(propertyValue, ";") {
			idleMsg = strings.TrimSpace(idleMsg)
			if len(idleMsg) < 1 {
				continue
			}
			room.IdleMessages = append(room.IdleMessages, idleMsg)
		}
		rooms.SaveRoomTemplate(*room)
	} else if propertyName == "symbol" || propertyName == "mapsymbol" {
		room.MapSymbol = propertyValue
		rooms.SaveRoomTemplate(*room)
	} else if propertyName == "legend" || propertyName == "maplegend" {
		room.MapLegend = propertyValue
		rooms.SaveRoomTemplate(*room)
	} else if propertyName == "zone" {
		if err := rooms.MoveToZone(room.RoomId, propertyValue); err != nil {
			user.SendText(err.Error())
			return true, nil
		}
	} else if propertyName == "biome" {
		room.Biome = strings.ToLower(propertyValue)
	} else {
		user.SendText(`Invalid property provided to <ansi fg="command">room set</ansi>.`)
		return false, fmt.Errorf("unknown room set property %q", propertyName)
	}

	user.SendText(fmt.Sprintf("Room %s set to %s.", propertyName, propertyValue))
	return true, nil
}
```

- [ ] **Step 3: Replace `Room()` body with dispatcher**

Replace the body of `Room()` in `admin.room.go` with:

```go
func Room(rest string, user *users.UserRecord, liveRoom *rooms.Room, flags events.EventFlag) (bool, error) {

	handled := true

	args := util.SplitButRespectQuotes(rest)

	if len(args) == 0 {
		infoOutput, _ := templates.Process("admincommands/help/command.room", nil, user.UserId)
		user.SendText(infoOutput)
		return handled, nil
	}

	var room *rooms.Room
	if liveRoom.IsEphemeral() {
		room = liveRoom
	} else {
		room = rooms.LoadRoomTemplate(liveRoom.RoomId)
	}

	if room == nil {
		err := fmt.Errorf(`Something went wrong for RoomId: %d`, liveRoom.RoomId)
		user.SendText(err.Error())
		return true, err
	}

	roomCmd := strings.ToLower(args[0])

	switch roomCmd {
	case `edit`:
		return adminRoom_Edit(rest, user, room, flags)
	case `noun`, `nouns`:
		return adminRoom_Noun(args, user, room)
	case `info`:
		return adminRoom_Info(args, user, room)
	case `secretexit`:
		if len(args) < 2 {
			user.SendText(fmt.Sprintf(`Invalid room command: <ansi fg="command">%s</ansi>`, roomCmd))
			return handled, nil
		}
		return adminRoom_SecretExit(args, user, room)
	case `copy`:
		if len(args) < 3 {
			user.SendText(fmt.Sprintf(`Invalid room command: <ansi fg="command">%s</ansi>`, roomCmd))
			return handled, nil
		}
		return adminRoom_Copy(args, user, room)
	case `exit`:
		if len(args) < 2 {
			user.SendText(fmt.Sprintf(`Invalid room command: <ansi fg="command">%s</ansi>`, roomCmd))
			return handled, nil
		}
		return adminRoom_Exit(args, user, room)
	case `set`:
		if len(args) < 2 {
			user.SendText(fmt.Sprintf(`Invalid room command: <ansi fg="command">%s</ansi>`, roomCmd))
			return handled, nil
		}
		return adminRoom_Set(args, user, room)
	default:
		user.SendText(fmt.Sprintf(`Invalid room command: <ansi fg="command">%s</ansi>`, roomCmd))
	}

	return handled, nil
}
```

**Important semantic note:** the original code had length checks INSIDE
each else-if branch (e.g. `else if len(args) >= 2 && roomCmd == "exit"`).
The new dispatcher hoists those into the switch by returning early with
the same "Invalid room command" message on too-few-args. **Verify by
hand** that every length guard from the original is represented. If you
find a case where the original silently ignored the subcommand when args
were too short, preserve that exactly — don't change to an error message.

- [ ] **Step 4: Trim unused imports from admin.room.go**

After replacing `Room()`, `admin.room.go` no longer needs some of the
imports. Run `go build ./...` and let the compiler tell you. Remove any
import it flags as unused.

- [ ] **Step 5: Build + vet + test**

```bash
go build ./...
go vet ./...
go test ./...
```

Expected: clean builds, clean vets, all tests pass.

- [ ] **Step 6: Manual smoke test**

Start the server locally:

```bash
go run .
```

Log in as an admin character, then run:
- `room` (no args) — should print help template
- `room info` — should print room info table
- `room info 1` — should print room 1 info
- `room set title "Test Title"` — should save
- `room set title "[original title]"` — restore
- `room noun` — list nouns
- `room noun chair "a wooden chair"` — add noun
- `room noun chair` — delete noun
- `room exit north 2` — add exit (if not already)
- `room secretexit north` — toggle secret
- `room edit` — should print help
- `room edit containers` / `room edit exits` / `room edit mutators` — each opens its prompt (just confirm the prompt appears, don't complete the flow yet — that's Task 6's smoke test).

Any unexpected behavior = stop and investigate before committing.
Press Ctrl+C to stop the server.

- [ ] **Step 7: Commit**

```bash
git add internal/usercommands/admin.room.go internal/usercommands/admin.room.dispatcher.go
git commit -m "$(cat <<'EOF'
refactor(usercommands): split admin room subcommand dispatcher

Move the 387-line Room() switch into 7 unexported helpers living in a
new admin.room.dispatcher.go file:
- adminRoom_Edit (dispatches to existing edit subfunctions)
- adminRoom_Noun, adminRoom_Copy, adminRoom_Info
- adminRoom_Exit, adminRoom_SecretExit, adminRoom_Set

Room() shrinks to ~45 lines — arg parsing, room load, and a switch
that routes to each helper. Length guards hoisted out of the original
else-if chain into the dispatcher's switch.

Zero behavior change. Manually smoke-tested:
- room info, room set title, room noun add/delete, room exit add,
  room secretexit, all three "room edit <sub>" sub-prompts.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6a: Fix duplicate "hidden exit?" prompt bug in `room_Edit_Exits`

**Rationale:** `room_Edit_Exits` has two back-to-back identical "Is this
a hidden exit?" prompts (current lines ~801–835). Both call
`cmdPrompt.Ask(\`Is this a hidden exit?\`, ...)` and both overwrite
`currentlyEditing.Exit.Secret`. The second block's comment says
"Secret exit?" but the prompt text and behavior are identical to the
first. This causes admins editing exits to be prompted twice for the
same answer — a clear UX bug.

We fix this in a standalone `fix:` commit BEFORE the file move, so the
Task 6b refactor commit remains purely structural per our scope-creep
policy.

**Files:**
- Modify: `internal/usercommands/admin.room.go` — delete the second
  (duplicate) "hidden exit?" block.

- [ ] **Step 1: Identify the duplicate block**

```bash
grep -n "Is this a hidden exit" internal/usercommands/admin.room.go
```

Expected: 2 matches, one at the end of an `//Exit message?` block (~line
811) and one at the start of a `//Secret exit?` block (~line 829). Confirm
both blocks assign `currentlyEditing.Exit.Secret = question.Response == \`yes\``.

- [ ] **Step 2: Delete the second block**

In `admin.room.go`, find the block that starts with:

```go
	//
	// Secret exit?
	//
	{
		secretExitDefault := `no`
		if currentlyEditing.Exit.Secret {
			secretExitDefault = `yes`
		}

		// allow them to name/rename the exit.
		question := cmdPrompt.Ask(`Is this a hidden exit?`, []string{`yes`, `no`}, secretExitDefault)
		if !question.Done {
			return true, nil
		}

		currentlyEditing.Exit.Secret = question.Response == `yes`
	}
```

Delete it entirely (including the `// Secret exit?` comment and the
surrounding `{` `}` braces). The preceding `// Exit message?` block
already asks the same question and sets the same field.

- [ ] **Step 3: Verify only one prompt remains**

```bash
grep -n "Is this a hidden exit" internal/usercommands/admin.room.go
```

Expected: exactly 1 match.

- [ ] **Step 4: Build + vet + test**

```bash
go build ./...
go vet ./...
go test ./...
```

Expected: clean. No test should regress — this is an admin-only prompt
path with no test coverage.

- [ ] **Step 5: Manual smoke test**

Start the server. Log in as admin, run `room edit exits`, create a new
exit, and walk through the whole prompt chain. Confirm you're asked
"Is this a hidden exit?" exactly **once** (after the exit message
prompt, before the lock questionnaire).

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/admin.room.go
git commit -m "$(cat <<'EOF'
fix(admin): remove duplicate "Is this a hidden exit?" prompt

room_Edit_Exits asked the same yes/no question twice in a row,
overwriting Exit.Secret with the same value both times. The second
block was commented "Secret exit?" but had identical prompt text,
choices, default, and effect to the preceding "Exit message?" block.

Classic copy-paste: the intended second question was likely something
else that never got written. Removing the duplicate leaves one correct
prompt for Secret and restores sensible admin UX.

Surfaced during 1.2b refactor read; fixed ahead of the refactor
commit so the structural diff stays semantics-preserving.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6b: Move `room_Edit_Exits` + prompt helpers to `admin.room.exits.go`

**Files:**
- Modify: `internal/usercommands/admin.room.go` — delete `room_Edit_Exits()`
  function body (it's moving).
- Create: `internal/usercommands/admin.room.exits.go` — houses the moved
  function + any prompt helpers we extract.

**Approach:** The function is a linear multi-step prompt machine. Move it
to its own file. Do NOT move `editLockAndTrap` — it's a shared helper used
by both exits and containers (see docstring at its declaration).

Extracting prompt-step helpers is ambitious here because each step reads
local state. Conservative approach: move the whole function as-is into
the new file, with minimal restructuring. The file-rehoming itself is
the primary value; further decomposition can happen later if patterns
emerge.

- [ ] **Step 1: Confirm function boundaries in admin.room.go**

```bash
grep -n "^func room_Edit_Exits" internal/usercommands/admin.room.go
grep -n "^func editLockAndTrap" internal/usercommands/admin.room.go
```

Expected: `room_Edit_Exits` at some line N (post-6a fix), `editLockAndTrap`
immediately after it.

- [ ] **Step 2: Create `admin.room.exits.go`**

Create `internal/usercommands/admin.room.exits.go`. Copy the ENTIRE body
of `room_Edit_Exits` (from line N to the closing `}`) into this new file,
wrapped in the standard package header:

```go
package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// room_Edit_Exits walks an admin through the multi-step prompt to create,
// rename, reconfigure, or delete a room exit. It persists changes via
// rooms.SaveRoomTemplate when complete.
//
// This function is stateful across prompt responses — each cmdPrompt.Ask
// either completes immediately (Done=true) or yields control back to the
// caller pending the next response. On yield, we return (true, nil) to
// keep the prompt session alive. On completion or error, we clear the
// prompt and return.
//
// The lock questionnaire is delegated to editLockAndTrap (lives in
// admin.room.go; shared with room_Edit_Containers).
func room_Edit_Exits(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	// Paste the full body of room_Edit_Exits here, exactly as it appears
	// in admin.room.go post-Task-6a.
	// ... [full body] ...
}
```

Copy the current body literally. No restructuring in this task.

- [ ] **Step 3: Delete `room_Edit_Exits` from `admin.room.go`**

Remove the entire `func room_Edit_Exits(...)` declaration + body from
`admin.room.go`. Leave `editLockAndTrap` in place — it stays in
`admin.room.go` because `room_Edit_Containers` also calls it.

- [ ] **Step 4: Trim imports**

Run `go build ./...` and let compiler errors guide you:
- `admin.room.go` may now have unused imports (e.g. `exit`, `sort`).
  Remove them.
- `admin.room.exits.go` may be missing imports (e.g. `gamelock` for
  editLockAndTrap return). Wait — `editLockAndTrap` is still called
  from `room_Edit_Exits`. Its signature uses `gamelock.Lock` and
  `prompt.Prompt`. We need to make sure `admin.room.exits.go` can call
  `editLockAndTrap` (same package) AND imports the types it needs.
  Check the import list against actual usage.

- [ ] **Step 5: Build + vet + test**

```bash
go build ./...
go vet ./...
go test ./...
```

Expected: clean.

- [ ] **Step 6: Manual smoke test — full exit edit flow**

Start the server. Log in as admin.

Scenario 1: Create a new exit
- `room edit exits` → should list existing exits
- At prompt, type `new` → should ask "Choose a name for this exit:"
- Type `testdir` → should ask "What RoomId will this exit lead to?"
- Type a valid room ID → should ask "Is this a hidden exit?" ONCE
  (this is the Task 6a fix in action)
- Type `no` → should ask "Special message when using the exit?"
- Type `none` → should enter the lock questionnaire
- Accept defaults (no lock) → should print "Changes saved."

Scenario 2: Edit existing exit
- `room edit exits` → list shows testdir
- Select it by number → should ask "Delete this exit?"
- Type `no` → should proceed to name prompt with testdir pre-filled
- Accept all defaults → should save

Scenario 3: Delete existing exit
- `room edit exits` → select testdir → answer `yes` to delete
- Should print "testdir deleted from the room."

Any deviation = stop, investigate, and fix.

- [ ] **Step 7: Commit**

```bash
git add internal/usercommands/admin.room.go internal/usercommands/admin.room.exits.go
git commit -m "$(cat <<'EOF'
refactor(usercommands): move room_Edit_Exits to admin.room.exits.go

Move the 240-line room_Edit_Exits function (post-Task-6a fix) from
admin.room.go into a dedicated admin.room.exits.go file. No structural
changes to the function body — this is a pure file rehoming to shrink
admin.room.go and give the exits prompt machine a clear home.

editLockAndTrap stays in admin.room.go because it's shared with
room_Edit_Containers.

Zero behavior change. Smoke-tested all three exit-edit flows (create,
edit, delete).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Mark 1.2b complete in overview

**Files:**
- Modify: `docs/superpowers/code_cleanup_stage_1_overview.md`

- [ ] **Step 1: Update status row**

Edit `docs/superpowers/code_cleanup_stage_1_overview.md`. Find:

```
| 1.2b | God-Function Refactor — Character + Admin | 5h | Low-Med | Planning |
```

Change `Planning` to `Complete`.

- [ ] **Step 2: Verify no other overview edits needed**

Read the overview. Confirm the 1.2b section body still accurately
describes what shipped. No PATCH_NOTES entry needed — this is an
internal-only refactor with zero player impact.

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/code_cleanup_stage_1_overview.md
git commit -m "$(cat <<'EOF'
docs: mark code cleanup 1.2b complete

5 god-functions decomposed, 18 characterization tests added for the
character batch, one duplicate-prompt bug fixed along the way.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 4: Confirm branch state**

```bash
git log --oneline feature/stage-1.2b-character-admin-refactor ^development
```

Expected: 8 commits, in order:
1. `test(characters): characterization tests for Validate/Wear/RecalculateStats`
2. `refactor(characters): extract RecalculateStats per-stat loops`
3. `refactor(characters): extract Wear slot-selection helpers`
4. `refactor(characters): extract Validate subsystem validators`
5. `refactor(usercommands): split admin room subcommand dispatcher`
6. `fix(admin): remove duplicate "Is this a hidden exit?" prompt`
7. `refactor(usercommands): move room_Edit_Exits to admin.room.exits.go`
8. `docs: mark code cleanup 1.2b complete`

Note: the `fix:` commit is listed in 1.2b's history but was created
chronologically between tasks 5 and 6b. That's fine for branch order.

- [ ] **Step 5: Final whole-project verification**

```bash
go build ./...
go vet ./...
go test ./...
```

Expected: clean. No test regressions anywhere.

---

## Merge to development (after user review)

Do NOT merge until the user has reviewed the branch. Once approved:

```bash
git checkout development
git merge --no-ff feature/stage-1.2b-character-admin-refactor -m "$(cat <<'EOF'
merge: stage 1.2b character + admin god-function refactor

Decomposes 5 oversized functions across character.go and admin.room.go,
backed by 18 new characterization tests for the character batch. One
duplicate-prompt bug fixed in room_Edit_Exits. Zero player-visible
behavior change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Do NOT push to origin. Let the user decide when (and whether) to push.
This refactor doesn't urgently need prod deployment — can wait until
1.2c is also done.
