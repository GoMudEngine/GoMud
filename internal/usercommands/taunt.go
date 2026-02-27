package usercommands

import (
	"fmt"
	"math"

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

func Taunt(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat to use taunt
	if user.Character.Aggro == nil {
		user.SendText("You must be in combat to taunt!")
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

	var targetPlayer *users.UserRecord
	var targetMob *mobs.Mob
	var targetChar *characters.Character
	var targetName string
	var targetType string

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetChar = &targetMob.Character
		targetName = targetMob.Character.Name
		targetType = "mob"
	} else if targetPlayerId > 0 {
		targetPlayer = users.GetByUserId(targetPlayerId)
		if targetPlayer == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetChar = targetPlayer.Character
		targetName = targetPlayer.Character.Name
		targetType = "user"
	} else {
		user.SendText("You have no target!")
		return true, nil
	}

	sourceName := user.Character.Name

	// Conviction attack: Charisma + rhetoric vs target Willpower + rhetoric
	attackerRhetoric := float64(user.Character.GetSkillLevel(skills.Rhetoric))
	attackScore := float64(user.Character.Stats.Charisma.ValueAdj) + attackerRhetoric

	defenderRhetoric := float64(targetChar.GetSkillLevel(skills.Rhetoric))
	defenseScore := float64(targetChar.Stats.Willpower.ValueAdj) + defenderRhetoric

	// Apply smooth conviction-based penalty to taunt hit chance
	cpPenalty := float64(configs.GetBalanceConfig().ConvictionPenaltyMax)
	convMult := combat.ResourceMultiplier(user.Character.Conviction,
		user.Character.ConvictionMax.Value, cpPenalty)
	attackScore *= convMult

	// Opposed roll for hit check
	attackSuccess, _, atkRoll, _ := dice.OpposedRollStat(attackScore, defenseScore)

	// Check for fumble (embarrassing taunt backfire)
	if atkRoll.ZScore <= -2.0 {
		sendTauntMessages(combat.TauntFumble, "", sourceName, targetName,
			"username", targetType, user, targetPlayer, room, targetPlayerId)

		// Fumble: small self-conviction damage
		selfDmg := user.Character.Stats.Charisma.ValueAdj / 10
		if selfDmg < 1 {
			selfDmg = 1
		}
		user.Character.Conviction -= selfDmg
		if user.Character.Conviction < 0 {
			user.Character.Conviction = 0
		}

		combat.RecordSpecialMove(combat.User, combat.Mob, "taunt", false, 0,
			user.Character, targetChar, util.GetRoundCount())

		// Still costs the round
		if user.Character.Aggro != nil {
			user.Character.Aggro.RoundsWaiting = 1
		}
		return true, nil
	}

	if attackSuccess {
		// Calculate conviction damage via unified pipeline
		rawDmg := combat.CalcRawDamage(
			user.Character.Stats.Charisma.ValueAdj,
			int(attackerRhetoric),
			0.5, // taunt base multiplier
			combat.ChannelConviction,
		)

		// Apply smooth conviction-based damage penalty
		rawDmg *= convMult

		// Crit check
		isCrit := atkRoll.ZScore >= 2.0

		var dmg int
		if isCrit {
			// Crit: bypass mitigation entirely — use raw damage
			dmgRoll := dice.RollStat(rawDmg)
			dmg = int(math.Round(dmgRoll.Value))
		} else {
			// Normal hit: apply target's conviction mitigation
			mitigPct := targetChar.GetConvictionMitigation()
			cap := combat.MitigationCap(combat.ChannelConviction)
			dmgMean := combat.ApplyMitigation(rawDmg, mitigPct, cap)
			dmgRoll := dice.RollStat(dmgMean)
			dmg = int(math.Round(dmgRoll.Value))
		}
		if dmg < 1 {
			dmg = 1
		}

		// Apply conviction damage
		targetChar.Conviction -= dmg
		if targetChar.Conviction < 0 {
			targetChar.Conviction = 0
		}

		// Get damage description
		convMaxRef := targetChar.ConvictionMax.Value
		if convMaxRef <= 0 {
			convMaxRef = 1
		}
		dmgDesc := combat.GetConvictionDamageDescription(dmg, convMaxRef)

		// Send messages based on intensity
		intensity := combat.TauntHit
		if isCrit {
			intensity = combat.TauntCritical
		}

		sendTauntMessages(intensity, dmgDesc, sourceName, targetName,
			"username", targetType, user, targetPlayer, room, targetPlayerId)

		// Record analytics
		combat.RecordSpecialMove(combat.User, combat.Mob, "taunt", true, dmg,
			user.Character, targetChar, util.GetRoundCount())

	} else {
		// Taunt missed
		sendTauntMessages(combat.TauntMiss, "", sourceName, targetName,
			"username", targetType, user, targetPlayer, room, targetPlayerId)

		// Record miss
		combat.RecordSpecialMove(combat.User, combat.Mob, "taunt", false, 0,
			user.Character, targetChar, util.GetRoundCount())
	}

	// Progress rhetoric skill
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.Rhetoric,
		Details: "taunt",
	})

	// Taunt costs the current combat round
	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}

// sendTauntMessages sends data-driven taunt messages to attacker, defender,
// and room. Falls back to generic text if no YAML messages are loaded.
func sendTauntMessages(intensity combat.TauntIntensity, dmgDesc, source, target, sourceType, targetType string,
	user *users.UserRecord, targetPlayer *users.UserRecord, room *rooms.Room, targetPlayerId int) {

	atkMsg := combat.GetTauntMessage(intensity, "toattacker", source, target, sourceType, targetType, dmgDesc)
	defMsg := combat.GetTauntMessage(intensity, "todefender", source, target, sourceType, targetType, dmgDesc)
	roomMsg := combat.GetTauntMessage(intensity, "toroom", source, target, sourceType, targetType, dmgDesc)

	// Fallback if no messages loaded
	if atkMsg == "" {
		switch intensity {
		case combat.TauntHit, combat.TauntCritical:
			atkMsg = fmt.Sprintf(`You hurl a scathing taunt at %s! (%s)`, target, dmgDesc)
		case combat.TauntMiss:
			atkMsg = fmt.Sprintf(`Your taunt falls flat against %s.`, target)
		case combat.TauntFumble:
			atkMsg = fmt.Sprintf(`You stumble over your words trying to taunt %s!`, target)
		}
	}

	user.SendText(atkMsg)

	if targetPlayer != nil && defMsg != "" {
		targetPlayer.SendText(defMsg)
	}

	if roomMsg != "" {
		room.SendText(roomMsg, user.UserId, targetPlayerId)
	}
}
