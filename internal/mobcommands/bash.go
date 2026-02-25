package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Bash(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use bash
	if mob.Character.Aggro == nil {
		return true, nil
	}

	// Must have a shield equipped
	if !mob.Character.HasShield() {
		return true, nil
	}

	// Check shared special move cooldown
	cfg := configs.GetGamePlayConfig()
	if !mob.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return true, nil
	}

	// Get current target from aggro
	targetPlayerId := mob.Character.Aggro.UserId
	targetMobId := mob.Character.Aggro.MobInstanceId

	var targetChar *users.UserRecord
	var targetMob *mobs.Mob
	var targetName string

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			return true, nil
		}
		targetName = targetMob.Character.Name
	} else if targetPlayerId > 0 {
		targetChar = users.GetByUserId(targetPlayerId)
		if targetChar == nil {
			return true, nil
		}
		targetName = targetChar.Character.Name
	} else {
		return true, nil
	}

	// Calculate knockdown chance using opposed roll
	// Attacker: Weapon Combat skill + Strength
	// Defender: Dexterity + combat skill
	attackerScore := float64(mob.Character.GetSkillLevel(skills.WeaponCombat)) + float64(mob.Character.Stats.Strength.ValueAdj)

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
	if mob.Character.Equipment.Weapon.ItemId > 0 {
		weapon := mob.Character.Equipment.Weapon
		attacks, dmg, variance, _ := weapon.GetDistributionDamage()
		if attacks > 0 {
			baseDamage = int(float64(dmg) * float64(cfg.BashDamagePercent))
			// Add some variance
			baseDamage += int(dice.Roll(0, variance).Value)
		}
	} else {
		// Unarmed bash (less damage)
		baseDamage = int(float64(mob.Character.Stats.Strength.ValueAdj) * float64(cfg.BashDamagePercent))
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
		// Roll for knockdown chance
		knockdownRoll := dice.RollStat(50) // Mean of 50
		if knockdownRoll.Value < float64(cfg.BashKnockdownChance) {
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
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi> damage)`, mob.Character.Name, combat.GetDamageDescription(baseDamage, targetMaxHP)))
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="username">%s</ansi> to the ground!`, mob.Character.Name, targetName),
				targetPlayerId,
			)
		} else {
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> strikes you! (<ansi fg="damage">%s</ansi> damage)`, mob.Character.Name, combat.GetDamageDescription(baseDamage, targetMaxHP)))
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> bashes <ansi fg="username">%s</ansi> with their shield!`, mob.Character.Name, targetName),
				targetPlayerId,
			)
		}
	} else {
		// Attack missed
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to bash you with their shield, but misses!`, mob.Character.Name))
		}

		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to bash <ansi fg="username">%s</ansi>, but misses!`, mob.Character.Name, targetName),
			targetPlayerId,
		)
	}

	// Stage 30.1: Record combat analytics
	bashDmgRecorded := 0
	if attackSuccess {
		bashDmgRecorded = baseDamage
	}
	bashTgtType := combat.Mob
	var bashTgtChar *characters.Character
	if targetMob != nil {
		bashTgtChar = &targetMob.Character
	} else {
		bashTgtType = combat.User
		bashTgtChar = targetChar.Character
	}
	combat.RecordSpecialMove(combat.Mob, bashTgtType, "bash", attackSuccess, bashDmgRecorded, &mob.Character, bashTgtChar, util.GetRoundCount())

	// Bash costs the current combat round
	if mob.Character.Aggro != nil {
		mob.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
