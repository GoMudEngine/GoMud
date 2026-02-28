package combat

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"strings"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// weaponSetup holds pre-computed weapon info for a single weapon swing.
type weaponSetup struct {
	weapon        items.Item
	weaponName    string
	weaponSubType items.ItemSubType
	attacks       int
	baseDmg       float64
	dmgVariance   float64
	critBuffs     []int
	weaponSpeed   float64
	weaponDmgMult float64
	isOffhand     bool
	penalty       int // dual wield penalty
}

// swingDamageParams holds per-swing damage values after pipeline calculations.
type swingDamageParams struct {
	dmgMean       float64
	dmgVariance   float64
	rawDmgForCrit float64
	critBuffs     []int
	msgSeed       int
}

// bestDefenseResult holds the outcome of best-of-all defense resolution.
type bestDefenseResult struct {
	margin       float64
	defenseType  string
	hitRoll      dice.RollResult
	defRoll      dice.RollResult
	defenseFloor bool // true if defense succeeded via floor save
}

// calcAttackCount computes the number of attack passes for a combat round.
func calcAttackCount(sourceChar *characters.Character, extraAttacks int) int {
	attackCount := 1 + int(sourceChar.Stats.Dexterity.ValueAdj/50)
	if attackCount < 1 {
		attackCount = 1
	}

	attackCount += extraAttacks

	// Apply smooth stamina-based attack count penalty
	spPenalty := float64(configs.GetBalanceConfig().StaminaPenaltyMax)
	attackCount = int(math.Ceil(float64(attackCount) *
		ResourceMultiplier(sourceChar.Stamina, sourceChar.StaminaMax.Value, spPenalty)))
	if attackCount < 1 {
		attackCount = 1
	}

	// Apply encumbrance penalty to attack count (weight-based)
	carriedWeight := sourceChar.GetCarriedWeight()
	capacity := sourceChar.CarryCapacity()
	if carriedWeight > capacity {
		overAmount := carriedWeight - capacity
		overRatio := overAmount / capacity
		encumbrancePenalty := math.Min(overRatio*0.5, 0.5)
		attackCount = int(math.Ceil(float64(attackCount) * (1.0 - encumbrancePenalty)))
		if attackCount < 1 {
			attackCount = 1
		}
	}

	return attackCount
}

// collectAttackWeapons gathers all weapons the character can attack with.
func collectAttackWeapons(sourceChar *characters.Character) []items.Item {
	attackWeapons := []items.Item{}

	if sourceChar.Equipment.Weapon.ItemId > 0 {
		attackWeapons = append(attackWeapons, sourceChar.Equipment.Weapon)
	}

	if sourceChar.Equipment.Offhand.ItemId > 0 && sourceChar.Equipment.Offhand.GetSpec().Type == items.Weapon {
		attackWeapons = append(attackWeapons, sourceChar.Equipment.Offhand)
	}

	// Extra arm weapons (from extra-arms mutation)
	if sourceChar.ExtraArms >= 1 && sourceChar.Equipment.ExtraArm1.ItemId > 0 && sourceChar.Equipment.ExtraArm1.GetSpec().Type == items.Weapon {
		attackWeapons = append(attackWeapons, sourceChar.Equipment.ExtraArm1)
	}
	if sourceChar.ExtraArms >= 2 && sourceChar.Equipment.ExtraArm2.ItemId > 0 && sourceChar.Equipment.ExtraArm2.GetSpec().Type == items.Weapon {
		attackWeapons = append(attackWeapons, sourceChar.Equipment.ExtraArm2)
	}

	// Put an empty weapon, so basically hands.
	if len(attackWeapons) == 0 {
		attackWeapons = append(attackWeapons, items.Item{
			ItemId: 0,
		})
	}

	return attackWeapons
}

// calcDualWieldPenalty computes the hit penalty for dual wielding.
func calcDualWieldPenalty(sourceChar *characters.Character, weapIdx, totalWeaps int) int {
	if totalWeaps <= 1 {
		return 0
	}

	dualWieldLevel := sourceChar.GetSkillLevel(skills.WeaponCombat)

	penalty := 0
	// Natural weapons (claws) ignore penalty
	if sourceChar.Equipment.Weapon.GetSpec().Subtype == items.Claws && sourceChar.Equipment.Offhand.GetSpec().Subtype == items.Claws {
		penalty = 0
	} else {
		penaltyReduction := float64(dualWieldLevel) / 50.0
		penalty = int(50.0 - (penaltyReduction * 40.0))
		if penalty < 10 {
			penalty = 10
		}
	}
	// Extra arm weapons get escalating additional penalties
	if weapIdx == 2 {
		penalty += 20
	} else if weapIdx >= 3 {
		penalty += 40
	}
	return penalty
}

// buildWeaponSetup pre-computes weapon info for a single weapon in the attack sequence.
func buildWeaponSetup(sourceChar *characters.Character, targetChar *characters.Character, weapon items.Item, idx, total int) weaponSetup {
	raceInfo := species.GetSpecies(sourceChar.SpeciesId)
	ws := weaponSetup{
		weapon:        weapon,
		weaponName:    raceInfo.UnarmedName,
		weaponSubType: items.Bludgeoning,
		weaponSpeed:   1.0,
		isOffhand:     idx > 0,
		penalty:       calcDualWieldPenalty(sourceChar, idx, total),
	}

	ws.attacks, ws.baseDmg, ws.dmgVariance, ws.critBuffs = sourceChar.GetDefaultDistributionDamage()

	if weapon.ItemId > 0 {
		itemSpec := weapon.GetSpec()
		ws.weaponName = weapon.DisplayName()
		ws.weaponSubType = itemSpec.Subtype
		ws.attacks, ws.baseDmg, ws.dmgVariance, ws.critBuffs = weapon.GetDistributionDamage()
		ws.weaponSpeed = itemSpec.GetSpeedMultiplier()

		// Racial bonus
		ws.baseDmg += float64(weapon.StatMod(string(statmods.RacialBonusPrefix) + strings.ToLower(targetChar.Species())))

		ws.weaponDmgMult = itemSpec.DamageMultiplier
		if ws.weaponDmgMult <= 0 {
			ws.weaponDmgMult = float64(configs.GetBalanceConfig().UnarmedDamageMultiplier)
		}
	} else {
		if speciesInfo := species.GetSpecies(sourceChar.SpeciesId); speciesInfo != nil && speciesInfo.DamageMultiplier > 0 {
			ws.weaponDmgMult = speciesInfo.DamageMultiplier
		} else {
			ws.weaponDmgMult = float64(configs.GetBalanceConfig().UnarmedDamageMultiplier)
		}
	}

	// Apply weapon speed multiplier, skill modifiers, and dual wielding bonuses
	ws.attacks = sourceChar.GetModifiedAttackCount(ws.attacks, ws.weaponSpeed, ws.isOffhand)

	// Stage 8.3: Apply position-based speed modifier
	positionSpeed := sourceChar.CombatPosition.GetSpeedMultiplier()
	ws.attacks = int(math.Ceil(float64(ws.attacks) * positionSpeed))
	if ws.attacks < 1 {
		ws.attacks = 1
	}

	// Stage 7.5: Reduce attacks to 1 if attempting recovery this round
	if sourceChar.HasCondition(characters.ConditionRecoveryPenalty) {
		ws.attacks = 1
	}

	// Hard cap: max 4 swings per weapon per pass
	if ws.attacks > 4 {
		ws.attacks = 4
	}

	return ws
}

// buildDamageParams computes the damage pipeline values for a weapon swing.
func buildDamageParams(sourceChar *characters.Character, targetChar *characters.Character, ws weaponSetup, statModBonus int, srcType SourceTarget) swingDamageParams {
	combatSkillLevel := sourceChar.GetCombatSkillLevel()
	rawDmg := CalcRawDamage(sourceChar.Stats.Strength.ValueAdj, combatSkillLevel, ws.weaponDmgMult, ChannelPhysical)

	// Apply mob damage multiplier
	if srcType == Mob {
		rawDmg *= float64(configs.GetBalanceConfig().MobDamageMultiplier)
	}

	// Apply target's physical mitigation
	dmgMean := ApplyMitigation(rawDmg, targetChar.GetPhysicalMitigation(), MitigationCap(ChannelPhysical))

	// Track pre-mitigation damage for crits
	rawDmgForCrit := rawDmg

	// Pipeline-proportional variance
	dmgVariance := dmgMean * float64(configs.GetGamePlayConfig().RollSpread)

	// Add statmod damage bonus
	dmgMean += float64(statModBonus)
	rawDmgForCrit += float64(statModBonus)

	// Apply smooth health-based melee damage penalty
	hpPenalty := float64(configs.GetBalanceConfig().HealthPenaltyMax)
	dmgMult := ResourceMultiplier(sourceChar.Health, sourceChar.HealthMax.Value, hpPenalty)
	dmgMean *= dmgMult
	rawDmgForCrit *= dmgMult

	// Stage 7.5: Apply prone damage penalty
	if sourceChar.CombatPosition == characters.PositionProne {
		dmgMean *= float64(configs.GetGamePlayConfig().ProneDamagePenalty)
		rawDmgForCrit *= float64(configs.GetGamePlayConfig().ProneDamagePenalty)
	}

	// Phase 24.2: Apply mutation damage multiplier
	if mutDmgMult := mutations.GetDamageMultiplier(sourceChar.Mutations); mutDmgMult != 0 {
		dmgMean *= (1.0 + mutDmgMult)
		rawDmgForCrit *= (1.0 + mutDmgMult)
	}

	// Message seed
	msgSeed := 0
	if configs.GetGamePlayConfig().ConsistentAttackMessages {
		msgSeed = ws.weapon.ItemId
	}

	return swingDamageParams{
		dmgMean:       dmgMean,
		dmgVariance:   dmgVariance,
		rawDmgForCrit: rawDmgForCrit,
		msgSeed:       msgSeed,
	}
}

// calcAttackScore computes the attack roll score with all modifiers.
func calcAttackScore(sourceChar *characters.Character, targetChar *characters.Character, penalty int) float64 {
	attackScore := float64(sourceChar.Stats.Dexterity.ValueAdj) + float64(sourceChar.GetCombatSkillLevel())
	attackScore -= float64(penalty)

	// Apply smooth stamina-based hit chance penalty
	spPenalty := float64(configs.GetBalanceConfig().StaminaPenaltyMax)
	staminaMult := ResourceMultiplier(sourceChar.Stamina, sourceChar.StaminaMax.Value, spPenalty)
	attackScore *= staminaMult

	// Stage 7.5: Apply prone attack multipliers
	cfg := configs.GetGamePlayConfig()
	if sourceChar.CombatPosition == characters.PositionProne {
		attackScore *= float64(cfg.ProneAttackMultiplier)
	}
	if targetChar.CombatPosition == characters.PositionProne {
		attackScore *= float64(cfg.ProneVulnerabilityMultiplier)
	}

	return attackScore
}

// calcCritThreshold computes the z-score threshold for critical hits.
func calcCritThreshold(sourceChar *characters.Character, targetChar *characters.Character) float64 {
	critThreshold := 2.0
	if sourceChar.HasBuffFlag(buffs.Accuracy) {
		critThreshold = 1.5
	}
	if targetChar.HasBuffFlag(buffs.Blink) {
		critThreshold = 2.5
	}
	// Skill advantage shifts crit threshold
	skillDiff := sourceChar.GetCombatSkillLevel() - targetChar.GetCombatSkillLevel()
	critThreshold -= float64(skillDiff) * 0.05

	// Floor after skill adjustment: never easier than Accuracy buff level (~6.7% crit)
	if critThreshold < 1.5 {
		critThreshold = 1.5
	}

	// Stage 8.3: Position-based crit modifiers
	if sourceChar.CombatPosition.IsGrapplePosition() && sourceChar.HasCondition(characters.ConditionGrappleController) {
		if sourceChar.CombatPosition == characters.PositionGrounded {
			critThreshold -= 0.4
		} else if sourceChar.CombatPosition == characters.PositionClinched {
			critThreshold -= 0.2
		}
	}
	if targetChar.CombatPosition == characters.PositionGrounded && !targetChar.HasCondition(characters.ConditionGrappleController) {
		critThreshold += 0.4
	}

	// Absolute floor after all modifiers: ~15.9% max crit (skilled + grapple controller)
	if critThreshold < 1.0 {
		critThreshold = 1.0
	}

	return critThreshold
}

// filterDefensesForThirdParty removes active defenses when the target is in a grapple
// and being attacked by a third party.
func filterDefensesForThirdParty(result *AttackResult, sourceChar *characters.Character, targetChar *characters.Character, defSeq []string) ([]string, bool) {
	isThirdParty := IsThirdPartyAttack(sourceChar, targetChar)
	if !isThirdParty {
		return defSeq, false
	}

	filteredDefenses := []string{}
	for _, def := range defSeq {
		if def == characters.DefenseBlock {
			filteredDefenses = append(filteredDefenses, def)
		}
	}

	// If no defenses remain, send vulnerability messages and auto-hit
	if len(filteredDefenses) == 0 {
		result.SendToTarget(fmt.Sprintf(
			`<ansi fg="red">You're too entangled to defend against %s's attack!</ansi>`,
			sourceChar.Name))
		result.SendToSource(fmt.Sprintf(
			`<ansi fg="attack-good">%s is helpless against your attack!</ansi>`,
			targetChar.Name))
		result.SendToSourceRoom(fmt.Sprintf(
			`<ansi fg="combat">%s is defenseless against %s's attack!</ansi>`,
			targetChar.Name, sourceChar.Name))
	}

	return filteredDefenses, true
}

// runBestOfAllDefense rolls every available defense and picks the one that won
// by the widest margin. Returns the best result.
func runBestOfAllDefense(result *AttackResult, sourceChar *characters.Character, targetChar *characters.Character, defSeq []string, atkScore float64, isThirdParty bool) bestDefenseResult {
	cfg := configs.GetGamePlayConfig()

	best := bestDefenseResult{
		margin: math.Inf(-1),
	}

	for _, defenseType := range defSeq {
		// Track defense attempt
		result.DefenseAttempts = append(result.DefenseAttempts, DefenseType(defenseType))

		// Stage 9.4: Track defense for stance calculation
		targetChar.IncrementDefenseCount()

		// Check if defender can afford this defense (don't deduct yet)
		cost := targetChar.GetDefenseStaminaCost(defenseType)
		if targetChar.Stamina < cost {
			continue
		}

		// Calculate defense score for this defense type
		defenseScore := targetChar.GetDefenseScore(defenseType)

		// Apply base effectiveness multipliers
		switch defenseType {
		case characters.DefenseDodge:
			defenseScore *= float64(cfg.DodgeEffectiveness)
		case characters.DefenseParry:
			defenseScore *= float64(cfg.ParryEffectiveness)
		case characters.DefenseBlock:
			defenseScore *= float64(cfg.BlockEffectiveness)
		}

		// Stage 7.5: Apply prone defense penalties
		if targetChar.CombatPosition == characters.PositionProne {
			switch defenseType {
			case "dodge":
				defenseScore *= float64(cfg.ProneDodgePenalty)
			case "parry":
				defenseScore *= float64(cfg.ProneParryPenalty)
			case "block":
				defenseScore *= float64(cfg.ProneBlockPenalty)
			}
		}

		// Stage 8.5: Apply third-party vulnerability penalty
		if isThirdParty {
			defenseScore *= float64(cfg.ThirdPartyGrapplePenalty)
		}

		// Stage 8.6: Apply failed grapple defense penalty
		if targetChar.HasCondition(characters.ConditionDefensePenalty) {
			defenseScore *= targetChar.GetConditionMagnitude(characters.ConditionDefensePenalty)
		}

		// Opposed roll: attack vs this defense
		_, _, hitRoll, defenseRoll := dice.OpposedRollStat(atkScore, defenseScore)

		// margin > 0 means defense won
		margin := defenseRoll.Value - hitRoll.Value
		if margin > best.margin {
			best.margin = margin
			best.defenseType = defenseType
			best.hitRoll = hitRoll
			best.defRoll = defenseRoll
		}
	}

	// Deduct stamina only for the winning defense
	if best.defenseType != "" {
		targetChar.DeductDefenseStamina(best.defenseType)
	}

	return best
}

// resolveDefenseOutcome processes the best defense result and sends appropriate messages.
// Returns whether the attack hit and the relevant roll result.
func resolveDefenseOutcome(result *AttackResult, best bestDefenseResult, sourceChar *characters.Character, targetChar *characters.Character, isThirdParty bool) (bool, dice.RollResult) {
	cfg := configs.GetGamePlayConfig()

	// Store z-scores from the best defense attempt
	if best.defenseType != "" {
		result.AttackZScore = best.hitRoll.ZScore
		result.DefenseZScore = best.defRoll.ZScore
	}

	if best.margin > 0 {
		// Attack hit floor: even outclassed attackers have a minimum chance
		attackFloor := float64(cfg.MinAttackHitChance)
		if attackFloor > 0 && util.Rand(100) < int(attackFloor*100) {
			// Floor save — attack hits despite defense winning the roll
			return true, best.hitRoll
		}

		// Best defense succeeded — attack is avoided
		result.DefenseUsed = DefenseType(best.defenseType)

		// Detect defense crits (z > 2.0) for Stage 8.4 crit outcomes
		if best.defRoll.ZScore > 2.0 {
			if best.defenseType == characters.DefenseParry {
				result.ParryCritDetected = true
			} else if best.defenseType == characters.DefenseDodge {
				result.DodgeCritDetected = true
			}
		}

		// Add defense success messages (Stage 9.3: narrative variety)
		var defenseVerb string
		var skillToProgress string
		var itemsDefenseType items.DefenseType
		switch best.defenseType {
		case characters.DefenseDodge:
			defenseVerb = "dodge"
			itemsDefenseType = items.DefenseDodge
			skillToProgress = string(skills.UnarmedCombat)
		case characters.DefenseParry:
			defenseVerb = "parry"
			itemsDefenseType = items.DefenseParry
			skillToProgress = string(skills.WeaponCombat)
		case characters.DefenseBlock:
			defenseVerb = "block"
			itemsDefenseType = items.DefenseBlock
			skillToProgress = string(skills.WeaponCombat)
		}

		// Trigger skill progression for successful defense
		targetChar.TrackSkillUse(skillToProgress)
		targetChar.CheckSkillProgression(skillToProgress, targetChar.GetUserId(), 1.0)

		// Get narrative defense messages based on defense z-score
		defenseMsgs := items.GetDefenseMessage(itemsDefenseType, best.defRoll.ZScore)

		// Prepare token replacements
		weaponName := species.GetSpecies(sourceChar.SpeciesId).UnarmedName
		if sourceChar.Equipment.Weapon.ItemId > 0 {
			weaponName = sourceChar.Equipment.Weapon.GetSpec().Name
		}

		tokenReplacements := map[items.TokenName]string{
			items.TokenDefender: targetChar.Name,
			items.TokenAttacker: sourceChar.Name,
			items.TokenWeapon:   weaponName,
			items.TokenStance:   targetChar.CalculateStanceString(),
			items.TokenPosition: targetChar.CalculatePositionString(),
			items.TokenMomentum: targetChar.CalculateMomentumString(),
		}

		// If we have custom defense messages, use them
		if len(defenseMsgs.Together.ToDefender) > 0 {
			toDefenderMsg := defenseMsgs.Together.ToDefender.Get()
			toAttackerMsg := defenseMsgs.Together.ToAttacker.Get()
			toRoomMsg := defenseMsgs.Together.ToRoom.Get()

			for token, value := range tokenReplacements {
				toDefenderMsg = toDefenderMsg.SetTokenValue(token, value)
				toAttackerMsg = toAttackerMsg.SetTokenValue(token, value)
				toRoomMsg = toRoomMsg.SetTokenValue(token, value)
			}

			result.SendToTarget(string(toDefenderMsg))
			result.SendToSource(string(toAttackerMsg))
			result.SendToSourceRoom(string(toRoomMsg))
			if sourceChar.RoomId != targetChar.RoomId {
				result.SendToTargetRoom(string(toRoomMsg))
			}
		} else {
			result.SendToSource(fmt.Sprintf(`<ansi fg="attack-bad">%s %ss your attack!</ansi>`, targetChar.Name, defenseVerb))
			result.SendToTarget(fmt.Sprintf(`<ansi fg="defense-good">You %s %s's attack!</ansi>`, defenseVerb, sourceChar.Name))
			result.SendToSourceRoom(fmt.Sprintf(`<ansi fg="combat">%s %ss %s's attack.</ansi>`, targetChar.Name, defenseVerb, sourceChar.Name))
			if sourceChar.RoomId != targetChar.RoomId {
				result.SendToTargetRoom(fmt.Sprintf(`<ansi fg="combat">%s %ss an attack.</ansi>`, targetChar.Name, defenseVerb))
			}
		}

		// Stage 8.5: Add third-party context if applicable
		if isThirdParty {
			result.SendToTarget(fmt.Sprintf(
				`<ansi fg="yellow">(Despite being entangled in a grapple!)</ansi>`))
		}

		return false, best.hitRoll
	}

	// No defense succeeded on the roll — check defense floor
	{
		// Default to dodge when no defense was attempted (e.g. no stamina)
		defType := best.defenseType
		if defType == "" {
			defType = characters.DefenseDodge
		}
		floor := float64(cfg.MinDefenseChance)
		if floor > 0 && util.Rand(100) < int(floor*100) {
			// Floor save — defense succeeds despite losing the roll
			result.DefenseUsed = DefenseType(defType)

			var defenseVerb string
			switch defType {
			case characters.DefenseDodge:
				defenseVerb = "dodge"
			case characters.DefenseParry:
				defenseVerb = "parry"
			case characters.DefenseBlock:
				defenseVerb = "block"
			default:
				defenseVerb = "avoid"
			}

			result.SendToSource(fmt.Sprintf(`<ansi fg="attack-bad">%s %ss your attack!</ansi>`, targetChar.Name, defenseVerb))
			result.SendToTarget(fmt.Sprintf(`<ansi fg="defense-good">You %s %s's attack!</ansi>`, defenseVerb, sourceChar.Name))
			result.SendToSourceRoom(fmt.Sprintf(`<ansi fg="combat">%s %ss %s's attack.</ansi>`, targetChar.Name, defenseVerb, sourceChar.Name))
			if sourceChar.RoomId != targetChar.RoomId {
				result.SendToTargetRoom(fmt.Sprintf(`<ansi fg="combat">%s %ss an attack.</ansi>`, targetChar.Name, defenseVerb))
			}

			return false, best.hitRoll
		}
	}

	// Attack hits
	return true, best.hitRoll
}

// calcHitDamage computes the damage for a successful hit, handling crits.
func calcHitDamage(result *AttackResult, lastHitRoll dice.RollResult, critThresh float64, backstab bool, sdp swingDamageParams) (int, bool) {
	if lastHitRoll.ZScore >= critThresh || backstab {
		result.Crit = true
		result.BuffTarget = sdp.critBuffs
		damageResult := dice.Roll(sdp.rawDmgForCrit, sdp.dmgVariance)
		dmg := int(math.Round(math.Max(0, damageResult.Value)))
		mudlog.Debug("CritDetected", "zScore", fmt.Sprintf("%.2f", lastHitRoll.ZScore), "threshold", fmt.Sprintf("%.2f", critThresh), "source", "attacker", "target", "defender", "rawDmg", fmt.Sprintf("%.1f", sdp.rawDmgForCrit), "mitigatedDmg", fmt.Sprintf("%.1f", sdp.dmgMean))
		return dmg, false // consume backstab
	}
	// Normal hit: use mitigated damage
	damageResult := dice.Roll(sdp.dmgMean, sdp.dmgVariance)
	return int(math.Round(math.Max(0, damageResult.Value))), backstab
}

// swingDamageParamsWithCritBuffs is a type alias to carry critBuffs through calcHitDamage
// critBuffs are stored via sdp so they pass through naturally.

// buildAttackMessages constructs and sends all combat messages for a swing.
func buildAttackMessages(result *AttackResult, sourceChar *characters.Character, targetChar *characters.Character,
	ws weaponSetup, sdp swingDamageParams, attackTargetDamage int, attackTargetReduction int,
	attackSourceDamage int, attackSourceReduction int,
	srcType, tgtType SourceTarget, prefix string) {

	// Calculate actual damage vs. expected damage pct
	pctDamage := 0.0
	if sdp.dmgMean > 0 {
		pctDamage = math.Ceil(float64(attackTargetDamage) / sdp.dmgMean * 100)
	}

	// Use fumble messages when a fumble is detected
	var msgs items.AttackOptions
	if result.Fumble {
		msgs = items.GetPreAttackMessage(ws.weaponSubType, items.Fumble)
	} else {
		msgs = items.GetAttackMessage(ws.weaponSubType, int(pctDamage))
	}

	var toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg items.ItemMessage

	tokenReplacements := map[items.TokenName]string{
		items.TokenItemName:     ws.weaponName,
		items.TokenSource:       sourceChar.Name,
		items.TokenSourceType:   string(srcType) + `name`,
		items.TokenTarget:       targetChar.Name,
		items.TokenTargetType:   string(tgtType) + `name`,
		items.TokenUsesLeft:     `[Invalid]`,
		items.TokenDamage:       GetDamageDescription(attackTargetDamage, targetChar.HealthMax.Value),
		items.TokenEntranceName: `unknown`,
		items.TokenExitName:     `unknown`,
		items.TokenStance:       sourceChar.CalculateStanceString(),
		items.TokenPosition:     sourceChar.CalculatePositionString(),
		items.TokenMomentum:     sourceChar.CalculateMomentumString(),
	}

	// Get source character's weapon skill level for message selection
	skillLevel := sourceChar.GetCombatSkillLevel()

	if sourceChar.RoomId == targetChar.RoomId {
		toAttackerMsg = msgs.Together.ToAttacker.GetForSkillLevel(skillLevel, sdp.msgSeed)
		toDefenderMsg = msgs.Together.ToDefender.GetForSkillLevel(skillLevel, sdp.msgSeed)
		toAttackerRoomMsg = msgs.Together.ToRoom.GetForSkillLevel(skillLevel, sdp.msgSeed)
		toDefenderRoomMsg = items.ItemMessage("")
	} else {
		toAttackerMsg = msgs.Separate.ToAttacker.GetForSkillLevel(skillLevel, sdp.msgSeed)
		toDefenderMsg = msgs.Separate.ToDefender.GetForSkillLevel(skillLevel, sdp.msgSeed)
		toAttackerRoomMsg = msgs.Separate.ToAttackerRoom.GetForSkillLevel(skillLevel, sdp.msgSeed)
		toDefenderRoomMsg = msgs.Separate.ToDefenderRoom.GetForSkillLevel(skillLevel, sdp.msgSeed)

		// Find the exit that leads to the target from the source (if any)
		if atkRoom := rooms.LoadRoom(sourceChar.RoomId); atkRoom != nil {
			for exitName, exit := range atkRoom.Exits {
				if exit.RoomId == targetChar.RoomId {
					tokenReplacements[items.TokenExitName] = exitName
					break
				}
			}
		}
		// find the exit that leads to the source from the target (if any)
		if defRoom := rooms.LoadRoom(targetChar.RoomId); defRoom != nil {
			for exitName, exit := range defRoom.Exits {
				if exit.RoomId == sourceChar.RoomId {
					tokenReplacements[items.TokenEntranceName] = exitName
					break
				}
			}
		}
	}

	if sourceChar.Equipment.Weapon.ItemId > 0 {
		tokenReplacements[items.TokenItemName] = sourceChar.Equipment.Weapon.DisplayName()
	}

	if srcType == Mob {
		tokenReplacements[items.TokenSource] = sourceChar.GetMobName(0).String()
	}

	if tgtType == Mob {
		tokenReplacements[items.TokenTarget] = targetChar.GetMobName(0).String()
	}

	for tokenName, tokenValue := range tokenReplacements {
		toAttackerMsg = toAttackerMsg.SetTokenValue(tokenName, tokenValue)
		toDefenderMsg = toDefenderMsg.SetTokenValue(tokenName, tokenValue)
		toAttackerRoomMsg = toAttackerRoomMsg.SetTokenValue(tokenName, tokenValue)
		if len(string(toDefenderRoomMsg)) > 0 {
			toDefenderRoomMsg = toDefenderRoomMsg.SetTokenValue(tokenName, tokenValue)
		}
	}

	if result.Crit {
		toAttackerMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toAttackerMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
		toDefenderMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toDefenderMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
		toAttackerRoomMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toAttackerRoomMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
		if len(string(toDefenderRoomMsg)) > 0 {
			toDefenderRoomMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toDefenderRoomMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
		}
	}

	if result.Fumble {
		toAttackerMsg = items.ItemMessage(`<ansi fg="red-bold">!!!</ansi> ` + string(toAttackerMsg) + ` <ansi fg="red-bold">!!!</ansi>`)
		toDefenderMsg = items.ItemMessage(`<ansi fg="red-bold">!!!</ansi> ` + string(toDefenderMsg) + ` <ansi fg="red-bold">!!!</ansi>`)
		toAttackerRoomMsg = items.ItemMessage(`<ansi fg="red-bold">!!!</ansi> ` + string(toAttackerRoomMsg) + ` <ansi fg="red-bold">!!!</ansi>`)
		if len(string(toDefenderRoomMsg)) > 0 {
			toDefenderRoomMsg = items.ItemMessage(`<ansi fg="red-bold">!!!</ansi> ` + string(toDefenderRoomMsg) + ` <ansi fg="red-bold">!!!</ansi>`)
		}
	}

	if len(prefix) > 0 {
		toAttackerMsg = items.ItemMessage(prefix + string(toAttackerMsg))
		toDefenderMsg = items.ItemMessage(prefix + string(toDefenderMsg))
		toAttackerRoomMsg = items.ItemMessage(prefix + string(toAttackerRoomMsg))
		if len(string(toDefenderRoomMsg)) > 0 {
			toDefenderRoomMsg = items.ItemMessage(prefix + string(toDefenderRoomMsg))
		}
	}

	// Send to attacker
	attackerMsg := string(toAttackerMsg)
	if attackSourceDamage > 0 && attackSourceReduction > 0 {
		attackerMsg += fmt.Sprintf(` <ansi fg="white">[%s was blocked]</ansi>`, GetDamageDescription(attackSourceReduction, sourceChar.HealthMax.Value))
	}

	result.SendToSource(string(attackerMsg))

	// Send to victim
	defenderMsg := string(toDefenderMsg)
	if attackTargetDamage > 0 && attackTargetReduction > 0 {
		defenderMsg += fmt.Sprintf(` <ansi fg="red">[you blocked %s]</ansi>`, GetDamageDescription(attackTargetReduction, targetChar.HealthMax.Value))
	}

	result.SendToTarget(string(defenderMsg))

	// Send to room
	result.SendToSourceRoom(
		string(toAttackerRoomMsg.SetTokenValue(items.TokenTarget, targetChar.Name).
			SetTokenValue(items.TokenTargetType, string(tgtType))),
	)

	// Send to defender room if separate
	if len(string(toDefenderRoomMsg)) > 0 {
		result.SendToTargetRoom(
			string(toDefenderRoomMsg.SetTokenValue(items.TokenTarget, targetChar.Name).SetTokenValue(items.TokenTargetType, string(tgtType))),
		)
	}
}

// applyPetDamage handles pet contribution to combat if applicable.
func applyPetDamage(result *AttackResult, sourceChar *characters.Character, targetChar *characters.Character, tgtType SourceTarget) {
	if petJoins, _ := dice.Percentile(20); !petJoins {
		return
	}
	if sourceChar.RoomId != targetChar.RoomId {
		return
	}
	if !sourceChar.Pet.Exists() || (sourceChar.Pet.Damage.BaseDamage <= 0 && sourceChar.Pet.Damage.DiceRoll == ``) {
		return
	}

	petDmg := sourceChar.Pet.Damage
	var petAttacks int
	var petBaseDmg, petVar float64
	if petDmg.BaseDamage > 0 {
		petAttacks = petDmg.Attacks
		if petAttacks < 1 {
			petAttacks = 1
		}
		petBaseDmg = float64(petDmg.BaseDamage)
		petVar = float64(petDmg.Variance)
	} else {
		petAttacks, _, _, _, _ = sourceChar.Pet.GetDiceRoll()
		petBaseDmg, petVar = dice.DiceToDistribution(petDmg.DiceCount, petDmg.SideCount, petDmg.BonusDamage)
	}

	for i := 0; i < petAttacks; i++ {
		attackTargetDamage := int(math.Round(math.Max(0, dice.Roll(petBaseDmg, petVar).Value)))

		result.DamageToTarget += attackTargetDamage

		toAttackerMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%s</ansi> to <ansi fg="%sname">%s</ansi>!`, sourceChar.Pet.DisplayName(), GetDamageDescription(attackTargetDamage, targetChar.HealthMax.Value), string(tgtType), targetChar.Name)
		result.SendToSource(toAttackerMsg)

		toDefenderMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%s</ansi> to you!`, sourceChar.Pet.DisplayName(), GetDamageDescription(attackTargetDamage, targetChar.HealthMax.Value))
		result.SendToTarget(toDefenderMsg)

		toAttackerRoomMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%s</ansi> to <ansi fg="%sname">%s</ansi>!`, sourceChar.Pet.DisplayName(), GetDamageDescription(attackTargetDamage, targetChar.HealthMax.Value), string(tgtType), targetChar.Name)
		result.SendToTargetRoom(toAttackerRoomMsg)
	}
}
