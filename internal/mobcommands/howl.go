package mobcommands

import (
	"fmt"
	"math"

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

// Howl is a wolf conviction attack — a taunt reskin for mob use.
func Howl(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

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

	var targetPlayer *users.UserRecord
	var targetMob *mobs.Mob
	var defender *characters.Character
	var targetName string

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			return true, nil
		}
		targetName = targetMob.Character.Name
		defender = &targetMob.Character
	} else if targetPlayerId > 0 {
		targetPlayer = users.GetByUserId(targetPlayerId)
		if targetPlayer == nil {
			return true, nil
		}
		targetName = targetPlayer.Character.Name
		defender = targetPlayer.Character
	} else {
		return true, nil
	}

	// Conviction attack: Charisma + rhetoric vs Willpower + rhetoric
	attackerRhetoric := float64(mob.Character.GetSkillLevel(skills.Rhetoric))
	attackScore := float64(mob.Character.Stats.Charisma.ValueAdj) + attackerRhetoric

	defenderRhetoric := float64(defender.GetSkillLevel(skills.Rhetoric))
	defenseScore := float64(defender.Stats.Willpower.ValueAdj) + defenderRhetoric

	// Apply conviction depletion penalty
	cpPenalty := float64(cfg.ConvictionPenaltyMax)
	convMult := combat.ResourceMultiplier(mob.Character.Conviction,
		mob.Character.ConvictionMax.Value, cpPenalty)
	attackScore *= convMult

	// Opposed roll
	attackSuccess, _, atkRoll, _ := dice.OpposedRollStat(attackScore, defenseScore)

	// Fumble: self-conviction damage
	if atkRoll.ZScore <= -2.0 {
		selfDmg := mob.Character.Stats.Charisma.ValueAdj / 10
		if selfDmg < 1 {
			selfDmg = 1
		}
		mob.Character.Conviction -= selfDmg
		if mob.Character.Conviction < 0 {
			mob.Character.Conviction = 0
		}

		sendAudioRoomText(room, mob,
			`Something lets out a pitiful howl that trails off weakly.`,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lets out a pitiful howl that trails off weakly.`, mob.Character.Name))

		combat.RecordSpecialMove(combat.Mob, combat.User, "howl", false, 0,
			&mob.Character, defender, util.GetRoundCount())

		if mob.Character.Aggro != nil {
			mob.Character.Aggro.RoundsWaiting = 1
		}
		return true, nil
	}

	if attackSuccess {
		rawDmg := combat.CalcRawDamage(
			mob.Character.Stats.Charisma.ValueAdj,
			int(attackerRhetoric),
			0.5,
			combat.ChannelConviction,
		)
		rawDmg *= convMult

		isCrit := atkRoll.ZScore >= 2.0

		var dmg int
		if isCrit {
			dmgRoll := dice.RollStat(rawDmg)
			dmg = int(math.Round(dmgRoll.Value))
		} else {
			mitigPct := defender.GetConvictionMitigation()
			cap := combat.MitigationCap(combat.ChannelConviction)
			dmgMean := combat.ApplyMitigation(rawDmg, mitigPct, cap)
			dmgRoll := dice.RollStat(dmgMean)
			dmg = int(math.Round(dmgRoll.Value))
		}
		if dmg < 1 {
			dmg = 1
		}

		defender.Conviction -= dmg
		if defender.Conviction < 0 {
			defender.Conviction = 0
		}

		convMaxRef := defender.ConvictionMax.Value
		if convMaxRef <= 0 {
			convMaxRef = 1
		}
		dmgDesc := combat.GetConvictionDamageDescription(dmg, convMaxRef)

		if targetPlayer != nil {
			if canSeeInDark(targetPlayer, room) {
				targetPlayer.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s menacing howl shakes your resolve! (<ansi fg="damage">%s</ansi>)`, mob.Character.Name, dmgDesc))
			} else {
				targetPlayer.SendText(fmt.Sprintf(`A menacing howl shakes your resolve! (<ansi fg="damage">%s</ansi>)`, dmgDesc))
			}
		}
		sendAudioRoomText(room, mob,
			fmt.Sprintf(`Something lets out a bone-chilling howl at <ansi fg="username">%s</ansi>!`, targetName),
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> throws back its head and lets out a bone-chilling howl at <ansi fg="username">%s</ansi>!`, mob.Character.Name, targetName))

		tgtType := combat.Mob
		if targetMob == nil {
			tgtType = combat.User
		}
		combat.RecordSpecialMove(combat.Mob, tgtType, "howl", true, dmg,
			&mob.Character, defender, util.GetRoundCount())

	} else {
		// Miss
		if targetPlayer != nil {
			if canSeeInDark(targetPlayer, room) {
				targetPlayer.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> howls, but you steel yourself against the sound.`, mob.Character.Name))
			} else {
				targetPlayer.SendText(`Something howls, but you steel yourself against the sound.`)
			}
		}
		sendAudioRoomText(room, mob,
			fmt.Sprintf(`Something howls menacingly at <ansi fg="username">%s</ansi>, but it has no effect.`, targetName),
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> howls menacingly at <ansi fg="username">%s</ansi>, but it has no effect.`, mob.Character.Name, targetName))

		tgtType := combat.Mob
		if targetMob == nil {
			tgtType = combat.User
		}
		combat.RecordSpecialMove(combat.Mob, tgtType, "howl", false, 0,
			&mob.Character, defender, util.GetRoundCount())
	}

	if mob.Character.Aggro != nil {
		mob.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
