package mobs

import (
	"github.com/GoMudEngine/GoMud/internal/util"
)

// EventContext is a minimal wrapper for behavior tree event data.
// Defined here to avoid importing behaviortree in mobs package.
type EventContext struct {
	EventType string
	UserId    int
	MobId     int
}

// TryMobBehaviorFunc is the signature for the behavior tree dispatch callback.
// Implementations are provided by the behaviortree package at init time.
type TryMobBehaviorFunc func(mobInstanceId int, ctx EventContext) bool

// AttackRejectedTryMobBehavior is a test-swappable wrapper that fires
// the player_attack_rejected btree event. Tests can inject a tracking stub.
// Production value is set by behaviortree.Register() at init time.
var AttackRejectedTryMobBehavior TryMobBehaviorFunc = func(mobInstanceId int, ctx EventContext) bool {
	return false // default no-op for tests or prior to registration
}

// FireAttackRejected fires the player_attack_rejected btree event on
// the given mob, subject to round-based dedupe on
// Character.LastAttackRejectedRound. Called from attack.go and cast.go's
// non-combatant rejection paths.
func FireAttackRejected(mob *Mob, userId int) {
	currentRound := util.GetRoundCount()
	if currentRound <= mob.Character.LastAttackRejectedRound {
		return
	}
	mob.Character.LastAttackRejectedRound = currentRound
	AttackRejectedTryMobBehavior(mob.InstanceId, EventContext{
		EventType: "player_attack_rejected",
		UserId:    userId,
	})
}
