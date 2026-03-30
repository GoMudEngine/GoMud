package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Trip(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat or specify a target to use trip
	if user.Character.Aggro == nil {
		if rest == "" {
			user.SendText("Trip whom?")
			return true, nil
		}
		targetPId, targetMId := room.FindByName(rest)
		if targetPId == user.UserId {
			user.SendText("You can't trip yourself.")
			return true, nil
		}
		if targetPId == 0 && targetMId == 0 {
			user.SendText("You don't see them here.")
			return true, nil
		}
		if targetMId > 0 {
			user.Character.SetAggro(0, targetMId, characters.DefaultAttack)
		} else {
			user.Character.SetAggro(targetPId, 0, characters.DefaultAttack)
		}
	}

	// Check shared special move cooldown
	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	// Resolve target
	targetPlayerId := user.Character.Aggro.UserId
	targetMobId := user.Character.Aggro.MobInstanceId

	var targetChar *users.UserRecord
	var targetMob *mobs.Mob
	var targetName string
	var defender *characters.Character

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetMob.Character.Name
		defender = &targetMob.Character
	} else if targetPlayerId > 0 {
		targetChar = users.GetByUserId(targetPlayerId)
		if targetChar == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetChar.Character.Name
		defender = targetChar.Character
	} else {
		user.SendText("You have no target!")
		return true, nil
	}

	// Execute skill move (trip uses Dexterity for attack stat)
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        user.Character,
		Defender:        defender,
		AttackStat:      user.Character.Stats.Dexterity.ValueAdj,
		AttackSkill:     user.Character.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     defender.Stats.Dexterity.ValueAdj,
		DefenseSkill:    defender.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.TripDamagePercent),
		KnockdownChance: int(cfg.TripKnockdownChance),
		SkillRank:       user.Character.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      user.Character.Stats.Strength.ValueAdj,
	})

	// Send messages
	if result.Hit {
		if result.KnockedDown {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> sends <ansi fg="mobname">%s</ansi> crashing to the ground! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> sweeps your legs, sending you crashing to the ground! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
			}
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> trips <ansi fg="mobname">%s</ansi>, sending them crashing to the ground!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		} else {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> strikes <ansi fg="mobname">%s</ansi>, but they stay on their feet! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip you, but you keep your footing! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
			}
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip <ansi fg="mobname">%s</ansi>, but they keep their footing!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		}
	} else {
		user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">trip</ansi> attempt misses <ansi fg="mobname">%s</ansi>!`, targetName))
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip you, but you avoid it!`, user.Character.Name))
		}
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to trip <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, targetName),
			user.UserId, targetPlayerId,
		)
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
	combat.RecordSpecialMove(combat.User, tgtType, "trip", result.Hit, dmgRecorded, user.Character, defender, util.GetRoundCount())

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
