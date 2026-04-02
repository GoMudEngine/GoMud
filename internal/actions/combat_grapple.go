package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// GrappleResult holds the outcome of a grapple attempt for the caller to use
// when formatting messages, firing events, and updating UI.
type GrappleResult struct {
	// Target is the resolved aggro target. Valid only when Executed is true.
	Target AggroTarget

	// MoveResult is the outcome from ExecuteGrappleMove. Valid only when
	// Executed is true.
	MoveResult combat.GrappleMoveResult

	// Executed reports whether the grapple was actually performed. False when
	// any early-exit condition fired (OnCooldown, NoTarget).
	Executed bool

	// OnCooldown is true when the special-move cooldown blocked the grapple.
	OnCooldown bool

	// NoTarget is true when there is no aggro target or the target is gone.
	NoTarget bool
}

// ExecuteGrapple performs the core grapple resolution shared between player
// and mob callers. It handles:
//   - Special-move cooldown (using SpecialMoveCooldown from balance config)
//   - Target resolution via ResolveAggroTarget
//   - ExecuteGrappleMove via combat package
//   - RecordAndWait (analytics + round consumption)
//
// Callers are responsible for all messaging, skill progression events, and
// any combat-initiation logic (e.g. SetAggro for player out-of-combat
// grapple).
func ExecuteGrapple(actor Actor) GrappleResult {
	char := actor.GetCharacter()

	// Must be in combat (aggro set) before this function is called.
	if char.Aggro == nil {
		return GrappleResult{NoTarget: true}
	}

	// Check special-move cooldown using the config value.
	cfg := configs.GetBalanceConfig()
	cooldownStr := fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)
	if !char.Cooldowns.Try("special-move", cooldownStr) {
		return GrappleResult{OnCooldown: true}
	}

	// Resolve the aggro target.
	target := ResolveAggroTarget(char.Aggro)
	if !target.Found {
		return GrappleResult{NoTarget: true}
	}

	// Execute the grapple move. Player actors pass their UserId; mobs pass 0.
	attackerId := actor.GetUserId()
	result := combat.ExecuteGrappleMove(char, target.Char, attackerId, actor.GetRoom())

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
	RecordAndWait(char, "grapple", sourceType, target.Char, targetType, result.Success, 0, util.GetRoundCount())

	return GrappleResult{
		Target:     target,
		MoveResult: result,
		Executed:   true,
	}
}
