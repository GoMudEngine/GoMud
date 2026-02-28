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

	// Execute skill move
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        user.Character,
		Defender:        defender,
		AttackStat:      user.Character.Stats.Strength.ValueAdj,
		AttackSkill:     user.Character.GetSkillLevel(skills.WeaponCombat),
		DefenseStat:     defender.Stats.Dexterity.ValueAdj,
		DefenseSkill:    defender.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.BashDamagePercent),
		KnockdownChance: int(cfg.BashKnockdownChance),
		SkillRank:       user.Character.GetSkillLevel(skills.WeaponCombat),
		DamageStat:      user.Character.Stats.Strength.ValueAdj,
	})

	// Send messages
	if result.Hit {
		if result.KnockedDown {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
			}
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		} else {
			user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">shield bash</ansi> strikes <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)`, targetName, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> strikes you! (<ansi fg="damage">%s</ansi>)`, user.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
			}
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> bashes <ansi fg="mobname">%s</ansi> with their shield!`, user.Character.Name, targetName),
				user.UserId, targetPlayerId,
			)
		}
	} else {
		user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">shield bash</ansi> misses <ansi fg="mobname">%s</ansi>!`, targetName))
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to bash you with their shield, but misses!`, user.Character.Name))
		}
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> attempts to bash <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, targetName),
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
	combat.RecordSpecialMove(combat.User, tgtType, "bash", result.Hit, dmgRecorded, user.Character, defender, util.GetRoundCount())

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
