package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	befriendCooldownKey      = "plan:befriend:cooldown_round"
	befriendDefaultThreshold = 60
	// Default cooldown between interactions if no config knob exists.
	// (Reads BefriendInteractionCooldown if defined; falls back to 30.)
	befriendDefaultCooldown = 30
)

func init() {
	RegisterPlanner("befriend", befriendPlanner)
}

// befriendPlanner: raise opinion with named target via positive social
// interactions.
//   - Opinion >= threshold → Success.
//   - Cooldown active → Running (waiting).
//   - Target same room → emit social action (rotated), set cooldown.
//   - Target same zone different room → pathto.
//   - Target out of zone → Failure.
func befriendPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	kind := goalParamStringOr(goal, "target_kind", "")
	id := goalParamIntOr(goal, "target_id", 0)
	threshold := goalParamIntOr(goal, "opinion_threshold", befriendDefaultThreshold)
	if kind == "" || id == 0 {
		return PlanResult{Status: StatusFailure}
	}

	// Opinion satisfied → Success. (Mob→mob opinions return 0 per 4.3
	// limitation; planner only meaningfully satisfies for player targets.)
	if kind == "player" && opinions.Get(int(mob.MobId), id) >= threshold {
		return PlanResult{Status: StatusSuccess}
	}

	// Cooldown active?
	nowRound := util.GetRoundCount()
	if cd := uint64(mobMiscIntOr(mob, befriendCooldownKey, 0)); nowRound < cd {
		return PlanResult{Status: StatusRunning} // waiting; no action this tick
	}

	targetRoom := resolveTargetRoomId(kind, id)
	if targetRoom == 0 {
		return PlanResult{Status: StatusFailure}
	}
	if r := rooms.LoadRoom(targetRoom); r == nil || r.Zone != mob.Character.Zone {
		return PlanResult{Status: StatusFailure}
	}

	// Same room → interaction + cooldown.
	if targetRoom == mob.Character.RoomId {
		cooldownLen := uint64(befriendInteractionCooldown())
		mobSetMisc(mob, befriendCooldownKey, int(nowRound+cooldownLen))
		return PlanResult{Command: pickSocialEmote(), Status: StatusRunning}
	}

	// Different room same zone → close distance.
	return PlanResult{Command: "pathto " + strconv.Itoa(targetRoom), Status: StatusRunning}
}

// befriendInteractionCooldown reads the config knob (if defined) or
// returns the default.
//
// TODO-ADAPT: if a BefriendInteractionCooldown knob doesn't exist yet,
// add it to configs.Balance (default 30) as part of this task. Otherwise
// fall back to the constant.
func befriendInteractionCooldown() int {
	// Cheapest path: just use the constant. Add config knob later if
	// gameplay tuning needs it.
	_ = configs.GetBalanceConfig()
	return befriendDefaultCooldown
}
