package catalog

import (
	"strconv"
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestWealthItem_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("wealth-item"); !ok {
		t.Fatalf("wealth-item not registered")
	}
}

func TestWealthItem_DedupKey_ByItemId(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-item")
	g1 := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	g2 := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 99}}
	if k1, k2 := meta.DedupKey(g1), meta.DedupKey(g2); k1 == k2 {
		t.Errorf("dedup keys collide: %s == %s (different ids)", k1, k2)
	}
}

func TestWealthItem_DedupKey_ByTag(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-item")
	g1 := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_tag": "iron-ingot"}}
	g2 := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_tag": "iron-ingot"}}
	if meta.DedupKey(g1) != meta.DedupKey(g2) {
		t.Errorf("dedup keys differ for same tag")
	}
}

func TestWealthItem_Predicate_ItemAbsent_False(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-item")
	mob := &mobs.Mob{}
	// Empty inventory.
	g := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	if meta.Predicate(g, mob) {
		t.Errorf("predicate with absent item: got true, want false")
	}
}

func TestWealthItem_ContextScore_Present_Zero(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-item")
	mob := &mobs.Mob{}
	mob.Character.Items = []items.Item{{ItemId: 42}}
	g := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	if got := meta.ContextScore(g, mob); got != 0 {
		t.Errorf("score when item present: got %f, want 0", got)
	}
	_ = strconv.Itoa(0) // silence import-unused if test rearranges
}

func TestWealthItem_ContextScore_Absent_OnePointZero(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-item")
	mob := &mobs.Mob{}
	g := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	got := meta.ContextScore(g, mob)
	// 1.0 baseline; the +0.5 shop-in-zone bump requires a real zone scan
	// which we don't have in unit tests, so the baseline is what fires.
	if got != 1.0 {
		t.Errorf("score when item absent (no zone shop scan): got %f, want 1.0", got)
	}
}
