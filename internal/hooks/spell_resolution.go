package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
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

	// --- Populate area targets for HelpArea ---
	if spellData.Type == spells.HelpArea {
		cs.TargetUserIds = room.GetPlayers(rooms.FindAll)
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
	success, _, atkRoll, _ := dice.OpposedRollStat(spellAttack, defVal)

	round := util.GetRoundCount()

	// Backfire on fumble
	if atkRoll.ZScore <= -2.0 {
		backfireDmg := magnitude / 4
		if backfireDmg < 1 {
			backfireDmg = 1
		}
		user.Character.Health -= backfireDmg
		user.SendText(`<ansi fg="red">Your spell backfires violently, wounding you!</ansi>`)
		room.SendText(fmt.Sprintf(
			`<ansi fg="red"><ansi fg="username">%s</ansi>'s spell backfires!</ansi>`, user.Character.Name), user.UserId)
		// Stage 30.1: Record backfire
		combat.RecordSpell(combat.User, combat.Mob, false, false, true, false, 0, atkRoll.ZScore, user.Character, &mob.Character, round)
		return
	}

	if !success {
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">Your %s fizzles against <ansi fg="mobname">%s</ansi>.</ansi>`,
			spellData.Name, mob.Character.Name))
		// Stage 30.1: Record fizzle
		combat.RecordSpell(combat.User, combat.Mob, false, false, false, true, 0, atkRoll.ZScore, user.Character, &mob.Character, round)
		return
	}

	isCrit := atkRoll.ZScore >= 2.0
	// Stage 30.1: Record spell hit (damage recorded as 0; actual damage applied in applyMobEffect)
	combat.RecordSpell(combat.User, combat.Mob, true, isCrit, false, false, 0, atkRoll.ZScore, user.Character, &mob.Character, round)
	applyMobEffect(user, mob, room, spellData, magnitude, isCrit)
}

// applyMobEffect applies the spell effect to a mob.
// user may be nil when the caster is a mob (guards all user.* references).
func applyMobEffect(user *users.UserRecord, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int, isCrit bool) {

	critTag := ""
	if isCrit {
		critTag = ` <ansi fg="yellow">[CRIT!]</ansi>`
	}

	switch spellData.EffectType {
	case "damage":
		dmgRoll := dice.RollStat(float64(magnitude))
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
			if user != nil {
				mob.Command(fmt.Sprintf("attack @%d", user.UserId))
			}
		}
		if user != nil {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> strikes <ansi fg="mobname">%s</ansi>! (<ansi fg="damage">%s</ansi>)%s</ansi>`,
				spellData.Name, mob.Character.Name, combat.GetDamageDescription(dmg, mob.Character.HealthMax.Value), critTag))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes <ansi fg="mobname">%s</ansi>!`,
				user.Character.Name, spellData.Name, mob.Character.Name), user.UserId)
		}

	case "dot":
		dotDuration := spellData.EffectDuration
		if dotDuration < 1 {
			dotDuration = 3
		}
		// Duration is in AutoHeal ticks (every 3 rounds), so multiply by 3 for round count
		mob.Character.AddCondition(characters.ConditionPoisoned, dotDuration*3, float64(magnitude), "spell")
		if mob.Character.Aggro == nil {
			mob.PreventIdle = true
			if user != nil {
				mob.Command(fmt.Sprintf("attack @%d", user.UserId))
			}
		}
		if user != nil {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> afflicts <ansi fg="mobname">%s</ansi>!%s</ansi>`,
				spellData.Name, mob.Character.Name, critTag))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> afflicts <ansi fg="mobname">%s</ansi>!`,
				user.Character.Name, spellData.Name, mob.Character.Name), user.UserId)
		}

	case "knockdown":
		dmgRoll := dice.RollStat(float64(magnitude))
		dmg := int(math.Round(dmgRoll.Value))
		if dmg < 1 {
			dmg = 1
		}
		if isCrit {
			dmg += magnitude
		}
		mob.Character.Health -= dmg
		mob.Character.CombatPosition = characters.PositionProne
		mob.Character.PositionRoundsMin = 1
		if mob.Character.Aggro == nil {
			mob.PreventIdle = true
			if user != nil {
				mob.Command(fmt.Sprintf("attack @%d", user.UserId))
			}
		}
		if user != nil {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> slams <ansi fg="mobname">%s</ansi> to the ground! (<ansi fg="damage">%s</ansi>)%s</ansi>`,
				spellData.Name, mob.Character.Name, combat.GetDamageDescription(dmg, mob.Character.HealthMax.Value), critTag))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> knocks <ansi fg="mobname">%s</ansi> to the ground!`,
				user.Character.Name, spellData.Name, mob.Character.Name), user.UserId)
		}

	case "buff":
		for _, buffId := range spellData.BuffIds {
			mob.AddBuff(buffId, "spell")
		}
		if user != nil {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> takes effect on <ansi fg="mobname">%s</ansi>!%s</ansi>`,
				spellData.Name, mob.Character.Name, critTag))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> affects <ansi fg="mobname">%s</ansi>!`,
				user.Character.Name, spellData.Name, mob.Character.Name), user.UserId)
		}

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
			if user != nil {
				user.SendText(fmt.Sprintf(
					`<ansi fg="red"><ansi fg="mobname">%s</ansi> cannot be tamed — it is not a wild animal.</ansi>`,
					mob.Character.Name))
			}
			return
		}
		if user != nil {
			mob.Character.Charm(user.UserId, 24, "")
			mob.Character.Aggro = nil
			user.Character.TrackCharmed(mob.InstanceId, true)
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan"><ansi fg="mobname">%s</ansi> calms and becomes your companion!</ansi>`,
				mob.Character.Name))
			room.SendText(fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> becomes docile and follows <ansi fg="username">%s</ansi>.`,
				mob.Character.Name, user.Character.Name), user.UserId)
		}

	default:
		if user != nil {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> takes effect on <ansi fg="mobname">%s</ansi>.</ansi>`,
				spellData.Name, mob.Character.Name))
		}
	}
}

// resolveAgainstPlayer performs the opposed roll and applies the effect to a player.
func resolveAgainstPlayer(user *users.UserRecord, target *users.UserRecord, room *rooms.Room, spellData *spells.SpellData, spellAttack float64, magnitude int) {

	defVal := spellDefenseValue(spellData.TargetDefenseType, target.Character)
	success, _, atkRoll, _ := dice.OpposedRollStat(spellAttack, defVal)

	// Backfire on fumble
	if atkRoll.ZScore <= -2.0 {
		backfireDmg := magnitude / 4
		if backfireDmg < 1 {
			backfireDmg = 1
		}
		user.Character.Health -= backfireDmg
		user.SendText(`<ansi fg="red">Your spell backfires violently, wounding you!</ansi>`)
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
	case "purge":
		target.Character.CancelBuffsWithFlag(buffs.Poison)
		target.Character.RemoveCondition(characters.ConditionPoisoned)
		user.SendText(fmt.Sprintf(
			`<ansi fg="green">Your <ansi fg="cyan-bold">%s</ansi> cleanses <ansi fg="username">%s</ansi> of afflictions.%s</ansi>`,
			spellData.Name, target.Character.Name, critTag))
		if target.UserId != user.UserId {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green"><ansi fg="username">%s</ansi>'s <ansi fg="cyan-bold">%s</ansi> purges the toxins from your body.</ansi>`,
				user.Character.Name, spellData.Name))
		} else {
			target.SendText(`<ansi fg="green">You purge the afflictions from your body.</ansi>`)
		}
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> cleanses <ansi fg="username">%s</ansi>.`,
			user.Character.Name, spellData.Name, target.Character.Name), user.UserId, target.UserId)

	case "heal":
		skillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
		// Magnitude from YAML is the regen multiplier (e.g. 3 = 3x base regen)
		regenMult := float64(magnitude)
		if regenMult < 1.0 {
			regenMult = 1.0
		}
		if isCrit {
			// Crit: boost the multiplier portion above 1x by 2x
			regenMult = 1.0 + (regenMult-1.0)*2.0
		}
		ticks := skillLevel / 10
		if ticks < 1 {
			ticks = 1
		}
		durationRounds := ticks * 3 // TickConditions runs every combat round; AutoHeal fires every 3
		target.Character.AddCondition(characters.ConditionRegen, durationRounds, regenMult, "heal spell")
		user.SendText(fmt.Sprintf(
			`<ansi fg="green">You weave restorative magic around <ansi fg="username">%s</ansi>.%s</ansi>`,
			target.Character.Name, critTag))
		if target.UserId != user.UserId {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green"><ansi fg="username">%s</ansi>'s <ansi fg="cyan-bold">%s</ansi> envelops you in healing energy. Your wounds begin to mend.</ansi>`,
				user.Character.Name, spellData.Name))
		} else {
			target.SendText(`<ansi fg="green">A warm glow of healing magic envelops you. Your wounds begin to mend.</ansi>`)
		}
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> envelops <ansi fg="username">%s</ansi> in healing light.`,
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
		duration := 10 + int(math.Round(float64(skillLevel)/5))
		target.Character.AddCondition(characters.ConditionShield, duration, float64(shieldBonus), "spell")
		target.SendText(`<ansi fg="cyan">A shimmering magical barrier forms around you, bolstering your defenses.</ansi>`)
		if target.UserId != user.UserId {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">A shimmering magical barrier forms around <ansi fg="username">%s</ansi>, bolstering their defenses.</ansi>`,
				target.Character.Name))
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

// resolveMobSpell is called when a mob's fold accumulation completes.
func resolveMobSpell(mob *mobs.Mob, cs *characters.CastingState, spellData *spells.SpellData, room *rooms.Room) {
	skillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
	spellAttack := characters.CalcSpellAttack(mob.Character.Stats.Willpower.ValueAdj, skillLevel)
	magnitude := spellData.EffectMagnitude

	if spellData.Type == spells.HarmArea {
		cs.TargetMobInstanceIds = room.GetMobs(rooms.FindAll)
		cs.TargetUserIds = room.GetPlayers(rooms.FindAll)
	}

	for _, mobInstId := range cs.TargetMobInstanceIds {
		if mobInstId == mob.InstanceId {
			// Self-cast (HelpSingle with self target)
			applyMobSelfEffect(mob, room, spellData, magnitude)
			continue
		}
		if target := mobs.GetInstance(mobInstId); target != nil && target.Character.Health > 0 {
			resolveMobSpellAgainstMob(mob, target, room, spellData, spellAttack, magnitude)
		}
	}
	for _, userId := range cs.TargetUserIds {
		if target := users.GetByUserId(userId); target != nil {
			resolveMobSpellAgainstPlayer(mob, target, room, spellData, spellAttack, magnitude)
		}
	}
}

// applyMobSelfEffect handles self-targeted help spells (heal, minor-shield).
func applyMobSelfEffect(mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int) {
	switch spellData.EffectType {
	case "heal":
		skillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
		regenMult := float64(magnitude)
		if regenMult < 1.0 {
			regenMult = 1.0
		}
		ticks := skillLevel / 10
		if ticks < 1 {
			ticks = 1
		}
		durationRounds := ticks * 3
		mob.Character.AddCondition(characters.ConditionRegen, durationRounds, regenMult, "heal spell")
		room.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> channels restorative magic.`, mob.Character.Name))
	case "shield":
		skillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
		shieldBonus := (mob.Character.Stats.Willpower.ValueAdj + skillLevel) / 3
		if shieldBonus < 1 {
			shieldBonus = 1
		}
		duration := 10 + int(math.Round(float64(skillLevel)/5))
		mob.Character.AddCondition(characters.ConditionShield, duration, float64(shieldBonus), "spell")
		room.SendText(fmt.Sprintf(
			`A shimmering barrier forms around <ansi fg="mobname">%s</ansi>.`, mob.Character.Name))
	}
}

func resolveMobSpellAgainstMob(caster *mobs.Mob, target *mobs.Mob, room *rooms.Room,
	spellData *spells.SpellData, spellAttack float64, magnitude int) {
	defVal := spellDefenseValue(spellData.TargetDefenseType, &target.Character)
	success, _, atkRoll, _ := dice.OpposedRollStat(spellAttack, defVal)
	if atkRoll.ZScore <= -2.0 {
		dmg := magnitude / 4
		if dmg < 1 {
			dmg = 1
		}
		caster.Character.Health -= dmg
		room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s spell backfires!`, caster.Character.Name))
		return
	}
	if !success {
		return
	}
	applyMobEffect(nil, target, room, spellData, magnitude, atkRoll.ZScore >= 2.0)
}

func resolveMobSpellAgainstPlayer(caster *mobs.Mob, target *users.UserRecord, room *rooms.Room,
	spellData *spells.SpellData, spellAttack float64, magnitude int) {
	defVal := spellDefenseValue(spellData.TargetDefenseType, target.Character)
	success, _, atkRoll, _ := dice.OpposedRollStat(spellAttack, defVal)
	round := util.GetRoundCount()
	if atkRoll.ZScore <= -2.0 {
		dmg := magnitude / 4
		if dmg < 1 {
			dmg = 1
		}
		caster.Character.Health -= dmg
		room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s spell backfires!`, caster.Character.Name))
		// Stage 30.1: Record backfire
		combat.RecordSpell(combat.Mob, combat.User, false, false, true, false, 0, atkRoll.ZScore, &caster.Character, target.Character, round)
		return
	}
	if !success {
		room.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s %s fizzles.`, caster.Character.Name, spellData.Name))
		// Stage 30.1: Record fizzle
		combat.RecordSpell(combat.Mob, combat.User, false, false, false, true, 0, atkRoll.ZScore, &caster.Character, target.Character, round)
		return
	}
	isCrit := atkRoll.ZScore >= 2.0
	// Stage 30.1: Record spell hit
	combat.RecordSpell(combat.Mob, combat.User, true, isCrit, false, false, 0, atkRoll.ZScore, &caster.Character, target.Character, round)
	critTag := ""
	if isCrit {
		critTag = ` <ansi fg="yellow">[CRIT!]</ansi>`
	}
	switch spellData.EffectType {
	case "damage":
		dmgRoll := dice.RollStat(float64(magnitude))
		dmg := int(math.Round(dmgRoll.Value))
		if dmg < 1 {
			dmg = 1
		}
		if isCrit {
			dmg += magnitude
		}
		// Stage 12.1: Magical Resistance mutation reduces incoming spell damage
		if resist := mutations.GetMagicalResistance(target.Character.Mutations); resist > 0 {
			dmg = int(float64(dmg) * (1.0 - resist))
			if dmg < 1 {
				dmg = 1
			}
			target.SendText(`<ansi fg="blue">Your magical resistance dampens the blow.</ansi>`)
		}
		target.Character.Health -= dmg
		target.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes you! (<ansi fg="damage">%s</ansi>)%s`,
			caster.Character.Name, spellData.Name,
			combat.GetDamageDescription(dmg, target.Character.HealthMax.Value), critTag))
		room.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes <ansi fg="username">%s</ansi>!`,
			caster.Character.Name, spellData.Name, target.Character.Name), target.UserId)
		if target.Character.Aggro == nil {
			target.Character.Aggro = &characters.Aggro{MobInstanceId: caster.InstanceId}
		}
	case "dot":
		dotDuration := spellData.EffectDuration
		if dotDuration < 1 {
			dotDuration = 3
		}
		target.Character.AddCondition(characters.ConditionPoisoned, dotDuration*3, float64(magnitude), "spell")
		target.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> afflicts you!%s`,
			caster.Character.Name, spellData.Name, critTag))
		room.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> afflicts <ansi fg="username">%s</ansi>!`,
			caster.Character.Name, spellData.Name, target.Character.Name), target.UserId)
		if target.Character.Aggro == nil {
			target.Character.Aggro = &characters.Aggro{MobInstanceId: caster.InstanceId}
		}
	case "knockdown":
		dmgRoll := dice.RollStat(float64(magnitude))
		dmg := int(math.Round(dmgRoll.Value))
		if dmg < 1 {
			dmg = 1
		}
		if isCrit {
			dmg += magnitude
		}
		if resist := mutations.GetMagicalResistance(target.Character.Mutations); resist > 0 {
			dmg = int(float64(dmg) * (1.0 - resist))
			if dmg < 1 {
				dmg = 1
			}
		}
		target.Character.Health -= dmg
		target.Character.CombatPosition = characters.PositionProne
		target.Character.PositionRoundsMin = 1
		target.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> slams you to the ground! (<ansi fg="damage">%s</ansi>)%s`,
			caster.Character.Name, spellData.Name,
			combat.GetDamageDescription(dmg, target.Character.HealthMax.Value), critTag))
		room.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> knocks <ansi fg="username">%s</ansi> to the ground!`,
			caster.Character.Name, spellData.Name, target.Character.Name), target.UserId)
		if target.Character.Aggro == nil {
			target.Character.Aggro = &characters.Aggro{MobInstanceId: caster.InstanceId}
		}
	case "buff":
		for _, buffId := range spellData.BuffIds {
			target.AddBuff(buffId, "spell")
		}
		target.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> takes effect on you!%s`,
			caster.Character.Name, spellData.Name, critTag))
		room.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> affects <ansi fg="username">%s</ansi>!`,
			caster.Character.Name, spellData.Name, target.Character.Name), target.UserId)
	default:
		target.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> takes effect on you.`,
			caster.Character.Name, spellData.Name))
	}
}
