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

// Hamstring is a wolf physical attack that applies a bleed condition.
func Hamstring(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.Aggro == nil {
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
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

	// Execute skill move (reuse trip's config for damage percent, no knockdown)
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        &mob.Character,
		Defender:        defender,
		AttackStat:      mob.Character.Stats.Dexterity.ValueAdj,
		AttackSkill:     mob.Character.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     defender.Stats.Dexterity.ValueAdj,
		DefenseSkill:    defender.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.TripDamagePercent),
		KnockdownChance: 0, // No knockdown — bleed instead
		SkillRank:       mob.Character.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      mob.Character.Stats.Strength.ValueAdj,
	})

	if result.Hit {
		// Apply bleed condition: duration 5, magnitude = Strength/10 (min 2)
		bleedDmg := mob.Character.Stats.Strength.ValueAdj / 10
		if bleedDmg < 2 {
			bleedDmg = 2
		}
		defender.AddCondition(characters.ConditionBleeding, 5, float64(bleedDmg), "hamstring")

		if targetChar != nil {
			targetChar.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> rakes its fangs across your legs, opening deep wounds! (<ansi fg="damage">%s</ansi> damage)`,
					mob.Character.Name, combat.GetDamageDescription(result.Damage, result.TargetMaxHP)))
		}
		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lunges low and rakes its fangs across <ansi fg="username">%s</ansi>'s legs!`,
				mob.Character.Name, targetName),
			targetPlayerId,
		)
	} else {
		if targetChar != nil {
			targetChar.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lunges at your legs, but you sidestep the attack!`,
					mob.Character.Name))
		}
		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lunges at <ansi fg="username">%s</ansi>'s legs, but misses!`,
				mob.Character.Name, targetName),
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
	combat.RecordSpecialMove(combat.Mob, tgtType, "hamstring", result.Hit, dmgRecorded, &mob.Character, defender, util.GetRoundCount())

	if mob.Character.Aggro != nil {
		mob.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
