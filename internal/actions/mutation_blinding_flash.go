package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

// MutationOpts parameterizes a mutation-active trigger attempt.
// TargetActor is used by single-target mutations (toxic-bite, blinding-spit).
// AoE and self-only mutations ignore it.
type MutationOpts struct {
	TargetActor Actor
}

// MutationResult is the structured outcome of a mutation-active trigger.
// On Triggered=false, BlockReason is one of:
//
//	"no-character"  - actor.GetCharacter() returned nil
//	"no-mutation"   - actor does not own the requested mutation
//	"not-in-combat" - combatRequired=true but actor is not in combat
//	"on-cooldown"   - the shared special-move cooldown is still active
//	"low-stamina"   - actor does not have enough stamina
//	"no-room"       - actor.GetRoom() returned nil
//
// On Triggered=true, AffectedCount counts targets that received the effect
// (0+ for AoE, 1 for self/single-target effects that hit).
type MutationResult struct {
	Triggered     bool
	BlockReason   string
	AffectedCount int
}

// TriggerBlindingFlash fires the blinding-flash mutation for any Actor (player
// or mob). AoE: applies ConditionBlinded (duration 3, magnitude 0.5) to each
// non-attacker mob in the room that fails an opposed Wil+UnarmedCombat vs
// Wil+CombatSkill roll. The attacker always takes a milder self-blind (duration
// 1, magnitude 0.7) afterward.
//
// Gates: must have "blinding-flash" mutation + be in combat + pass the shared
// special-move cooldown + have at least 14 stamina. Stamina is only deducted on
// full success.
//
// Player actors receive descriptive messages; mob actors are silent (MobActor
// SendText is a no-op). The original numeric count message ("N creature(s) are
// blinded") is replaced with a descriptive variant per the no-hard-numbers rule.
func TriggerBlindingFlash(actor Actor, opts MutationOpts) MutationResult {
	pre := mutationPreamble(actor, "blinding-flash", true, 14)
	if !pre.OK {
		return MutationResult{BlockReason: pre.BlockReason}
	}

	room := actor.GetRoom()
	if room == nil {
		return MutationResult{BlockReason: "no-room"}
	}

	char := actor.GetCharacter()
	attackerScore := float64(char.GetSkillLevel(skills.UnarmedCombat)) +
		float64(char.Stats.Willpower.ValueAdj)

	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`<ansi fg="white-bold">Blinding light erupts from your skin in a searing flash!</ansi>`)
	}
	room.SendTextVisual(
		messaging.CategoryMutation,
		fmt.Sprintf(
			`<ansi fg="white-bold">A blinding flash of light erupts from <ansi fg="username">%s</ansi>!</ansi>`,
			actor.GetName(),
		),
		actor.GetUserId(),
	)

	// AoE blind: roll against every mob in the room except the attacker.
	blindedCount := 0
	for _, mobInstId := range room.GetMobs(rooms.FindAll) {
		if mobInstId == actor.GetMobInstanceId() {
			continue
		}
		victim := mobs.GetInstance(mobInstId)
		if victim == nil {
			continue
		}
		defenderScore := float64(victim.Character.Stats.Willpower.ValueAdj) +
			float64(victim.Character.GetCombatSkillLevel())
		attackSuccess, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)
		if attackSuccess {
			victim.Character.AddCondition(characters.ConditionBlinded, 3, 0.5, "blinding-flash")
			blindedCount++
		}
	}

	if blindedCount > 0 && actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`<ansi fg="white">Creatures in the area are blinded by the flash!</ansi>`)
	}

	// Attacker takes a shorter self-blind as the afterimage sears their vision.
	char.AddCondition(characters.ConditionBlinded, 1, 0.7, "blinding-flash self")
	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			`<ansi fg="yellow">The afterimage sears your own vision briefly.</ansi>`)
	}

	actor.OnSkillUse(string(skills.UnarmedCombat))

	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	return MutationResult{Triggered: true, AffectedCount: blindedCount}
}
