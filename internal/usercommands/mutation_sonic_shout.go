package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func SonicShout(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !mutations.HasMutation(user.Character.Mutations, "sonic-shout") {
		user.SendText(messaging.CategorySystem, "You don't have that ability.")
		return true, nil
	}

	if !user.Character.IsInCombat() {
		user.SendText(messaging.CategorySystem, "You must be in combat to use sonic shout!")
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText(messaging.CategorySystem, "You need a moment to recover before attempting another special move.")
		return true, nil
	}

	// Stamina cost
	staminaCost := 15
	if user.Character.Stamina < staminaCost {
		user.SendText(messaging.CategorySystem, "You're too exhausted to muster a sonic shout!")
		return true, nil
	}
	user.Character.Stamina -= staminaCost

	// AoE attack: hit all mobs in the room
	attackerScore := float64(user.Character.GetSkillLevel(skills.UnarmedCombat)) + float64(user.Character.Stats.Willpower.ValueAdj)
	baseDamage := int(float64(user.Character.Stats.Willpower.ValueAdj) * 0.08)
	if baseDamage < 1 {
		baseDamage = 1
	}

	user.SendText(messaging.CategorySystem, `<ansi fg="magenta-bold">You unleash a devastating sonic shout that reverberates through the room!</ansi>`)
	room.SendTextVisual(messaging.CategoryMutation, 
		fmt.Sprintf(`<ansi fg="magenta-bold"><ansi fg="username">%s</ansi> unleashes a devastating sonic shout!</ansi>`, user.Character.Name),
		user.UserId,
	)

	for _, mobInstId := range room.GetMobs(rooms.FindAll) {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil {
			continue
		}

		defenderScore := float64(mob.Character.Stats.Willpower.ValueAdj) + float64(mob.Character.GetCombatSkillLevel())
		attackSuccess, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

		if attackSuccess {
			mob.Character.Health -= baseDamage
			if mob.Character.Health < 1 {
				mob.Character.Health = 0
			}
			// Knockdown
			knockdownRoll := dice.RollStat(50)
			if knockdownRoll.Value < 40 {
				_ = mob.Character.Position.TransitionToProne(
					position.ProneData{MinRecoveryRounds: 2},
					state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward},
				)
			}
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`The shout staggers <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`,
				mob.Character.Name, combat.GetDamageDescription(baseDamage, mob.Character.HealthMax.Value)))
		}
		// Stage 30.1: Record combat analytics per target
		shoutDmg := 0
		if attackSuccess {
			shoutDmg = baseDamage
		}
		combat.RecordSpecialMove(combat.User, combat.Mob, "sonic_shout", attackSuccess, shoutDmg, user.Character, &mob.Character, util.GetRoundCount())
	}

	// Self-deafen: reduced perception for 3 rounds
	user.Character.AddCondition(characters.ConditionBlinded, 3, 0.7, "sonic-shout self-deafen")
	user.SendText(messaging.CategorySystem, `<ansi fg="yellow">The echoes leave your own senses ringing!</ansi>`)

	events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: skills.UnarmedCombat, Details: "sonic-shout"})

	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
