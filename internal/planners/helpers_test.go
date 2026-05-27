package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// ─── pickSocialEmote ─────────────────────────────────────────────────────────

func TestPickSocialEmote_ReturnsNonEmpty(t *testing.T) {
	got := pickSocialEmote()
	if got == "" {
		t.Errorf("pickSocialEmote returned empty string")
	}
}

func TestPickSocialEmote_ReturnsValidEmote(t *testing.T) {
	valid := map[string]bool{
		"emote nods":         true,
		"emote bows":         true,
		"emote smiles warmly": true,
		"emote waves":        true,
		"emote grins":        true,
	}
	for i := 0; i < 20; i++ {
		got := pickSocialEmote()
		if !valid[got] {
			t.Errorf("pickSocialEmote returned unexpected emote %q", got)
		}
	}
}

// ─── zoneAdjacentTo ──────────────────────────────────────────────────────────

func TestZoneAdjacentTo_Cached(t *testing.T) {
	// First call computes; second call returns cached.
	zoneAdjacencyCache = nil
	first := zoneAdjacentTo("stillwater")
	if zoneAdjacencyCache == nil {
		t.Fatalf("cache not populated after first call")
	}
	second := zoneAdjacentTo("stillwater")
	if len(first) != len(second) {
		t.Errorf("first=%v second=%v differ in length", first, second)
	}
}

func TestZoneAdjacentTo_UnknownZoneReturnsEmpty(t *testing.T) {
	zoneAdjacencyCache = nil
	result := zoneAdjacentTo("no_such_zone_xyz")
	if result == nil {
		t.Errorf("expected non-nil slice for unknown zone, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice for unknown zone, got %v", result)
	}
}

// ─── pickGiftItemFromInventory ───────────────────────────────────────────────

func TestPickGiftItemFromInventory_NilMob_ReturnsNil(t *testing.T) {
	if got := pickGiftItemFromInventory(nil); got != nil {
		t.Errorf("expected nil for nil mob, got %v", got)
	}
}

func TestPickGiftItemFromInventory_EmptyMob_ReturnsNil(t *testing.T) {
	mob := &mobs.Mob{}
	if got := pickGiftItemFromInventory(mob); got != nil {
		t.Errorf("expected nil for empty inventory, got %v", got)
	}
}

func TestPickGiftItemFromInventory_QuestItemSkipped(t *testing.T) {
	// Register a test item spec with a QuestToken so it gets skipped.
	items.RegisterTestItemSpec(&items.ItemSpec{
		ItemId:     99901,
		Name:       "test quest token item",
		QuestToken: "somequest-start",
		Value:      500,
	})

	mob := &mobs.Mob{}
	mob.Character.Items = []items.Item{
		items.New(99901),
	}
	got := pickGiftItemFromInventory(mob)
	if got != nil {
		t.Errorf("expected nil (quest item should be skipped), got item %v", got)
	}
}

func TestPickGiftItemFromInventory_PicksHighestValue(t *testing.T) {
	items.RegisterTestItemSpec(&items.ItemSpec{
		ItemId: 99902,
		Name:   "cheap trinket",
		Value:  10,
	})
	items.RegisterTestItemSpec(&items.ItemSpec{
		ItemId: 99903,
		Name:   "expensive gem",
		Value:  200,
	})

	mob := &mobs.Mob{}
	mob.Character.Items = []items.Item{
		items.New(99902),
		items.New(99903),
	}
	got := pickGiftItemFromInventory(mob)
	if got == nil {
		t.Fatal("expected non-nil item")
	}
	spec := got.GetSpec()
	if spec.ItemId != 99903 {
		t.Errorf("expected highest-value item (99903), got %d", spec.ItemId)
	}
}

// ─── pickRandomExit ──────────────────────────────────────────────────────────

func TestPickRandomExit_NilMob_ReturnsEmpty(t *testing.T) {
	if got := pickRandomExit(nil); got != "" {
		t.Errorf("expected empty string for nil mob, got %q", got)
	}
}

// pickRandomExit with a real room requires live room state — integration
// coverage deferred to Task 23 smoke.

// ─── mobMiscIntOr ────────────────────────────────────────────────────────────

func TestMobMiscIntOr_NilMob_ReturnsDefault(t *testing.T) {
	if got := mobMiscIntOr(nil, "key", 42); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestMobMiscIntOr_MissingKey_ReturnsDefault(t *testing.T) {
	mob := &mobs.Mob{}
	if got := mobMiscIntOr(mob, "missing", 7); got != 7 {
		t.Errorf("expected 7, got %d", got)
	}
}

func TestMobMiscIntOr_SetAndRead(t *testing.T) {
	mob := &mobs.Mob{}
	mobSetMisc(mob, "plan:test:count", 99)
	if got := mobMiscIntOr(mob, "plan:test:count", 0); got != 99 {
		t.Errorf("expected 99, got %d", got)
	}
}

// ─── mobMiscStringOr ─────────────────────────────────────────────────────────

func TestMobMiscStringOr_NilMob_ReturnsDefault(t *testing.T) {
	if got := mobMiscStringOr(nil, "key", "def"); got != "def" {
		t.Errorf("expected \"def\", got %q", got)
	}
}

func TestMobMiscStringOr_SetAndRead(t *testing.T) {
	mob := &mobs.Mob{}
	mobSetMisc(mob, "plan:test:phase", "approach")
	if got := mobMiscStringOr(mob, "plan:test:phase", ""); got != "approach" {
		t.Errorf("expected \"approach\", got %q", got)
	}
}

// ─── goalParamIntOr ───────────────────────────────────────────────────────────

func TestGoalParamIntOr_NilGoal_ReturnsDefault(t *testing.T) {
	if got := goalParamIntOr(nil, "k", 5); got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
}

func TestGoalParamIntOr_MissingKey_ReturnsDefault(t *testing.T) {
	g := &goals.Goal{Params: map[string]any{"other": 1}}
	if got := goalParamIntOr(g, "missing", 3); got != 3 {
		t.Errorf("expected 3, got %d", got)
	}
}

func TestGoalParamIntOr_ReadsInt(t *testing.T) {
	g := &goals.Goal{Params: map[string]any{"hp_pct": 30}}
	if got := goalParamIntOr(g, "hp_pct", 0); got != 30 {
		t.Errorf("expected 30, got %d", got)
	}
}

func TestGoalParamIntOr_ReadsInt64(t *testing.T) {
	g := &goals.Goal{Params: map[string]any{"hp_pct": int64(25)}}
	if got := goalParamIntOr(g, "hp_pct", 0); got != 25 {
		t.Errorf("expected 25, got %d", got)
	}
}

// ─── goalParamStringOr ────────────────────────────────────────────────────────

func TestGoalParamStringOr_NilGoal_ReturnsDefault(t *testing.T) {
	if got := goalParamStringOr(nil, "k", "fallback"); got != "fallback" {
		t.Errorf("expected \"fallback\", got %q", got)
	}
}

func TestGoalParamStringOr_ReadsString(t *testing.T) {
	g := &goals.Goal{Params: map[string]any{"target": "thornwall"}}
	if got := goalParamStringOr(g, "target", ""); got != "thornwall" {
		t.Errorf("expected \"thornwall\", got %q", got)
	}
}

// ─── findShopInZoneSelling / findShopInZoneBuying / findFactionMemberInZone /
//     findHostileInZone / findCraftingStationInZone / pickKnownRecipeForSkill
//
// These helpers depend on live mob/zone/shop/room state — integration coverage
// deferred to per-planner tasks (8-20) and the Task 23 smoke session.
