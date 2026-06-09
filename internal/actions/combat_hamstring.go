package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// HamstringResult holds the outcome of a hamstring attempt for the caller to
// use when formatting messages, firing events, and updating UI.
type HamstringResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// MoveResult is the outcome from ExecuteSkillMove. Valid only when Executed is true.
	MoveResult combat.SkillMoveResult

	// Executed reports whether the hamstring was actually performed. False when any
	// early-exit condition fired (OnCooldown, NoTarget).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the hamstring.
	OnCooldown bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool

	// NotBeast is true when the actor lacks the beast anatomy required for
	// hamstring (not fanged-or-clawed, or has "hands" marking it as a
	// tool-user). Returned by the defense-in-depth identity gate; the AI path
	// is pre-gated by CanUseHamstring so this fires only on direct dispatch.
	NotBeast bool

	// BleedDmg is the per-tick bleed damage applied on a hit (Strength/10, min 2).
	BleedDmg int
}

// ExecuteHamstring performs the core hamstring resolution shared between player
// and mob callers. It handles:
//   - Special-move cooldown (using SpecialMoveCooldown from balance config)
//   - Target resolution via ResolveAggroTarget
//   - ExecuteSkillMove via combat package (UnarmedCombat skill, Dexterity attack
//     stat, Dexterity defense stat, TripDamagePercent, Strength damage stat,
//     no knockdown)
//   - On hit: apply ConditionBleeding (duration 5, magnitude = Strength/10 min 2)
//   - combat.RecordSpecialMove for analytics + RoundsWaiting = 1
//   - OnSkillUse(UnarmedCombat) on hit for progression
//
// Callers are responsible for all messaging and any combat-initiation logic.
func ExecuteHamstring(actor Actor) HamstringResult {
	char := actor.GetCharacter()

	// Must be in combat (aggro set) before this function is called.
	if char.Aggro == nil {
		return HamstringResult{NoTarget: true}
	}

	// Check special-move cooldown using the config value.
	cfg := configs.GetBalanceConfig()
	if !char.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return HamstringResult{OnCooldown: true}
	}

	// Resolve the aggro target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return HamstringResult{NoTarget: true}
	}

	// Beast identity gate (defense-in-depth): only handless fanged/clawed
	// creatures hamstring. Unreachable via the AI path (CanUseHamstring gates
	// it) but reachable via a direct player command or btree dispatch to a
	// non-beast mob.
	sp := species.GetSpecies(char.SpeciesId)
	if sp == nil || (sp.NaturalAttack != items.Bite && sp.NaturalAttack != items.Claws) || char.HasBodyPart("hands") {
		return HamstringResult{NotBeast: true}
	}

	// Execute the skill move (reuse trip's config for damage percent, no knockdown).
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        char,
		Defender:        target.Char,
		AttackStat:      char.Stats.Dexterity.ValueAdj,
		AttackSkill:     char.GetSkillLevel(skills.UnarmedCombat),
		DefenseStat:     target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.TripDamagePercent),
		KnockdownChance: 0, // No knockdown — bleed instead
		SkillRank:       char.GetSkillLevel(skills.UnarmedCombat),
		DamageStat:      char.Stats.Strength.ValueAdj,
	})

	// On hit: apply bleed condition (duration 5, magnitude = Strength/10, min 2).
	bleedDmg := 0
	if result.Hit {
		bleedDmg = char.Stats.Strength.ValueAdj / 10
		if bleedDmg < 2 {
			bleedDmg = 2
		}
		target.Char.AddCondition(characters.ConditionBleeding, 5, float64(bleedDmg), "hamstring")
	}

	// Determine source/target types for analytics.
	sourceType := combat.User
	if !actor.IsPlayer() {
		sourceType = combat.Mob
	}
	targetType := combat.User
	if target.MobInstanceId > 0 {
		targetType = combat.Mob
	}

	// Record combat analytics (matches existing hamstring.go pattern: RecordSpecialMove).
	dmgRecorded := 0
	if result.Hit {
		dmgRecorded = result.Damage
	}
	combat.RecordSpecialMove(sourceType, targetType, "hamstring", result.Hit, dmgRecorded, char, target.Char, util.GetRoundCount())

	// Consume the combat round.
	if char.Aggro != nil {
		char.Aggro.RoundsWaiting = 1
	}

	// Progression: unarmed-combat on hit.
	if result.Hit {
		actor.OnSkillUse(string(skills.UnarmedCombat))
	}

	return HamstringResult{
		Target:     target,
		MoveResult: result,
		Executed:   true,
		BleedDmg:   bleedDmg,
	}
}
