package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestClearPlanState_RemovesPrefixedKeys(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.MiscData = map[string]any{
		"plan:wealth-gold:target_shop_room": 12,
		"plan:befriend:cooldown_round":      uint64(5000),
		"plan:visit-zone:next_hop_zone":     "stillwater",
	}
	ClearPlanState(mob)
	if len(mob.Character.MiscData) != 0 {
		t.Errorf("expected 0 keys after Clear, got %d: %v", len(mob.Character.MiscData), mob.Character.MiscData)
	}
}

func TestClearPlanState_LeavesUnprefixedKeysUntouched(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.MiscData = map[string]any{
		"plan:wealth-gold:target_shop_room": 12,
		"faction_kills_inflicted:bandits":   3,
		"conversation_line_idx":             2,
		"some_other_key":                    "value",
	}
	ClearPlanState(mob)
	if _, has := mob.Character.MiscData["plan:wealth-gold:target_shop_room"]; has {
		t.Errorf("plan: key not wiped")
	}
	if mob.Character.MiscData["faction_kills_inflicted:bandits"] != 3 {
		t.Errorf("non-prefixed key wiped (faction_kills): %v", mob.Character.MiscData)
	}
	if mob.Character.MiscData["conversation_line_idx"] != 2 {
		t.Errorf("non-prefixed key wiped (conversation_line_idx): %v", mob.Character.MiscData)
	}
	if mob.Character.MiscData["some_other_key"] != "value" {
		t.Errorf("non-prefixed key wiped (some_other_key): %v", mob.Character.MiscData)
	}
}

func TestClearPlanState_NilMob_NoOp(t *testing.T) {
	ClearPlanState(nil) // must not panic
}

func TestClearPlanState_NilMiscData_NoOp(t *testing.T) {
	mob := &mobs.Mob{} // MiscData is nil
	ClearPlanState(mob) // must not panic
}

func TestClearPlanState_EmptyMiscData_NoOp(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.MiscData = map[string]any{}
	ClearPlanState(mob)
	if len(mob.Character.MiscData) != 0 {
		t.Errorf("len=%d, want 0", len(mob.Character.MiscData))
	}
}
