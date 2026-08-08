package combat

import (
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// unloadedMeleeDamageCap is the maximum effective damage multiplier a ranged
// (Shooting-subtype) weapon may contribute on the MELEE auto-attack swing path.
// Ranged weapons keep their full multiplier on the deliberate SHOOT path; in
// melee they are improvised clubs and clamp here.
const unloadedMeleeDamageCap = 0.30

// combatContext carries per-round environmental info into the combat engine.
type combatContext struct {
	sourceCanSee bool // source has nightvision OR room visibility >= 1
	targetCanSee bool // target has nightvision OR room visibility >= 1
	// Chunk 3.3: when true, the first hit of this round is guaranteed to
	// crit regardless of the z-score threshold. Set by the round dispatcher
	// when the defender was snapshotted as Sleeping at round start.
	// Uses combatContext (not a separate parameter) so it threads through
	// calculateCombat without touching Attack*Vs* signatures.
	forceCrit bool
}

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
//
// There is deliberately NO variance field here. The spread of a damage roll is
// always derived from the mean actually being rolled, via dice.RollStat (which
// applies StdDevFor(mean) = mean * RollSpread). Carrying a pre-computed
// variance alongside two different means (dmgMean for a normal hit,
// rawDmgForCrit for a crit) is what let the crit roll inherit the mitigated
// mean's spread — roughly half the intended width against an armored target,
// and staler still once the statmod/health/prone/mutation/warcry modifiers
// below moved the means. See internal/dice/context.md and
// combat.ExecuteSkillMove, which has always rolled this way.
type swingDamageParams struct {
	dmgMean       float64
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

// calcSwingCount computes the number of swings for a single weapon per round.
// Merges dex + weapon speed + skill into one formula, replacing the old
// outer-loop calcAttackCount × inner-loop ws.attacks double multiplication.
//
// Formula: swings = max(1, round(1 + (dex - 50) / 100 × weaponSpeed × (1 + skill / softCap)))
// Then apply stamina, encumbrance, position, and recovery modifiers.
// Hard cap: 4 per weapon.
func calcSwingCount(sourceChar *characters.Character, weaponSpeed float64, extraAttacks int, isOffhand bool) int {
	bal := configs.GetBalanceConfig()
	softCap := float64(bal.SkillSoftCap)
	if softCap <= 0 {
		softCap = 50
	}

	dex := float64(sourceChar.GetEffectiveDexterity())
	skillLevel := float64(sourceChar.GetCombatSkillLevel()) * float64(bal.SkillWeight)

	// Core swing count formula
	swings := 1.0 + (dex-50.0)/100.0*weaponSpeed*(1.0+skillLevel/softCap)
	swings += float64(extraAttacks)

	// Offhand penalty: skill governs dual-wield speed
	if isOffhand {
		dualSkill := float64(sourceChar.GetSkillLevel(skills.WeaponCombat))
		if sourceChar.IsUnarmedStyle() {
			dualSkill = float64(sourceChar.GetSkillLevel(skills.UnarmedCombat))
		}
		dualSkill *= float64(bal.SkillWeight)
		dualWieldMod := 0.5 + (dualSkill/50.0)*0.5
		swings *= dualWieldMod
	}

	// Apply smooth stamina-based swing count penalty
	spPenalty := float64(bal.StaminaPenaltyMax)
	swings *= ResourceMultiplier(sourceChar.Stamina, sourceChar.StaminaMax.Value, spPenalty)

	// Apply encumbrance penalty (weight-based)
	carriedWeight := sourceChar.GetCarriedWeight()
	capacity := sourceChar.CarryCapacity()
	if carriedWeight > capacity {
		overAmount := carriedWeight - capacity
		overRatio := overAmount / capacity
		encumbrancePenalty := math.Min(overRatio*0.5, 0.5)
		swings *= (1.0 - encumbrancePenalty)
	}

	// Haste buff: significant attack speed boost
	if sourceChar.HasBuffFlag(buffs.Haste) {
		swings *= float64(bal.HasteSwingMultiplier)
	}

	// Position-based speed modifier (chunk 4b R1: FSM-driven via the new
	// helper; legacy CombatPosition.GetSpeedMultiplier is sunset in S5).
	swings *= sourceChar.GetPositionSpeedMultiplier()

	// Round to nearest int, minimum 1
	result := int(math.Round(swings))
	if result < 1 {
		result = 1
	}

	// Recovery penalty: force to 1
	if sourceChar.HasCondition(characters.ConditionRecoveryPenalty) {
		result = 1
	}

	// Hard cap: max 4 swings per weapon
	if result > 4 {
		result = 4
	}

	return result
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
	if sourceChar.ExtraArms >= 3 && sourceChar.Equipment.ExtraArm3.ItemId > 0 && sourceChar.Equipment.ExtraArm3.GetSpec().Type == items.Weapon {
		attackWeapons = append(attackWeapons, sourceChar.Equipment.ExtraArm3)
	}
	if sourceChar.ExtraArms >= 4 && sourceChar.Equipment.ExtraArm4.ItemId > 0 && sourceChar.Equipment.ExtraArm4.GetSpec().Type == items.Weapon {
		attackWeapons = append(attackWeapons, sourceChar.Equipment.ExtraArm4)
	}

	// Empty hand slots become fist attacks (unless holding a shield or
	// blocked by a 2H weapon in the pair-partner slot — a 2H physically
	// occupies the partner arm even though its ItemId reads as 0).
	emptyArm := items.Item{ItemId: 0}

	// Main hand empty → fist. Weapon is always the first slot of pair A,
	// so it can never be blocked by a 2H.
	if sourceChar.Equipment.Weapon.ItemId == 0 {
		attackWeapons = append(attackWeapons, emptyArm)
	}
	// Offhand empty → fist, unless blocked by a 2H in the main hand.
	if sourceChar.Equipment.Offhand.ItemId == 0 &&
		!sourceChar.Equipment.IsBlockedBy2H("offhand") {
		attackWeapons = append(attackWeapons, emptyArm)
	}
	// Extra arm empty slots → fist. Even-numbered extras (2, 4) may be
	// blocked by a 2H in the paired odd-numbered slot (1, 3).
	if sourceChar.ExtraArms >= 1 && sourceChar.Equipment.ExtraArm1.ItemId == 0 {
		attackWeapons = append(attackWeapons, emptyArm)
	}
	if sourceChar.ExtraArms >= 2 && sourceChar.Equipment.ExtraArm2.ItemId == 0 &&
		!sourceChar.Equipment.IsBlockedBy2H("extra arm 2") {
		attackWeapons = append(attackWeapons, emptyArm)
	}
	if sourceChar.ExtraArms >= 3 && sourceChar.Equipment.ExtraArm3.ItemId == 0 {
		attackWeapons = append(attackWeapons, emptyArm)
	}
	if sourceChar.ExtraArms >= 4 && sourceChar.Equipment.ExtraArm4.ItemId == 0 &&
		!sourceChar.Equipment.IsBlockedBy2H("extra arm 4") {
		attackWeapons = append(attackWeapons, emptyArm)
	}

	// Fallback: must have at least one attack
	if len(attackWeapons) == 0 {
		attackWeapons = append(attackWeapons, items.Item{ItemId: 0})
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
	// Natural weapons (claws, fists, bare hands) ignore dual-wield penalty
	mainSub := sourceChar.Equipment.Weapon.GetSpec().Subtype
	offSub := sourceChar.Equipment.Offhand.GetSpec().Subtype
	mainIsNatural := mainSub == items.Claws || mainSub == items.Fist || sourceChar.Equipment.Weapon.ItemId == 0
	offIsNatural := offSub == items.Claws || offSub == items.Fist || sourceChar.Equipment.Offhand.ItemId == 0
	if mainIsNatural && offIsNatural {
		penalty = 0
	} else {
		penaltyReduction := float64(dualWieldLevel) / 50.0
		penalty = int(50.0 - (penaltyReduction * 40.0))
		if penalty < 10 {
			penalty = 10
		}
	}
	// Extra arm weapons get escalating penalties: +20 per arm beyond offhand
	if weapIdx >= 2 {
		penalty += (weapIdx - 1) * 20
	}
	return penalty
}

// buildWeaponSetup pre-computes weapon info for a single weapon in the attack sequence.
// Note: swing count is now computed separately by calcSwingCount, not here.
func buildWeaponSetup(sourceChar *characters.Character, targetChar *characters.Character, weapon items.Item, idx, total int) weaponSetup {
	bal := configs.GetBalanceConfig()
	// GetSpecies is a bare map lookup and returns nil for an unknown
	// SpeciesId, so the name must be defaulted before any dereference.
	// Matches the fallback used further down in this file.
	raceInfo := species.GetSpecies(sourceChar.SpeciesId)
	unarmedName := "fists"
	if raceInfo != nil {
		unarmedName = raceInfo.UnarmedName
	}
	ws := weaponSetup{
		weapon:        weapon,
		weaponName:    unarmedName,
		weaponSubType: items.Unarmed,
		weaponSpeed:   float64(bal.UnarmedSpeedMultiplier),
		isOffhand:     idx > 0,
		penalty:       calcDualWieldPenalty(sourceChar, idx, total),
	}

	ws.attacks, ws.baseDmg, ws.dmgVariance, ws.critBuffs = sourceChar.GetDefaultDistributionDamage()

	// Non-human basic attacks render through the species' natural-attack
	// subtype (bite/claws/slam/...) instead of generic. A real equipped weapon
	// overrides this below. Humanoids leave NaturalAttack empty and stay on
	// Unarmed -> generic.
	if raceInfo != nil && raceInfo.NaturalAttack != "" {
		ws.weaponSubType = raceInfo.NaturalAttack
	}

	if weapon.ItemId > 0 {
		itemSpec := weapon.GetSpec()
		ws.weaponName = weapon.DisplayName()
		ws.weaponSubType = itemSpec.Subtype
		ws.attacks, ws.baseDmg, ws.dmgVariance, ws.critBuffs = weapon.GetDistributionDamage()
		ws.weaponSpeed = itemSpec.GetSpeedMultiplier()

		// Racial bonus
		ws.baseDmg += float64(weapon.StatMod(string(statmods.RacialBonusPrefix) + strings.ToLower(targetChar.Species())))

		gearMul := mutations.GearEffectivenessMultiplier(sourceChar.Mutations)
		// T3 (chunk 4c): apply reach-utility penalty — long weapons are
		// penalized in grapple positions. CalcReachAdjustedItemMult uses
		// DamageMultiplier as the base and scales by radius/reach. The
		// resulting effective multiplier is then further scaled by gearMul.
		adjustedMult := CalcReachAdjustedItemMult(weapon, sourceChar)
		// Ranged (Shooting-subtype) weapons reserve their full damage multiplier
		// for the deliberate SHOOT path (actions.ExecuteFire). In the auto-attack
		// MELEE swing path a bow/crossbow/pistol is just an awkward improvised
		// club, so clamp the melee multiplier — otherwise a high-multiplier
		// arbalest would club like a god-maul.
		if itemSpec.Subtype == items.Shooting && adjustedMult > unloadedMeleeDamageCap {
			adjustedMult = unloadedMeleeDamageCap
		}
		ws.weaponDmgMult = adjustedMult * gearMul
		if ws.weaponDmgMult <= 0 {
			ws.weaponDmgMult = float64(bal.UnarmedDamageMultiplier)
		}
	} else {
		// Natural-attack path (mob unarmed, species DamageMultiplier).
		// Apply reach-utility directly: mob claws/bite/fist are short-reach
		// and stay effective in grapples, so this is mostly a correctness
		// guard — the numbers don't change for natural attacks in practice.
		naturalReach := items.ResolveNaturalReach(ws.weaponSubType)
		posRadius := 0.0 // default: no grapple penalty if Position not initialized
		if sourceChar.Position != nil {
			posRadius = PositionReachRadius(sourceChar.Position.State())
		}
		reachMult := ReachUtility(naturalReach, posRadius)
		if speciesInfo := species.GetSpecies(sourceChar.SpeciesId); speciesInfo != nil && speciesInfo.DamageMultiplier > 0 {
			ws.weaponDmgMult = speciesInfo.DamageMultiplier * reachMult
		} else {
			ws.weaponDmgMult = float64(bal.UnarmedDamageMultiplier) * reachMult
		}
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

	// Add statmod damage bonus
	dmgMean += float64(statModBonus)
	rawDmgForCrit += float64(statModBonus)

	// Apply smooth health-based melee damage penalty
	hpPenalty := float64(configs.GetBalanceConfig().HealthPenaltyMax)
	dmgMult := ResourceMultiplier(sourceChar.Health, sourceChar.HealthMax.Value, hpPenalty)
	dmgMean *= dmgMult
	rawDmgForCrit *= dmgMult

	// Stage 7.5: Apply prone damage penalty. Chunk 4b R1: FSM-driven —
	// Supine attackers swing just as poorly as Prone, so include both.
	if sourceChar.IsProne() || sourceChar.IsSupine() {
		dmgMean *= float64(configs.GetBalanceConfig().ProneDamagePenalty)
		rawDmgForCrit *= float64(configs.GetBalanceConfig().ProneDamagePenalty)
	}

	// Phase 24.2: Apply mutation damage multiplier
	if mutDmgMult := mutations.GetDamageMultiplier(sourceChar.Mutations); mutDmgMult != 0 {
		// #22 crash-site: inside the buried hull, mutation-driven power is
		// suppressed. GetDamageMultiplier returns the bonus fraction (applied
		// as 1.0+bonus), so dampen the full multiplier and re-extract the bonus
		// (penalties, i.e. multiplier <= 1.0, are left untouched by DampenBonus).
		if sourceChar.HasBuffFlag(buffs.Dampened) {
			factor := float64(configs.GetBalanceConfig().CrashSiteSuppressionFactor)
			mutDmgMult = mutations.DampenBonus(1.0+mutDmgMult, factor) - 1.0
		}
		dmgMean *= (1.0 + mutDmgMult)
		rawDmgForCrit *= (1.0 + mutDmgMult)
	}

	// Warcry condition: applies a physical damage multiplier from rhetoric shout
	if sourceChar.HasCondition(characters.ConditionWarcry) {
		warcryMult := 1.0 + sourceChar.GetConditionMagnitude(characters.ConditionWarcry)
		dmgMean *= warcryMult
		rawDmgForCrit *= warcryMult
	}

	// Message seed
	msgSeed := 0
	if configs.GetBalanceConfig().ConsistentAttackMessages {
		msgSeed = ws.weapon.ItemId
	}

	return swingDamageParams{
		dmgMean:       dmgMean,
		rawDmgForCrit: rawDmgForCrit,
		msgSeed:       msgSeed,
	}
}

// calcAttackScore computes the attack roll score with all modifiers.
func calcAttackScore(sourceChar *characters.Character, targetChar *characters.Character, penalty int, ctx combatContext) float64 {
	bal := configs.GetBalanceConfig()
	attackScore := float64(sourceChar.GetEffectiveDexterity()) + float64(sourceChar.GetCombatSkillLevel())*float64(bal.SkillWeight)
	attackScore -= float64(penalty)

	// Apply smooth stamina-based hit chance penalty
	spPenalty := float64(bal.StaminaPenaltyMax)
	staminaMult := ResourceMultiplier(sourceChar.Stamina, sourceChar.StaminaMax.Value, spPenalty)
	attackScore *= staminaMult

	// Stage 7.5: Apply prone attack multipliers. Chunk 4b R1: FSM-driven —
	// Supine maps to the same modifier as Prone (legacy enum couldn't
	// distinguish the two).
	if sourceChar.IsProne() || sourceChar.IsSupine() {
		attackScore *= float64(bal.ProneAttackMultiplier)
	}
	if targetChar.IsProne() || targetChar.IsSupine() {
		attackScore *= float64(bal.ProneVulnerabilityMultiplier)
	}

	// Darkness penalty: attacker can't see
	if !ctx.sourceCanSee {
		attackScore *= float64(bal.DarknessCombatPenalty)
	}

	// Winged Flight: a flyer beats the earthbound on the melee opposed roll —
	// striking from a superior angle (attacker flying) or staying out of an
	// earthbound reach (defender flying). Cancels flyer-vs-flyer / grounded-vs-
	// grounded. Applied additively to the attacker's score.
	if edge := flightEdge(mutations.IsFlying(sourceChar.Mutations), mutations.IsFlying(targetChar.Mutations),
		int(bal.FlightOpposedEdge)); edge != 0 {
		attackScore += float64(edge)
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

	// Stage 8.3: Position-based crit modifiers. Chunk 4b R1: FSM-driven —
	// IsController replaces the legacy ConditionGrappleController gate
	// (S4 sunset), and IsGroundGrapple / IsStandingGrapple replace the
	// legacy PositionGrounded / PositionClinched enum reads.
	if sourceChar.IsController() {
		if sourceChar.IsGroundGrapple() {
			critThreshold -= 0.4
		} else if sourceChar.IsStandingGrapple() {
			critThreshold -= 0.2
		}
	}
	if targetChar.IsGroundGrapple() && !targetChar.IsController() {
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

	// If no defenses remain, send vulnerability messages and auto-hit.
	// Vulnerability prose is hit-prep — the swing is about to land.
	if len(filteredDefenses) == 0 {
		hitCat := messaging.CategoryHitMelee
		if sourceChar.Equipment.Weapon.ItemId > 0 {
			hitCat = CategoryForWeaponSubtype(sourceChar.Equipment.Weapon.GetSpec().Subtype)
		}
		result.SendToTarget(hitCat, fmt.Sprintf(
			`<ansi fg="red">You're too entangled to defend against %s's attack!</ansi>`,
			sourceChar.Name))
		result.SendToSource(hitCat, fmt.Sprintf(
			`<ansi fg="attack-good">%s is helpless against your attack!</ansi>`,
			targetChar.Name))
		result.SendToSourceRoom(hitCat, fmt.Sprintf(
			`<ansi fg="combat">%s is defenseless against %s's attack!</ansi>`,
			targetChar.Name, sourceChar.Name))
	}

	return filteredDefenses, true
}

// runBestOfAllDefense rolls every available defense and picks the one that won
// by the widest margin. Returns the best result.
func runBestOfAllDefense(result *AttackResult, sourceChar *characters.Character, targetChar *characters.Character, defSeq []string, atkScore float64, isThirdParty bool, ctx combatContext) bestDefenseResult {
	bal := configs.GetBalanceConfig()

	best := bestDefenseResult{
		margin: math.Inf(-1),
	}

	// Roll the attack ONCE — all defense types contest the same swing.
	atkStdDev := dice.StdDevFor(atkScore)
	attackRoll := dice.Roll(atkScore, atkStdDev)

	// Always record the attack roll so crit/fumble z-scores are available
	// even when no defense is attempted (empty sequence or all skipped).
	best.hitRoll = attackRoll

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
			defenseScore *= float64(bal.DodgeEffectiveness)
		case characters.DefenseParry:
			defenseScore *= float64(bal.ParryEffectiveness)
		case characters.DefenseBlock:
			defenseScore *= float64(bal.BlockEffectiveness)
		}

		// Stage 7.5: Apply position-based defense penalties. Chunk 4b R1:
		// FSM-driven — Prone/Supine collapse to the legacy "prone"
		// penalty bucket, IsStandingGrapple matches the legacy
		// "clinched" bucket, IsGroundGrapple matches the legacy
		// "grounded" bucket.
		switch {
		case targetChar.IsProne() || targetChar.IsSupine():
			switch defenseType {
			case "dodge":
				defenseScore *= float64(bal.ProneDodgePenalty)
			case "parry":
				defenseScore *= float64(bal.ProneParryPenalty)
			case "block":
				defenseScore *= float64(bal.ProneBlockPenalty)
			}
		case targetChar.IsStandingGrapple():
			switch defenseType {
			case "dodge":
				defenseScore *= float64(bal.ClinchDodgePenalty)
			case "parry":
				defenseScore *= float64(bal.ClinchParryPenalty)
			case "block":
				defenseScore *= float64(bal.ClinchBlockPenalty)
			}
		case targetChar.IsGroundGrapple():
			switch defenseType {
			case "dodge":
				defenseScore *= float64(bal.GroundedDodgePenalty)
			case "parry":
				defenseScore *= float64(bal.GroundedParryPenalty)
			case "block":
				defenseScore *= float64(bal.GroundedBlockPenalty)
			}
		}

		// Rally condition: applies a defense score multiplier from rhetoric shout
		if targetChar.HasCondition(characters.ConditionRally) {
			defenseScore *= 1.0 + targetChar.GetConditionMagnitude(characters.ConditionRally)
		}

		// Stage 8.5: Apply third-party vulnerability penalty
		if isThirdParty {
			defenseScore *= float64(bal.ThirdPartyGrapplePenalty)
		}

		// Stage 8.6: Apply failed grapple defense penalty
		if targetChar.HasCondition(characters.ConditionDefensePenalty) {
			defenseScore *= targetChar.GetConditionMagnitude(characters.ConditionDefensePenalty)
		}

		// Darkness penalty: defender can't see
		if !ctx.targetCanSee {
			defenseScore *= float64(bal.DarknessCombatPenalty)
		}

		// Incorporeal mutation: physical defense bonus (channel-scoped
		// to physical attacks; this function only handles physical
		// swings — spells use a different resolution path).
		defenseScore += mutations.GetPhysicalDefenseBonus(targetChar.Mutations)

		// Roll this defense against the single attack roll
		defenseRoll := dice.Roll(defenseScore, atkStdDev)

		// margin > 0 means defense won
		margin := defenseRoll.Value - attackRoll.Value
		if margin > best.margin {
			best.margin = margin
			best.defenseType = defenseType
			best.hitRoll = attackRoll
			best.defRoll = defenseRoll
		}
	}

	// Deduct stamina only for the winning defense
	if best.defenseType != "" {
		targetChar.DeductDefenseStamina(best.defenseType)
	}

	return best
}

// hitResolution holds the outcome of the full hitroll pipeline.
type hitResolution struct {
	hit          bool
	crit         bool
	fumble       bool
	doubleFumble bool
	defenseCrit  bool
	hitRoll      dice.RollResult
}

// doubleFumbleMessages are comedy flavor text for when both combatants fumble.
var doubleFumbleMessages = []struct {
	toAttacker string
	toDefender string
	toRoom     string
}{
	{
		toAttacker: `You trip over your own feet and %s stumbles trying to capitalize!`,
		toDefender: `%s trips over their own feet and you stumble trying to capitalize!`,
		toRoom:     `%s and %s both stumble in a spectacular display of ineptitude.`,
	},
	{
		toAttacker: `You swing wildly and lose your balance — %s flails just as badly!`,
		toDefender: `%s swings wildly and loses balance — and you flail just as badly!`,
		toRoom:     `%s and %s both flail about in an embarrassing tangle of limbs.`,
	},
	{
		toAttacker: `Your weapon slips at the exact moment %s trips over their own guard!`,
		toDefender: `Your guard tangles at the exact moment %s's weapon slips free!`,
		toRoom:     `%s's weapon slips and %s's guard tangles — both stumble to the ground!`,
	},
	{
		toAttacker: `You overcommit and tumble forward — %s overreacts and falls too!`,
		toDefender: `%s overcommits and tumbles forward — you overreact and fall too!`,
		toRoom:     `%s overcommits and %s overreacts — both crash to the ground!`,
	},
	{
		toAttacker: `You slip on something and %s panics into a heap beside you!`,
		toDefender: `%s slips on something and you panic into a heap beside them!`,
		toRoom:     `%s slips and %s panics — both end up in a heap on the ground!`,
	},
}

// handleDoubleFumble applies prone to both combatants and sends comedy text.
func handleDoubleFumble(result *AttackResult, sourceChar *characters.Character, targetChar *characters.Character) {
	// Both go prone via FSM. Position can be nil on a pre-Validate()
	// Character (mirrors the guard in HandleGrappleCritFailure); prod
	// combatants are always Validated, so this only shields the
	// under-initialized-fixture case rather than any live path.
	r := state.TransitionReason{Trigger: position.TriggerKnockdownFaceForward}
	if sourceChar.Position != nil {
		if err := sourceChar.Position.TransitionToProne(position.ProneData{}, r); err != nil {
			mudlog.Warn("handleDoubleFumble: source TransitionToProne failed", "err", err)
		}
	}
	if targetChar.Position != nil {
		if err := targetChar.Position.TransitionToProne(position.ProneData{}, r); err != nil {
			mudlog.Warn("handleDoubleFumble: target TransitionToProne failed", "err", err)
		}
	}

	// Pick a random comedy message
	msg := doubleFumbleMessages[util.Rand(len(doubleFumbleMessages))]

	// Double-fumble: both sides fumble, no defense, no clean
	// weapon category to pick. Use CategoryHitMelee as the combat-
	// neutral hit-band default.
	fumbleCat := messaging.CategoryHitMelee
	result.SendToSource(fumbleCat, fmt.Sprintf(`<ansi fg="fumble-text">!!!</ansi> `+
		`<ansi fg="yellow">`+msg.toAttacker+`</ansi>`+
		` <ansi fg="fumble-text">!!!</ansi>`, targetChar.Name))
	result.SendToTarget(fumbleCat, fmt.Sprintf(`<ansi fg="fumble-text">!!!</ansi> `+
		`<ansi fg="yellow">`+msg.toDefender+`</ansi>`+
		` <ansi fg="fumble-text">!!!</ansi>`, sourceChar.Name))
	result.SendToSourceRoom(fumbleCat, fmt.Sprintf(`<ansi fg="fumble-text">!!!</ansi> `+
		`<ansi fg="yellow">`+msg.toRoom+`</ansi>`+
		` <ansi fg="fumble-text">!!!</ansi>`, sourceChar.Name, targetChar.Name))
}

// resolveDefenseOutcome processes the best defense result with the new
// crit/fumble priority: fumbles → crits → normal → floors.
// Returns the full hitResolution including crit/fumble flags.
//
// forceCrit bypasses the Z-score threshold check and treats the attack as a
// confirmed crit regardless of the roll result. The stored ZScore is bumped to
// critThreshold+0.5 so downstream crit-magnitude logic sees a confident crit.
// Pass false for all normal combat rounds; T14 passes true for sleeping-victim
// first-hit-crit.
func resolveDefenseOutcome(result *AttackResult, best bestDefenseResult, sourceChar *characters.Character, targetChar *characters.Character, critThreshold float64, isThirdParty bool, forceCrit bool) hitResolution {
	bal := configs.GetBalanceConfig()
	fumbleThreshold := -2.0
	defCritThreshold := 2.0

	// forceCrit: override the attack roll's Z-score before any threshold check
	// so every downstream branch (fumble guards, crit vs crit comparisons, etc.)
	// sees a clearly-crit roll.
	if forceCrit && best.hitRoll.ZScore < critThreshold+0.5 {
		best.hitRoll.ZScore = critThreshold + 0.5
	}

	res := hitResolution{
		hitRoll: best.hitRoll,
	}

	// Store z-scores from the best defense attempt
	if best.defenseType != "" {
		result.AttackZScore = best.hitRoll.ZScore
		result.DefenseZScore = best.defRoll.ZScore
	}

	attackFumble := best.hitRoll.ZScore <= fumbleThreshold
	defenseFumble := best.defenseType != "" && best.defRoll.ZScore <= fumbleThreshold
	attackCrit := best.hitRoll.ZScore >= critThreshold
	defenseCrit := best.defenseType != "" && best.defRoll.ZScore >= defCritThreshold

	// ── Step 1: Fumble resolution (absolute) ────────────────────────────────

	if attackFumble && defenseFumble {
		// Double fumble: miss, both go prone, comedy text
		res.fumble = true
		res.doubleFumble = true
		res.hit = false
		result.Fumble = true
		result.DoubleFumble = true
		handleDoubleFumble(result, sourceChar, targetChar)
		mudlog.Debug("DoubleFumble", "atkZ", fmt.Sprintf("%.2f", best.hitRoll.ZScore),
			"defZ", fmt.Sprintf("%.2f", best.defRoll.ZScore),
			"source", sourceChar.Name, "target", targetChar.Name)
		return res
	}

	if attackFumble {
		// Attack fumble: always miss, no exceptions
		res.fumble = true
		res.hit = false
		result.Fumble = true
		mudlog.Debug("AttackFumble", "zScore", fmt.Sprintf("%.2f", best.hitRoll.ZScore),
			"source", sourceChar.Name, "target", targetChar.Name)
		return res
	}

	if defenseFumble {
		// Defense fumble: guarantees a hit (but NOT auto-crit)
		res.hit = true
		mudlog.Debug("DefenseFumble", "defZ", fmt.Sprintf("%.2f", best.defRoll.ZScore),
			"source", sourceChar.Name, "target", targetChar.Name)
		// Still check if the attack roll was also a crit
		if attackCrit {
			res.crit = true
		}
		return res
	}

	// ── Step 2: Crit resolution (trumps normal rolls) ───────────────────────

	if attackCrit && defenseCrit {
		// Both crit: compare raw values, higher wins
		if best.hitRoll.Value >= best.defRoll.Value {
			res.hit = true
			res.crit = true
			mudlog.Debug("CritVsCrit-AtkWins", "atkVal", fmt.Sprintf("%.1f", best.hitRoll.Value),
				"defVal", fmt.Sprintf("%.1f", best.defRoll.Value),
				"source", sourceChar.Name, "target", targetChar.Name)
		} else {
			res.hit = false
			res.defenseCrit = true
			setDefenseCritFlags(result, best)
			sendDefenseMessages(result, best, sourceChar, targetChar, isThirdParty)
			mudlog.Debug("CritVsCrit-DefWins", "atkVal", fmt.Sprintf("%.1f", best.hitRoll.Value),
				"defVal", fmt.Sprintf("%.1f", best.defRoll.Value),
				"source", sourceChar.Name, "target", targetChar.Name)
		}
		return res
	}

	if attackCrit {
		// Attack crit vs normal defense: always hits
		res.hit = true
		res.crit = true
		mudlog.Debug("AttackCrit", "zScore", fmt.Sprintf("%.2f", best.hitRoll.ZScore),
			"threshold", fmt.Sprintf("%.2f", critThreshold),
			"source", sourceChar.Name, "target", targetChar.Name)
		return res
	}

	if defenseCrit {
		// Defense crit vs normal attack: always avoids
		res.hit = false
		res.defenseCrit = true
		setDefenseCritFlags(result, best)
		sendDefenseMessages(result, best, sourceChar, targetChar, isThirdParty)
		mudlog.Debug("DefenseCrit", "defZ", fmt.Sprintf("%.2f", best.defRoll.ZScore),
			"source", sourceChar.Name, "target", targetChar.Name)
		return res
	}

	// ── Step 3: Normal resolution ───────────────────────────────────────────

	if best.margin > 0 {
		// Defense won the roll normally — check attack floor (last resort)
		attackFloor := float64(bal.MinAttackHitChance)
		if attackFloor > 0 && util.Rand(100) < int(attackFloor*100) {
			res.hit = true
			return res
		}

		// Defense succeeded
		res.hit = false
		sendDefenseMessages(result, best, sourceChar, targetChar, isThirdParty)
		return res
	}

	// Attack won the roll normally — check defense floor (last resort)
	{
		floor := float64(bal.MinDefenseChance)
		if floor > 0 && util.Rand(100) < int(floor*100) {
			res.hit = false
			// Floor save — defense succeeds via last resort
			defType := best.defenseType
			if defType == "" {
				defType = characters.DefenseDodge
			}
			sendFloorDefenseMessages(result, defType, sourceChar, targetChar)
			return res
		}
	}

	// Normal hit
	res.hit = true
	return res
}

// setDefenseCritFlags marks parry/dodge/block crit flags on the result.
func setDefenseCritFlags(result *AttackResult, best bestDefenseResult) {
	switch best.defenseType {
	case characters.DefenseParry:
		result.ParryCritDetected = true
	case characters.DefenseDodge:
		result.DodgeCritDetected = true
	case characters.DefenseBlock:
		result.BlockCritDetected = true
	}
}

// sendDefenseMessages sends narrative messages for a successful defense.
func sendDefenseMessages(result *AttackResult, best bestDefenseResult, sourceChar *characters.Character, targetChar *characters.Character, isThirdParty bool) {
	result.DefenseUsed = DefenseType(best.defenseType)

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
	weaponName := "fists"
	attackName := "strike" // Generic term for unarmed attacks
	if raceInfo := species.GetSpecies(sourceChar.SpeciesId); raceInfo != nil {
		weaponName = raceInfo.UnarmedName
	}
	if sourceChar.Equipment.Weapon.ItemId > 0 {
		weaponName = sourceChar.Equipment.Weapon.GetSpec().Name
		attackName = weaponName
	}

	tokenReplacements := map[items.TokenName]string{
		items.TokenDefender: targetChar.Name,
		items.TokenAttacker: sourceChar.Name,
		items.TokenWeapon:   weaponName,
		items.TokenAttack:   attackName,
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

		defCat := CategoryForDefenseVerb(defenseVerb)
		result.SendToTarget(defCat, string(toDefenderMsg))
		result.SendToSource(defCat, string(toAttackerMsg))
		result.SendToSourceRoom(defCat, string(toRoomMsg))
		if sourceChar.RoomId != targetChar.RoomId {
			result.SendToTargetRoom(defCat, string(toRoomMsg))
		}
	} else {
		defCat := CategoryForDefenseVerb(defenseVerb)
		result.SendToSource(defCat, fmt.Sprintf(`<ansi fg="attack-bad">%s %ss your attack!</ansi>`, targetChar.Name, defenseVerb))
		result.SendToTarget(defCat, fmt.Sprintf(`<ansi fg="defense-good">You %s %s's attack!</ansi>`, defenseVerb, sourceChar.Name))
		result.SendToSourceRoom(defCat, fmt.Sprintf(`<ansi fg="combat">%s %ss %s's attack.</ansi>`, targetChar.Name, defenseVerb, sourceChar.Name))
		if sourceChar.RoomId != targetChar.RoomId {
			result.SendToTargetRoom(defCat, fmt.Sprintf(`<ansi fg="combat">%s %ss an attack.</ansi>`, targetChar.Name, defenseVerb))
		}
	}

	// Stage 8.5: Add third-party context if applicable
	if isThirdParty {
		result.SendToTarget(CategoryForDefenseVerb(defenseVerb), fmt.Sprintf(
			`<ansi fg="yellow">(Despite being entangled in a grapple!)</ansi>`))
	}
}

// sendFloorDefenseMessages sends messages for a defense floor save.
func sendFloorDefenseMessages(result *AttackResult, defType string, sourceChar *characters.Character, targetChar *characters.Character) {
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

	defCat := CategoryForDefenseVerb(defenseVerb)
	result.SendToSource(defCat, fmt.Sprintf(`<ansi fg="attack-bad">%s %ss your attack!</ansi>`, targetChar.Name, defenseVerb))
	result.SendToTarget(defCat, fmt.Sprintf(`<ansi fg="defense-good">You %s %s's attack!</ansi>`, defenseVerb, sourceChar.Name))
	result.SendToSourceRoom(defCat, fmt.Sprintf(`<ansi fg="combat">%s %ss %s's attack.</ansi>`, targetChar.Name, defenseVerb, sourceChar.Name))
	if sourceChar.RoomId != targetChar.RoomId {
		result.SendToTargetRoom(defCat, fmt.Sprintf(`<ansi fg="combat">%s %ss an attack.</ansi>`, targetChar.Name, defenseVerb))
	}
}

// calcHitDamage computes the damage for a successful hit, handling crits.
// The isCrit flag is determined during hitroll resolution, not re-derived here.
func calcHitDamage(result *AttackResult, isCrit bool, backstab bool, sdp swingDamageParams) (int, bool) {
	if isCrit || backstab {
		result.Crit = true
		result.BuffTarget = sdp.critBuffs
		// Crits bypass mitigation, so they roll around the UNmitigated mean —
		// and therefore must take their spread from that same mean. RollStat
		// derives stdDev = mean * RollSpread internally, which is the only way
		// to keep the two in step.
		damageResult := dice.RollStat(sdp.rawDmgForCrit)
		dmg := int(math.Round(math.Max(0, damageResult.Value)))
		mudlog.Debug("CritDamage", "rawDmg", fmt.Sprintf("%.1f", sdp.rawDmgForCrit), "mitigatedDmg", fmt.Sprintf("%.1f", sdp.dmgMean))
		return dmg, false // consume backstab
	}
	// Normal hit: use mitigated damage
	damageResult := dice.RollStat(sdp.dmgMean)
	return int(math.Round(math.Max(0, damageResult.Value))), backstab
}

// swingDamageParamsWithCritBuffs is a type alias to carry critBuffs through calcHitDamage
// critBuffs are stored via sdp so they pass through naturally.

// meleeDisplaySubtype computes the subtype used to select AUTO-ATTACK melee
// swing narration. Two rules apply, in order:
//
//  1. Shooting-subtype weapons (bows/crossbows/slings/guns) ALWAYS narrate as
//     improvised Bludgeoning in the melee path — a swung bow is a club, never
//     "fires/snipes/arrow". The deliberate SHOOT path (actions.ExecuteFire /
//     combat_fire.go) has its own messages and never routes through here, so
//     this is unconditional regardless of grapple position.
//  2. Otherwise, when ShouldBludgeon fires (weapon reach exceeds the grapple
//     radius of the attacker's position), bladed subtypes (Slashing, Cleaving,
//     Stabbing) narrate as Bludgeoning — the fiction tracks the pommel/hilt
//     strike that the reach damage penalty already reflects.
//
// Natural-blunt subtypes (Fist, Claws, Bite, Sting, Slam, Gore, Whipping) and
// caster subtypes (Wand, Sceptre, Staff) keep their own vocabulary.
func meleeDisplaySubtype(weaponSubType items.ItemSubType, weaponReach, posRadius float64) items.ItemSubType {
	if weaponSubType == items.Shooting {
		return items.Bludgeoning
	}
	if ShouldBludgeon(weaponReach, posRadius) {
		switch weaponSubType {
		case items.Slashing, items.Cleaving, items.Stabbing:
			return items.Bludgeoning
		}
	}
	return weaponSubType
}

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

	// T4 (chunk 4c): compute the display subtype for attack-message selection.
	// See meleeDisplaySubtype for the swap rules.
	displaySubtype := ws.weaponSubType
	{
		var weaponReach float64
		if ws.weapon.ItemId > 0 {
			spec := ws.weapon.GetSpec()
			weaponReach = items.ResolveReach(&spec)
		} else {
			weaponReach = items.ResolveNaturalReach(ws.weaponSubType)
		}
		posRadius := 0.0
		if sourceChar.Position != nil {
			posRadius = PositionReachRadius(sourceChar.Position.State())
		}
		displaySubtype = meleeDisplaySubtype(ws.weaponSubType, weaponReach, posRadius)
	}

	// Use fumble messages when a fumble is detected
	var msgs items.AttackOptions
	isFeint := false
	if result.Fumble {
		msgs = items.GetPreAttackMessage(displaySubtype, items.Fumble)
	} else {
		msgs = items.GetAttackMessage(displaySubtype, int(pctDamage))
		// Feint check: skilled attackers can turn misses into deliberate-looking feints
		if int(pctDamage) == 0 && !result.Fumble {
			isFeint = checkFeint(sourceChar.GetCombatSkillLevel())
		}
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
		items.TokenBodyPart:     GetRandomBodyPart(),
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

	// Feint: replace miss messages with feint-flavored text for skilled attackers
	if isFeint {
		feintMsg := getFeintMessage()
		toAttackerMsg = items.ItemMessage(feintMsg.toAttacker)
		toDefenderMsg = items.ItemMessage(feintMsg.toDefender)
		toAttackerRoomMsg = items.ItemMessage(feintMsg.toRoom)
		// Apply name tokens to feint messages
		toAttackerMsg = toAttackerMsg.SetTokenValue(items.TokenTarget, tokenReplacements[items.TokenTarget])
		toAttackerMsg = toAttackerMsg.SetTokenValue(items.TokenTargetType, tokenReplacements[items.TokenTargetType])
		toDefenderMsg = toDefenderMsg.SetTokenValue(items.TokenSource, tokenReplacements[items.TokenSource])
		toDefenderMsg = toDefenderMsg.SetTokenValue(items.TokenSourceType, tokenReplacements[items.TokenSourceType])
		toAttackerRoomMsg = toAttackerRoomMsg.SetTokenValue(items.TokenSource, tokenReplacements[items.TokenSource])
		toAttackerRoomMsg = toAttackerRoomMsg.SetTokenValue(items.TokenSourceType, tokenReplacements[items.TokenSourceType])
		toAttackerRoomMsg = toAttackerRoomMsg.SetTokenValue(items.TokenTarget, tokenReplacements[items.TokenTarget])
		toAttackerRoomMsg = toAttackerRoomMsg.SetTokenValue(items.TokenTargetType, tokenReplacements[items.TokenTargetType])
	}

	if result.Crit {
		toAttackerMsg = items.ItemMessage(`<ansi fg="crit-text">***</ansi> ` + string(toAttackerMsg) + ` <ansi fg="crit-text">***</ansi>`)
		toDefenderMsg = items.ItemMessage(`<ansi fg="crit-text">***</ansi> ` + string(toDefenderMsg) + ` <ansi fg="crit-text">***</ansi>`)
		toAttackerRoomMsg = items.ItemMessage(`<ansi fg="crit-text">***</ansi> ` + string(toAttackerRoomMsg) + ` <ansi fg="crit-text">***</ansi>`)
		if len(string(toDefenderRoomMsg)) > 0 {
			toDefenderRoomMsg = items.ItemMessage(`<ansi fg="crit-text">***</ansi> ` + string(toDefenderRoomMsg) + ` <ansi fg="crit-text">***</ansi>`)
		}
	}

	if result.Fumble {
		toAttackerMsg = items.ItemMessage(`<ansi fg="fumble-text">!!!</ansi> ` + string(toAttackerMsg) + ` <ansi fg="fumble-text">!!!</ansi>`)
		toDefenderMsg = items.ItemMessage(`<ansi fg="fumble-text">!!!</ansi> ` + string(toDefenderMsg) + ` <ansi fg="fumble-text">!!!</ansi>`)
		toAttackerRoomMsg = items.ItemMessage(`<ansi fg="fumble-text">!!!</ansi> ` + string(toAttackerRoomMsg) + ` <ansi fg="fumble-text">!!!</ansi>`)
		if len(string(toDefenderRoomMsg)) > 0 {
			toDefenderRoomMsg = items.ItemMessage(`<ansi fg="fumble-text">!!!</ansi> ` + string(toDefenderRoomMsg) + ` <ansi fg="fumble-text">!!!</ansi>`)
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

	// Per-swing hit-band Category from the weapon subtype.
	hitCat := CategoryForWeaponSubtype(ws.weaponSubType)

	// Send to attacker
	attackerMsg := string(toAttackerMsg)
	if attackSourceDamage > 0 && attackSourceReduction > 0 {
		attackerMsg += fmt.Sprintf(` <ansi fg="white">[%s was blocked]</ansi>`, GetDamageDescription(attackSourceReduction, sourceChar.HealthMax.Value))
	}

	result.SendToSource(hitCat, string(attackerMsg))

	// Send to victim
	defenderMsg := string(toDefenderMsg)
	if attackTargetDamage > 0 && attackTargetReduction > 0 {
		defenderMsg += fmt.Sprintf(` <ansi fg="red">[you blocked %s]</ansi>`, GetDamageDescription(attackTargetReduction, targetChar.HealthMax.Value))
	}

	result.SendToTarget(hitCat, string(defenderMsg))

	// Send to room
	result.SendToSourceRoom(hitCat,
		string(toAttackerRoomMsg.SetTokenValue(items.TokenTarget, targetChar.Name).
			SetTokenValue(items.TokenTargetType, string(tgtType))),
	)

	// Send to defender room if separate
	if len(string(toDefenderRoomMsg)) > 0 {
		result.SendToTargetRoom(hitCat,
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

	// Pet bites and claws are physical damage, so they run through the target's
	// physical mitigation exactly like melee (buildDamageParams), ranged
	// (ExecuteSkillMove) and the mob-parity helpers. Before this, pet damage was
	// the one physical source that ignored armor entirely — an armored boss took
	// the same pet damage as an unarmored newbie.
	//
	// Both the mean AND the spread are mitigated. The pet's variance comes from
	// its authored dice spec, not from a stat, so it is not stat-proportional
	// (dice.Roll, not dice.RollStat, is correct here) — but scaling only the
	// mean would leave the roll's spread at full width against armor. Scaling
	// both by the same factor is exactly equivalent to mitigating the rolled
	// value, and preserves the pet's authored coefficient of variation.
	//
	// Pets have no crit path: nothing here inspects a Z-score or sets
	// result.Crit, so the "crits bypass mitigation" convention used by the melee
	// (combat_helpers.go), spell (combat_shared_helpers.go) and taunt
	// (combat_taunt.go) channels has nothing to attach to. Deliberately not
	// adding one — a pet crit is a separate design decision, not a bug fix.
	petMitigation := targetChar.GetPhysicalMitigation()
	petMitigationCap := MitigationCap(ChannelPhysical)
	petBaseDmg = ApplyMitigation(petBaseDmg, petMitigation, petMitigationCap)
	petVar = ApplyMitigation(petVar, petMitigation, petMitigationCap)

	// Pet damage is claws/bite/etc — natural-sharp band.
	petCat := messaging.CategoryHitNaturalSharp
	for i := 0; i < petAttacks; i++ {
		attackTargetDamage := int(math.Round(math.Max(0, dice.Roll(petBaseDmg, petVar).Value)))

		result.DamageToTarget += attackTargetDamage

		toAttackerMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%s</ansi> to <ansi fg="%sname">%s</ansi>!`, sourceChar.Pet.DisplayName(), GetDamageDescription(attackTargetDamage, targetChar.HealthMax.Value), string(tgtType), targetChar.Name)
		result.SendToSource(petCat, toAttackerMsg)

		toDefenderMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%s</ansi> to you!`, sourceChar.Pet.DisplayName(), GetDamageDescription(attackTargetDamage, targetChar.HealthMax.Value))
		result.SendToTarget(petCat, toDefenderMsg)

		toAttackerRoomMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%s</ansi> to <ansi fg="%sname">%s</ansi>!`, sourceChar.Pet.DisplayName(), GetDamageDescription(attackTargetDamage, targetChar.HealthMax.Value), string(tgtType), targetChar.Name)
		result.SendToSourceRoom(petCat, toAttackerRoomMsg)
		if sourceChar.RoomId != targetChar.RoomId {
			result.SendToTargetRoom(petCat, toAttackerRoomMsg)
		}
	}
}

// feintMessage holds the three message variants for a feint.
type feintMessage struct {
	toAttacker string
	toDefender string
	toRoom     string
}

// feintMessages are weapon-agnostic feint flavor messages.
// Tokens: {target}/{targettype} for attacker POV, {source}/{sourcetype} for defender POV.
var feintMessages = []feintMessage{
	{
		toAttacker: `You feint at <ansi fg="{targettype}">{target}</ansi>, testing their defenses.`,
		toDefender: `<ansi fg="{sourcetype}">{source}</ansi> feints at you, probing for weakness.`,
		toRoom:     `<ansi fg="{sourcetype}">{source}</ansi> feints toward <ansi fg="{targettype}">{target}</ansi>, testing for openings.`,
	},
	{
		toAttacker: `You make a deliberate feint, drawing <ansi fg="{targettype}">{target}</ansi>'s guard wide.`,
		toDefender: `<ansi fg="{sourcetype}">{source}</ansi> feints deliberately, drawing your guard.`,
		toRoom:     `<ansi fg="{sourcetype}">{source}</ansi> makes a deliberate feint at <ansi fg="{targettype}">{target}</ansi>.`,
	},
	{
		toAttacker: `You throw a calculated misdirection at <ansi fg="{targettype}">{target}</ansi>.`,
		toDefender: `<ansi fg="{sourcetype}">{source}</ansi> throws a calculated misdirection your way.`,
		toRoom:     `<ansi fg="{sourcetype}">{source}</ansi> throws a misdirection at <ansi fg="{targettype}">{target}</ansi>.`,
	},
	{
		toAttacker: `You probe <ansi fg="{targettype}">{target}</ansi>'s defenses with a quick false strike.`,
		toDefender: `<ansi fg="{sourcetype}">{source}</ansi> probes your defenses with a quick false strike.`,
		toRoom:     `<ansi fg="{sourcetype}">{source}</ansi> probes <ansi fg="{targettype}">{target}</ansi>'s defenses with a quick feint.`,
	},
	{
		toAttacker: `You shift your weight and feint low, reading <ansi fg="{targettype}">{target}</ansi>'s reaction.`,
		toDefender: `<ansi fg="{sourcetype}">{source}</ansi> feints low, reading your reaction intently.`,
		toRoom:     `<ansi fg="{sourcetype}">{source}</ansi> feints low toward <ansi fg="{targettype}">{target}</ansi>, studying their stance.`,
	},
	{
		toAttacker: `You commit to a false opening, watching how <ansi fg="{targettype}">{target}</ansi> responds.`,
		toDefender: `<ansi fg="{sourcetype}">{source}</ansi> opens up deliberately, watching your response.`,
		toRoom:     `<ansi fg="{sourcetype}">{source}</ansi> makes a calculated false opening toward <ansi fg="{targettype}">{target}</ansi>.`,
	},
	{
		toAttacker: `You disguise a measuring strike as a real attack toward <ansi fg="{targettype}">{target}</ansi>.`,
		toDefender: `<ansi fg="{sourcetype}">{source}</ansi> disguises a measuring strike as a real attack.`,
		toRoom:     `<ansi fg="{sourcetype}">{source}</ansi> throws a measured feint toward <ansi fg="{targettype}">{target}</ansi>.`,
	},
	{
		toAttacker: `You draw <ansi fg="{targettype}">{target}</ansi>'s attention high with a deceptive flourish.`,
		toDefender: `<ansi fg="{sourcetype}">{source}</ansi> draws your attention with a deceptive flourish.`,
		toRoom:     `<ansi fg="{sourcetype}">{source}</ansi> flourishes deceptively toward <ansi fg="{targettype}">{target}</ansi>.`,
	},
}

// checkFeint returns true if a miss should be presented as an intentional feint.
// Probability scales smoothly from near-zero at rank 1 to ~33% at soft cap, capped at 75%.
func checkFeint(skillRank int) bool {
	if skillRank <= 0 {
		return false
	}
	bal := configs.GetBalanceConfig()
	softCap := float64(bal.SkillSoftCap)
	if softCap <= 0 {
		softCap = 50
	}
	ratio := float64(skillRank) / softCap
	feintChance := math.Min(0.75, 0.33*math.Pow(ratio, 1.5))
	return util.Rand(1000) < int(feintChance*1000)
}

// getFeintMessage returns a random feint message set.
func getFeintMessage() feintMessage {
	return feintMessages[util.Rand(len(feintMessages))]
}
