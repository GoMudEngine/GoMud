package actions

import (
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// SneakResult holds the outcome of a Sneak call.
type SneakResult struct {
	// Success is true when the actor entered the hidden state.
	Success bool
	// SpottedByName is the name of the observer who detected the actor.
	// Empty when Success is true.
	SpottedByName string
	// AlreadyHidden is true when the actor already has the hidden buff.
	AlreadyHidden bool
	// InCombat is true when the actor could not attempt the action because
	// they were engaged in combat.
	InCombat bool
	// RollHappened is true when at least one opposed roll was made against an
	// observer. Callers use this to gate skill progression — no roll means no
	// practice (room was empty).
	RollHappened bool
}

// Sneak attempts to put actor into the hidden (sneaking) state.
//
// It rolls the actor's sneak score against every observer in the room. Party
// members of a player actor are excluded from the observer checks. If any
// observer wins the opposed roll the attempt fails and SpottedByName is set.
//
// On success the hidden buff (id 9) is applied via the event queue and the
// "sneaking" misc-data key is set immediately so other systems can react
// before the buff processes on the next tick.
//
// Callers are responsible for:
//   - Skill-gate checks (player side only)
//   - Failure cooldowns (player side only)
//   - Skill progression and quest engine notifications
//   - Player-facing messaging
func Sneak(actor Actor) SneakResult {
	char := actor.GetCharacter()

	// Already hidden — nothing to do.
	if char.IsHidden() {
		return SneakResult{AlreadyHidden: true}
	}

	// Can't hide while fighting.
	if char.Aggro != nil {
		return SneakResult{InCombat: true}
	}

	// Transition to Concealing; the activity veto fires here if the actor
	// is busy crafting/casting. On veto error we return without a result
	// since the caller's skill-gate messaging handles the veto path.
	if err := char.Awareness.TransitionToConcealing(
		awareness.ConcealingData{},
		state.TransitionReason{Trigger: awareness.TriggerSneakCommand},
	); err != nil {
		// Activity veto or invalid transition (e.g. already Concealing).
		return SneakResult{}
	}

	room := actor.GetRoom()

	// Build party exclusion set for player actors.
	// Mob actors have no party (GetUserId returns 0); parties.Get(0) returns nil.
	partySet := map[int]bool{}
	if uid := actor.GetUserId(); uid > 0 {
		partySet[uid] = true
		if party := parties.Get(uid); party != nil {
			for _, memberId := range party.GetMembers() {
				partySet[memberId] = true
			}
		}
	}

	// Exclude the mob actor itself from the observer lists.
	selfMobId := actor.GetMobInstanceId()

	rollHappened := false

	// Check each player in the room. Score is computed per-observer so
	// that NightVision observers (effectiveLit=true) apply the appropriate
	// light modifier even in a dark room.
	for _, observerId := range room.GetPlayers() {
		if partySet[observerId] {
			continue
		}
		observer := users.GetByUserId(observerId)
		if observer == nil {
			continue
		}
		sneakScore := CalcSneakScoreVsObserver(char, observer.Character, room)
		observerScore := CalcSearchScore(observer.Character)
		rollHappened = true
		success, _, _, _ := dice.OpposedRollStat(sneakScore, observerScore)
		if !success {
			// Notify the observing player.
			observer.SendText(messaging.CategorySystem,
				`<ansi fg="username">`+actor.GetName()+`</ansi> tries to hide but you notice them.`,
			)
			char.Awareness.ResolveConcealment(false, state.TransitionReason{
				Trigger: awareness.TriggerSneakFailed,
			})
			return SneakResult{SpottedByName: observer.Character.Name, RollHappened: true}
		}
	}

	// Check each mob in the room.
	for _, mobInstanceId := range room.GetMobs() {
		if mobInstanceId == selfMobId {
			continue // don't roll against yourself
		}
		m := mobs.GetInstance(mobInstanceId)
		if m == nil {
			continue
		}
		sneakScore := CalcSneakScoreVsObserver(char, &m.Character, room)
		observerScore := CalcSearchScore(&m.Character)
		rollHappened = true
		success, _, _, _ := dice.OpposedRollStat(sneakScore, observerScore)
		if !success {
			char.Awareness.ResolveConcealment(false, state.TransitionReason{
				Trigger: awareness.TriggerSneakFailed,
			})
			return SneakResult{SpottedByName: m.Character.Name, RollHappened: true}
		}
	}

	// All observers failed to spot the actor — transition to Hidden.
	// The buff-#9 mirror cascade in Awareness_Cascades.go handles AddBuff automatically.
	char.Awareness.ResolveConcealment(true, state.TransitionReason{
		Trigger: awareness.TriggerSneakSuccess,
	})
	char.SetMiscData(`sneaking`, true)

	return SneakResult{Success: true, RollHappened: rollHappened}
}
