package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Kick(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat to use kick
	if user.Character.Aggro == nil {
		user.SendText("You must be in combat to kick!")
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
	// Attacker: Unarmed Combat skill + Strength (power kick)
	// Defender: Dexterity + combat skill
	attackerScore := float64(user.Character.GetSkillLevel(skills.UnarmedCombat)) + float64(user.Character.Stats.Strength.ValueAdj)

	var defenderScore float64

	if targetMob != nil {
		defenderScore = float64(targetMob.Character.GetCombatSkillLevel()) + float64(targetMob.Character.Stats.Dexterity.ValueAdj)
	} else {
		defenderScore = float64(targetChar.Character.GetCombatSkillLevel()) + float64(targetChar.Character.Stats.Dexterity.ValueAdj)
	}

	// Perform opposed roll
	attackSuccess, _, _, _ := dice.OpposedRoll(attackerScore, defenderScore, 15.0)

	// Calculate damage (moderate - more than trip, less than bash)
	baseDamage := int(float64(user.Character.Stats.Strength.ValueAdj) * float64(cfg.KickDamagePercent))
	if baseDamage < 1 {
		baseDamage = 1
	}

	// Get target's max HP for damage description
	targetMaxHP := 0
	if targetMob != nil {
		targetMaxHP = targetMob.Character.HealthMax.Value
	} else if targetChar != nil {
		targetMaxHP = targetChar.Character.HealthMax.Value
	}

	// Apply damage and determine knockdown
	knockedDown := false
	if attackSuccess {
		// Roll for knockdown chance (moderate - between trip and bash)
		knockdownRoll := dice.Roll(50, 15.0) // Mean of 50
		if knockdownRoll.Value < float64(cfg.KickKnockdownChance) {
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
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">kick</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(baseDamage, targetMaxHP)))

			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s powerful <ansi fg="yellow-bold">kick</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(baseDamage, targetMaxHP)))
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> kicks <ansi fg="mobname">%s</ansi>, knocking them to the ground!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		} else {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">kick</ansi> strikes <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(baseDamage, targetMaxHP)))

			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> kicks you hard! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(baseDamage, targetMaxHP)))
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> kicks <ansi fg="mobname">%s</ansi>!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		}
	} else {
		// Attack missed
		user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">kick</ansi> misses <ansi fg="mobname">%s</ansi>!`, targetName))

		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to kick you, but misses!`, user.Character.Name))
		}

		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to kick <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, targetName),
			user.UserId, targetPlayerId,
		)
	}

	// Progress unarmed combat skill
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.UnarmedCombat,
		Details: "kick",
	})

	// Kick costs the current combat round
	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
