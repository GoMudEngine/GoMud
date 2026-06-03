package characters

import "github.com/GoMudEngine/GoMud/internal/util"

// ForceTauntAggro sets the character's combat target to the taunter and
// applies a taunt-hold lock for holdRounds rounds. While the lock is active,
// SetAggro ignores basic-attack re-aggro that would switch the target away
// from the taunter (the reactive per-round `attack` re-target, the combat
// reciprocal, target-switch AI). This is what lets a taunt actually hold
// aggro instead of being clobbered the instant an ally hits the target.
//
// The lock fields are set BEFORE the SetAggro call so the gate sees the new
// taunter as the locked target and lets the set through (this is also why a
// newer taunt cleanly overrides an active hold from a previous taunter).
func (c *Character) ForceTauntAggro(userId, mobInstanceId, holdRounds int) {
	if holdRounds < 1 {
		holdRounds = 1
	}
	c.tauntHoldUserId = userId
	c.tauntHoldMobInstanceId = mobInstanceId
	c.tauntHoldUntilRound = util.GetRoundCount() + uint64(holdRounds)
	c.SetAggro(userId, mobInstanceId, DefaultAttack)
}

// tauntHoldActive reports whether a taunt-hold lock is currently in force.
func (c *Character) tauntHoldActive() bool {
	if c.tauntHoldUserId == 0 && c.tauntHoldMobInstanceId == 0 {
		return false
	}
	return c.tauntHoldUntilRound > util.GetRoundCount()
}

// tauntHoldBlocks reports whether an incoming SetAggro should be ignored
// because a taunt hold pins the target onto a different taunter. Only basic
// attack-type aggro (DefaultAttack/Shooting/SurpriseAttack) is pinned;
// SpellCast and Flee (self/room-directed) always pass, as does a set that
// matches the locked taunter.
func (c *Character) tauntHoldBlocks(userId, mobInstanceId int, aggroType AggroType) bool {
	if !c.tauntHoldActive() {
		return false
	}
	switch aggroType {
	case DefaultAttack, Shooting, SurpriseAttack:
		return userId != c.tauntHoldUserId || mobInstanceId != c.tauntHoldMobInstanceId
	default:
		return false
	}
}

// clearTauntHold drops any active taunt-hold lock. Called from EndAggro so a
// dead/fled taunter doesn't leave the enemy pinned and unable to re-acquire.
func (c *Character) clearTauntHold() {
	c.tauntHoldUntilRound = 0
	c.tauntHoldUserId = 0
	c.tauntHoldMobInstanceId = 0
}
