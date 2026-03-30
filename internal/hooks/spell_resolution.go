package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// resolveSpell is called when fold accumulation completes.
// It dispatches to per-target resolution based on spell type and effect.
func resolveSpell(user *users.UserRecord, cs *characters.CastingState, spellData *spells.SpellData, room *rooms.Room) {

	skillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
	spellAttack := characters.CalcSpellAttack(user.Character.Stats.Willpower.ValueAdj, skillLevel)
	magnitude := spellData.EffectMagnitude

	// --- Identify: resolve against caster's item, no targets ---
	if spellData.EffectType == "identify" {
		resolveIdentify(user, cs.SpellRest, room)
		return
	}

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
		if mob.Character.RoomId != room.RoomId {
			continue // target left the room before spell resolved
		}
		resolveAgainstMob(user, mob, room, spellData, spellAttack, magnitude)
	}

	// --- Resolve against player targets ---
	for _, targetUserId := range cs.TargetUserIds {
		targetUser := users.GetByUserId(targetUserId)
		if targetUser == nil {
			continue
		}
		if targetUser.Character.RoomId != room.RoomId {
			user.SendText(fmt.Sprintf(`Your spell fizzles — <ansi fg="username">%s</ansi> is no longer here.`, targetUser.Character.Name))
			continue // target left the room before spell resolved
		}
		// Skip downed players for harm spells — they're already down.
		if targetUser.Character.Health < 1 &&
			(spellData.Type == spells.HarmSingle || spellData.Type == spells.HarmArea || spellData.Type == spells.HarmMulti) {
			continue
		}
		if spellData.TargetDefenseType == "" {
			// Help spell with no defense — always applies
			applyPlayerEffect(user, targetUser, room, spellData, magnitude, false)
		} else {
			resolveAgainstPlayer(user, targetUser, room, spellData, spellAttack, magnitude)
		}
	}

	// --- Run spell script onMagic (if present) ---
	spellAggro := characters.SpellAggroInfo{
		SpellId:              cs.SpellId,
		SpellRest:            cs.SpellRest,
		TargetUserIds:        cs.TargetUserIds,
		TargetMobInstanceIds: cs.TargetMobInstanceIds,
	}
	scripting.TrySpellScriptEvent("onMagic", user.UserId, 0, spellAggro)

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
			`<ansi fg="yellow">Your %s fizzles against %s.</ansi>`,
			spellData.Name, mobDisplayName(mob, room, user.UserId)))
		// Stage 30.1: Record fizzle
		combat.RecordSpell(combat.User, combat.Mob, false, false, false, true, 0, atkRoll.ZScore, user.Character, &mob.Character, round)
		return
	}

	isCrit := atkRoll.ZScore >= 2.0
	dmgDealt := applyMobEffect(user, mob, room, spellData, magnitude, isCrit)
	// Stage 30.1: Record spell hit with actual damage
	combat.RecordSpell(combat.User, combat.Mob, true, isCrit, false, false, dmgDealt, atkRoll.ZScore, user.Character, &mob.Character, round)
}

// applyMobEffect applies the spell effect to a mob and returns damage dealt (0 for non-damage effects).
// user may be nil when the caster is a mob (guards all user.* references).
func applyMobEffect(user *users.UserRecord, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int, isCrit bool) int {

	dmgDealt := 0

	critTag := ""
	if isCrit {
		critTag = ` <ansi fg="yellow">[CRIT!]</ansi>`
	}

	viewerId := 0
	if user != nil {
		viewerId = user.UserId
	}
	mName := mobDisplayName(mob, room, viewerId)

	switch spellData.EffectType {
	case "damage":
		var casterChar *characters.Character
		if user != nil {
			casterChar = user.Character
		}
		dmg := calcSpellDamageForCharacter(spellData, casterChar, &mob.Character, magnitude, isCrit)
		dmgDealt = dmg
		mob.Character.Health -= dmg
		// Set aggro on both sides immediately
		if mob.Character.Aggro == nil {
			mob.PreventIdle = true
			if user != nil {
				mob.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
			}
		}
		if user != nil && user.Character.Aggro == nil {
			user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
		}
		if user != nil {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> strikes %s! (<ansi fg="damage">%s</ansi>)%s</ansi>`,
				spellData.Name, mName, combat.GetDamageDescription(dmg, mob.Character.HealthMax.Value), critTag))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes %s!`,
				user.Character.Name, spellData.Name, mName), user.UserId)
		}

	case "dot":
		dotDuration := spellData.EffectDuration
		if dotDuration < 1 {
			dotDuration = 3
		}
		// Duration is in AutoHeal ticks (every 3 rounds), so multiply by 3 for round count
		mob.Character.AddCondition(characters.ConditionPoisoned, dotDuration*3, float64(magnitude), "spell")
		// Set aggro on both sides immediately
		if mob.Character.Aggro == nil {
			mob.PreventIdle = true
			if user != nil {
				mob.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
			}
		}
		if user != nil && user.Character.Aggro == nil {
			user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
		}
		if user != nil {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> afflicts %s!%s</ansi>`,
				spellData.Name, mName, critTag))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> afflicts %s!`,
				user.Character.Name, spellData.Name, mName), user.UserId)
		}

	case "knockdown":
		var casterChar2 *characters.Character
		if user != nil {
			casterChar2 = user.Character
		}
		dmg := calcSpellDamageForCharacter(spellData, casterChar2, &mob.Character, magnitude, isCrit)
		dmgDealt = dmg
		mob.Character.Health -= dmg
		mob.Character.CombatPosition = characters.PositionProne
		mob.Character.PositionRoundsMin = 1
		// Set aggro on both sides immediately
		if mob.Character.Aggro == nil {
			mob.PreventIdle = true
			if user != nil {
				mob.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
			}
		}
		if user != nil && user.Character.Aggro == nil {
			user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
		}
		if user != nil {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> slams %s to the ground! (<ansi fg="damage">%s</ansi>)%s</ansi>`,
				spellData.Name, mName, combat.GetDamageDescription(dmg, mob.Character.HealthMax.Value), critTag))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> knocks %s to the ground!`,
				user.Character.Name, spellData.Name, mName), user.UserId)
		}

	case "buff":
		for _, buffId := range spellData.BuffIds {
			mob.AddBuff(buffId, "spell")
		}
		if user != nil {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> takes effect on %s!%s</ansi>`,
				spellData.Name, mName, critTag))
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> affects %s!`,
				user.Character.Name, spellData.Name, mName), user.UserId)
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
					`<ansi fg="red">%s cannot be tamed — it is not a wild animal.</ansi>`,
					mName))
			}
			return 0
		}
		if user != nil {
			mob.Character.Charm(user.UserId, 24, "")
			mob.Character.Aggro = nil
			user.Character.TrackCharmed(mob.InstanceId, true)
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">%s calms and becomes your companion!</ansi>`,
				mName))
			room.SendText(fmt.Sprintf(
				`%s becomes docile and follows <ansi fg="username">%s</ansi>.`,
				mName, user.Character.Name), user.UserId)
		}

	default:
		if user != nil {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> takes effect on %s.</ansi>`,
				spellData.Name, mName))
		}
	}
	return dmgDealt
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

	// Set reciprocal aggro for harm spells
	if spellData.Type == spells.HarmSingle || spellData.Type == spells.HarmArea || spellData.Type == spells.HarmMulti {
		if user.Character.Aggro == nil {
			user.Character.SetAggro(target.UserId, 0, characters.DefaultAttack)
		}
		if target.Character.Aggro == nil {
			target.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
		}
	}
}

// applyPlayerEffect applies the spell effect to a player target.
func applyPlayerEffect(user *users.UserRecord, target *users.UserRecord, room *rooms.Room, spellData *spells.SpellData, magnitude int, isCrit bool) {

	critTag := ""
	if isCrit {
		critTag = ` <ansi fg="yellow">[CRIT!]</ansi>`
	}

	switch spellData.EffectType {
	case "damage":
		dmg := calcSpellDamageForCharacter(spellData, user.Character, target.Character, magnitude, isCrit)
		target.Character.Health -= dmg
		dmgDesc := combat.GetDamageDescription(dmg, target.Character.HealthMax.Value)
		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> strikes <ansi fg="username">%s</ansi>! (<ansi fg="damage">%s</ansi>)%s</ansi>`,
			spellData.Name, target.Character.Name, dmgDesc, critTag))
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes <ansi fg="username">%s</ansi>!`,
			user.Character.Name, spellData.Name, target.Character.Name), user.UserId, target.UserId)
		target.SendText(fmt.Sprintf(
			`<ansi fg="red"><ansi fg="username">%s</ansi>'s <ansi fg="cyan-bold">%s</ansi> strikes you! (<ansi fg="damage">%s</ansi>)</ansi>`,
			user.Character.Name, spellData.Name, combat.GetDamageDescription(dmg, target.Character.HealthMax.Value)))

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
		durationRounds := ticks * 6 // TickConditions runs every combat round; AutoHeal fires every 3
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
		weightedSkill := int(math.Round(float64(skillLevel) * float64(configs.GetBalanceConfig().SkillWeight)))
		shieldBonus := (user.Character.Stats.Willpower.ValueAdj + weightedSkill) / 3
		if shieldBonus < 1 {
			shieldBonus = 1
		}
		duration := 20 + int(math.Round(float64(skillLevel)*2/5))
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

// calcSpellDamage and calcMobSpellDamage have been unified into
// calcSpellDamageForCharacter() in combat_shared_helpers.go (Stage 38.1).

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
		if target := mobs.GetInstance(mobInstId); target != nil && target.Character.Health > 0 && target.Character.RoomId == room.RoomId {
			resolveMobSpellAgainstMob(mob, target, room, spellData, spellAttack, magnitude)
		}
	}
	for _, userId := range cs.TargetUserIds {
		if target := users.GetByUserId(userId); target != nil && target.Character.RoomId == room.RoomId {
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
		durationRounds := ticks * 6
		mob.Character.AddCondition(characters.ConditionRegen, durationRounds, regenMult, "heal spell")
		room.SendText(fmt.Sprintf(
			`%s channels restorative magic.`, mobDisplayName(mob, room, 0)))
	case "shield":
		skillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
		weightedSkill := int(math.Round(float64(skillLevel) * float64(configs.GetBalanceConfig().SkillWeight)))
		shieldBonus := (mob.Character.Stats.Willpower.ValueAdj + weightedSkill) / 3
		if shieldBonus < 1 {
			shieldBonus = 1
		}
		duration := 20 + int(math.Round(float64(skillLevel)*2/5))
		mob.Character.AddCondition(characters.ConditionShield, duration, float64(shieldBonus), "spell")
		room.SendText(fmt.Sprintf(
			`A shimmering barrier forms around %s.`, mobDisplayName(mob, room, 0)))
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
	mobSpellDmg := 0
	critTag := ""
	if isCrit {
		critTag = ` <ansi fg="yellow">[CRIT!]</ansi>`
	}
	switch spellData.EffectType {
	case "damage":
		dmg := calcSpellDamageForCharacter(spellData, &caster.Character, target.Character, magnitude, isCrit)
		mobSpellDmg = dmg
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
		dmg := calcSpellDamageForCharacter(spellData, &caster.Character, target.Character, magnitude, isCrit)
		mobSpellDmg = dmg
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
	// Stage 30.1: Record spell hit with actual damage
	combat.RecordSpell(combat.Mob, combat.User, true, isCrit, false, false, mobSpellDmg, atkRoll.ZScore, &caster.Character, target.Character, round)
}

// resolveIdentify finds the named item on the caster and renders
// the identify template with descriptive item properties.
func resolveIdentify(user *users.UserRecord, itemName string, room *rooms.Room) {

	if itemName == "" {
		user.SendText("Identify what? (Usage: cast identify <item>)")
		return
	}

	// Search backpack and equipped items as a single pool
	matchItem, _, found := user.Character.FindItem(itemName)

	if !found {
		user.SendText("You can't seem to identify that.")
		return
	}

	iSpec := matchItem.GetSpec()

	type identifyDetails struct {
		Item     *items.Item
		ItemSpec *items.ItemSpec
	}

	details := identifyDetails{
		Item:     &matchItem,
		ItemSpec: &iSpec,
	}

	user.SendText(
		fmt.Sprintf(`You concentrate on the <ansi fg="item">%s</ansi>...`,
			matchItem.DisplayName()),
	)
	room.SendText(
		fmt.Sprintf(
			`<ansi fg="username">%s</ansi> concentrates on their <ansi fg="item">%s</ansi>...`,
			user.Character.Name, matchItem.DisplayName()),
		user.UserId,
	)

	identifyTxt, _ := templates.Process("descriptions/identify", details, user.UserId)
	user.SendText(identifyTxt)
}
