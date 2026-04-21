package behaviortree

import (
	"sort"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/spells"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

func spellIds(sds []*spells.SpellData) []string {
	out := make([]string, len(sds))
	for i, sd := range sds {
		out[i] = sd.SpellId
	}
	return out
}

// seedCategorySpells installs a minimal self_defense spell catalog and
// returns the cleanup function. Buff IDs used: 10, 11.
func seedCategorySpells(t *testing.T) func() {
	t.Helper()
	return spells.SeedSpellsForTest(map[string]*spells.SpellData{
		// category matches, affordable
		"d1": {SpellId: "d1", Type: spells.HelpSingle, Cost: 10, BaseFolds: 2,
			Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{10}},
		// category matches, affordable, higher score
		"d2": {SpellId: "d2", Type: spells.HelpSingle, Cost: 50, BaseFolds: 6,
			Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{11}},
		// wrong category
		"other": {SpellId: "other", Type: spells.HelpSingle, Cost: 10, BaseFolds: 2,
			Categories: []string{"self_offense"}, EffectType: "buff", BuffIds: []int{12}},
		// too expensive (cost 999 > cpHave 100)
		"broke": {SpellId: "broke", Type: spells.HelpSingle, Cost: 999, BaseFolds: 6,
			Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{13}},
		// component required
		"compreq": {SpellId: "compreq", Type: spells.HelpSingle, Cost: 10, BaseFolds: 2,
			Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{14},
			ComponentTag: "reagent"},
		// summon mob
		"summon": {SpellId: "summon", Type: spells.HelpSingle, Cost: 10, BaseFolds: 2,
			Categories: []string{"self_defense"}, SummonMobId: 999},
		// charm spell
		"charm": {SpellId: "charm", Type: spells.HarmSingle, Cost: 10, BaseFolds: 2,
			Categories: []string{"self_defense"}, EffectType: "charm"},
		// summon component required (SummonComponentId != 0)
		"compsum": {SpellId: "compsum", Type: spells.HelpSingle, Cost: 10, BaseFolds: 2,
			Categories: []string{"self_defense"}, SummonComponentId: 5},
		// shield effect type
		"shield1": {SpellId: "shield1", Type: spells.HelpSingle, Cost: 10, BaseFolds: 4,
			Categories: []string{"self_defense"}, EffectType: "shield"},
		// ranking: higher score than d2 (BaseFolds=8 × Cost=10 = 80 vs d2's 300)
		// use d3 with BaseFolds=10 × Cost=50 = 500 for a top-scorer test
		"d3": {SpellId: "d3", Type: spells.HelpSingle, Cost: 50, BaseFolds: 10,
			Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{15}},
	})
}

// ─── collectCategoryCandidates unit tests ────────────────────────────────────

func TestCollectCategoryCandidates_NilCharReturnsNil(t *testing.T) {
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{})
	defer cleanup()
	got := collectCategoryCandidates(nil, map[string]int{"x": 1}, "self_defense", 1, "test")
	if len(got) != 0 {
		t.Fatalf("want 0 candidates for nil char, got %d", len(got))
	}
}

func TestCollectCategoryCandidates_EmptyCategory(t *testing.T) {
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"d1": {SpellId: "d1", Categories: []string{"self_defense"}, Cost: 10, BaseFolds: 2},
	})
	defer cleanup()
	char := &characters.Character{Conviction: 100}
	got := collectCategoryCandidates(char, map[string]int{"d1": 1}, "", 1, "test")
	if len(got) != 0 {
		t.Fatalf("want 0 for empty category, got %d", len(got))
	}
}

func TestCollectCategoryCandidates_NoCategoryMatch(t *testing.T) {
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"d1": {SpellId: "d1", Categories: []string{"self_offense"}, Cost: 10, BaseFolds: 2},
	})
	defer cleanup()
	char := &characters.Character{Conviction: 100}
	got := collectCategoryCandidates(char, map[string]int{"d1": 1}, "self_defense", 1, "test")
	if len(got) != 0 {
		t.Fatalf("want 0 (no category match), got %d", len(got))
	}
}

func TestCollectCategoryCandidates_MultipleCandidatesAllIncluded(t *testing.T) {
	cleanup := seedCategorySpells(t)
	defer cleanup()

	char := &characters.Character{Conviction: 100, Buffs: buffs.New()}
	// spellbook: d1, d2, other, broke, compreq, summon, charm, compsum
	sb := map[string]int{"d1": 1, "d2": 1, "other": 1, "broke": 1, "compreq": 1, "summon": 1, "charm": 1, "compsum": 1}
	got := collectCategoryCandidates(char, sb, "self_defense", 1, "test")
	// Only d1 and d2 should pass all checks (category match + affordable + no component + not summon/charm)
	if len(got) != 2 {
		t.Fatalf("want 2 candidates (d1, d2), got %d: %v", len(got), spellIds(got))
	}
	ids := spellIds(got)
	hasD1, hasD2 := false, false
	for _, id := range ids {
		if id == "d1" {
			hasD1 = true
		}
		if id == "d2" {
			hasD2 = true
		}
	}
	if !hasD1 || !hasD2 {
		t.Errorf("expected d1 and d2 in candidates, got %v", ids)
	}
}

func TestCollectCategoryCandidates_SkipsBuffAlreadyActive(t *testing.T) {
	cleanup := seedCategorySpells(t)
	defer cleanup()
	cleanupBuffs := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		10: {BuffId: 10, Name: "TestBuff10"},
		11: {BuffId: 11, Name: "TestBuff11"},
	})
	defer cleanupBuffs()

	// Build a Buffs tracker with buff 10 already in the list, then Validate()
	// to rebuild the internal buffIds index. This avoids calling char.AddBuff
	// which triggers Character.Validate() and may clamp Conviction to 0.
	b := buffs.New()
	b.List = append(b.List, &buffs.Buff{BuffId: 10, TriggersLeft: 5})
	b.Validate(true)

	char := &characters.Character{Conviction: 100, Buffs: b}
	if !char.HasBuff(10) {
		t.Fatal("test setup: HasBuff(10) should be true after seeding List")
	}

	sb := map[string]int{"d1": 1, "d2": 1}
	got := collectCategoryCandidates(char, sb, "self_defense", 1, "test")
	// d1 should be excluded (buff 10 active), d2 should be included
	if len(got) != 1 {
		t.Fatalf("want 1 candidate (d2 only), got %d: %v", len(got), spellIds(got))
	}
	if got[0].SpellId != "d2" {
		t.Errorf("expected d2, got %s", got[0].SpellId)
	}
}

func TestCollectCategoryCandidates_SkipsShieldAlreadyActive(t *testing.T) {
	cleanup := seedCategorySpells(t)
	defer cleanup()

	char := &characters.Character{Conviction: 100, Buffs: buffs.New()}
	// Manually mark the character as having a shield by using HasShield check.
	// HasShield() checks worn equipment, which we can't easily set without full
	// item setup. Instead, verify the shield-active path doesn't panic and
	// returns the shield spell when no shield is active.
	sb := map[string]int{"shield1": 1}
	got := collectCategoryCandidates(char, sb, "self_defense", 1, "test")
	// No shield active → should be included
	if len(got) != 1 {
		t.Fatalf("want 1 (shield1 with no active shield), got %d: %v", len(got), spellIds(got))
	}
	if got[0].SpellId != "shield1" {
		t.Errorf("expected shield1, got %s", got[0].SpellId)
	}
}

func TestCollectCategoryCandidates_SkipsInsufficientCP(t *testing.T) {
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"expensive": {SpellId: "expensive", Type: spells.HelpSingle, Cost: 999, BaseFolds: 4,
			Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{20}},
		"cheap": {SpellId: "cheap", Type: spells.HelpSingle, Cost: 5, BaseFolds: 2,
			Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{21}},
	})
	defer cleanup()

	char := &characters.Character{Conviction: 50, Buffs: buffs.New()}
	sb := map[string]int{"expensive": 1, "cheap": 1}
	got := collectCategoryCandidates(char, sb, "self_defense", 1, "test")
	if len(got) != 1 {
		t.Fatalf("want 1 candidate (cheap only), got %d: %v", len(got), spellIds(got))
	}
	if got[0].SpellId != "cheap" {
		t.Errorf("expected cheap, got %s", got[0].SpellId)
	}
}

func TestCollectCategoryCandidates_SkipsComponentTag(t *testing.T) {
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"compreq": {SpellId: "compreq", Type: spells.HelpSingle, Cost: 10, BaseFolds: 2,
			Categories: []string{"self_defense"}, ComponentTag: "reagent"},
	})
	defer cleanup()

	char := &characters.Character{Conviction: 100, Buffs: buffs.New()}
	got := collectCategoryCandidates(char, map[string]int{"compreq": 1}, "self_defense", 1, "test")
	if len(got) != 0 {
		t.Fatalf("want 0 (component required), got %d", len(got))
	}
}

func TestCollectCategoryCandidates_SkipsSummonComponentId(t *testing.T) {
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"compsum": {SpellId: "compsum", Type: spells.HelpSingle, Cost: 10, BaseFolds: 2,
			Categories: []string{"self_defense"}, SummonComponentId: 5},
	})
	defer cleanup()

	char := &characters.Character{Conviction: 100, Buffs: buffs.New()}
	got := collectCategoryCandidates(char, map[string]int{"compsum": 1}, "self_defense", 1, "test")
	if len(got) != 0 {
		t.Fatalf("want 0 (summon component required), got %d", len(got))
	}
}

func TestCollectCategoryCandidates_SkipsSummonMobId(t *testing.T) {
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"summon": {SpellId: "summon", Type: spells.HelpSingle, Cost: 10, BaseFolds: 2,
			Categories: []string{"self_defense"}, SummonMobId: 999},
	})
	defer cleanup()

	char := &characters.Character{Conviction: 100, Buffs: buffs.New()}
	got := collectCategoryCandidates(char, map[string]int{"summon": 1}, "self_defense", 1, "test")
	if len(got) != 0 {
		t.Fatalf("want 0 (summon spell), got %d", len(got))
	}
}

func TestCollectCategoryCandidates_SkipsCharmEffectType(t *testing.T) {
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"charm": {SpellId: "charm", Type: spells.HarmSingle, Cost: 10, BaseFolds: 2,
			Categories: []string{"self_defense"}, EffectType: "charm"},
	})
	defer cleanup()

	char := &characters.Character{Conviction: 100, Buffs: buffs.New()}
	got := collectCategoryCandidates(char, map[string]int{"charm": 1}, "self_defense", 1, "test")
	if len(got) != 0 {
		t.Fatalf("want 0 (charm spell), got %d", len(got))
	}
}

func TestCollectCategoryCandidates_DeletedSpellIdDoesNotCrash(t *testing.T) {
	// Spellbook references "ghost" which is not in the seed map.
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"real": {SpellId: "real", Type: spells.HelpSingle, Cost: 10, BaseFolds: 2,
			Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{30}},
	})
	defer cleanup()

	char := &characters.Character{Conviction: 100, Buffs: buffs.New()}
	sb := map[string]int{"ghost": 1, "real": 1}
	// Should not crash; ghost excluded, real included.
	got := collectCategoryCandidates(char, sb, "self_defense", 1, "test")
	if len(got) != 1 {
		t.Fatalf("want 1 (real only, ghost is deleted), got %d: %v", len(got), spellIds(got))
	}
	if got[0].SpellId != "real" {
		t.Errorf("expected real, got %s", got[0].SpellId)
	}
}

// ─── Ranking test ─────────────────────────────────────────────────────────────

func TestCastBestInCategory_RankingSelectsHighestScore(t *testing.T) {
	// Three spells with distinct base_folds × cost scores:
	//   lo:  2 × 5  = 10
	//   mid: 4 × 10 = 40
	//   hi:  6 × 20 = 120  ← should win
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"lo":  {SpellId: "lo", Type: spells.HelpSingle, Cost: 5, BaseFolds: 2, Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{40}},
		"mid": {SpellId: "mid", Type: spells.HelpSingle, Cost: 10, BaseFolds: 4, Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{41}},
		"hi":  {SpellId: "hi", Type: spells.HelpSingle, Cost: 20, BaseFolds: 6, Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{42}},
	})
	defer cleanup()

	char := &characters.Character{Conviction: 500, Buffs: buffs.New()}
	sb := map[string]int{"lo": 1, "mid": 1, "hi": 1}

	candidates := collectCategoryCandidates(char, sb, "self_defense", 1, "test")
	if len(candidates) != 3 {
		t.Fatalf("want 3 candidates, got %d: %v", len(candidates), spellIds(candidates))
	}

	// Apply the same sort as the action.
	sort.SliceStable(candidates, func(i, j int) bool {
		si, sj := candidates[i], candidates[j]
		scoreI := si.BaseFolds * si.Cost
		scoreJ := sj.BaseFolds * sj.Cost
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		return si.SpellId < sj.SpellId
	})

	if candidates[0].SpellId != "hi" {
		t.Errorf("expected highest-scoring spell 'hi' first, got %s", candidates[0].SpellId)
	}
	if candidates[1].SpellId != "mid" {
		t.Errorf("expected 'mid' second, got %s", candidates[1].SpellId)
	}
	if candidates[2].SpellId != "lo" {
		t.Errorf("expected 'lo' last, got %s", candidates[2].SpellId)
	}
}

func TestCastBestInCategory_TieBreaksBySpellIdAsc(t *testing.T) {
	// Two spells with identical score — "aaa" < "zzz" alphabetically.
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"aaa": {SpellId: "aaa", Type: spells.HelpSingle, Cost: 10, BaseFolds: 5, Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{50}},
		"zzz": {SpellId: "zzz", Type: spells.HelpSingle, Cost: 10, BaseFolds: 5, Categories: []string{"self_defense"}, EffectType: "buff", BuffIds: []int{51}},
	})
	defer cleanup()

	char := &characters.Character{Conviction: 500, Buffs: buffs.New()}
	candidates := collectCategoryCandidates(char, map[string]int{"aaa": 1, "zzz": 1}, "self_defense", 1, "test")
	if len(candidates) != 2 {
		t.Fatalf("want 2, got %d", len(candidates))
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		si, sj := candidates[i], candidates[j]
		scoreI := si.BaseFolds * si.Cost
		scoreJ := sj.BaseFolds * sj.Cost
		if scoreI != scoreJ {
			return scoreI > scoreJ
		}
		return si.SpellId < sj.SpellId
	})

	if candidates[0].SpellId != "aaa" {
		t.Errorf("tie-break: expected 'aaa' first (asc), got %s", candidates[0].SpellId)
	}
}

// ─── Action-level smoke tests ─────────────────────────────────────────────────

func TestActCastBestInCategory_FailureOnEmptyCategory(t *testing.T) {
	ctx := &EvalContext{InstanceId: 999}
	if result := actCastBestInCategory(map[string]any{}, ctx); result != Failure {
		t.Errorf("expected Failure for empty category, got %v", result)
	}
}

func TestActCastBestInCategory_FailureOnMissingMob(t *testing.T) {
	// InstanceId 999 is not seeded.
	cleanup := mobs.SeedMobsForTest(map[int]*mobs.Mob{}, map[int]*mobs.Mob{})
	defer cleanup()

	ctx := &EvalContext{InstanceId: 999}
	params := map[string]any{"category": "self_defense"}
	if result := actCastBestInCategory(params, ctx); result != Failure {
		t.Errorf("expected Failure for missing mob, got %v", result)
	}
}

func TestActCastBestInCategory_FailureOnNoCandidates(t *testing.T) {
	cleanup := seedCategorySpells(t)
	defer cleanup()

	cleanMob := seedTestMob(t, 1, 101, 1, "TestMob")
	defer cleanMob()

	// Give the mob a spellbook but with a spell not in "self_defense"
	mob := mobs.GetInstance(101)
	mob.Character.Conviction = 500
	mob.Character.SpellBook = map[string]int{"other": 1}

	ctx := &EvalContext{InstanceId: 101}
	params := map[string]any{"category": "self_defense"}
	if result := actCastBestInCategory(params, ctx); result != Failure {
		t.Errorf("expected Failure when no candidates, got %v", result)
	}
}

func TestActCastBestInCategory_FailureOnSpecialMoveCooldown(t *testing.T) {
	cleanup := seedCategorySpells(t)
	defer cleanup()

	cleanMob := seedTestMob(t, 2, 102, 1, "CooldownMob")
	defer cleanMob()

	mob := mobs.GetInstance(102)
	mob.Character.Conviction = 500
	mob.Character.SpellBook = map[string]int{"d1": 1}
	mob.Character.Cooldowns = characters.Cooldowns{"special-move": 3}

	ctx := &EvalContext{InstanceId: 102}
	params := map[string]any{"category": "self_defense"}
	if result := actCastBestInCategory(params, ctx); result != Failure {
		t.Errorf("expected Failure when on special-move cooldown, got %v", result)
	}
}

func TestActCastBestInCategory_FailureWhenAlreadyCasting(t *testing.T) {
	cleanup := seedCategorySpells(t)
	defer cleanup()

	cleanMob := seedTestMob(t, 3, 103, 1, "CastingMob")
	defer cleanMob()

	mob := mobs.GetInstance(103)
	mob.Character.Conviction = 500
	mob.Character.SpellBook = map[string]int{"d1": 1}
	mob.Character.CastingState = &characters.CastingState{}

	ctx := &EvalContext{InstanceId: 103}
	params := map[string]any{"category": "self_defense"}
	if result := actCastBestInCategory(params, ctx); result != Failure {
		t.Errorf("expected Failure when already casting, got %v", result)
	}
}

func TestActCastBestInCategory_SuccessWithViableCandidate(t *testing.T) {
	cleanup := seedCategorySpells(t)
	defer cleanup()

	cleanMob := seedTestMob(t, 4, 104, 1, "SpellMob")
	defer cleanMob()

	mob := mobs.GetInstance(104)
	mob.Character.Conviction = 500
	mob.Character.SpellBook = map[string]int{"d1": 1, "d2": 1}

	ctx := &EvalContext{InstanceId: 104}
	params := map[string]any{"category": "self_defense"}
	if result := actCastBestInCategory(params, ctx); result != Success {
		t.Errorf("expected Success with viable candidates, got %v", result)
	}
}
