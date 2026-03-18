package usercommands

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// executeSurpriseAttack fires an extra crit hit from stealth before normal
// combat begins. It swings all of the attacker's equipped weapons, with
// escalating hit penalties for each additional weapon.
//
// targetMobInstanceId XOR targetPlayerId identifies the target (one is 0).
func executeSurpriseAttack(attacker *users.UserRecord, room *rooms.Room, targetMobInstanceId int, targetPlayerId int) {

	cfg := configs.GetBalanceConfig()

	// --- Resolve target character pointer ---
	var targetName string
	var defenderMaxHP int
	var defenderHealth *int
	var defenderMitigation float64

	if targetMobInstanceId > 0 {
		m := mobs.GetInstance(targetMobInstanceId)
		if m == nil {
			return
		}
		dupIdx := room.GetMobDuplicateIndex(m.InstanceId)
		targetName = m.Character.GetMobNameIndexed(attacker.UserId, dupIdx).String()
		defenderMaxHP = m.Character.HealthMax.Value
		defenderHealth = &m.Character.Health
		defenderMitigation = m.Character.GetPhysicalMitigation()
	} else if targetPlayerId > 0 {
		p := users.GetByUserId(targetPlayerId)
		if p == nil {
			return
		}
		targetName = p.Character.Name
		defenderMaxHP = p.Character.HealthMax.Value
		defenderHealth = &p.Character.Health
		defenderMitigation = p.Character.GetPhysicalMitigation()
	} else {
		return
	}

	// --- Attacker stats ---
	dex := attacker.Character.Stats.Dexterity.ValueAdj
	skillRank := attacker.Character.GetSkillLevel(skills.Skullduggery)

	// Surprise multiplier: dex + skill / 100, minimum 1.0
	surpriseMult := math.Max(1.0, float64(dex+skillRank)/100.0)

	// --- Collect attack weapons (same helper used by combat engine) ---
	type weaponEntry struct {
		itemId      int
		dmgMult     float64
		hitPenalty  float64 // fraction (0.0 = no penalty, 0.10 = 10% miss chance added)
	}

	weapons := []weaponEntry{}

	// Primary weapon
	if attacker.Character.Equipment.Weapon.ItemId > 0 {
		spec := attacker.Character.Equipment.Weapon.GetSpec()
		wm := spec.DamageMultiplier
		if wm <= 0 {
			wm = float64(cfg.UnarmedDamageMultiplier)
		}
		weapons = append(weapons, weaponEntry{itemId: attacker.Character.Equipment.Weapon.ItemId, dmgMult: wm, hitPenalty: 0.0})
	}

	// Offhand weapon
	if attacker.Character.Equipment.Offhand.ItemId > 0 {
		offSpec := attacker.Character.Equipment.Offhand.GetSpec()
		if offSpec.Type == items.Weapon {
			wm := offSpec.DamageMultiplier
			if wm <= 0 {
				wm = float64(cfg.UnarmedDamageMultiplier)
			}
			weapons = append(weapons, weaponEntry{itemId: attacker.Character.Equipment.Offhand.ItemId, dmgMult: wm, hitPenalty: float64(cfg.SurpriseAttackOffhandPenalty)})
		}
	}

	// Extra arm 1
	if attacker.Character.ExtraArms >= 1 && attacker.Character.Equipment.ExtraArm1.ItemId > 0 {
		ea1Spec := attacker.Character.Equipment.ExtraArm1.GetSpec()
		if ea1Spec.Type == items.Weapon {
			wm := ea1Spec.DamageMultiplier
			if wm <= 0 {
				wm = float64(cfg.UnarmedDamageMultiplier)
			}
			weapons = append(weapons, weaponEntry{itemId: attacker.Character.Equipment.ExtraArm1.ItemId, dmgMult: wm, hitPenalty: float64(cfg.SurpriseAttackExtraArm1Penalty)})
		}
	}

	// Extra arm 2
	if attacker.Character.ExtraArms >= 2 && attacker.Character.Equipment.ExtraArm2.ItemId > 0 {
		ea2Spec := attacker.Character.Equipment.ExtraArm2.GetSpec()
		if ea2Spec.Type == items.Weapon {
			wm := ea2Spec.DamageMultiplier
			if wm <= 0 {
				wm = float64(cfg.UnarmedDamageMultiplier)
			}
			weapons = append(weapons, weaponEntry{itemId: attacker.Character.Equipment.ExtraArm2.ItemId, dmgMult: wm, hitPenalty: float64(cfg.SurpriseAttackExtraArm2Penalty)})
		}
	}

	// Fallback: unarmed
	if len(weapons) == 0 {
		weapons = append(weapons, weaponEntry{itemId: 0, dmgMult: float64(cfg.UnarmedDamageMultiplier), hitPenalty: 0.0})
	}

	// --- Swing each weapon ---
	totalDamage := 0
	anyHit := false

	for _, w := range weapons {
		// Hit check: roll 0-99, if result >= (penaltyFraction * 100) → hit
		penaltyPct := int(w.hitPenalty * 100)
		roll := util.Rand(100)
		if roll < penaltyPct {
			// Missed this weapon swing due to penalty
			continue
		}

		// Damage pipeline: CalcRawDamage → surprise mult (bypass half mitigation) →
		// dice.RollStat for variance
		rawDmg := combat.CalcRawDamage(
			attacker.Character.Stats.Strength.ValueAdj,
			skillRank,
			w.dmgMult,
			combat.ChannelPhysical,
		)

		// Surprise attack: apply only half of normal mitigation (crit-like bypass)
		halfMitigation := defenderMitigation * 0.5
		dmgMean := combat.ApplyMitigation(rawDmg, halfMitigation, combat.MitigationCap(combat.ChannelPhysical))

		// Apply surprise multiplier
		dmgMean *= surpriseMult

		// Variance roll
		dmgResult := dice.RollStat(dmgMean)
		dmg := int(math.Round(math.Max(1.0, dmgResult.Value)))

		totalDamage += dmg
		anyHit = true

		// Apply damage to target
		*defenderHealth -= dmg
		if *defenderHealth < 0 {
			*defenderHealth = 0
		}
	}

	if !anyHit {
		// All weapons missed — still reveal position but no damage message
		attacker.SendText(
			fmt.Sprintf(`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> You lunge at <ansi fg="mobname">%s</ansi> from the shadows, but miss!`,
				targetName),
		)
		room.SendText(
			fmt.Sprintf(`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> <ansi fg="username">%s</ansi> lunges at %s from the shadows, but misses!`,
				attacker.Character.Name, targetName),
			attacker.UserId,
		)
		return
	}

	// --- Send hit messages ---
	dmgDesc := combat.GetDamageDescription(totalDamage, defenderMaxHP)

	attacker.SendText(
		fmt.Sprintf(`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> You strike <ansi fg="mobname">%s</ansi> from the shadows! (<ansi fg="damage">%s</ansi>)`,
			targetName, dmgDesc),
	)

	room.SendText(
		fmt.Sprintf(`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> <ansi fg="username">%s</ansi> strikes %s from the shadows!`,
			attacker.Character.Name, targetName),
		attacker.UserId,
	)

	// Notify victim if player target
	if targetPlayerId > 0 {
		if p := users.GetByUserId(targetPlayerId); p != nil {
			p.SendText(
				fmt.Sprintf(`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> <ansi fg="username">%s</ansi> strikes you from the shadows! (<ansi fg="damage">%s</ansi>)`,
					attacker.Character.Name, dmgDesc),
			)
		}
	}

	// --- Skill progression ---
	attacker.Character.TrackSkillUse(string(skills.Skullduggery))
	attacker.Character.CheckSkillProgression(string(skills.Skullduggery), attacker.UserId, 1.0)
}
