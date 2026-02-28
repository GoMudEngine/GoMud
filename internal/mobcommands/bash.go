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

	// Resolve target
	targetPlayerId := mob.Character.Aggro.UserId
	targetMobId := mob.Character.Aggro.MobInstanceId

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

	// Execute skill move
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        &mob.Character,
		Defender:        defender,
		AttackStat:      mob.Character.Stats.Strength.ValueAdj,
		AttackSkill:     mob.Character.GetSkillLevel(skills.WeaponCombat),
		DefenseStat:     defender.Stats.Dexterity.ValueAdj,
		DefenseSkill:    defender.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.BashDamagePercent),
		KnockdownChance: int(cfg.BashKnockdownChance),
		SkillRank:       mob.Character.GetSkillLevel(skills.WeaponCombat),
		DamageStat:      mob.Character.Stats.Strength.ValueAdj,
	})

	// Send messages
	if result.Hit {
		if result.KnockedDown {
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks you to the ground! (<ansi fg="damage">%s</ansi> damage)`, mob.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
			}
			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> knocks <ansi fg="username">%s</ansi> to the ground!`, mob.Character.Name, targetName),
				targetPlayerId,
			)
		} else {
			if targetChar != nil {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s <ansi fg="yellow-bold">shield bash</ansi> strikes you! (<ansi fg="damage">%s</ansi> damage)`, mob.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
			}
			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> bashes <ansi fg="username">%s</ansi> with their shield!`, mob.Character.Name, targetName),
				targetPlayerId,
			)
		}
	} else {
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to bash you with their shield, but misses!`, mob.Character.Name))
		}
		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> attempts to bash <ansi fg="username">%s</ansi>, but misses!`, mob.Character.Name, targetName),
			targetPlayerId,
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
	combat.RecordSpecialMove(combat.Mob, tgtType, "bash", result.Hit, dmgRecorded, &mob.Character, defender, util.GetRoundCount())

	// Bash costs the current combat round
	if mob.Character.Aggro != nil {
		mob.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
