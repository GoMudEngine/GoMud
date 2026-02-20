package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// resolveSpell is called when fold accumulation completes.
// It dispatches to per-target resolution based on spell type and effect.
func resolveSpell(user *users.UserRecord, cs *characters.CastingState, spellData *spells.SpellData, room *rooms.Room) {

	skillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
	spellAttack := characters.CalcSpellAttack(user.Character.Stats.Willpower.ValueAdj, skillLevel)
	magnitude := spellData.EffectMagnitude

	// --- Populate area targets for HarmArea ---
	if spellData.Type == spells.HarmArea {
		cs.TargetMobInstanceIds = room.GetMobs(rooms.FindAll)
		// HarmArea hits everyone in the room (all mobs); players are excluded in this stage
	}

	// --- Resolve against mob targets ---
	for _, mobInstId := range cs.TargetMobInstanceIds {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil || mob.Character.Health < 1 {
			continue
		}
		resolveAgainstMob(user, mob, room, spellData, spellAttack, magnitude)
	}

	// --- Resolve against player targets ---
	for _, targetUserId := range cs.TargetUserIds {
		targetUser := users.GetByUserId(targetUserId)
		if targetUser == nil {
			continue
		}
		if spellData.TargetDefenseType == "" {
			// Help spell with no defense — always applies
			applyPlayerEffect(user, targetUser, room, spellData, magnitude, false)
		} else {
			resolveAgainstPlayer(user, targetUser, room, spellData, spellAttack, magnitude)
		}
	}

	// --- Consume component if required ---
	if spellData.ComponentTag != "" {
		consumeSpellComponent(user, spellData.ComponentTag)
	}
}

// resolveAgainstMob performs the opposed roll and applies the effect to a mob.
func resolveAgainstMob(user *users.UserRecord, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, spellAttack float64, magnitude int) {

	defVal := spellDefenseValue(spellData.TargetDefenseType, &mob.Character)
	success, _, atkRoll, _ := dice.OpposedRoll(spellAttack, defVal, dice.StdDevFor(spellAttack))

	// Backfire on fumble
	if atkRoll.ZScore <= -2.0 {
		backfireDmg := magnitude / 4
		if backfireDmg < 1 {
			backfireDmg = 1
		}
		user.Character.Health -= backfireDmg
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">Your spell backfires! You take %d damage!</ansi>`, backfireDmg))
		room.SendText(fmt.Sprintf(
			`<ansi fg="red"><ansi fg="username">%s</ansi>'s spell backfires!</ansi>`, user.Character.Name), user.UserId)
		return
	}

	if !success {
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">Your %s fizzles against <ansi fg="mobname">%s</ansi>.</ansi>`,
			spellData.Name, mob.Character.Name))
		return
	}

	isCrit := atkRoll.ZScore >= 2.0
	applyMobEffect(user, mob, room, spellData, magnitude, isCrit)
}

// applyMobEffect applies the spell effect to a mob.
func applyMobEffect(user *users.UserRecord, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int, isCrit bool) {

	critTag := ""
	if isCrit {
		critTag = ` <ansi fg="yellow">[CRIT!]</ansi>`
	}

	switch spellData.EffectType {
	case "damage":
		dmgRoll := dice.Roll(float64(magnitude), dice.StdDevFor(float64(magnitude)))
		dmg := int(math.Round(dmgRoll.Value))
		if dmg < 1 {
			dmg = 1
		}
		if isCrit {
			dmg += magnitude
		}
		mob.Character.Health -= dmg
		// Set mob aggro toward the caster if not already fighting
		if mob.Character.Aggro == nil {
			mob.PreventIdle = true
			mob.Command(fmt.Sprintf("attack @%d", user.UserId))
		}
		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> strikes <ansi fg="mobname">%s</ansi> for <ansi fg="red">%d</ansi> damage!%s</ansi>`,
			spellData.Name, mob.Character.Name, dmg, critTag))
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes <ansi fg="mobname">%s</ansi>!`,
			user.Character.Name, spellData.Name, mob.Character.Name), user.UserId)

	case "buff":
		for _, buffId := range spellData.BuffIds {
			mob.AddBuff(buffId, "spell")
		}
		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> takes effect on <ansi fg="mobname">%s</ansi>!%s</ansi>`,
			spellData.Name, mob.Character.Name, critTag))
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> affects <ansi fg="mobname">%s</ansi>!`,
			user.Character.Name, spellData.Name, mob.Character.Name), user.UserId)

	case "tame":
		// Tame is restricted to animal group mobs
		isAnimal := false
		for _, g := range mob.Groups {
			if g == "animal" {
				isAnimal = true
				break
			}
		}
		if !isAnimal {
			user.SendText(fmt.Sprintf(
				`<ansi fg="red"><ansi fg="mobname">%s</ansi> cannot be tamed — it is not a wild animal.</ansi>`,
				mob.Character.Name))
			return
		}
		mob.Character.Charm(user.UserId, 24, "")
		mob.Character.Aggro = nil
		user.Character.TrackCharmed(mob.InstanceId, true)
		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan"><ansi fg="mobname">%s</ansi> calms and becomes your companion!</ansi>`,
			mob.Character.Name))
		room.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> becomes docile and follows <ansi fg="username">%s</ansi>.`,
			mob.Character.Name, user.Character.Name), user.UserId)

	default:
		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> takes effect on <ansi fg="mobname">%s</ansi>.</ansi>`,
			spellData.Name, mob.Character.Name))
	}
}

// resolveAgainstPlayer performs the opposed roll and applies the effect to a player.
func resolveAgainstPlayer(user *users.UserRecord, target *users.UserRecord, room *rooms.Room, spellData *spells.SpellData, spellAttack float64, magnitude int) {

	defVal := spellDefenseValue(spellData.TargetDefenseType, target.Character)
	success, _, atkRoll, _ := dice.OpposedRoll(spellAttack, defVal, dice.StdDevFor(spellAttack))

	// Backfire on fumble
	if atkRoll.ZScore <= -2.0 {
		backfireDmg := magnitude / 4
		if backfireDmg < 1 {
			backfireDmg = 1
		}
		user.Character.Health -= backfireDmg
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">Your spell backfires! You take %d damage!</ansi>`, backfireDmg))
		room.SendText(fmt.Sprintf(
			`<ansi fg="red"><ansi fg="username">%s</ansi>'s spell backfires!</ansi>`, user.Character.Name), user.UserId)
		return
	}

	if !success {
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">Your %s fizzles against <ansi fg="username">%s</ansi>.</ansi>`,
			spellData.Name, target.Character.Name))
		return
	}

	isCrit := atkRoll.ZScore >= 2.0
	applyPlayerEffect(user, target, room, spellData, magnitude, isCrit)
}

// applyPlayerEffect applies the spell effect to a player target.
func applyPlayerEffect(user *users.UserRecord, target *users.UserRecord, room *rooms.Room, spellData *spells.SpellData, magnitude int, isCrit bool) {

	critTag := ""
	if isCrit {
		critTag = ` <ansi fg="yellow">[CRIT!]</ansi>`
	}

	switch spellData.EffectType {
	case "heal":
		healRoll := dice.Roll(float64(magnitude), dice.StdDevFor(float64(magnitude)))
		heal := int(math.Round(healRoll.Value))
		if heal < 1 {
			heal = 1
		}
		if isCrit {
			heal += magnitude
		}
		actual := target.Character.Heal(heal)
		user.SendText(fmt.Sprintf(
			`<ansi fg="green">Your <ansi fg="cyan-bold">%s</ansi> restores <ansi fg="green-bold">%d</ansi> health to <ansi fg="username">%s</ansi>!%s</ansi>`,
			spellData.Name, actual, target.Character.Name, critTag))
		if target.UserId != user.UserId {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green"><ansi fg="username">%s</ansi>'s <ansi fg="cyan-bold">%s</ansi> restores <ansi fg="green-bold">%d</ansi> health!</ansi>`,
				user.Character.Name, spellData.Name, actual))
		}
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> heals <ansi fg="username">%s</ansi>.`,
			user.Character.Name, spellData.Name, target.Character.Name), user.UserId, target.UserId)

	case "buff":
		for _, buffId := range spellData.BuffIds {
			target.AddBuff(buffId, "spell")
		}
		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> takes effect on <ansi fg="username">%s</ansi>!%s</ansi>`,
			spellData.Name, target.Character.Name, critTag))
		if target.UserId != user.UserId {
			target.SendText(fmt.Sprintf(
				`<ansi fg="cyan"><ansi fg="username">%s</ansi>'s <ansi fg="cyan-bold">%s</ansi> takes effect on you!</ansi>`,
				user.Character.Name, spellData.Name))
		}

	case "shield":
		skillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
		shieldBonus := (user.Character.Stats.Willpower.ValueAdj + skillLevel) / 3
		if shieldBonus < 1 {
			shieldBonus = 1
		}
		target.Character.AddCondition(characters.ConditionShield, 10, float64(shieldBonus), "spell")
		target.SendText(fmt.Sprintf(
			`<ansi fg="cyan">A magical barrier forms around you! +%d armor for 10 rounds.</ansi>`, shieldBonus))
		if target.UserId != user.UserId {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">A magical barrier forms around <ansi fg="username">%s</ansi>! +%d armor for 10 rounds.</ansi>`,
				target.Character.Name, shieldBonus))
		}
		room.SendText(fmt.Sprintf(
			`A shimmering barrier surrounds <ansi fg="username">%s</ansi>.`, target.Character.Name), target.UserId)

	default:
		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> takes effect on <ansi fg="username">%s</ansi>.</ansi>`,
			spellData.Name, target.Character.Name))
	}
}

// spellDefenseValue computes the defender's stat for the opposed roll.
func spellDefenseValue(defenseType string, target *characters.Character) float64 {
	switch defenseType {
	case "physical":
		equip := target.Equipment
		armor := 0
		if equip.Head.ItemId > 0 {
			armor += equip.Head.GetSpec().DamageReduction
		}
		if equip.Body.ItemId > 0 {
			armor += equip.Body.GetSpec().DamageReduction
		}
		if equip.Legs.ItemId > 0 {
			armor += equip.Legs.GetSpec().DamageReduction
		}
		if equip.Feet.ItemId > 0 {
			armor += equip.Feet.GetSpec().DamageReduction
		}
		if equip.Gloves.ItemId > 0 {
			armor += equip.Gloves.GetSpec().DamageReduction
		}
		if equip.Neck.ItemId > 0 {
			armor += equip.Neck.GetSpec().DamageReduction
		}
		if equip.Ring.ItemId > 0 {
			armor += equip.Ring.GetSpec().DamageReduction
		}
		if equip.Offhand.ItemId > 0 {
			armor += equip.Offhand.GetSpec().DamageReduction
		}
		defVal := float64(armor)
		// Add Minor Shield bonus if active
		defVal += target.GetConditionMagnitude(characters.ConditionShield)
		return defVal

	case "mental":
		return float64(target.Stats.Willpower.ValueAdj)

	default:
		return 0.0
	}
}

// consumeSpellComponent removes the first matching component item from caster's inventory.
func consumeSpellComponent(user *users.UserRecord, tag string) {
	for i, itm := range user.Character.Items {
		if itm.GetSpec().ComponentTag == tag {
			user.Character.Items = append(user.Character.Items[:i], user.Character.Items[i+1:]...)
			user.SendText(fmt.Sprintf(
				`<ansi fg="yellow">You consume a %s as a spell component.</ansi>`, tag))
			return
		}
	}
}
