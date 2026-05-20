package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Charge is a boar trip variant — same mechanics, different messages.
func Charge(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if !mob.Character.IsInCombat() {
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	if !mob.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return true, nil
	}

	// Resolve target
	targetPlayerId := mob.Character.CurrentCombatTarget().UserId
	targetMobId := mob.Character.CurrentCombatTarget().MobInstanceId

	var targetChar *users.UserRecord
	var targetMob *mobs.Mob
	var targetName string
	var defender *characters.Character

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			return true, nil
		}
		targetName = targetMob.Character.Name
		defender = &targetMob.Character
	} else if targetPlayerId > 0 {
		targetChar = users.GetByUserId(targetPlayerId)
		if targetChar == nil {
			return true, nil
		}
		targetName = targetChar.Character.Name
		defender = targetChar.Character
	} else {
		return true, nil
	}

	// Execute skill move (same params as trip)
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        &mob.Character,
		Defender:        defender,
		AttackStat:      mob.Character.Stats.Dexterity.ValueAdj,
		AttackSkill:     mob.Character.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     defender.Stats.Dexterity.ValueAdj,
		DefenseSkill:    defender.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.TripDamagePercent),
		KnockdownChance: int(cfg.TripKnockdownChance),
		SkillRank:       mob.Character.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      mob.Character.Stats.Strength.ValueAdj,
	})

	// Send messages
	mobName := mob.Character.Name
	canSee := targetChar == nil || canSeeInDark(targetChar, room)
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)
	if result.Hit {
		if result.KnockedDown {
			if targetChar != nil {
				if canSee {
					targetChar.SendTextLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges and slams into you, sending you sprawling! (<ansi fg="damage">%s</ansi> damage)`, mobName, dmgDesc))
				} else {
					targetChar.SendTextLegacy(fmt.Sprintf(`Something charges and slams into you, sending you sprawling! (<ansi fg="damage">%s</ansi> damage)`, dmgDesc))
				}
			}
			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges and slams into <ansi fg="username">%s</ansi>, sending them sprawling!`, mobName, targetName),
				targetPlayerId)
		} else {
			if targetChar != nil {
				if canSee {
					targetChar.SendTextLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges at you, but you keep your footing! (<ansi fg="damage">%s</ansi> damage)`, mobName, dmgDesc))
				} else {
					targetChar.SendTextLegacy(fmt.Sprintf(`Something charges at you, but you keep your footing! (<ansi fg="damage">%s</ansi> damage)`, dmgDesc))
				}
			}
			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges at <ansi fg="username">%s</ansi>, but they keep their footing!`, mobName, targetName),
				targetPlayerId)
		}
	} else {
		if targetChar != nil {
			if canSee {
				targetChar.SendTextLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges past you, missing entirely!`, mobName))
			} else {
				targetChar.SendTextLegacy(`Something charges past you, missing entirely!`)
			}
		}
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> charges past <ansi fg="username">%s</ansi>, missing entirely!`, mobName, targetName),
			targetPlayerId)
	}

	// Record combat analytics
	dmgRecorded := 0
	if result.Hit {
		dmgRecorded = result.Damage
	}
	tgtType := combat.Mob
	if targetMob == nil {
		tgtType = combat.User
	}
	combat.RecordSpecialMove(combat.Mob, tgtType, "charge", result.Hit, dmgRecorded, &mob.Character, defender, util.GetRoundCount())

	if mob.Character.Aggro != nil {
		mob.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
