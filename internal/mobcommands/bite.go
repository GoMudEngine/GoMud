package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Bite is a vampire-only special attack that deals moderate physical damage
// and heals the vampire for half the damage inflicted (life drain).
func Bite(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat.
	if mob.Character.Aggro == nil {
		return true, nil
	}

	// Check special-move cooldown (shared with bash/kick/trip).
	cfg := configs.GetBalanceConfig()
	cooldownStr := fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)
	if !mob.Character.Cooldowns.Try("special-move", cooldownStr) {
		return true, nil
	}

	// Resolve the aggro target.
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return true, nil
	}

	// Execute the skill move.
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        &mob.Character,
		Defender:        target.Char,
		AttackStat:      mob.Character.Stats.Strength.ValueAdj,
		AttackSkill:     mob.Character.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   0.60,
		KnockdownChance: 0,
		SkillRank:       mob.Character.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      mob.Character.Stats.Strength.ValueAdj,
	})

	// On hit: drain life — heal the vampire for 50% of damage dealt.
	if result.Hit && result.Damage > 0 {
		drain := int(float64(result.Damage) * 0.50)
		mob.Character.Health += drain
		if mob.Character.Health > mob.Character.HealthMax.Value {
			mob.Character.Health = mob.Character.HealthMax.Value
		}
	}

	// Record analytics and consume the combat round.
	actions.RecordAndWait(&mob.Character, "bite", combat.Mob, target.Char, combat.User, result.Hit,
		func() int {
			if result.Hit {
				return result.Damage
			}
			return 0
		}(), util.GetRoundCount())

	// Messaging — darkness-aware.
	mobName := mob.Character.Name
	dmgDesc := combat.GetDamageDescription(result.Damage, result.TargetMaxHP)

	var targetUser *users.UserRecord
	if target.UserId > 0 {
		targetUser = users.GetByUserId(target.UserId)
	}
	canSee := targetUser == nil || canSeeInDark(targetUser, room)

	if result.Hit {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> sinks its fangs into you, drawing strength from the wound! (<ansi fg="damage">%s</ansi> damage)`,
					mobName, dmgDesc))
			} else {
				targetUser.SendText(fmt.Sprintf(
					`Something sinks its fangs into you in the darkness, drawing strength from the wound! (<ansi fg="damage">%s</ansi> damage)`,
					dmgDesc))
			}
		}
		if result.Damage > 0 {
			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> sinks its fangs into <ansi fg="username">%s</ansi> and draws strength from the wound!`,
					mobName, target.Name),
				target.UserId)
		} else {
			sendRoomText(room,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> sinks its fangs into <ansi fg="username">%s</ansi>!`,
					mobName, target.Name),
				target.UserId)
		}
	} else {
		if targetUser != nil {
			if canSee {
				targetUser.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> snaps its fangs at you, but misses!`,
					mobName))
			} else {
				targetUser.SendText(`Something snaps its fangs at you in the darkness, but misses!`)
			}
		}
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> snaps its fangs at <ansi fg="username">%s</ansi>, but misses!`,
				mobName, target.Name),
			target.UserId)
	}

	return true, nil
}
