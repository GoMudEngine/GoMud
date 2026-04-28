package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// calcSpellDuration computes a universal spell duration in rounds based on
// the spell's fold count, the caster's spellcasting skill, and willpower.
// Higher folds, skill, and willpower all extend duration.
// Formula: baseFolds × (10 + willpower/20 + spellcastingSkill/2)
func calcSpellDuration(baseFolds int, spellcastingSkill int, willpower int) int {
	if baseFolds < 1 {
		baseFolds = 4
	}
	duration := float64(baseFolds) * (10.0 + float64(willpower)/20.0 + float64(spellcastingSkill)/2.0)
	if duration < 10 {
		duration = 10
	}
	return int(math.Round(duration))
}

// resolveSpell is called when fold accumulation completes for a player caster.
// It dispatches to per-target resolution based on spell type and effect.
//
// Why this is NOT merged with resolveMobSpell:
//   - resolveSpell handles the "identify" spell type (no mob equivalent).
//   - HarmArea populates only mob targets for players; resolveMobSpell also
//     hits players in the room (mobs can cleave all occupants).
//   - HelpArea is player-only (mobs never cast area healing in this engine).
//   - Player targets go through resolveAgainstPlayer which has a help-spell
//     shortcut (TargetDefenseType == "") absent in the mob path.
//   - Post-resolution: player fires the onMagic script and consumes a
//     component; mob does neither.
//   - The per-target helpers (resolveAgainstMob vs resolveMobSpellAgainstMob,
//     resolveAgainstPlayer vs resolveMobSpellAgainstPlayer) have fundamentally
//     different signatures, messaging, and combat-record calls.
//
// Extracting the 6-line loop skeleton into a shared wrapper would require
// function-parameter callbacks or an interface, adding abstraction without
// meaningful savings. Keep them separate and well-documented instead.
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
		allMobs := room.GetMobs(rooms.FindAll)
		filtered := make([]int, 0, len(allMobs))
		for _, mId := range allMobs {
			// Don't damage any player-owned companions or non-combatants
			if m := mobs.GetInstance(mId); m != nil && (m.Character.IsCharmed() || m.IsNonCombatant()) {
				continue
			}
			filtered = append(filtered, mId)
		}
		cs.TargetMobInstanceIds = filtered
	}

	// --- Populate area targets for HelpArea ---
	if spellData.Type == spells.HelpArea {
		cs.TargetUserIds = room.GetPlayers(rooms.FindAll)
	}

	// --- Resolve against mob targets ---
	// castFumbled tracks whether ANY per-target roll fumbled (ZScore <= -2.0).
	// A fumble gates the post-target effects (summon, charm, Go hooks) below
	// so a summon-spell caster who fumbles doesn't still get the companion.
	castFumbled := false
	targetsResolved := 0
	for _, mobInstId := range cs.TargetMobInstanceIds {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil || mob.Character.Health < 1 {
			continue
		}
		if mob.Character.RoomId != room.RoomId {
			continue // target left the room before spell resolved
		}
		if resolveAgainstMob(user, mob, room, spellData, spellAttack, magnitude) {
			castFumbled = true
		}
		targetsResolved++
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
			if resolveAgainstPlayer(user, targetUser, room, spellData, spellAttack, magnitude) {
				castFumbled = true
			}
		}
		targetsResolved++
	}

	// --- Empty room / no valid targets feedback ---
	// Skip for summon/charm spells — they handle their own targeting via Go functions
	isSummonOrCharm := spellData != nil && (spellData.SummonMobId > 0 || spellData.EffectType == "charm")
	if targetsResolved == 0 && !isSummonOrCharm {
		user.SendText(`<ansi fg="cyan">Your spell erupts outward but finds no targets.</ansi>`)
		sendVisualRoomText(room, fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s spell crackles through the air harmlessly.`,
			user.Character.Name), user.UserId)
	}

	// --- Run spell script onMagic (if present) ---
	// Send YAML magic text (if defined).
	if spellData != nil && (spellData.MagicUserText != "" || spellData.MagicRoomText != "") {
		tCtx := textutil.TokenContext{
			SourceName:      user.Character.GetCharacterName(true),
			SourcePlainName: user.Character.GetCharacterName(false),
		}
		if len(cs.TargetUserIds) > 0 {
			if tUser := users.GetByUserId(cs.TargetUserIds[0]); tUser != nil {
				tCtx.TargetName = tUser.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tUser.Character.GetCharacterName(false)
			}
		} else if len(cs.TargetMobInstanceIds) > 0 {
			if tMob := mobs.GetInstance(cs.TargetMobInstanceIds[0]); tMob != nil {
				tCtx.TargetName = tMob.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tMob.Character.GetCharacterName(false)
			}
		}
		cfg := textutil.SendTextConfig{
			UserSendFunc: func(msg string) { user.SendText(msg) },
			RoomSendFunc: func(msg string, skip ...int) {
				if r := rooms.LoadRoom(user.Character.RoomId); r != nil {
					r.SendText(msg, skip...)
				}
			},
			ExcludeId: user.UserId,
		}
		textutil.SendPhaseText(spellData.MagicUserText, spellData.MagicRoomText, tCtx, "pink", cfg)
	}
	// Fumble gate for the post-target effects (summon / charm / Go hooks).
	// A fumbled cast consumed conviction + component but should NOT also land
	// the primary effect. A single flavor message; individual blocks skip
	// silently so we don't spam the player.
	if castFumbled && spellData != nil &&
		(spellData.SummonMobId > 0 || spellData.EffectType == "charm" ||
			cs.SpellId == "fold-anchor" || cs.SpellId == "fold-recall" || cs.SpellId == "purge-affliction") {
		user.SendText(`<ansi fg="red">The weave unravels — the spell fails to take shape.</ansi>`)
	}

	// Resolve companion summon (if configured)
	if !castFumbled && spellData != nil && spellData.SummonMobId > 0 {
		resolveCompanionSummon(user, spellData, cs.SpellRest, room)
	}
	// Resolve charm spell
	if !castFumbled && spellData != nil && spellData.EffectType == "charm" {
		if len(cs.TargetMobInstanceIds) > 0 {
			if targetMob := mobs.GetInstance(cs.TargetMobInstanceIds[0]); targetMob != nil {
				resolveCharmSpell(user, targetMob, room)
			}
		}
	}

	// --- Go spell hooks — dispatch before JS scripts ---
	// Fumble aborts the hook body but falls through to the component-consume
	// block below so the catalyst is still used up.
	if !castFumbled {
		switch cs.SpellId {
		case "fold-anchor":
			resolveFoldAnchor(actions.NewUserActorInRoom(user, room))
			return
		case "fold-recall":
			resolveFoldRecall(user)
			return
		case "purge-affliction":
			if len(cs.TargetUserIds) > 0 {
				if targetUser := users.GetByUserId(cs.TargetUserIds[0]); targetUser != nil {
					resolvePurgeAffliction(user, targetUser)
				}
			} else {
				resolvePurgeAffliction(user, user) // self-cast
			}
			return
		}
	}

	// --- Consume component if required ---
	if spellData.ComponentTag != "" {
		consumeSpellComponent(user, spellData.ComponentTag)
	}
}

// resolveAgainstMob performs the opposed roll and applies the effect to a mob.
// Returns true if the cast fumbled (ZScore <= -2.0). A fumble aborts any
// post-target spell effects (summon, charm, Go hooks) in the caller's main
// flow; component consumption still fires (the failed binding uses up the
// catalyst regardless).
func resolveAgainstMob(user *users.UserRecord, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, spellAttack float64, magnitude int) (fumbled bool) {

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
		sendVisualRoomText(room, fmt.Sprintf(
			`<ansi fg="red"><ansi fg="username">%s</ansi>'s spell backfires!</ansi>`, user.Character.Name), user.UserId)
		// Stage 30.1: Record backfire
		combat.RecordSpell(combat.User, combat.Mob, false, false, true, false, 0, atkRoll.ZScore, user.Character, &mob.Character, round)
		return true
	}

	if !success {
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">Your %s fizzles against %s.</ansi>`,
			spellData.Name, mobDisplayName(mob, room, user.UserId)))
		// Stage 30.1: Record fizzle
		combat.RecordSpell(combat.User, combat.Mob, false, false, false, true, 0, atkRoll.ZScore, user.Character, &mob.Character, round)
		return false
	}

	isCrit := atkRoll.ZScore >= 2.0
	dmgDealt := applyMobEffect(user, user.Character, mob, room, spellData, magnitude, isCrit)
	// Stage 30.1: Record spell hit with actual damage
	combat.RecordSpell(combat.User, combat.Mob, true, isCrit, false, false, dmgDealt, atkRoll.ZScore, user.Character, &mob.Character, round)
	return false
}

// setMobSpellAggro sets reciprocal aggro between the caster and the
// mob target immediately after a hostile spell lands.
//
// Note: applyMobEffect_buff does NOT call this helper — its aggro block
// is gated on spell Type being Harm*. Kept inline there.
func setMobSpellAggro(user *users.UserRecord, mob *mobs.Mob) {
	if mob.Character.Aggro == nil {
		mob.PreventIdle = true
		if user != nil {
			mob.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
		}
	}
	if user != nil && user.Character.Aggro == nil {
		user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
	}
}

// applyMobEffect_damage handles the "damage" EffectType case for applyMobEffect.
// Returns damage dealt to the mob.
func applyMobEffect_damage(
	user *users.UserRecord,
	casterChar *characters.Character,
	mob *mobs.Mob,
	room *rooms.Room,
	spellData *spells.SpellData,
	magnitude int,
	isCrit bool,
	critTag string,
	mName string,
) int {
	dmg := calcSpellDamageForCharacter(spellData, casterChar, &mob.Character, magnitude, isCrit)
	// Spell Deflection: defender attempts to partially deflect
	deflected := false
	critDeflect := false
	if !isCrit && casterChar != nil {
		deflectMult := combat.TrySpellDeflection(casterChar, &mob.Character, 0)
		if deflectMult < 1.0 {
			deflected = true
			if deflectMult == 0.0 {
				critDeflect = true
			}
			dmg = int(math.Round(float64(dmg) * deflectMult))
			if dmg < 1 && deflectMult > 0 {
				dmg = 1
			}
		}
	}
	mob.Character.Health -= dmg
	setMobSpellAggro(user, mob)
	if user != nil {
		if critDeflect {
			user.SendText(fmt.Sprintf(
				`<ansi fg="yellow">%s completely unravels your <ansi fg="cyan-bold">%s</ansi>!</ansi>`,
				mName, spellData.Name))
			sendVisualRoomText(room, fmt.Sprintf(
				`%s unravels <ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> completely!`,
				mName, user.Character.Name, spellData.Name), user.UserId)
		} else if deflected {
			user.SendText(fmt.Sprintf(
				`<ansi fg="yellow">%s partially deflects your <ansi fg="cyan-bold">%s</ansi>! (<ansi fg="damage">%s</ansi>)</ansi>`,
				mName, spellData.Name, combat.GetDamageDescription(dmg, mob.Character.HealthMax.Value)))
			sendVisualRoomText(room, fmt.Sprintf(
				`%s partially deflects <ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi>!`,
				mName, user.Character.Name, spellData.Name), user.UserId)
		} else {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> strikes %s! (<ansi fg="damage">%s</ansi>)%s</ansi>`,
				spellData.Name, mName, combat.GetDamageDescription(dmg, mob.Character.HealthMax.Value), critTag))
			sendVisualRoomText(room, fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes %s!`,
				user.Character.Name, spellData.Name, mName), user.UserId)
		}
	}
	return dmg
}

// applyMobEffect_dot handles the "dot" EffectType case for applyMobEffect.
// Returns 0 (no immediate damage; condition is applied for periodic ticks).
func applyMobEffect_dot(
	user *users.UserRecord,
	mob *mobs.Mob,
	room *rooms.Room,
	spellData *spells.SpellData,
	magnitude int,
	critTag string,
	mName string,
) int {
	casterSkill := 0
	casterWil := 100
	if user != nil {
		casterSkill = user.Character.GetSkillLevel(skills.Spellcasting)
		casterWil = user.Character.Stats.Willpower.ValueAdj
	}
	dotDuration := calcSpellDuration(spellData.BaseFolds, casterSkill, casterWil) / 3
	if dotDuration < 3 {
		dotDuration = 3
	}
	mob.Character.AddCondition(characters.ConditionPoisoned, dotDuration, float64(magnitude), "spell")
	setMobSpellAggro(user, mob)
	if user != nil {
		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> afflicts %s!%s</ansi>`,
			spellData.Name, mName, critTag))
		sendVisualRoomText(room, fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> afflicts %s!`,
			user.Character.Name, spellData.Name, mName), user.UserId)
	}
	return 0
}

// applyMobEffect_knockdown handles the "knockdown" EffectType case for applyMobEffect.
// Returns damage dealt to the mob.
func applyMobEffect_knockdown(
	user *users.UserRecord,
	casterChar *characters.Character,
	mob *mobs.Mob,
	room *rooms.Room,
	spellData *spells.SpellData,
	magnitude int,
	isCrit bool,
	critTag string,
	mName string,
) int {
	dmg := calcSpellDamageForCharacter(spellData, casterChar, &mob.Character, magnitude, isCrit)
	// Spell Deflection: defender attempts to partially deflect (damage only, knockdown still applies)
	kdDeflected := false
	if !isCrit && casterChar != nil {
		deflectMult := combat.TrySpellDeflection(casterChar, &mob.Character, 0)
		if deflectMult < 1.0 {
			kdDeflected = true
			dmg = int(math.Round(float64(dmg) * deflectMult))
			if dmg < 1 && deflectMult > 0 {
				dmg = 1
			}
		}
	}
	mob.Character.Health -= dmg
	mob.Character.CombatPosition = characters.PositionProne
	mob.Character.PositionRoundsMin = 1
	setMobSpellAggro(user, mob)
	if user != nil {
		if kdDeflected {
			user.SendText(fmt.Sprintf(
				`<ansi fg="yellow">%s partially deflects your <ansi fg="cyan-bold">%s</ansi>, but is knocked down! (<ansi fg="damage">%s</ansi>)</ansi>`,
				mName, spellData.Name, combat.GetDamageDescription(dmg, mob.Character.HealthMax.Value)))
		} else {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> slams %s to the ground! (<ansi fg="damage">%s</ansi>)%s</ansi>`,
				spellData.Name, mName, combat.GetDamageDescription(dmg, mob.Character.HealthMax.Value), critTag))
		}
		sendVisualRoomText(room, fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> knocks %s to the ground!`,
			user.Character.Name, spellData.Name, mName), user.UserId)
	}
	return dmg
}

func applyMobEffect_buff(
	user *users.UserRecord,
	mob *mobs.Mob,
	room *rooms.Room,
	spellData *spells.SpellData,
	critTag string,
	mName string,
) int {
	for _, buffId := range spellData.BuffIds {
		mob.AddBuff(buffId, "spell")
		// Compute tick snapshot for config-driven buffs
		if user != nil {
			if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil && buffSpec.TickPool != "" {
				skillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
				scalingMult := combat.SkillMultiplier(skillLevel)
				// Apply weapon spell damage multiplier if equipped
				if user.Character.Equipment.Weapon.ItemId > 0 {
					if weaponSpec := items.GetItemSpec(user.Character.Equipment.Weapon.ItemId); weaponSpec != nil && weaponSpec.SpellDamageMultiplier > 0 {
						scalingMult *= weaponSpec.SpellDamageMultiplier
					}
				}
				var maxPool int
				switch buffSpec.TickPool {
				case "health":
					maxPool = mob.Character.HealthMax.Value
				case "stamina":
					maxPool = mob.Character.StaminaMax.Value
				case "conviction":
					maxPool = mob.Character.ConvictionMax.Value
				}
				tickAmt := buffs.ComputeTickAmount(maxPool, buffSpec.TickPercent, buffSpec.TickVariance, buffSpec.TickMin, scalingMult)
				mob.Character.Buffs.SetTickAmount(buffId, tickAmt)
			}
		}
	}
	// Conditional aggro for harmful buff spells — kept inline because it is
	// gated on Harm* spell types; not consolidated in Task 7's setMobSpellAggro.
	if spellData.Type == spells.HarmSingle || spellData.Type == spells.HarmArea || spellData.Type == spells.HarmMulti {
		if mob.Character.Aggro == nil {
			mob.PreventIdle = true
			if user != nil {
				mob.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
			}
		}
		if user != nil && user.Character.Aggro == nil {
			user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
		}
	}
	if user != nil {
		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> takes effect on %s!%s</ansi>`,
			spellData.Name, mName, critTag))
		sendVisualRoomText(room, fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> affects %s!`,
			user.Character.Name, spellData.Name, mName), user.UserId)
	}
	return 0
}

func applyMobEffect_default(
	user *users.UserRecord,
	spellData *spells.SpellData,
	mName string,
) int {
	if user != nil {
		user.SendText(fmt.Sprintf(
			`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> takes effect on %s.</ansi>`,
			spellData.Name, mName))
	}
	return 0
}

// applyMobEffect applies the spell effect to a mob and returns damage dealt (0 for non-damage effects).
// user may be nil when the caster is a mob (guards all user.* references).
// casterChar is the caster's Character pointer (may be nil for mob-on-mob when unavailable).
func applyMobEffect(user *users.UserRecord, casterChar *characters.Character, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int, isCrit bool) int {
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
		return applyMobEffect_damage(user, casterChar, mob, room, spellData, magnitude, isCrit, critTag, mName)
	case "dot":
		return applyMobEffect_dot(user, mob, room, spellData, magnitude, critTag, mName)
	case "knockdown":
		return applyMobEffect_knockdown(user, casterChar, mob, room, spellData, magnitude, isCrit, critTag, mName)
	case "buff":
		return applyMobEffect_buff(user, mob, room, spellData, critTag, mName)
	default:
		return applyMobEffect_default(user, spellData, mName)
	}
}

// resolveAgainstPlayer performs the opposed roll and applies the effect to a player.
// Returns true if the cast fumbled (ZScore <= -2.0). See resolveAgainstMob for
// the fumble semantics carrying over to summon/charm/Go-hook gating.
func resolveAgainstPlayer(user *users.UserRecord, target *users.UserRecord, room *rooms.Room, spellData *spells.SpellData, spellAttack float64, magnitude int) (fumbled bool) {

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
		sendVisualRoomText(room, fmt.Sprintf(
			`<ansi fg="red"><ansi fg="username">%s</ansi>'s spell backfires!</ansi>`, user.Character.Name), user.UserId)
		return true
	}

	if !success {
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">Your %s fizzles against <ansi fg="username">%s</ansi>.</ansi>`,
			spellData.Name, target.Character.Name))
		return false
	}

	isCrit := atkRoll.ZScore >= 2.0
	applyPlayerEffect(user, target, room, spellData, magnitude, isCrit)

	// Crit received → stat progression for the defender
	if isCrit && (spellData.Type == spells.HarmSingle || spellData.Type == spells.HarmArea || spellData.Type == spells.HarmMulti) {
		// Determine damage channel from spell effect
		switch spellData.EffectType {
		case "damage":
			target.Character.OnCritReceived("magical", target.UserId)
		}
	}

	// Set reciprocal aggro for harm spells
	if spellData.Type == spells.HarmSingle || spellData.Type == spells.HarmArea || spellData.Type == spells.HarmMulti {
		if user.Character.Aggro == nil {
			user.Character.SetAggro(target.UserId, 0, characters.DefaultAttack)
		}
		if target.Character.Aggro == nil {
			target.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
		}
	}
	return false
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
		deflected := false
		critDeflect := false
		if !isCrit {
			deflectMult := combat.TrySpellDeflection(
				user.Character, target.Character, target.UserId)
			if deflectMult < 1.0 {
				deflected = true
				if deflectMult == 0.0 {
					critDeflect = true
				}
				dmg = int(math.Round(float64(dmg) * deflectMult))
				if dmg < 1 && deflectMult > 0 {
					dmg = 1
				}
			}
		}
		if critDeflect {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green">You read <ansi fg="username">%s</ansi>'s spell perfectly `+
					`and unravel it before it reaches you!</ansi>`,
				user.Character.Name))
			user.SendText(fmt.Sprintf(
				`<ansi fg="yellow"><ansi fg="username">%s</ansi> completely unravels `+
					`your spell!</ansi>`,
				target.Character.Name))
			sendVisualRoomText(room, fmt.Sprintf(
				`<ansi fg="username">%s</ansi> unravels <ansi fg="username">%s</ansi>'s `+
					`spell completely!`,
				target.Character.Name, user.Character.Name), user.UserId, target.UserId)
			return
		}
		target.Character.Health -= dmg
		dmgDesc := combat.GetDamageDescription(dmg, target.Character.HealthMax.Value)
		if deflected {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green">You partially deflect `+
					`<ansi fg="username">%s</ansi>'s `+
					`<ansi fg="cyan-bold">%s</ansi>! `+
					`(<ansi fg="damage">%s</ansi>)</ansi>`,
				user.Character.Name, spellData.Name, dmgDesc))
			user.SendText(fmt.Sprintf(
				`<ansi fg="yellow"><ansi fg="username">%s</ansi> partially deflects `+
					`your <ansi fg="cyan-bold">%s</ansi>! `+
					`(<ansi fg="damage">%s</ansi>)</ansi>`,
				target.Character.Name, spellData.Name, dmgDesc))
			sendVisualRoomText(room, fmt.Sprintf(
				`<ansi fg="username">%s</ansi> partially deflects `+
					`<ansi fg="username">%s</ansi>'s `+
					`<ansi fg="cyan">%s</ansi>!`,
				target.Character.Name, user.Character.Name, spellData.Name),
				user.UserId, target.UserId)
		} else {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">Your <ansi fg="cyan-bold">%s</ansi> strikes `+
					`<ansi fg="username">%s</ansi>! `+
					`(<ansi fg="damage">%s</ansi>)%s</ansi>`,
				spellData.Name, target.Character.Name, dmgDesc, critTag))
			sendVisualRoomText(room, fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes `+
					`<ansi fg="username">%s</ansi>!`,
				user.Character.Name, spellData.Name, target.Character.Name),
				user.UserId, target.UserId)
			target.SendText(fmt.Sprintf(
				`<ansi fg="red"><ansi fg="username">%s</ansi>'s `+
					`<ansi fg="cyan-bold">%s</ansi> strikes you! `+
					`(<ansi fg="damage">%s</ansi>)</ansi>`,
				user.Character.Name, spellData.Name,
				combat.GetDamageDescription(dmg, target.Character.HealthMax.Value)))
		}

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
		sendVisualRoomText(room, fmt.Sprintf(
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
		durationRounds := calcSpellDuration(spellData.BaseFolds, skillLevel, user.Character.Stats.Willpower.ValueAdj) / 2
		if durationRounds < 6 {
			durationRounds = 6
		}
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
		sendVisualRoomText(room, fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="cyan">%s</ansi> envelops <ansi fg="username">%s</ansi> in healing light.`,
			user.Character.Name, spellData.Name, target.Character.Name), user.UserId, target.UserId)

	case "buff":
		for _, buffId := range spellData.BuffIds {
			target.AddBuff(buffId, "spell")
			// Compute tick snapshot for config-driven buffs
			if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil && buffSpec.TickPool != "" {
				skillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
				scalingMult := combat.SkillMultiplier(skillLevel)
				// Apply weapon spell damage multiplier if equipped
				if user.Character.Equipment.Weapon.ItemId > 0 {
					if weaponSpec := items.GetItemSpec(user.Character.Equipment.Weapon.ItemId); weaponSpec != nil && weaponSpec.SpellDamageMultiplier > 0 {
						scalingMult *= weaponSpec.SpellDamageMultiplier
					}
				}
				var maxPool int
				switch buffSpec.TickPool {
				case "health":
					maxPool = target.Character.HealthMax.Value
				case "stamina":
					maxPool = target.Character.StaminaMax.Value
				case "conviction":
					maxPool = target.Character.ConvictionMax.Value
				}
				tickAmt := buffs.ComputeTickAmount(maxPool, buffSpec.TickPercent, buffSpec.TickVariance, buffSpec.TickMin, scalingMult)
				target.Character.Buffs.SetTickAmount(buffId, tickAmt)
			}
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
		// Scale shield strength by spell magnitude (100 = 1.0x baseline)
		if magnitude > 0 {
			shieldBonus = int(math.Round(float64(shieldBonus) * float64(magnitude) / 100.0))
			if shieldBonus < 1 {
				shieldBonus = 1
			}
		}
		duration := calcSpellDuration(spellData.BaseFolds, skillLevel, user.Character.Stats.Willpower.ValueAdj)
		if isCrit {
			shieldBonus = int(float64(shieldBonus) * 1.5)
		}
		target.Character.AddCondition(characters.ConditionShield, duration, float64(shieldBonus), "spell")
		target.SendText(`<ansi fg="cyan">A shimmering magical barrier forms around you, bolstering your defenses.</ansi>`)
		if target.UserId != user.UserId {
			user.SendText(fmt.Sprintf(
				`<ansi fg="cyan">A shimmering magical barrier forms around <ansi fg="username">%s</ansi>, bolstering their defenses.</ansi>`,
				target.Character.Name))
		}
		sendVisualRoomText(room, fmt.Sprintf(
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
// resolveMobSpell is called when fold accumulation completes for a mob caster.
// It dispatches to per-target resolution based on spell type and effect.
//
// Why this is NOT merged with resolveSpell (see that function for details):
//   - HarmArea here populates both mob AND player targets; player casters only
//     hit mobs (players in the room are excluded from player-cast area spells).
//   - Mob targets include a self-cast branch (applyMobSelfEffect) for help
//     spells; player casters never self-target via this dispatcher.
//   - No onMagic script, no component consumption.
//   - Per-target helpers are entirely separate from the player equivalents.
func resolveMobSpell(mob *mobs.Mob, cs *characters.CastingState, spellData *spells.SpellData, room *rooms.Room) {
	skillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
	spellAttack := characters.CalcSpellAttack(mob.Character.Stats.Willpower.ValueAdj, skillLevel)
	magnitude := spellData.EffectMagnitude

	if spellData.Type == spells.HarmArea {
		allMobs := room.GetMobs(rooms.FindAll)
		filtered := make([]int, 0, len(allMobs))
		charmedByUserId := mob.Character.GetCharmedUserId()
		for _, mId := range allMobs {
			if mId == mob.InstanceId {
				continue // don't target self
			}
			// If this mob is charmed by a player, don't hit that player's other companions
			// Also never hit non-combatant mobs (shopkeepers etc.)
			if m := mobs.GetInstance(mId); m != nil {
				if m.IsNonCombatant() {
					continue
				}
				if charmedByUserId > 0 && m.Character.IsCharmed(charmedByUserId) {
					continue
				}
			}
			filtered = append(filtered, mId)
		}
		cs.TargetMobInstanceIds = filtered
		cs.TargetUserIds = room.GetPlayers(rooms.FindAll)
		// If charmed, don't hit the owner
		if charmedByUserId > 0 {
			ownerFiltered := make([]int, 0, len(cs.TargetUserIds))
			for _, pId := range cs.TargetUserIds {
				if pId != charmedByUserId {
					ownerFiltered = append(ownerFiltered, pId)
				}
			}
			cs.TargetUserIds = ownerFiltered
		}
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
		durationRounds := calcSpellDuration(spellData.BaseFolds, skillLevel, mob.Character.Stats.Willpower.ValueAdj) / 2
		if durationRounds < 6 {
			durationRounds = 6
		}
		mob.Character.AddCondition(characters.ConditionRegen, durationRounds, regenMult, "heal spell")
		sendVisualRoomText(room, fmt.Sprintf(
			`%s channels restorative magic.`, mobDisplayName(mob, room, 0)))
	case "buff":
		for _, buffId := range spellData.BuffIds {
			mob.AddBuff(buffId, "spell")
			// Compute tick snapshot for config-driven buffs (matches
			// applyMobEffect_buff for consistency across all caster paths).
			if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil && buffSpec.TickPool != "" {
				skillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
				scalingMult := combat.SkillMultiplier(skillLevel)
				var maxPool int
				switch buffSpec.TickPool {
				case "health":
					maxPool = mob.Character.HealthMax.Value
				case "stamina":
					maxPool = mob.Character.StaminaMax.Value
				case "conviction":
					maxPool = mob.Character.ConvictionMax.Value
				}
				tickAmt := buffs.ComputeTickAmount(maxPool, buffSpec.TickPercent, buffSpec.TickVariance, buffSpec.TickMin, scalingMult)
				mob.Character.Buffs.SetTickAmount(buffId, tickAmt)
			}
		}
	case "shield":
		skillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
		weightedSkill := int(math.Round(float64(skillLevel) * float64(configs.GetBalanceConfig().SkillWeight)))
		shieldBonus := (mob.Character.Stats.Willpower.ValueAdj + weightedSkill) / 3
		if shieldBonus < 1 {
			shieldBonus = 1
		}
		// Scale shield strength by spell magnitude (100 = 1.0x baseline)
		if magnitude > 0 {
			shieldBonus = int(math.Round(float64(shieldBonus) * float64(magnitude) / 100.0))
			if shieldBonus < 1 {
				shieldBonus = 1
			}
		}
		duration := calcSpellDuration(spellData.BaseFolds, skillLevel, mob.Character.Stats.Willpower.ValueAdj)
		mob.Character.AddCondition(characters.ConditionShield, duration, float64(shieldBonus), "spell")
		sendVisualRoomText(room, fmt.Sprintf(
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
		sendVisualRoomText(room, fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s spell backfires!`, caster.Character.Name))
		return
	}
	if !success {
		return
	}
	applyMobEffect(nil, &caster.Character, target, room, spellData, magnitude, atkRoll.ZScore >= 2.0)
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
		sendVisualRoomText(room, fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s spell backfires!`, caster.Character.Name))
		// Stage 30.1: Record backfire
		combat.RecordSpell(combat.Mob, combat.User, false, false, true, false, 0, atkRoll.ZScore, &caster.Character, target.Character, round)
		return
	}
	if !success {
		sendVisualRoomText(room, fmt.Sprintf(
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
		deflected := false
		critDeflect := false
		if !isCrit {
			deflectMult := combat.TrySpellDeflection(
				&caster.Character, target.Character, target.UserId)
			if deflectMult < 1.0 {
				deflected = true
				if deflectMult == 0.0 {
					critDeflect = true
				}
				dmg = int(math.Round(float64(dmg) * deflectMult))
				if dmg < 1 && deflectMult > 0 {
					dmg = 1
				}
			}
		}
		if critDeflect {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green">You read `+
					`<ansi fg="mobname">%s</ansi>'s spell perfectly `+
					`and unravel it before it reaches you!</ansi>`,
				caster.Character.Name))
			sendVisualRoomText(room, fmt.Sprintf(
				`<ansi fg="username">%s</ansi> unravels `+
					`<ansi fg="mobname">%s</ansi>'s spell completely!`,
				target.Character.Name, caster.Character.Name), target.UserId)
			break
		}
		mobSpellDmg = dmg
		target.Character.Health -= dmg
		if deflected {
			target.SendText(fmt.Sprintf(
				`<ansi fg="green">You partially deflect `+
					`<ansi fg="mobname">%s</ansi>'s `+
					`<ansi fg="cyan">%s</ansi>! `+
					`(<ansi fg="damage">%s</ansi>)</ansi>`,
				caster.Character.Name, spellData.Name,
				combat.GetDamageDescription(dmg, target.Character.HealthMax.Value)))
			sendVisualRoomText(room, fmt.Sprintf(
				`<ansi fg="username">%s</ansi> partially deflects `+
					`<ansi fg="mobname">%s</ansi>'s `+
					`<ansi fg="cyan">%s</ansi>!`,
				target.Character.Name, caster.Character.Name, spellData.Name),
				target.UserId)
		} else {
			target.SendText(fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> `+
					`strikes you! (<ansi fg="damage">%s</ansi>)%s`,
				caster.Character.Name, spellData.Name,
				combat.GetDamageDescription(dmg, target.Character.HealthMax.Value), critTag))
			sendVisualRoomText(room, fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> strikes `+
					`<ansi fg="username">%s</ansi>!`,
				caster.Character.Name, spellData.Name, target.Character.Name), target.UserId)
		}
		if target.Character.Aggro == nil {
			target.Character.Aggro = &characters.Aggro{MobInstanceId: caster.InstanceId}
		}
		// Magical crit received → willpower progression for defender
		if isCrit {
			target.Character.OnCritReceived("magical", target.UserId)
		}
	case "dot":
		dotDuration := calcSpellDuration(spellData.BaseFolds, caster.Character.GetSkillLevel(skills.Spellcasting), caster.Character.Stats.Willpower.ValueAdj) / 3
		if dotDuration < 3 {
			dotDuration = 3
		}
		target.Character.AddCondition(characters.ConditionPoisoned, dotDuration, float64(magnitude), "spell")
		target.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> afflicts you!%s`,
			caster.Character.Name, spellData.Name, critTag))
		sendVisualRoomText(room, fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> afflicts <ansi fg="username">%s</ansi>!`,
			caster.Character.Name, spellData.Name, target.Character.Name), target.UserId)
		if target.Character.Aggro == nil {
			target.Character.Aggro = &characters.Aggro{MobInstanceId: caster.InstanceId}
		}
	case "knockdown":
		dmg := calcSpellDamageForCharacter(spellData, &caster.Character, target.Character, magnitude, isCrit)
		if !isCrit {
			deflectMult := combat.TrySpellDeflection(
				&caster.Character, target.Character, target.UserId)
			if deflectMult < 1.0 {
				dmg = int(math.Round(float64(dmg) * deflectMult))
				if dmg < 1 && deflectMult > 0 {
					dmg = 1
				}
			}
		}
		mobSpellDmg = dmg
		target.Character.Health -= dmg
		target.Character.CombatPosition = characters.PositionProne
		target.Character.PositionRoundsMin = 1
		target.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> slams you `+
				`to the ground! (<ansi fg="damage">%s</ansi>)%s`,
			caster.Character.Name, spellData.Name,
			combat.GetDamageDescription(dmg, target.Character.HealthMax.Value), critTag))
		sendVisualRoomText(room, fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> knocks `+
				`<ansi fg="username">%s</ansi> to the ground!`,
			caster.Character.Name, spellData.Name, target.Character.Name), target.UserId)
		if target.Character.Aggro == nil {
			target.Character.Aggro = &characters.Aggro{MobInstanceId: caster.InstanceId}
		}
	case "buff":
		for _, buffId := range spellData.BuffIds {
			target.AddBuff(buffId, "spell")
		}
		// Set aggro for harmful buff spells
		if spellData.Type == spells.HarmSingle || spellData.Type == spells.HarmArea || spellData.Type == spells.HarmMulti {
			if target.Character.Aggro == nil {
				target.Character.Aggro = &characters.Aggro{MobInstanceId: caster.InstanceId}
			}
		}
		target.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s <ansi fg="cyan">%s</ansi> takes effect on you!%s`,
			caster.Character.Name, spellData.Name, critTag))
		sendVisualRoomText(room, fmt.Sprintf(
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
	sendVisualRoomText(room, 
		fmt.Sprintf(
			`<ansi fg="username">%s</ansi> concentrates on their <ansi fg="item">%s</ansi>...`,
			user.Character.Name, matchItem.DisplayName()),
		user.UserId,
	)

	identifyTxt, _ := templates.Process("descriptions/identify", details, user.UserId)
	user.SendText(identifyTxt)
}
