package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	befriendFactionFocusKey    = "plan:befriend-faction:focus_mob_instance_id"
	befriendFactionCooldownKey = "plan:befriend-faction:cooldown_round"
)

func init() {
	RegisterPlanner("befriend-faction", befriendFactionPlanner)
}

// befriendFactionPlanner: get into proximity with faction members.
// Picks one member as focus (sticky), pathing to / emoting near them.
// Actual rep accumulation is via 4.5's reactive counter-writing hook.
//
// Branches:
//   - nil mob or no faction_id param → Failure
//   - No members in zone (fresh pick fails) → Failure
//   - Focus id in MiscData is stale (nil instance or wrong zone) → re-pick
//   - Cooldown active → Running (waiting)
//   - Focus in same room → emote + set cooldown → Running
//   - Focus in zone different room → pathto → Running
func befriendFactionPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	factionId := goalParamStringOr(goal, "faction_id", "")
	if factionId == "" {
		return PlanResult{Status: StatusFailure}
	}

	// Resolve or pick a focus member.
	focusId := mobMiscIntOr(mob, befriendFactionFocusKey, 0)
	var focus *mobs.Mob
	if focusId != 0 {
		focus = mobs.GetInstance(focusId)
	}
	// Stale focus (gone or wrong zone) → re-pick.
	if focus == nil || focus.Character.Zone != mob.Character.Zone {
		picked, ok := findFactionMemberInZone(mob, factionId, false)
		if !ok {
			return PlanResult{Status: StatusFailure}
		}
		focus = picked
		mobSetMisc(mob, befriendFactionFocusKey, focus.InstanceId)
	}

	// Cooldown active?
	nowRound := util.GetRoundCount()
	if cd := uint64(mobMiscIntOr(mob, befriendFactionCooldownKey, 0)); nowRound < cd {
		return PlanResult{Status: StatusRunning}
	}

	// Same room → emote + cooldown.
	if focus.Character.RoomId == mob.Character.RoomId {
		cooldownLen := uint64(befriendInteractionCooldown())
		mobSetMisc(mob, befriendFactionCooldownKey, int(nowRound+cooldownLen))
		return PlanResult{Command: pickSocialEmote(), Status: StatusRunning}
	}

	// Different room same zone → pathto.
	return PlanResult{Command: "pathto " + strconv.Itoa(focus.Character.RoomId), Status: StatusRunning}
}
