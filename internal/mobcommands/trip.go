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

func Trip(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use trip
	if mob.Character.Aggro == nil {
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
	// Attacker: Unarmed Combat skill + Dexterity (agility for tripping)
	// Defender: Dexterity + combat skill
	attackerScore := float64(mob.Character.GetSkillLevel(skills.UnarmedCombat)) + float64(mob.Character.Stats.Dexterity.ValueAdj)

	var defenderScore float64

	if targetMob != nil {
		defenderScore = float64(targetMob.Character.GetCombatSkillLevel()) + float64(targetMob.Character.Stats.Dexterity.ValueAdj)
	} else {
		defenderScore = float64(targetChar.Character.GetCombatSkillLevel()) + float64(targetChar.Character.Stats.Dexterity.ValueAdj)
	}

	// Perform opposed roll
	attackSuccess, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	// Calculate damage via unified pipeline (low damage - primarily a setup move)
	skillRank := mob.Character.GetSkillLevel(skills.UnarmedCombat)
	rawDmg := combat.CalcRawDamage(mob.Character.Stats.Strength.ValueAdj, skillRank, float64(cfg.TripDamagePercent), combat.ChannelPhysical)

	// Apply target's physical mitigation
	var targetMitig float64
	if targetMob != nil {
		targetMitig = targetMob.Character.GetPhysicalMitigation()
	} else {
		targetMitig = targetChar.Character.GetPhysicalMitigation()
	}
	dmgMean := combat.ApplyMitigation(rawDmg, targetMitig, combat.MitigationCap(combat.ChannelPhysical))
	dmgRoll := dice.RollStat(dmgMean)
	baseDamage := int(dmgRoll.Value)
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
		// Roll for knockdown chance (higher than bash since it's the primary purpose)
		knockdownRoll := dice.RollStat(50) // Mean of 50
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
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> sweeps your legs, sending you crashing to the ground! (<ansi fg="damage">%s</ansi> damage)`, mob.Character.Name, combat.GetDamageDescription(baseDamage, targetMaxHP)))
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> trips <ansi fg="username">%s</ansi>, sending them crashing to the ground!`, mob.Character.Name, targetName),
				targetPlayerId,
			)
		} else {
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to trip you, but you keep your footing! (<ansi fg="damage">%s</ansi> damage)`, mob.Character.Name, combat.GetDamageDescription(baseDamage, targetMaxHP)))
			}

			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to trip <ansi fg="username">%s</ansi>, but they keep their footing!`, mob.Character.Name, targetName),
				targetPlayerId,
			)
		}
	} else {
		// Attack missed
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to trip you, but you avoid it!`, mob.Character.Name))
		}

		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to trip <ansi fg="username">%s</ansi>, but misses!`, mob.Character.Name, targetName),
			targetPlayerId,
		)
	}

	// Stage 30.1: Record combat analytics
	tripDmgRecorded := 0
	if attackSuccess {
		tripDmgRecorded = baseDamage
	}
	tripTgtType := combat.Mob
	var tripTgtChar *characters.Character
	if targetMob != nil {
		tripTgtChar = &targetMob.Character
	} else {
		tripTgtType = combat.User
		tripTgtChar = targetChar.Character
	}
	combat.RecordSpecialMove(combat.Mob, tripTgtType, "trip", attackSuccess, tripDmgRecorded, &mob.Character, tripTgtChar, util.GetRoundCount())

	// Trip costs the current combat round
	if mob.Character.Aggro != nil {
		mob.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
