package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestWealthItem_Registered(t *testing.T) {
	if LookupPlanner("wealth-item") == nil {
		t.Fatalf("wealth-item planner not registered")
	}
}

func TestWealthItem_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("wealth-item")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "wealth-item"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (no params)", res.Status)
	}
}

func TestWealthItem_ItemPresentByItemId_Success(t *testing.T) {
	fn := LookupPlanner("wealth-item")
	mob := &mobs.Mob{}
	mob.Character.Items = []items.Item{{ItemId: 42}}
	g := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	res := fn(mob, g)
	if res.Status != StatusSuccess {
		t.Errorf("status=%v, want StatusSuccess (item in backpack by id)", res.Status)
	}
}

func TestWealthItem_ItemAbsent_NoShopInZone_Wanders(t *testing.T) {
	fn := LookupPlanner("wealth-item")
	mob := &mobs.Mob{}
	// findShopInZoneSelling returns false when no live shop state is loaded
	// (unit-test environment). Expect wander branch.
	g := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	res := fn(mob, g)
	if res.Command != "wander" {
		t.Errorf("command=%q, want wander (no shop in zone)", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want StatusRunning", res.Status)
	}
}

func TestWealthItem_TagOnly_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("wealth-item")
	// Both tag="" and itemId=0 → Failure even if Params map is present.
	g := &goals.Goal{Type: "wealth-item", Params: map[string]any{}}
	res := fn(&mobs.Mob{}, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (empty params map)", res.Status)
	}
}

// ─── mobHasItem unit tests ────────────────────────────────────────────────────

func TestMobHasItem_NilMob_False(t *testing.T) {
	if mobHasItem(nil, "sword", 0) {
		t.Errorf("expected false for nil mob")
	}
}

func TestMobHasItem_MatchById(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Items = []items.Item{{ItemId: 99}}
	if !mobHasItem(mob, "", 99) {
		t.Errorf("expected true: item 99 is in backpack")
	}
}

func TestMobHasItem_NoMatch(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Items = []items.Item{{ItemId: 1}}
	if mobHasItem(mob, "", 99) {
		t.Errorf("expected false: item 99 not in backpack")
	}
}

func TestMobHasItem_EmptyInventory_False(t *testing.T) {
	mob := &mobs.Mob{}
	if mobHasItem(mob, "blade", 0) {
		t.Errorf("expected false: empty inventory")
	}
}

// ─── matchesPlannerItem unit tests ───────────────────────────────────────────

func TestMatchesPlannerItem_IdMatch(t *testing.T) {
	if !matchesPlannerItem(10, "", "", 10) {
		t.Errorf("expected true: id match")
	}
}

func TestMatchesPlannerItem_TagMatch(t *testing.T) {
	if !matchesPlannerItem(0, "iron-ingot", "iron-ingot", 0) {
		t.Errorf("expected true: tag match")
	}
}

func TestMatchesPlannerItem_NoMatch(t *testing.T) {
	if matchesPlannerItem(5, "copper", "iron", 10) {
		t.Errorf("expected false: neither id nor tag matches")
	}
}

func TestMatchesPlannerItem_ZeroIdIgnored(t *testing.T) {
	// wantId=0 must not match any gotId (even gotId=0).
	if matchesPlannerItem(0, "x", "y", 0) {
		t.Errorf("expected false: wantId=0 should not match")
	}
}

func TestMatchesPlannerItem_EmptyTagIgnored(t *testing.T) {
	// wantTag="" must not match any gotTag (even gotTag="").
	if matchesPlannerItem(5, "", "", 0) {
		t.Errorf("expected false: wantTag empty should not match")
	}
}
