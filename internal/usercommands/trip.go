package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Trip(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat to use trip
	if user.Character.Aggro == nil {
		user.SendText("You must be in combat to trip someone!")
		return true, nil
	}

	// Check shared special move cooldown
	cfg := configs.GetGamePlayConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		remaining := user.Character.Cooldowns["special-move"]
		roundWord := "round"
		if remaining > 1 {
			roundWord = "rounds"
		}
		user.SendText(fmt.Sprintf("You can't use another special move yet! (%d %s remaining)", remaining, roundWord))
		return true, nil
	}

	// Get current target from aggro
	targetPlayerId := user.Character.Aggro.UserId
	targetMobId := user.Character.Aggro.MobInstanceId

	var targetChar *users.UserRecord
	var targetMob *mobs.Mob
	var targetName string

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetMob.Character.Name
	} else if targetPlayerId > 0 {
		targetChar = users.GetByUserId(targetPlayerId)
		if targetChar == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetChar.Character.Name
	} else {
		user.SendText("You have no target!")
		return true, nil
	}

	// Calculate knockdown chance using opposed roll
	// Attacker: Unarmed Combat skill + Dexterity (agility for tripping)
	// Defender: Dexterity + combat skill
	attackerScore := float64(user.Character.GetSkillLevel(skills.UnarmedCombat)) + float64(user.Character.Stats.Dexterity.ValueAdj)

	var defenderScore float64

	if targetMob != nil {
		defenderScore = float64(targetMob.Character.GetCombatSkillLevel()) + float64(targetMob.Character.Stats.Dexterity.ValueAdj)
	} else {
		defenderScore = float64(targetChar.Character.GetCombatSkillLevel()) + float64(targetChar.Character.Stats.Dexterity.ValueAdj)
	}

	// Perform opposed roll
	attackSuccess, _, _, _ := dice.OpposedRoll(attackerScore, defenderScore, 15.0)

	// Calculate damage (low damage - primarily a setup move)
	baseDamage := int(float64(user.Character.Stats.Strength.ValueAdj) * float64(cfg.TripDamagePercent))
	if baseDamage < 1 {
		baseDamage = 1
	}

	// Apply damage and determine knockdown
	knockedDown := false
	if attackSuccess {
		// Roll for knockdown chance (higher than bash since it's the primary purpose)
		knockdownRoll := dice.Roll(50, 15.0) // Mean of 50
		if knockdownRoll.Value < float64(cfg.TripKnockdownChance) {
			knockedDown = true
		}

		// Apply damage
		if targetMob != nil {
			targetMob.Character.Health -= baseDamage
			if targetMob.Character.Health < 1 {
				targetMob.Character.Health = 0
			}
			if knockedDown {
				targetMob.Character.CombatPosition = characters.PositionProne
				targetMob.Character.PositionRoundsMin = 2 // Guarantees 1 full round prone
			}
		} else if targetChar != nil {
			targetChar.Character.Health -= baseDamage
			if targetChar.Character.Health < 1 {
				targetChar.Character.Health = 0
			}
			if knockedDown {
				targetChar.Character.CombatPosition = characters.PositionProne
				targetChar.Character.PositionRoundsMin = 2 // Guarantees 1 full round prone
			}
		}

		// Send messages
		if knockedDown {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> sends <ansi fg="mobname">%s</ansi> crashing to the ground! (<ansi fg="damage">%d</ansi> damage)`, targetName, baseDamage))

			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> sweeps your legs, sending you crashing to the ground! (<ansi fg="damage">%d</ansi> damage)`, user.Character.Name, baseDamage))
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> trips <ansi fg="mobname">%s</ansi>, sending them crashing to the ground!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		} else {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> strikes <ansi fg="mobname">%s</ansi>, but they stay on their feet! (<ansi fg="damage">%d</ansi> damage)`, targetName, baseDamage))

			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip you, but you keep your footing! (<ansi fg="damage">%d</ansi> damage)`, user.Character.Name, baseDamage))
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip <ansi fg="mobname">%s</ansi>, but they keep their footing!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		}
	} else {
		// Attack missed
		user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> attempt misses <ansi fg="mobname">%s</ansi>!`, targetName))

		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip you, but you avoid it!`, user.Character.Name))
		}

		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, targetName),
			user.UserId, targetPlayerId,
		)
	}

	// Progress unarmed combat skill
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.UnarmedCombat,
		Details: "trip",
	})

	// Trip costs the current combat round
	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
