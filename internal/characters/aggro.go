package characters

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/combatphase"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type AggroType int

const (
	// Enumerated Aggro Types
	DefaultAttack AggroType = iota // Regular H2H combat, everything can decay to this. Starts at zero
	Shooting
	SurpriseAttack
	SpellCast
	Flee
)

type SpellAggroInfo struct {
	SpellId              string
	SpellRest            string
	TargetUserIds        []int
	TargetMobInstanceIds []int
}

type Aggro struct {
	Type          AggroType
	MobInstanceId int
	UserId        int
	SpellInfo     SpellAggroInfo // If Type is SpellCast, this is the spell info
	ExitName      string         // For example, firing a weapon in a direction
	RoundsWaiting int            // How many rounds must pass before this triggers
}

// userUntargetableFn is registered from hooks at boot. Returns true if
// the user with the given id is protected from incoming aggro (e.g.,
// post-respawn grace period). Called from SetAggro before setting
// aggro on a player target. nil = no check (safe default for tests).
var userUntargetableFn func(userId int) bool

// SetUserUntargetableCheck registers the untargetable-user check used
// by SetAggro's player-target gate. Follows the callback pattern used
// by rooms.SetCompanionTransport / rooms.SetBTreeStateEvictor to
// avoid the users→characters import cycle (characters cannot import
// users directly).
//
// Repeated registrations overwrite; pass nil to disable.
func SetUserUntargetableCheck(fn func(userId int) bool) {
	userUntargetableFn = fn
}

// returns description unless description is a hash
// which points to another description location.
func (c *Character) TrackPlayerDamage(userId int, damageAmt int) {

	roundNow := util.GetRoundCount()
	if len(c.PlayerDamage) == 0 {
		c.PlayerDamage = map[int]int{}
	} else {
		if roundNow-c.LastPlayerDamage > 30 {
			clear(c.PlayerDamage)
		}
	}

	c.PlayerDamage[userId] = c.PlayerDamage[userId] + damageAmt
	c.LastPlayerDamage = roundNow

}

func (c *Character) SetAggroRemote(exitName string, userId int, mobInstanceId int, aggroType AggroType, roundsWaitTime ...int) {
	c.SetAggro(userId, mobInstanceId, aggroType, roundsWaitTime...)
	c.Aggro.ExitName = exitName
}

func (c *Character) SetAggro(userId int, mobInstanceId int, aggroType AggroType, roundsWaitTime ...int) {
	// Grace-period guard: don't acquire aggro on a grace-protected
	// player. Other target shapes (mob, spellcast) are unaffected.
	if userId > 0 && userUntargetableFn != nil && userUntargetableFn(userId) {
		return
	}

	// Stage 8.3: Clear grapple state if switching targets
	if c.Aggro != nil {
		if c.Aggro.UserId != userId || c.Aggro.MobInstanceId != mobInstanceId {
			c.ClearGrappleState()
		}
	}

	var combatAddlWaitRounds int = 0

	if len(roundsWaitTime) > 0 {
		for _, waitAmt := range roundsWaitTime {
			combatAddlWaitRounds += waitAmt
		}
	} else {
		combatAddlWaitRounds = c.Equipment.Weapon.GetSpec().WaitRounds + c.Equipment.Offhand.GetSpec().WaitRounds
	}

	if aggroType == DefaultAttack {
		if c.Equipment.Weapon.GetSpec().Subtype == items.Shooting {
			aggroType = Shooting
		}
	}

	c.Aggro = &Aggro{
		UserId:        userId,
		MobInstanceId: mobInstanceId,
		Type:          aggroType,
		RoundsWaiting: combatAddlWaitRounds,
	}

	// Task 15: Dual-write — also transition Combat Phase so new
	// readers (IsInCombat, CurrentCombatTarget, etc.) stay correct.
	// Errors are intentionally ignored: vetoes may legitimately reject
	// the transition (non-combatant, dead target, etc.) while the
	// legacy Aggro write has already done its job.
	// TODO Task 18: remove legacy Aggro write once CombatPhase is sole
	// source of truth.
	if c.CombatPhase != nil {
		trigger := combatphase.TriggerAttackCommand
		if aggroType == SurpriseAttack {
			trigger = combatphase.TriggerSurpriseAttack
		}
		_ = c.CombatPhase.TransitionToEngaging(combatphase.EngagingData{
			Target: state.ActorRef{
				UserId:        userId,
				MobInstanceId: mobInstanceId,
			},
			RoundsUntil: combatAddlWaitRounds,
		}, state.TransitionReason{
			Trigger: trigger,
			Actor:   state.ActorRef{UserId: c.userId},
			Target:  state.ActorRef{UserId: userId, MobInstanceId: mobInstanceId},
		})
	}
}

func (c *Character) EndAggro() {
	c.Aggro = nil
	c.ClearGrappleState()
	// Task 15: Dual-write — also clear Combat Phase so new readers stay correct.
	// TODO Task 18: this becomes the sole cleanup path once Aggro is removed.
	if c.CombatPhase != nil && c.CombatPhase.IsInCombat() {
		c.CombatPhase.ForceIdle(state.TransitionReason{
			Trigger: combatphase.TriggerForceIdle,
		})
	}
}

// ClearGrappleState clears all grapple-related state
// Stage 8.3: Called when combat ends, targets change, or participant dies
func (c *Character) ClearGrappleState() {
	c.GrappleControllerId = 0
	c.RemoveCondition(ConditionGrappleController)
	// Reset to standing if in a grapple position
	if c.CombatPosition.IsGrapplePosition() {
		c.CombatPosition = PositionStanding
	}
}

func (c *Character) IsAggro(targetUserId int, targetMobInstanceId int) bool {

	if c.Aggro != nil {

		if c.Aggro.MobInstanceId > 0 && c.Aggro.MobInstanceId == targetMobInstanceId {
			return true
		}

		if c.Aggro.UserId > 0 && c.Aggro.UserId == targetUserId {
			return true
		}

		if c.Aggro.Type == SpellCast {
			if len(c.Aggro.SpellInfo.TargetUserIds) > 0 {
				for _, uId := range c.Aggro.SpellInfo.TargetUserIds {
					if uId == targetUserId {
						return true
					}
				}
			}

			if len(c.Aggro.SpellInfo.TargetMobInstanceIds) > 0 {
				for _, mId := range c.Aggro.SpellInfo.TargetMobInstanceIds {
					if mId == targetMobInstanceId {
						return true
					}
				}
			}
		}

	}
	return false
}
