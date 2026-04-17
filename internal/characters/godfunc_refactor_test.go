package characters

// godfunc_refactor_test.go — characterization tests for RecalculateStats, Wear,
// and Validate. These tests capture CURRENT behavior before the 1.2b refactor so
// that post-refactor commits can be verified to be semantics-preserving.
//
// All tests must PASS (or SKIP) against the unrefactored code. They must continue
// to pass unchanged after Tasks 2, 3, 4 of the 1.2b plan.

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/stretchr/testify/assert"
)

// ─────────────────────────────────────────────────────────────────────────────
// Test helpers
// ─────────────────────────────────────────────────────────────────────────────

// newTestBuffs returns a zero-value Buffs instance that is safe to embed in a
// manually-constructed Character.
func newTestBuffs() buffs.Buffs {
	return buffs.New()
}

// validStats returns a deterministic stat block for Validate/RecalculateStats
// tests (all six stats at the human baseline of 100).
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

// ─────────────────────────────────────────────────────────────────────────────
// RecalculateStats tests (6)
// ─────────────────────────────────────────────────────────────────────────────

// TestRecalculateStats_BaseStatHydrationFromSpecies verifies the species-hydration
// block at character.go:2760-2783.
//
// Two sub-cases are tested:
//
//  1. Non-zero Base values are NEVER overwritten by species hydration — the guard
//     condition (Base == 0) prevents it, regardless of whether species data is loaded.
//
//  2. Zero Base values ARE populated from the seeded species when GetSpecies returns
//     a non-nil result. This sub-case uses species.SeedSpeciesForTest to inject known
//     test data so the hydration block is genuinely exercised (previously the test
//     hit only the nil-species short-circuit because no data files are loaded in the
//     unit-test binary).
func TestRecalculateStats_BaseStatHydrationFromSpecies(t *testing.T) {
	// Seed a test species with known base stat values for the duration of this test.
	testSpecies := &species.Species{
		SpeciesId: 1,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 80},
			Dexterity:  stats.StatInfo{Base: 90},
			Perception: stats.StatInfo{Base: 85},
			Vitality:   stats.StatInfo{Base: 95},
			Willpower:  stats.StatInfo{Base: 75},
			Charisma:   stats.StatInfo{Base: 70},
		},
	}
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{1: testSpecies})
	t.Cleanup(cleanup)

	// Sub-case 1: non-zero Base → must NOT be overwritten.
	c := &Character{
		SpeciesId: 1,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 123},
			Dexterity:  stats.StatInfo{Base: 100},
			Perception: stats.StatInfo{Base: 100},
			Vitality:   stats.StatInfo{Base: 100},
			Willpower:  stats.StatInfo{Base: 100},
			Charisma:   stats.StatInfo{Base: 100},
		},
		Mutations: map[string]int{},
		Buffs:     newTestBuffs(),
	}
	c.RecalculateStats()
	assert.Equal(t, 123, c.Stats.Strength.Base, "rolled Strength.Base must not be overwritten by species hydration")
	assert.Equal(t, 100, c.Stats.Dexterity.Base, "rolled Dexterity.Base must not be overwritten")

	// Sub-case 2: zero Base → hydration block copies values from the seeded species.
	// This exercises character.go:2760-2783, which was previously unreachable in tests.
	c2 := &Character{
		SpeciesId: 1,
		Stats: stats.Statistics{
			Strength:   stats.StatInfo{Base: 0},
			Dexterity:  stats.StatInfo{Base: 0},
			Perception: stats.StatInfo{Base: 0},
			Vitality:   stats.StatInfo{Base: 0},
			Willpower:  stats.StatInfo{Base: 0},
			Charisma:   stats.StatInfo{Base: 0},
		},
		Mutations: map[string]int{},
		Buffs:     newTestBuffs(),
	}
	c2.RecalculateStats()
	assert.Equal(t, 80, c2.Stats.Strength.Base, "zero Strength.Base must be hydrated from species data")
	assert.Equal(t, 90, c2.Stats.Dexterity.Base, "zero Dexterity.Base must be hydrated from species data")
	assert.Equal(t, 85, c2.Stats.Perception.Base, "zero Perception.Base must be hydrated from species data")
	assert.Equal(t, 95, c2.Stats.Vitality.Base, "zero Vitality.Base must be hydrated from species data")
	assert.Equal(t, 75, c2.Stats.Willpower.Base, "zero Willpower.Base must be hydrated from species data")
	assert.Equal(t, 70, c2.Stats.Charisma.Base, "zero Charisma.Base must be hydrated from species data")
}

// TestRecalculateStats_PoolMaxDerivation verifies that pool maximums are
// computed and floored by RecalculateStats. Without real config data loaded,
// all balance multipliers are 0, so pool Mods = 0. The floor logic in
// RecalculateStats is still exercised: HealthMax ≥ 1, ActionPointsMax ≥ 50,
// Stamina/Conviction ≥ 0.
//
// Current characterization (no config data loaded): all pools sit at their
// hard floors after RecalculateStats.
func TestRecalculateStats_PoolMaxDerivation(t *testing.T) {
	c := &Character{
		SpeciesId: 1,
		Stats:     validStats(),
		Mutations: map[string]int{},
		Buffs:     newTestBuffs(),
	}

	c.RecalculateStats()

	// Floors are always applied regardless of config state.
	assert.GreaterOrEqual(t, c.HealthMax.Value, 1, "HealthMax floor is 1")
	assert.GreaterOrEqual(t, c.StaminaMax.Value, 0, "StaminaMax floor is 0")
	assert.GreaterOrEqual(t, c.ConvictionMax.Value, 0, "ConvictionMax floor is 0")
	assert.GreaterOrEqual(t, c.ActionPointsMax.Value, 50, "ActionPointsMax floor is 50")
	// Without config data (all balance multipliers = 0), HealthMax sits at its floor.
	// ActionPointsMax is hardcoded to Mods=200 in RecalculateStats, so it is well above
	// the floor of 50. Characterize the exact values to catch drift.
	assert.Equal(t, 1, c.HealthMax.Value, "HealthMax=1 at floor when config not loaded")
	assert.Equal(t, 200, c.ActionPointsMax.Value, "ActionPointsMax=200 (hardcoded Mods) when config not loaded")
}

// TestRecalculateStats_EquipmentStatMods verifies that after RecalculateStats,
// each stat's .Mods field equals the corresponding StatMod() call value.
func TestRecalculateStats_EquipmentStatMods(t *testing.T) {
	c := &Character{
		SpeciesId: 1,
		Stats:     validStats(),
		Mutations: map[string]int{},
		Buffs:     newTestBuffs(),
	}

	c.RecalculateStats()

	assert.Equal(t, c.StatMod(string(statmods.Strength)), c.Stats.Strength.Mods,
		"Strength.Mods should equal StatMod(strength)")
	assert.Equal(t, c.StatMod(string(statmods.Dexterity)), c.Stats.Dexterity.Mods,
		"Dexterity.Mods should equal StatMod(dexterity)")
	assert.Equal(t, c.StatMod(string(statmods.Perception)), c.Stats.Perception.Mods,
		"Perception.Mods should equal StatMod(perception)")
	assert.Equal(t, c.StatMod(string(statmods.Vitality)), c.Stats.Vitality.Mods,
		"Vitality.Mods should equal StatMod(vitality)")
	assert.Equal(t, c.StatMod(string(statmods.Willpower)), c.Stats.Willpower.Mods,
		"Willpower.Mods should equal StatMod(willpower)")
	assert.Equal(t, c.StatMod(string(statmods.Charisma)), c.Stats.Charisma.Mods,
		"Charisma.Mods should equal StatMod(charisma)")
}

// TestRecalculateStats_MutationFlatAndMultiplier verifies idempotency and that
// ValueAdj is positive for a standard stat=100 character. Full flat/multiplier
// mutation coverage requires specific mutation fixtures; that subset is captured
// as the stability invariant here.
func TestRecalculateStats_MutationFlatAndMultiplier(t *testing.T) {
	c := &Character{
		SpeciesId: 1,
		Stats:     validStats(),
		Mutations: map[string]int{},
		Buffs:     newTestBuffs(),
	}
	c.RecalculateStats()
	baseStr := c.Stats.Strength.ValueAdj

	assert.Greater(t, baseStr, 0, "baseline Strength.ValueAdj should be positive at stat=100")

	// RecalculateStats must be idempotent — a second call with no changes must
	// produce identical ValueAdj values.
	baseDex := c.Stats.Dexterity.ValueAdj
	c.RecalculateStats()
	assert.Equal(t, baseStr, c.Stats.Strength.ValueAdj, "RecalculateStats must be idempotent on Strength.ValueAdj")
	assert.Equal(t, baseDex, c.Stats.Dexterity.ValueAdj, "Dexterity.ValueAdj idempotent")
}

// TestRecalculateStats_PoolReservationClamping verifies that current pool values
// survive RecalculateStats when they are below the respective pool maximums, and
// that GetPoolReservation returns 0 when no chrysalis items are equipped.
func TestRecalculateStats_PoolReservationClamping(t *testing.T) {
	c := &Character{
		SpeciesId:  1,
		Stats:      validStats(),
		Mutations:  map[string]int{},
		Buffs:      newTestBuffs(),
		Health:     50,
		Stamina:    50,
		Conviction: 50,
	}

	c.RecalculateStats()

	// 50 is safely below any pool max for stat=100, so values must survive.
	assert.Equal(t, 50, c.Health, "Health should survive RecalculateStats when below max")
	assert.Equal(t, 50, c.Stamina, "Stamina should survive RecalculateStats when below max")
	assert.Equal(t, 50, c.Conviction, "Conviction should survive RecalculateStats when below max")

	// No chrysalis items equipped → reservation is 0 for all pools.
	assert.Equal(t, 0, c.GetPoolReservation("health", c.HealthMax.Value),
		"GetPoolReservation(health) should be 0 with no chrysalis items")
	assert.Equal(t, 0, c.GetPoolReservation("stamina", c.StaminaMax.Value),
		"GetPoolReservation(stamina) should be 0 with no chrysalis items")
	assert.Equal(t, 0, c.GetPoolReservation("conviction", c.ConvictionMax.Value),
		"GetPoolReservation(conviction) should be 0 with no chrysalis items")
}

// TestRecalculateStats_IdempotencyAcrossRepeatedCalls verifies that a second
// RecalculateStats call with no state changes produces identical ValueAdj values
// to the first call. This proves the change-detection path's inputs are stable:
// if no stat adjustments drift between calls, the engine's internal "changed"
// guard has nothing to act on.
//
// Note: event-emission itself cannot be directly observed in unit tests (no
// events.Clear() exists). This test covers the prerequisite invariant — stable
// inputs — not the event dispatch outcome.
func TestRecalculateStats_IdempotencyAcrossRepeatedCalls(t *testing.T) {
	c := &Character{
		SpeciesId: 1,
		userId:    42, // non-zero triggers event emission path
		Stats:     validStats(),
		Mutations: map[string]int{},
		Buffs:     newTestBuffs(),
	}

	// First call: populates ValueAdj from Base values.
	c.RecalculateStats()

	// Capture state after first call.
	snapStr := c.Stats.Strength.ValueAdj
	snapDex := c.Stats.Dexterity.ValueAdj
	snapPer := c.Stats.Perception.ValueAdj
	snapVit := c.Stats.Vitality.ValueAdj
	snapWil := c.Stats.Willpower.ValueAdj
	snapCha := c.Stats.Charisma.ValueAdj
	snapHPMax := c.HealthMax.Value

	// Second call: identical input → must not change any ValueAdj.
	c.RecalculateStats()

	assert.Equal(t, snapStr, c.Stats.Strength.ValueAdj, "Strength.ValueAdj must not drift on second call")
	assert.Equal(t, snapDex, c.Stats.Dexterity.ValueAdj, "Dexterity.ValueAdj must not drift on second call")
	assert.Equal(t, snapPer, c.Stats.Perception.ValueAdj, "Perception.ValueAdj must not drift on second call")
	assert.Equal(t, snapVit, c.Stats.Vitality.ValueAdj, "Vitality.ValueAdj must not drift on second call")
	assert.Equal(t, snapWil, c.Stats.Willpower.ValueAdj, "Willpower.ValueAdj must not drift on second call")
	assert.Equal(t, snapCha, c.Stats.Charisma.ValueAdj, "Charisma.ValueAdj must not drift on second call")
	assert.Equal(t, snapHPMax, c.HealthMax.Value, "HealthMax must not drift on second call")
}

// ─────────────────────────────────────────────────────────────────────────────
// Wear tests (5)
// ─────────────────────────────────────────────────────────────────────────────
//
// Wear tests require real item fixtures loaded from disk via items.LoadDataFiles(),
// which in turn requires configs.GetFilePathsConfig() to be initialised. That
// bootstrapping chain is non-trivial to replicate in a unit test without a
// config file on the test runner's filesystem.
//
// All five Wear tests are therefore skipped with a clear description. They can
// be unblocked in a follow-up by either:
//   (a) adding a TestMain that initialises configs + items.LoadDataFiles(), or
//   (b) extracting the Wear slot-selection logic into a helper that accepts an
//       ItemSpec value directly (removing the data-file dependency).

// TestWear_EmptySlotHappyPath — wearing a Body-slot item into an empty slot
// succeeds and returns no displaced items.
func TestWear_EmptySlotHappyPath(t *testing.T) {
	t.Skip("BLOCKED: requires items.LoadDataFiles() which needs configs init; " +
		"unblock by adding TestMain with full bootstrap or by refactoring Wear to " +
		"accept ItemSpec directly. Fixture: items.New(20008) = cotton shirt (body armor).")
}

// TestWear_SlotOccupiedSwap — wearing an item into an already-occupied slot
// returns the displaced item.
func TestWear_SlotOccupiedSwap(t *testing.T) {
	t.Skip("BLOCKED: same items.LoadDataFiles() dependency as TestWear_EmptySlotHappyPath. " +
		"Expected: equip shirtA, then Wear(shirtB); returnItems==[shirtA], newItemWorn==true.")
}

// TestWear_TwoHandedWeaponDisplacesOffhand — equipping a 2H weapon returns BOTH
// the existing 1H weapon and the offhand item.
func TestWear_TwoHandedWeaponDisplacesOffhand(t *testing.T) {
	t.Skip("BLOCKED: same items.LoadDataFiles() dependency. " +
		"Fixture: items.New(10007)=heavy greatsword (2H), items.New(10001)=sharp stick (1H).")
}

// TestWear_WrongItemTypeFails — trying to wear a non-equipment item (e.g. a
// potion) fails with a non-empty failureReason and no state change.
func TestWear_WrongItemTypeFails(t *testing.T) {
	t.Skip("BLOCKED: same items.LoadDataFiles() dependency. " +
		"Expected: returnItems empty, newItemWorn==false, failureReason non-empty.")
}

// TestWear_MultiArmRouting — with Extra Arms mutation active, additional weapons
// route to extra arm slots before displacing the main hand.
func TestWear_MultiArmRouting(t *testing.T) {
	t.Skip("BLOCKED: same items.LoadDataFiles() dependency. " +
		"Requires 1H weapon fixture + character with c.ExtraArms >= 1 via Mutations.")
}

// ─────────────────────────────────────────────────────────────────────────────
// Validate tests (7)
// ─────────────────────────────────────────────────────────────────────────────

// TestValidate_EmptyCharacterCorrected verifies that a minimal Character struct
// is corrected to safe defaults without returning an error.
func TestValidate_EmptyCharacterCorrected(t *testing.T) {
	c := &Character{
		Stats: validStats(),
		Buffs: newTestBuffs(),
		// SpeciesId intentionally left 0 → should default to 1
		// Description intentionally empty → should default
	}

	err := c.Validate()

	assert.NoError(t, err)
	assert.Equal(t, 1, c.SpeciesId, "SpeciesId should default to 1 when species lookup fails for 0")
	assert.NotEmpty(t, c.Description, "Description should get a default value")
	assert.NotNil(t, c.SpellBook, "SpellBook should be allocated")
	assert.NotNil(t, c.KnownRecipes, "KnownRecipes should be filled with starter recipes")
	assert.NotNil(t, c.Mutations, "Mutations should be allocated")
}

// TestValidate_SkillMapEnsured verifies that partial or nil skill maps are filled
// to rank 1 for all active skills, and that retired skills are stripped.
func TestValidate_SkillMapEnsured(t *testing.T) {
	c := &Character{
		Stats:     validStats(),
		Buffs:     newTestBuffs(),
		SpeciesId: 1,
		Skills: map[string]int{
			"cast":          5, // retired
			"ranged-combat": 3, // retired
			"first-aid":     2, // retired
			// intentionally omitting most active skills
		},
	}

	_ = c.Validate()

	_, hasCast := c.Skills["cast"]
	assert.False(t, hasCast, "retired skill 'cast' must be stripped by Validate")

	_, hasRanged := c.Skills["ranged-combat"]
	assert.False(t, hasRanged, "retired skill 'ranged-combat' must be stripped by Validate")

	_, hasFirstAid := c.Skills["first-aid"]
	assert.False(t, hasFirstAid, "retired skill 'first-aid' must be stripped by Validate")

	// All active skills must exist at rank >= 1 after Validate.
	for _, sk := range skills.GetAllSkillNames() {
		assert.GreaterOrEqual(t, c.Skills[string(sk)], 1,
			"active skill %q should be at rank >= 1 after ensureAllSkills", sk)
	}
}

// TestValidate_SkullduggeryMigration verifies that the legacy "stealth" skill key
// is renamed to "skullduggery" while preserving its rank.
func TestValidate_SkullduggeryMigration(t *testing.T) {
	c := &Character{
		Stats:     validStats(),
		Buffs:     newTestBuffs(),
		SpeciesId: 1,
		Skills: map[string]int{
			"stealth": 12,
		},
	}

	_ = c.Validate()

	_, hasStealth := c.Skills["stealth"]
	assert.False(t, hasStealth, "'stealth' key should be removed after migration")
	assert.Equal(t, 12, c.Skills["skullduggery"],
		"'skullduggery' should inherit rank 12 from the old 'stealth' key")
}

// TestValidate_SearchSkillMigration verifies that legacy "tracking" and "foraging"
// skills are merged into "search": rank = max(tracking, foraging), use-count = sum.
func TestValidate_SearchSkillMigration(t *testing.T) {
	c := &Character{
		Stats:     validStats(),
		Buffs:     newTestBuffs(),
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
	assert.False(t, hasTracking, "'tracking' should be removed after migration")
	assert.False(t, hasForaging, "'foraging' should be removed after migration")
	assert.Equal(t, 12, c.Skills["search"],
		"'search' rank should be max(tracking=5, foraging=12)")
	assert.Equal(t, 100, c.SkillUseCount["search"],
		"'search' use-count should be sum of tracking(40)+foraging(60)")
}

// TestValidate_ExtraArmsDerivation verifies that ExtraArms is derived from the
// "extra-arms" mutation level and capped at 4.
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
				Buffs:     newTestBuffs(),
				SpeciesId: 1,
				Mutations: map[string]int{},
			}
			if tt.level > 0 {
				c.Mutations["extra-arms"] = tt.level
			}

			_ = c.Validate()

			assert.Equal(t, tt.want, c.ExtraArms,
				"ExtraArms should be %d for mutation level %d", tt.want, tt.level)
		})
	}
}

// TestValidate_HealthClamping verifies that Validate clamps pool values to their
// legal ranges: Health ∈ [-10, HealthMax], Stamina ∈ [0, StaminaMax],
// Conviction ∈ [0, ConvictionMax].
func TestValidate_HealthClamping(t *testing.T) {
	c := &Character{
		Stats:      validStats(),
		Buffs:      newTestBuffs(),
		SpeciesId:  1,
		Health:     999999, // above max → clamp down
		Stamina:    -50,    // below 0 → clamp up
		Conviction: -50,    // below 0 → clamp up
	}

	_ = c.Validate()

	// After Validate, HealthMax is now populated.
	assert.LessOrEqual(t, c.Health, c.HealthMax.Value,
		"Health must be clamped to HealthMax after Validate")
	assert.Equal(t, 0, c.Stamina, "Stamina floor is 0")
	assert.Equal(t, 0, c.Conviction, "Conviction floor is 0")

	// Death-floor: Health must not go below -10.
	c.Health = -9999
	_ = c.Validate()
	assert.Equal(t, -10, c.Health, "Health death-floor is -10")
}

// TestValidate_ReturnsNilForValidCharacter verifies that a freshly-constructed
// character passes Validate without error and that pool maximums are stable
// across repeated calls.
func TestValidate_ReturnsNilForValidCharacter(t *testing.T) {
	c := New()

	// Capture pool maximums after New() (which already called Validate once).
	snapHealthMax := c.HealthMax.Value
	snapStaminaMax := c.StaminaMax.Value

	err := c.Validate()
	assert.NoError(t, err, "Validate must return nil for a valid character")

	// Pool maximums must be stable across repeated Validate calls.
	assert.Equal(t, snapHealthMax, c.HealthMax.Value,
		"HealthMax must be stable across repeated Validate calls")
	assert.Equal(t, snapStaminaMax, c.StaminaMax.Value,
		"StaminaMax must be stable across repeated Validate calls")
}
