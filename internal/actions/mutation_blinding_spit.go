package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// TriggerBlindingSpit fires the blinding-spit mutation for any Actor (player
// or mob). Single-target: applies ConditionBlinded (duration 3, magnitude 0.5)
// to opts.TargetActor on a successful opposed Dex+UnarmedCombat vs Dex+CombatSkill
// roll.
//
// Gates: must have "blinding-spit" mutation + be in combat + pass the shared
// special-move cooldown + have at least 10 stamina. Stamina is deducted by the
// preamble on full success; no-target is therefore checked after the preamble
// (matching the original player command ordering where target resolution follows
// the stamina deduction).
//
// Player actors receive descriptive messages; mob actors are silent (MobActor
// SendText is a no-op). When the target is a player, the victim also receives
// a first-person message via users.GetByUserId.
func TriggerBlindingSpit(actor Actor, opts MutationOpts) MutationResult {
	pre := mutationPreamble(actor, "blinding-spit", true, 10)
	if !pre.OK {
		return MutationResult{BlockReason: pre.BlockReason}
	}

	// Target resolution follows the preamble (matching the original player
	// command ordering). Stamina has already been deducted; no refund on
	// no-target (mirrors original behavior).
	if opts.TargetActor == nil || opts.TargetActor.GetCharacter() == nil {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, "You have no target!")
		}
		return MutationResult{BlockReason: "no-target"}
	}

	char := actor.GetCharacter()
	target := opts.TargetActor.GetCharacter()
	targetName := opts.TargetActor.GetName()

	attackerScore := float64(char.GetSkillLevel(skills.UnarmedCombat)) +
		float64(char.Stats.Dexterity.ValueAdj)
	defenderScore := float64(target.Stats.Dexterity.ValueAdj) +
		float64(target.GetCombatSkillLevel())

	attackSuccess, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	room := actor.GetRoom()

	if attackSuccess {
		target.AddCondition(characters.ConditionBlinded, 3, 0.5, "blinding-spit")

		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf(
					`<ansi fg="yellow-bold">You spit a stream of caustic fluid into <ansi fg="mobname">%s</ansi>'s eyes!</ansi>`,
					targetName,
				),
			)
		}

		// Notify the victim if they are a player.
		targetUserId := opts.TargetActor.GetUserId()
		if targetUserId > 0 {
			if tUser := users.GetByUserId(targetUserId); tUser != nil {
				tUser.SendText(messaging.CategorySystem,
					fmt.Sprintf(
						`<ansi fg="red"><ansi fg="username">%s</ansi> spits caustic fluid into your eyes! You can barely see!</ansi>`,
						actor.GetName(),
					),
				)
			}
		}

		if room != nil {
			room.SendTextVisual(
				messaging.CategoryMutation,
				fmt.Sprintf(
					`<ansi fg="username">%s</ansi> spits a stream of fluid into <ansi fg="mobname">%s</ansi>'s eyes!`,
					actor.GetName(), targetName,
				),
				actor.GetUserId(), targetUserId,
			)
		}

		actor.OnSkillUse(string(skills.UnarmedCombat))

		if char.Aggro != nil {
			char.Aggro.RoundsWaiting = 1
		}

		return MutationResult{Triggered: true, AffectedCount: 1}
	}

	// Miss path.
	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem,
			fmt.Sprintf(
				`<ansi fg="red">Your blinding spit misses <ansi fg="mobname">%s</ansi>!</ansi>`,
				targetName,
			),
		)
	}

	targetUserId := opts.TargetActor.GetUserId()
	if targetUserId > 0 {
		if tUser := users.GetByUserId(targetUserId); tUser != nil {
			tUser.SendText(messaging.CategorySystem,
				fmt.Sprintf(
					`<ansi fg="username">%s</ansi> spits at you, but you dodge the caustic stream!`,
					actor.GetName(),
				),
			)
		}
	}

	if room != nil {
		room.SendTextVisual(
			messaging.CategoryMutation,
			fmt.Sprintf(
				`<ansi fg="username">%s</ansi> spits at <ansi fg="mobname">%s</ansi>, but misses!`,
				actor.GetName(), targetName,
			),
			actor.GetUserId(), targetUserId,
		)
	}

	actor.OnSkillUse(string(skills.UnarmedCombat))

	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	return MutationResult{Triggered: true, AffectedCount: 0}
}
