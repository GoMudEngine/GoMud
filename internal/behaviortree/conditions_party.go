package behaviortree

// conditions_party.go — party-state query conditions for NPC party behavior
// trees.
//
// Each condition looks up the caller's party via
// parties.GetByMobInstanceId(ctx.InstanceId) and returns Failure (not panic)
// when the caller isn't in a party, so behavior tree selectors fall through
// to other branches gracefully.

import (
	"github.com/GoMudEngine/GoMud/internal/parties"
)

func init() {
	conditionRegistry["party_member_below_pct"] = condPartyMemberBelowPct
	conditionRegistry["party_in_combat"] = condPartyInCombat
	conditionRegistry["party_leader_in_combat"] = condPartyLeaderInCombat
	conditionRegistry["party_in_room"] = condPartyInRoom
	conditionRegistry["party_at_home"] = condPartyAtHome
	conditionRegistry["party_help_inactive"] = condPartyHelpInactive
}

// condPartyMemberBelowPct returns Success if any party member's resource
// pool is below the threshold percentage.
// Params:
//   - pool    string ("hp" | "sp" | "cp")
//   - percent int   (1-100, exclusive lower bound)
func condPartyMemberBelowPct(params map[string]any, ctx *EvalContext) Result {
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil {
		return Failure
	}
	pool := getStringParam(params, "pool")
	pct := getIntParam(params, "percent")
	if pool == "" || pct <= 0 {
		return Failure
	}
	for _, m := range p.Members {
		c := m.GetCharacter()
		if c == nil {
			continue
		}
		var current, max int
		switch pool {
		case "hp":
			current = c.Health
			max = c.HealthMax.Value
		case "sp":
			current = c.Stamina
			max = c.StaminaMax.Value
		case "cp":
			current = c.Conviction
			max = c.ConvictionMax.Value
		default:
			return Failure
		}
		if max > 0 && (current*100)/max < pct {
			return Success
		}
	}
	return Failure
}

// condPartyInCombat returns Success if any party member is in combat
// (has a non-nil Aggro).
// No params.
func condPartyInCombat(params map[string]any, ctx *EvalContext) Result {
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil {
		return Failure
	}
	for _, m := range p.Members {
		c := m.GetCharacter()
		if c != nil && c.Aggro != nil {
			return Success
		}
	}
	return Failure
}

// condPartyLeaderInCombat returns Success if specifically the party leader
// is in combat (has a non-nil Aggro).
// No params.
func condPartyLeaderInCombat(params map[string]any, ctx *EvalContext) Result {
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil || p.Leader == nil {
		return Failure
	}
	c := p.Leader.GetCharacter()
	if c != nil && c.Aggro != nil {
		return Success
	}
	return Failure
}

// condPartyInRoom returns Success if all party members are in the same room.
// Returns Failure when the party has no members or any member has no room.
// No params.
func condPartyInRoom(params map[string]any, ctx *EvalContext) Result {
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil || len(p.Members) == 0 {
		return Failure
	}
	first := p.Members[0].GetRoom()
	if first == nil {
		return Failure
	}
	for _, m := range p.Members[1:] {
		r := m.GetRoom()
		if r == nil || r.RoomId != first.RoomId {
			return Failure
		}
	}
	return Success
}

// condPartyHelpInactive returns Success when the party has no active help
// call (HelpRoomId == 0). Used to suppress retreat / flee branches while a
// rally is in progress — without this gate, a leader who just arrived at
// the rally room would immediately order the whole party back home if any
// fighting member's HP was already below the flee threshold.
//
// Returns Failure when the caller is not in a party.
// No params.
func condPartyHelpInactive(params map[string]any, ctx *EvalContext) Result {
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil {
		return Failure
	}
	if p.HelpRoomId == 0 {
		return Success
	}
	return Failure
}

// condPartyAtHome returns Success if all party members are in the party's
// HomeRoomId. Returns Failure when HomeRoomId is 0 (unset) or any member
// is outside the home room.
// No params.
func condPartyAtHome(params map[string]any, ctx *EvalContext) Result {
	p := parties.GetByMobInstanceId(ctx.InstanceId)
	if p == nil || p.HomeRoomId == 0 {
		return Failure
	}
	for _, m := range p.Members {
		r := m.GetRoom()
		if r == nil || r.RoomId != p.HomeRoomId {
			return Failure
		}
	}
	return Success
}
