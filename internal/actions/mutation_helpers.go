package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// MutationOpts carries optional inputs to a mutation-active trigger (e.g. a
// pre-resolved target actor for single-target actives). Shared by the actives
// still in the actions package (cocoon, venom coat).
type MutationOpts struct {
	TargetActor Actor
}

// MutationResult reports the outcome of a mutation-active trigger for the
// behaviortree dispatch layer.
type MutationResult struct {
	Triggered     bool
	BlockReason   string
	AffectedCount int
}

// preambleResult is the outcome of a mutation-active preamble check.
// On OK=false, BlockReason is set to one of:
//
//	"no-character"   – actor.GetCharacter() returned nil
//	"busy"           – actor's Activity machine is occupied (crafting etc.)
//	"no-mutation"    – actor does not own the requested mutation
//	"not-in-combat"  – combatRequired=true but actor is not in combat
//	"on-cooldown"    – the shared special-move cooldown is still active
//	"low-stamina"    – actor.Stamina < staminaCost
type preambleResult struct {
	OK          bool
	BlockReason string
}

// mutationPreamble runs the shared gates for any mutation-active command:
//
//  0. Activity exclusivity — actor must not be Crafting/Salvaging/Casting
//  1. Mutation ownership  — actor must have mutationKey in Mutations map
//  2. Combat requirement  — if combatRequired, actor must be in combat
//  3. Cooldown            — the shared "special-move" cooldown must be free
//  4. Stamina cost        — actor must have at least staminaCost stamina
//
// On success, staminaCost is deducted from the actor's Stamina and OK=true
// is returned. Stamina is ONLY deducted on full success, never on an
// intermediate block.
//
// Player actors receive a descriptive message for each block condition and
// for no other reason (matches the pattern used in forage.go, salvage.go,
// cast.go). Mob actors are silently gated (MobActor.SendText is a no-op).
//
// Current callers: venom-coat and cocoon. (The six legacy actives this
// comment used to name — blinding-flash, blinding-spit, healing-gel,
// pacifism-aura, sonic-shout, toxic-bite — were RETIRED in the 0.14.0
// Chrysalis migration and have no Trigger functions anymore.)
func mutationPreamble(actor Actor, mutationKey string, combatRequired bool, staminaCost int) preambleResult {
	char := actor.GetCharacter()
	if char == nil {
		return preambleResult{BlockReason: "no-character"}
	}

	// Gate 0: activity exclusivity (2026-08-03 crafting-focus audit) —
	// firing an active mid-craft/mid-cast was the rally/warcry bug again
	// through a different door. Applies to mobs too: a crafting shopkeep
	// should not venom-coat between hammer strokes.
	if char.IsActing() {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				"You can't use that ability while focused on your work. Finish or be interrupted first.")
		}
		return preambleResult{BlockReason: "busy"}
	}

	// Gate 1: mutation ownership.
	if !mutations.HasMutation(char.Mutations, mutationKey) {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, "You don't have that ability.")
		}
		return preambleResult{BlockReason: "no-mutation"}
	}

	// Gate 2: combat requirement.
	if combatRequired && !char.IsInCombat() {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf("You must be in combat to use %s!", mutationKey))
		}
		return preambleResult{BlockReason: "not-in-combat"}
	}

	// Gate 3: shared special-move cooldown.
	cfg := configs.GetBalanceConfig()
	cooldownStr := fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)
	if !char.Cooldowns.Try("special-move", cooldownStr) {
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem,
				"You need a moment to recover before attempting another special move.")
		}
		return preambleResult{BlockReason: "on-cooldown"}
	}

	// Gate 4: stamina cost.
	if char.Stamina < staminaCost {
		// Cooldown was consumed by the Try call above; roll it back so the
		// actor isn't punished with a cooldown for a failed attempt.
		delete(char.Cooldowns, "special-move")
		if actor.IsPlayer() {
			actor.SendText(messaging.CategorySystem, "You're too exhausted!")
		}
		return preambleResult{BlockReason: "low-stamina"}
	}
	char.Stamina -= staminaCost

	return preambleResult{OK: true}
}
