package actions

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TauntResult holds the outcome of a taunt/howl conviction attack for the
// caller to use when formatting messages and firing events.
type TauntResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// Executed reports whether the taunt was actually performed. False when any
	// early-exit condition fired (OnCooldown, NoTarget).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the taunt.
	OnCooldown bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool

	// Hit reports whether the conviction attack landed.
	Hit bool

	// Crit reports whether the attack was a critical success (ZScore >= 2.0).
	Crit bool

	// Fumble reports whether the attack was a critical failure (ZScore <= -2.0).
	Fumble bool

	// Damage is the conviction damage dealt to the target on a hit.
	Damage int

	// DmgDesc is the player-facing description of the conviction damage.
	DmgDesc string

	// SelfDamage is the self-conviction damage taken on a fumble.
	SelfDamage int

	// Deflected reports whether the target partially resisted the conviction damage.
	Deflected bool

	// CritDeflected reports whether the target fully negated the conviction damage.
	CritDeflected bool
}

// ExecuteTaunt performs the shared conviction-attack resolution used by both
// the player "taunt" command and the mob "howl" command. It handles:
//   - Aggro / cooldown guards
//   - Target resolution via ResolveAggroTarget
//   - Skill-weighted conviction vs. willpower opposed roll
//   - Conviction depletion penalty via ResourceMultiplier
//   - Fumble self-damage, normal hit with mitigation, crit bypass
//   - OnSkillUse progression trigger (fires on all outcomes)
//   - RecordSpecialMove analytics + RoundsWaiting consumption
//
// Callers are responsible for all player-facing messages and any
// out-of-combat aggro setup (e.g. player targeting before entering combat).
func ExecuteTaunt(actor Actor) TauntResult {
	char := actor.GetCharacter()

	// Must be in combat (aggro set) before calling.
	if char.Aggro == nil {
		return TauntResult{NoTarget: true}
	}

	// Check shared special-move cooldown.
	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return TauntResult{OnCooldown: true}
	}

	// Resolve target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return TauntResult{NoTarget: true}
	}

	// Conviction attack: Charisma + (rhetoric * SkillWeight) vs
	//                    Willpower   + (rhetoric * SkillWeight).
	skillWeight := float64(cfg.SkillWeight)
	attackerRhetoric := float64(char.GetSkillLevel(skills.Rhetoric)) * skillWeight
	attackScore := float64(char.Stats.Charisma.ValueAdj) + attackerRhetoric

	defenderRhetoric := float64(target.Char.GetSkillLevel(skills.Rhetoric)) * skillWeight
	defenseScore := float64(target.Char.Stats.Willpower.ValueAdj) + defenderRhetoric

	// Apply smooth conviction-depletion penalty to hit chance.
	cpPenalty := float64(cfg.ConvictionPenaltyMax)
	convMult := combat.ResourceMultiplier(char.Conviction, char.ConvictionMax.Value, cpPenalty)
	attackScore *= convMult

	// Opposed roll for hit/miss/fumble/crit classification.
	attackSuccess, _, atkRoll, _ := dice.OpposedRollStat(attackScore, defenseScore)

	// Determine source/target types for analytics.
	sourceType := combat.User
	if !actor.IsPlayer() {
		sourceType = combat.Mob
	}
	targetType := combat.User
	if target.MobInstanceId > 0 {
		targetType = combat.Mob
	}

	// Fumble: self-conviction damage and early return.
	if atkRoll.ZScore <= -2.0 {
		selfDmg := char.Stats.Charisma.ValueAdj / 10
		if selfDmg < 1 {
			selfDmg = 1
		}
		char.Conviction -= selfDmg
		if char.Conviction < 0 {
			char.Conviction = 0
		}

		actor.OnSkillUse(string(skills.Rhetoric))

		combat.RecordSpecialMove(sourceType, targetType, "taunt", false, 0,
			char, target.Char, util.GetRoundCount())

		if char.Aggro != nil {
			char.Aggro.RoundsWaiting = 1
		}

		return TauntResult{
			Target:     target,
			Executed:   true,
			Fumble:     true,
			SelfDamage: selfDmg,
		}
	}

	if attackSuccess {
		// Calculate conviction damage via the unified pipeline.
		rawDmg := combat.CalcRawDamage(
			char.Stats.Charisma.ValueAdj,
			int(attackerRhetoric),
			0.5, // taunt base item multiplier
			combat.ChannelConviction,
		)

		// Apply smooth conviction-depletion damage penalty.
		rawDmg *= convMult

		isCrit := atkRoll.ZScore >= 2.0

		var dmg int
		if isCrit {
			// Crit: bypass mitigation entirely.
			dmgRoll := dice.RollStat(rawDmg)
			dmg = int(math.Round(dmgRoll.Value))
		} else {
			// Normal hit: apply target conviction mitigation.
			mitigPct := target.Char.GetConvictionMitigation()
			cap := combat.MitigationCap(combat.ChannelConviction)
			dmgMean := combat.ApplyMitigation(rawDmg, mitigPct, cap)
			dmgRoll := dice.RollStat(dmgMean)
			dmg = int(math.Round(dmgRoll.Value))
		}
		if dmg < 1 {
			dmg = 1
		}

		// Stoic Resolve: defender attempts to partially resist
		deflected := false
		critDeflect := false
		if !isCrit {
			defenderUserId := 0
			if target.UserId > 0 {
				defenderUserId = target.UserId
			}
			resolveMult := combat.TryStoicResolve(char, target.Char, defenderUserId)
			if resolveMult < 1.0 {
				deflected = true
				if resolveMult == 0.0 {
					critDeflect = true
				}
				dmg = int(math.Round(float64(dmg) * resolveMult))
				if dmg < 1 && resolveMult > 0 {
					dmg = 1
				}
			}
		}

		// Apply conviction damage to target.
		target.Char.Conviction -= dmg
		if target.Char.Conviction < 0 {
			target.Char.Conviction = 0
		}

		// Build player-facing damage description.
		convMaxRef := target.Char.ConvictionMax.Value
		if convMaxRef <= 0 {
			convMaxRef = 1
		}
		dmgDesc := combat.GetConvictionDamageDescription(dmg, convMaxRef)

		// Crit received: conviction progression for target player.
		if isCrit && target.UserId > 0 {
			target.Char.OnCritReceived("conviction", target.UserId)
		}

		actor.OnSkillUse(string(skills.Rhetoric))

		combat.RecordSpecialMove(sourceType, targetType, "taunt", true, dmg,
			char, target.Char, util.GetRoundCount())

		if char.Aggro != nil {
			char.Aggro.RoundsWaiting = 1
		}

		return TauntResult{
			Target:        target,
			Executed:      true,
			Hit:           true,
			Crit:          isCrit,
			Damage:        dmg,
			DmgDesc:       dmgDesc,
			Deflected:     deflected,
			CritDeflected: critDeflect,
		}
	}

	// Miss.
	actor.OnSkillUse(string(skills.Rhetoric))

	combat.RecordSpecialMove(sourceType, targetType, "taunt", false, 0,
		char, target.Char, util.GetRoundCount())

	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	return TauntResult{
		Target:   target,
		Executed: true,
		Hit:      false,
	}
}
