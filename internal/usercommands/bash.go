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
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Bash(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat to use bash
	if user.Character.Aggro == nil {
		user.SendText("You must be in combat to use shield bash!")
		return true, nil
	}

	// Must have a shield equipped
	if !user.Character.HasShield() {
		user.SendText("You need a shield equipped to perform a shield bash!")
		return true, nil
	}

	// Check shared special move cooldown
	cfg := configs.GetGamePlayConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
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
	// Attacker: Weapon Combat skill + Strength
	// Defender: Dexterity + combat skill
	attackerScore := float64(user.Character.GetSkillLevel(skills.WeaponCombat)) + float64(user.Character.Stats.Strength.ValueAdj)

	var defenderScore float64

	if targetMob != nil {
		defenderScore = float64(targetMob.Character.GetCombatSkillLevel()) + float64(targetMob.Character.Stats.Dexterity.ValueAdj)
	} else {
		defenderScore = float64(targetChar.Character.GetCombatSkillLevel()) + float64(targetChar.Character.Stats.Dexterity.ValueAdj)
	}

	// Perform opposed roll
	attackSuccess, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	// Calculate damage (percentage of normal attack damage)
	baseDamage := 0
	if user.Character.Equipment.Weapon.ItemId > 0 {
		weapon := user.Character.Equipment.Weapon
		attacks, dmg, variance, _ := weapon.GetDistributionDamage()
		if attacks > 0 {
			baseDamage = int(float64(dmg) * float64(cfg.BashDamagePercent))
			// Add some variance
			baseDamage += int(dice.Roll(0, variance).Value)
		}
	} else {
		// Unarmed bash (less damage)
		baseDamage = int(float64(user.Character.Stats.Strength.ValueAdj) * float64(cfg.BashDamagePercent))
	}

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
		// Roll for knockdown chance (flat percentage roll)
		if util.Rand(100) < int(cfg.BashKnockdownChance) {
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
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(baseDamage, targetMaxHP)))

			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(baseDamage, targetMaxHP)))
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		} else {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">shield bash</ansi> strikes <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(baseDamage, targetMaxHP)))

			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> strikes you! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(baseDamage, targetMaxHP)))
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> bashes <ansi fg="mobname">%s</ansi> with their shield!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		}
	} else {
		// Attack missed
		user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">shield bash</ansi> misses <ansi fg="mobname">%s</ansi>!`, targetName))

		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to bash you with their shield, but misses!`, user.Character.Name))
		}

		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to bash <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, targetName),
			user.UserId, targetPlayerId,
		)
	}

	// Stage 30.1: Record combat analytics
	dmgRecorded := 0
	if attackSuccess {
		dmgRecorded = baseDamage
	}
	tgtType := combat.Mob
	var tgtCharPtr *characters.Character
	if targetMob != nil {
		tgtCharPtr = &targetMob.Character
	} else {
		tgtType = combat.User
		tgtCharPtr = targetChar.Character
	}
	combat.RecordSpecialMove(combat.User, tgtType, "bash", attackSuccess, dmgRecorded, user.Character, tgtCharPtr, util.GetRoundCount())

	// Progress weapon combat skill
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.WeaponCombat,
		Details: "bash",
	})

	// Bash costs the current combat round
	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
