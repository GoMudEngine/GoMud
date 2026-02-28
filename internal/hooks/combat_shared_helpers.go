package hooks

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// =============================================================================
// Stage 38.1: Shared combat helpers for mob/player unification
//
// Each helper operates on *characters.Character so it works for both players
// and mobs. Callers handle messaging (since mob vs player messaging differs).
// =============================================================================

// calcSpellDamageForCharacter computes spell damage using the pipeline (if
// spell has DamageMultiplier > 0) or the legacy path. Works for any caster.
func calcSpellDamageForCharacter(spellData *spells.SpellData, caster *characters.Character, target *characters.Character, magnitude int, isCrit bool) int {

	if spellData.DamageMultiplier > 0 && caster != nil {
		skillLevel := caster.GetSkillLevel(skills.Spellcasting)
		rawDmg := combat.CalcRawDamage(caster.Stats.Willpower.ValueAdj, skillLevel, spellData.DamageMultiplier, combat.ChannelMagical)

		// Apply weapon spell damage multiplier (caster weapons)
		if caster.Equipment.Weapon.ItemId > 0 {
			if sdm := caster.Equipment.Weapon.GetSpec().SpellDamageMultiplier; sdm > 0 {
				rawDmg *= sdm
			}
		}

		// Apply smooth conviction-based spell damage penalty
		cpPenalty := float64(configs.GetBalanceConfig().ConvictionPenaltyMax)
		rawDmg *= combat.ResourceMultiplier(caster.Conviction,
			caster.ConvictionMax.Value, cpPenalty)

		// Apply mitigation based on defense type
		var mitigPct, cap float64
		switch spellData.TargetDefenseType {
		case "physical":
			mitigPct = target.GetPhysicalMitigation()
			cap = combat.MitigationCap(combat.ChannelPhysical)
		case "mental":
			mitigPct = target.GetMagicalMitigation()
			cap = combat.MitigationCap(combat.ChannelMagical)
		default:
			mitigPct = 0
			cap = 0.75
		}

		if isCrit {
			// Crit: bypass mitigation entirely — use raw damage
			dmgRoll := dice.RollStat(rawDmg)
			dmg := int(math.Round(dmgRoll.Value))
			if dmg < 1 {
				dmg = 1
			}
			return dmg
		}

		dmgMean := combat.ApplyMitigation(rawDmg, mitigPct, cap)
		dmgRoll := dice.RollStat(dmgMean)
		dmg := int(math.Round(dmgRoll.Value))
		if dmg < 1 {
			dmg = 1
		}
		return dmg
	}

	// Legacy fallback: magnitude-based damage (no mitigation, no stat scaling)
	mudlog.Warn("calcSpellDamageForCharacter", "spell", spellData.SpellId, "msg", "using legacy magnitude path — add damage_multiplier to spell YAML")
	dmgRoll := dice.RollStat(float64(magnitude))
	dmg := int(math.Round(dmgRoll.Value))
	if dmg < 1 {
		dmg = 1
	}
	if isCrit {
		dmg += magnitude
	}
	return dmg
}

// checkConcentrationBreak tests whether a casting character's concentration
// breaks from taking damage. Returns true if concentration broke.
// Caller is responsible for clearing CastingState and sending messages.
func checkConcentrationBreak(ch *characters.Character, damage int) bool {
	if ch.CastingState == nil || damage <= 0 {
		return false
	}
	maxHealth := ch.HealthMax.Value
	damagePct := damage * 100 / maxHealth
	if damagePct < 1 {
		damagePct = 1
	}
	chance := characters.CalcConcentrationChance(
		ch.Stats.Willpower.ValueAdj, damagePct)
	roll := util.Rand(100)
	util.LogRoll(`Concentration`, roll, chance)
	return roll >= chance
}

// WeaponBreakResult holds the outcome of a weapon break test.
type WeaponBreakResult struct {
	Broke           bool
	BrokenItemName  string
	BrokenItem      items.Item
	ReplacementItem items.Item
}

// tryWeaponBreak tests offhand breakage for any character and performs the
// item swap (remove broken offhand, create broken-item #20). Returns the
// result so callers can emit ItemOwnership events and messages.
// The room param is used as a fallback for item drops; nil-safe.
func tryWeaponBreak(defender *characters.Character, roundResult combat.AttackResult, room *rooms.Room) WeaponBreakResult {
	result := WeaponBreakResult{}

	if !roundResult.Hit {
		return result
	}
	if defender.Equipment.Offhand.ItemId <= 0 {
		return result
	}

	modifier := 0
	if roundResult.Crit {
		modifier = int(defender.Equipment.Offhand.GetSpec().BreakChance)
	}

	if !defender.Equipment.Offhand.BreakTest(modifier) {
		return result
	}

	result.Broke = true
	result.BrokenItemName = defender.Equipment.Offhand.NameSimple()
	result.BrokenItem = defender.Equipment.Offhand

	defender.RemoveFromBody(defender.Equipment.Offhand)

	itm := items.New(20) // Broken item
	result.ReplacementItem = itm
	if !defender.StoreItem(itm) {
		if room != nil {
			room.AddItem(itm, false)
		}
	}

	return result
}

// CritEffectResult holds the outcome of applying crit effects from defense.
type CritEffectResult struct {
	Disarmed    bool
	DisarmItem  combat.DisarmResult
	GrappleSet  bool
}

// applyCritEffects processes parry/dodge crit effects for any combat pairing.
// The defender is the one who parried/dodged (and gains the crit benefit).
// The attacker is the one whose weapon may be disarmed or guard penetrated.
// Returns a result struct so callers can route messages appropriately.
func applyCritEffects(attacker, defender *characters.Character, roundResult combat.AttackResult, room *rooms.Room) CritEffectResult {
	result := CritEffectResult{}

	// Parry crit: defender disarms attacker (10% chance)
	if roundResult.ParryCritDetected {
		disarmResult := combat.AttemptCritDisarm(defender, attacker, 10.0)
		if disarmResult.Success {
			if room != nil {
				room.AddItem(disarmResult.Weapon, false)
			}
			result.Disarmed = true
			result.DisarmItem = disarmResult
		}
	}

	// Dodge crit: defender gets grapple opportunity
	// Bug fix: always check cooldown (PvM path was missing this)
	if roundResult.DodgeCritDetected {
		if defender.Cooldowns["special-move"] <= 0 {
			combat.SetGrappleOpportunity(defender)
			result.GrappleSet = true
		}
	}

	return result
}

// simulateFoldRound simulates the fold doubling loop to determine how many
// folds will be gained this round (without actually modifying state).
// Returns the foldDelta.
func simulateFoldRound(cs *characters.CastingState) int {
	simFolds := cs.FoldsAccumulated
	for i := 0; i < cs.FoldsPerRound; i++ {
		if simFolds == 0 {
			simFolds = 1
		} else {
			simFolds *= 2
		}
		if simFolds >= cs.FoldsNeeded {
			simFolds = cs.FoldsNeeded
			break
		}
	}
	return simFolds - cs.FoldsAccumulated
}

// calcFoldConvictionCost computes the conviction cost for a fold round,
// proportional to folds gained vs total folds needed.
func calcFoldConvictionCost(cs *characters.CastingState, foldDelta int) int {
	if cs.TotalConvictionCost <= 0 || cs.FoldsNeeded <= 0 {
		return 0
	}
	roundCost := (cs.TotalConvictionCost * foldDelta) / cs.FoldsNeeded
	if roundCost < 1 {
		roundCost = 1
	}
	return roundCost
}

// advanceFolds performs the real fold doubling loop, advancing
// FoldsAccumulated. Returns true when folds reach FoldsNeeded (spell ready).
func advanceFolds(cs *characters.CastingState) bool {
	for i := 0; i < cs.FoldsPerRound; i++ {
		if cs.FoldsAccumulated == 0 {
			cs.FoldsAccumulated = 1
		} else {
			cs.FoldsAccumulated *= 2
		}
		if cs.FoldsAccumulated > cs.FoldsNeeded {
			cs.FoldsAccumulated = cs.FoldsNeeded
		}
		if cs.FoldsAccumulated >= cs.FoldsNeeded {
			return true
		}
	}
	return false
}
