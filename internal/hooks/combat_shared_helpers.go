package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/users"
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

		// Apply weapon spell damage multiplier (caster weapons), scaled
		// by gear-effectiveness for incorporeal casters.
		if caster.Equipment.Weapon.ItemId > 0 {
			if sdm := caster.Equipment.Weapon.GetSpec().SpellDamageMultiplier; sdm > 0 {
				rawDmg *= sdm * mutations.GearEffectivenessMultiplier(caster.Mutations)
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

// CritEffectResult holds the outcome of applying defense crit effects.
type CritEffectResult struct {
	// Parry crit → riposte (free counter-attack)
	Riposte       bool
	RiposteDamage int
	RiposteMaxHP  int
	// Dodge crit → auto-trip (ignores cooldown)
	AutoTrip   bool
	TripResult combat.SkillMoveResult
	// Block crit → auto-bash (ignores cooldown)
	AutoBash   bool
	BashResult combat.SkillMoveResult
	// Messages for all crit effects
	DefenderMsg string
	AttackerMsg string
	RoomMsg     string
}

// applyCritEffects processes parry/dodge/block crit effects for any combat
// pairing. The attacker is the one who swung. The defender is the one who
// rolled the defense crit and gains the benefit.
func applyCritEffects(attacker, defender *characters.Character, roundResult combat.AttackResult, room *rooms.Room) CritEffectResult {
	result := CritEffectResult{}
	cfg := configs.GetBalanceConfig()

	// ── Parry crit → riposte: free counter-swing ────────────────────────
	if roundResult.ParryCritDetected {
		raw := combat.CalcRawDamage(
			defender.Stats.Strength.ValueAdj,
			defender.GetCombatSkillLevel(),
			0.5, // riposte hits at half weapon damage
			combat.ChannelPhysical,
		)
		dmgMean := combat.ApplyMitigation(raw, attacker.GetPhysicalMitigation(),
			combat.MitigationCap(combat.ChannelPhysical))
		roll := dice.RollStat(dmgMean)
		dmg := int(roll.Value)
		if dmg < 1 {
			dmg = 1
		}

		attacker.Health -= dmg
		if attacker.Health < 0 {
			attacker.Health = 0
		}
		result.Riposte = true
		result.RiposteDamage = dmg
		result.RiposteMaxHP = attacker.HealthMax.Value

		dmgDesc := combat.GetDamageDescription(dmg, attacker.HealthMax.Value)
		result.DefenderMsg = fmt.Sprintf(
			`<ansi fg="cyan-bold">⚔ RIPOSTE!</ansi> You turn the parry into a swift counter-strike! (<ansi fg="damage">%s</ansi>)`, dmgDesc)
		result.AttackerMsg = fmt.Sprintf(
			`<ansi fg="cyan-bold">⚔ RIPOSTE!</ansi> %s turns the parry into a lightning counter-strike! (<ansi fg="damage">%s</ansi>)`, defender.Name, dmgDesc)
		result.RoomMsg = fmt.Sprintf(
			`<ansi fg="cyan-bold">⚔ RIPOSTE!</ansi> %s turns a deft parry into a counter-strike against %s!`, defender.Name, attacker.Name)
	}

	// ── Dodge crit → auto-trip (ignores cooldown) ───────────────────────
	if roundResult.DodgeCritDetected {
		tripResult := combat.ExecuteSkillMove(combat.SkillMoveParams{
			Attacker:        defender,
			Defender:        attacker,
			AttackStat:      defender.Stats.Dexterity.ValueAdj,
			AttackSkill:     defender.GetSkillLevel(skills.UnarmedCombat),
			DefenseStat:     attacker.Stats.Dexterity.ValueAdj,
			DefenseSkill:    attacker.GetCombatSkillLevel(),
			DamagePercent:   float64(cfg.TripDamagePercent),
			KnockdownChance: int(cfg.TripKnockdownChance),
			SkillRank:       defender.GetSkillLevel(skills.UnarmedCombat),
			DamageStat:      defender.Stats.Dexterity.ValueAdj,
		})
		result.AutoTrip = true
		result.TripResult = tripResult

		if tripResult.Hit {
			dmgDesc := combat.GetDamageDescription(tripResult.Damage, tripResult.TargetMaxHP)
			if tripResult.KnockedDown {
				result.DefenderMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">⚡ SWEEP!</ansi> You dodge and sweep their legs out! They crash to the ground! (<ansi fg="damage">%s</ansi>)`, dmgDesc)
				result.AttackerMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">⚡ SWEEP!</ansi> %s dodges and sweeps your legs! You crash to the ground! (<ansi fg="damage">%s</ansi>)`, defender.Name, dmgDesc)
				result.RoomMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">⚡ SWEEP!</ansi> %s dodges and sweeps %s to the ground!`, defender.Name, attacker.Name)
			} else {
				result.DefenderMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">⚡ SWEEP!</ansi> You dodge and lash out at their legs! (<ansi fg="damage">%s</ansi>)`, dmgDesc)
				result.AttackerMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">⚡ SWEEP!</ansi> %s dodges and lashes out at your legs! (<ansi fg="damage">%s</ansi>)`, defender.Name, dmgDesc)
				result.RoomMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">⚡ SWEEP!</ansi> %s dodges and kicks at %s's legs!`, defender.Name, attacker.Name)
			}
		} else {
			result.DefenderMsg = `<ansi fg="cyan-bold">⚡ SWEEP!</ansi> You dodge and try to sweep their legs, but they keep their footing!`
			result.AttackerMsg = fmt.Sprintf(
				`<ansi fg="cyan-bold">⚡ SWEEP!</ansi> %s dodges and tries to sweep your legs, but you keep your footing!`, defender.Name)
			result.RoomMsg = fmt.Sprintf(
				`<ansi fg="cyan-bold">⚡ SWEEP!</ansi> %s dodges and tries to sweep %s's legs, but misses!`, defender.Name, attacker.Name)
		}
	}

	// ── Block crit → auto-bash (ignores cooldown) ───────────────────────
	if roundResult.BlockCritDetected {
		bashResult := combat.ExecuteSkillMove(combat.SkillMoveParams{
			Attacker:        defender,
			Defender:        attacker,
			AttackStat:      defender.Stats.Strength.ValueAdj,
			AttackSkill:     defender.GetSkillLevel(skills.WeaponCombat),
			DefenseStat:     attacker.Stats.Dexterity.ValueAdj,
			DefenseSkill:    attacker.GetCombatSkillLevel(),
			DamagePercent:   float64(cfg.BashDamagePercent),
			KnockdownChance: int(cfg.BashKnockdownChance),
			SkillRank:       defender.GetSkillLevel(skills.WeaponCombat),
			DamageStat:      defender.Stats.Strength.ValueAdj,
		})
		result.AutoBash = true
		result.BashResult = bashResult

		if bashResult.Hit {
			dmgDesc := combat.GetDamageDescription(bashResult.Damage, bashResult.TargetMaxHP)
			if bashResult.KnockedDown {
				result.DefenderMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">🛡 SHIELD SLAM!</ansi> You catch the blow on your shield and slam them back! They stumble and fall! (<ansi fg="damage">%s</ansi>)`, dmgDesc)
				result.AttackerMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">🛡 SHIELD SLAM!</ansi> %s catches your blow on their shield and slams you back! You stumble and fall! (<ansi fg="damage">%s</ansi>)`, defender.Name, dmgDesc)
				result.RoomMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">🛡 SHIELD SLAM!</ansi> %s blocks and shield-slams %s to the ground!`, defender.Name, attacker.Name)
			} else {
				result.DefenderMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">🛡 SHIELD SLAM!</ansi> You catch the blow and slam your shield into them! (<ansi fg="damage">%s</ansi>)`, dmgDesc)
				result.AttackerMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">🛡 SHIELD SLAM!</ansi> %s catches your blow and slams their shield into you! (<ansi fg="damage">%s</ansi>)`, defender.Name, dmgDesc)
				result.RoomMsg = fmt.Sprintf(
					`<ansi fg="cyan-bold">🛡 SHIELD SLAM!</ansi> %s blocks and slams their shield into %s!`, defender.Name, attacker.Name)
			}
		} else {
			result.DefenderMsg = `<ansi fg="cyan-bold">🛡 SHIELD SLAM!</ansi> You catch the blow and try to slam them, but they brace against it!`
			result.AttackerMsg = fmt.Sprintf(
				`<ansi fg="cyan-bold">🛡 SHIELD SLAM!</ansi> %s catches your blow and tries to slam you, but you brace against it!`, defender.Name)
			result.RoomMsg = fmt.Sprintf(
				`<ansi fg="cyan-bold">🛡 SHIELD SLAM!</ansi> %s blocks and tries to slam %s, but they hold firm!`, defender.Name, attacker.Name)
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

// clearCastingActivity fires Activity.TransitionToFree with the given trigger
// if the character's Activity machine is currently in the Casting state.
// Used alongside legacy CastingState = nil through Task 11.
func clearCastingActivity(ch *characters.Character, trigger string) {
	if ch.Activity != nil && ch.Activity.IsCasting() {
		_ = ch.Activity.TransitionToFree(state.TransitionReason{
			Trigger: trigger,
			Actor:   ch.Activity.Self(),
		})
	}
}

// cancelCraftOrSalvageOnDamage cancels the character's Activity if it is
// Crafting or Salvaging. This is a hard cancel — no roll — because any
// damage interrupts physical work immediately. Casting interrupts are
// handled separately by clearCastingActivity (concentration break via
// willpower roll). Both helpers may be called together at the same damage
// site: they are independent and each no-ops when the state doesn't match.
func cancelCraftOrSalvageOnDamage(ch *characters.Character) {
	if ch.Activity == nil {
		return
	}
	switch ch.Activity.State() {
	case activity.Crafting:
		_ = ch.Activity.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerDamageInterrupt,
			Actor:   ch.Activity.Self(),
		})
		ch.CraftingState = nil
	case activity.Salvaging:
		_ = ch.Activity.TransitionToFree(state.TransitionReason{
			Trigger: activity.TriggerDamageInterrupt,
			Actor:   ch.Activity.Self(),
		})
		ch.CraftingState = nil
		delete(ch.MiscData, "salvage_item_uuid")
		delete(ch.MiscData, "salvage_spoiled_potion")
	}
}

// =============================================================================
// processFoldRound — shared fold casting step for both players and mobs.
//
// Handles all state mutations that are identical for every caster type:
//   - Prone → concentration break
//   - Target-gone check
//   - Simulate fold advance → compute conviction cost
//   - Insufficient conviction → break
//   - Deduct conviction + advance folds
//
// The caller (handlePlayerFoldCasting / handleMobFoldCasting) is responsible
// for all messaging and spell resolution, because those differ by caster type.
// =============================================================================

// FoldRoundResult describes the outcome of a single fold casting step.
// Exactly one of the boolean fields will be true; StillCasting and
// CastComplete are mutually exclusive; the break fields are all terminal.
type FoldRoundResult struct {
	// Terminal states — caller should return after messaging.
	ProneBroke             bool // caster fell prone, concentration broken
	TargetGone             bool // all targets are dead/gone
	SpellDataMissing       bool // spells.GetSpell returned nil
	InsufficientConviction bool // not enough CP to pay this fold's cost

	// Ongoing states.
	StillCasting bool // folds advanced but not yet complete; send in-progress msg
	CastComplete bool // folds complete; caller should resolve the spell

	// Values the caller needs for messaging / resolution.
	FoldDelta      int                      // folds simulated this round
	ConvictionCost int                      // CP deducted from caster this round
	SpellData      *spells.SpellData        // non-nil when SpellDataMissing==false
	CastingState   *characters.CastingState // same pointer as char.CastingState
}

// processFoldRound advances one round of fold casting for any caster character.
// It mutates char.CastingState (deducting conviction, advancing folds, or
// clearing the state on terminal conditions).  Returns a FoldRoundResult
// describing what happened so the caller can emit the right messages.
func processFoldRound(char *characters.Character) FoldRoundResult {
	cs := char.CastingState // caller must have verified non-nil

	// Prone → immediate concentration break.
	if char.CombatPosition == characters.PositionProne {
		clearCastingActivity(char, activity.TriggerConcentrationBreak)
		char.CastingState = nil
		return FoldRoundResult{ProneBroke: true, CastingState: cs}
	}

	// Target-gone check: any dead/nil target breaks the spell.
	spellData := spells.GetSpell(cs.SpellId)
	targetGone := false
	for _, mobInstId := range cs.TargetMobInstanceIds {
		m := mobs.GetInstance(mobInstId)
		if m == nil || m.Character.Health < 1 {
			targetGone = true
			break
		}
	}
	if !targetGone {
		for _, targetUserId := range cs.TargetUserIds {
			u := users.GetByUserId(targetUserId)
			if u == nil {
				targetGone = true
				break
			}
			// For harm spells, downed players count as gone.
			if u.Character.Health < 1 && spellData != nil &&
				(spellData.Type == spells.HarmSingle || spellData.Type == spells.HarmArea || spellData.Type == spells.HarmMulti) {
				targetGone = true
				break
			}
		}
	}
	if targetGone {
		clearCastingActivity(char, activity.TriggerConcentrationBreak)
		char.CastingState = nil
		return FoldRoundResult{TargetGone: true, CastingState: cs}
	}

	if spellData == nil {
		clearCastingActivity(char, activity.TriggerConcentrationBreak)
		char.CastingState = nil
		return FoldRoundResult{SpellDataMissing: true, CastingState: cs}
	}

	// Simulate fold advance → compute conviction cost.
	foldDelta := simulateFoldRound(cs)
	roundCost := calcFoldConvictionCost(cs, foldDelta)

	if roundCost > 0 && char.Conviction < roundCost {
		clearCastingActivity(char, activity.TriggerConcentrationBreak)
		char.CastingState = nil
		return FoldRoundResult{
			InsufficientConviction: true,
			FoldDelta:              foldDelta,
			ConvictionCost:         roundCost,
			SpellData:              spellData,
			CastingState:           cs,
		}
	}

	// Deduct conviction and advance folds.
	char.Conviction -= roundCost
	cs.ConvictionSpent += roundCost

	complete := advanceFolds(cs)
	if complete {
		clearCastingActivity(char, activity.TriggerCastComplete)
		char.CastingState = nil
		return FoldRoundResult{
			CastComplete:   true,
			FoldDelta:      foldDelta,
			ConvictionCost: roundCost,
			SpellData:      spellData,
			CastingState:   cs,
		}
	}
	return FoldRoundResult{
		StillCasting:   true,
		FoldDelta:      foldDelta,
		ConvictionCost: roundCost,
		SpellData:      spellData,
		CastingState:   cs,
	}
}
