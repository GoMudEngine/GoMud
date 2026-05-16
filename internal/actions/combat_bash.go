package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// BashResult holds the outcome of a bash attempt for the caller to use when
// formatting messages, firing events, and updating UI.
type BashResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// MoveResult is the outcome from ExecuteSkillMove. Valid only when Executed is true.
	MoveResult combat.SkillMoveResult

	// Executed reports whether the bash was actually performed. False when any
	// early-exit condition fired (OnCooldown, NoTarget, NoShield).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the bash.
	OnCooldown bool

	// Crafting is true when the actor is mid-craft and cannot bash.
	Crafting bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool

	// NoShield is true when the actor has no shield equipped.
	NoShield bool
}

// ExecuteBash performs the core bash resolution shared between player and mob
// callers. It handles:
//   - Shield check
//   - Special-move cooldown (using SpecialMoveCooldown from balance config)
//   - Target resolution via ResolveAggroTarget
//   - ExecuteSkillMove via combat package
//   - RecordAndWait (analytics + round consumption)
//
// Callers are responsible for all messaging, skill progression events, and
// any combat-initiation logic (e.g. SetAggro for player out-of-combat bash).
func ExecuteBash(actor Actor) BashResult {
	char := actor.GetCharacter()

	// Don't interrupt any active activity (cast/craft/salvage) to swing a shield.
	if char.IsActing() {
		return BashResult{Crafting: true}
	}

	// Must be in combat (aggro set) before this function is called.
	if char.Aggro == nil {
		return BashResult{NoTarget: true}
	}

	// Must have a shield equipped.
	if !char.HasShield() {
		return BashResult{NoShield: true}
	}

	// Check special-move cooldown using the config value.
	cfg := configs.GetBalanceConfig()
	cooldownStr := fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)
	if !char.Cooldowns.Try("special-move", cooldownStr) {
		return BashResult{OnCooldown: true}
	}

	// Resolve the aggro target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return BashResult{NoTarget: true}
	}

	// Execute the skill move.
	result := combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:        char,
		Defender:        target.Char,
		AttackStat:      char.Stats.Strength.ValueAdj,
		AttackSkill:     char.GetSkillLevel(skills.WeaponCombat),
		DefenseStat:     target.Char.Stats.Dexterity.ValueAdj,
		DefenseSkill:    target.Char.GetCombatSkillLevel(),
		DamagePercent:   float64(cfg.BashDamagePercent),
		KnockdownChance: int(cfg.BashKnockdownChance),
		SkillRank:       char.GetSkillLevel(skills.WeaponCombat),
		DamageStat:      char.Stats.Strength.ValueAdj,
	})

	// Determine source/target types for analytics.
	sourceType := combat.User
	if !actor.IsPlayer() {
		sourceType = combat.Mob
	}
	targetType := combat.User
	if target.MobInstanceId > 0 {
		targetType = combat.Mob
	}

	// Record analytics and consume the combat round.
	dmgRecorded := 0
	if result.Hit {
		dmgRecorded = result.Damage
	}
	RecordAndWait(char, "bash", sourceType, target.Char, targetType, result.Hit, dmgRecorded, util.GetRoundCount())

	// Progression: weapon-combat on hit (moved from user/mob wrappers)
	if result.Hit {
		actor.OnSkillUse(string(skills.WeaponCombat))
	}

	return BashResult{
		Target:     target,
		MoveResult: result,
		Executed:   true,
	}
}
