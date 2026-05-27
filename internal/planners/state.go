package planners

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// PlanKeyPrefix is the MiscData key prefix every planner uses for its
// intermediate state. ClearPlanState wipes all keys with this prefix
// on goal switch.
const PlanKeyPrefix = "plan:"

// ClearPlanState wipes all "plan:" prefixed keys from
// mob.Character.MiscData. Wired into goals.Recompute via a
// SetPlanStateClear callback registered in main.go (Task 4).
//
// Nil-safe on mob == nil and MiscData == nil.
func ClearPlanState(mob *mobs.Mob) {
	if mob == nil || mob.Character.MiscData == nil {
		return
	}
	for k := range mob.Character.MiscData {
		if strings.HasPrefix(k, PlanKeyPrefix) {
			delete(mob.Character.MiscData, k)
		}
	}
}
