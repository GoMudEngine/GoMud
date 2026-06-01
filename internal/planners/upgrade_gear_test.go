package planners

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestUpgradeGear_Registered(t *testing.T) {
	if LookupPlanner("upgrade-gear") == nil {
		t.Fatalf("upgrade-gear planner not registered")
	}
}

func TestUpgradeGear_NilMob_Failure(t *testing.T) {
	fn := LookupPlanner("upgrade-gear")
	res := fn(nil, &goals.Goal{Type: "upgrade-gear"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (nil mob)", res.Status)
	}
}

func TestUpgradeGear_NothingInStock_Idle(t *testing.T) {
	fn := LookupPlanner("upgrade-gear")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	mob.Character.Gold = 1000
	res := fn(mob, &goals.Goal{Type: "upgrade-gear"})
	if res.Command != "" {
		t.Errorf("command=%q, want empty (nothing in stock, nothing to sell)", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want StatusRunning", res.Status)
	}
}

func TestUpgradeGear_PendingEquip_EmitsGearup(t *testing.T) {
	fn := LookupPlanner("upgrade-gear")
	mob := &mobs.Mob{}
	mob.Character.MiscData = map[string]any{
		upgradePendingEquipKey: 1,
	}
	res := fn(mob, &goals.Goal{Type: "upgrade-gear"})
	if res.Command != "gearup" {
		t.Errorf("command=%q, want gearup", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want StatusRunning", res.Status)
	}
	if mobMiscIntOr(mob, upgradePendingEquipKey, 0) != 0 {
		t.Errorf("pending-equip flag should be cleared after gearup")
	}
}

func TestUpgradeGear_MiscKeys_HavePlanPrefix(t *testing.T) {
	for _, k := range []string{upgradePendingEquipKey, upgradeSellShopRoomKey} {
		if !strings.HasPrefix(k, PlanKeyPrefix) {
			t.Errorf("key %q does not start with PlanKeyPrefix %q", k, PlanKeyPrefix)
		}
	}
}
