package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// GrappleResult represents the outcome of a grapple attempt
type GrappleResult struct {
	Success          bool
	Margin           float64
	NewPosition      characters.CombatPosition
	AttackScore      float64
	DefenseScore     float64
	AttackRoll       float64
	DefenseRoll      float64
	PositionPenalty  float64 // For defender if prone
	AttackZScore     float64 // For crit detection (Stage 8.4)
	DefenseZScore    float64 // For reference (Stage 8.4)
}

// SubmissionResult represents the outcome of a submission attempt (Stage 8.6)
type SubmissionResult struct {
	Success      bool
	Margin       float64
	AttackZScore float64
	Choice       string // "yield" or "resist"
}


// AttemptGrapple performs a grapple attempt from attacker to defender.
// Returns a GrappleResult with the outcome and details.
//
// Grapple calculation:
// attackScore = attacker.Dex + attacker.CombatSkill + weapon.GrappleModifier
// defenseScore = defender.Dex + defender.CombatSkill
//
// Position modifiers:
// - If defender is prone: defenseScore *= 0.3  (-70% defense when already down)
// - If attacker is prone: attackScore *= 0.5   (-50% offense when attacking from ground)
//
// Position transitions on success:
// - Standing → Clinched
// - Prone → Grounded (direct, skip Clinched)
func AttemptGrapple(attacker *characters.Character, defender *characters.Character) GrappleResult {
	result := GrappleResult{}

	// Base scores: Dex + Combat Skill
	attackerCombatSkill := float64(attacker.GetCombatSkillLevel())
	defenderCombatSkill := float64(defender.GetCombatSkillLevel())

	// Check for 1-round grapple opportunity from prior dodge crit (Stage 8.4)
	opportunityBonus := GetGrappleOpportunityBonus(attacker)

	result.AttackScore = (float64(attacker.Stats.Dexterity.ValueAdj) + attackerCombatSkill) * opportunityBonus
	result.DefenseScore = float64(defender.Stats.Dexterity.ValueAdj) + defenderCombatSkill

	// Clear opportunity after use if it was active (Stage 8.4)
	if opportunityBonus > 1.0 {
		ClearGrappleOpportunity(attacker)
	}

	// Add weapon grapple modifier (if wielding a weapon)
	if attacker.Equipment.Weapon.ItemId != 0 {
		weaponSpec := items.GetItemSpec(attacker.Equipment.Weapon.ItemId)
		if weaponSpec != nil {
			result.AttackScore += weaponSpec.GrappleModifier
		}
	}

	// Position modifiers
	if defender.CombatPosition == characters.PositionProne {
		// Defender at -70% defense when already down (brutal!)
		result.PositionPenalty = -0.7
		result.DefenseScore *= 0.3
	}

	if attacker.CombatPosition == characters.PositionProne {
		// Attacker at -50% offense when attacking from ground
		result.AttackScore *= 0.5
	}

	// Opposed roll
	success, margin, attackRoll, defenseRoll := dice.OpposedRollStat(result.AttackScore, result.DefenseScore)

	result.Success = success
	result.Margin = margin
	result.AttackRoll = attackRoll.Value
	result.DefenseRoll = defenseRoll.Value
	result.AttackZScore = attackRoll.ZScore   // Stage 8.4: For crit detection
	result.DefenseZScore = defenseRoll.ZScore // Stage 8.4: For reference

	// Determine new position on success
	if success {
		if defender.CombatPosition == characters.PositionProne {
			// Prone → Grounded (direct, skip Clinched)
			result.NewPosition = characters.PositionGrounded
		} else {
			// Standing → Clinched
			result.NewPosition = characters.PositionClinched
		}
	}

	return result
}

// ApplyGrappleResult applies the grapple result to both characters.
// Sets positions and tracks the grapple controller.
//
// Chunk 4b W1 cutover: fires position.TransitionPair onto the new
// FSM in parallel with the legacy CombatPosition + GrappleControllerId
// writes. If the pair transition fails (e.g. invalid source state for
// the chosen target) the legacy fields are NOT updated either, so the
// two views stay consistent across the migration window. Both writes
// disappear in S1 along with CombatPosition itself.
func ApplyGrappleResult(attacker *characters.Character, defender *characters.Character, result GrappleResult, attackerId int) {
	if !result.Success {
		return
	}

	// Pick the FSM target. Either side already on the ground lands the
	// pair directly into SideControl (skips Clinch); both standing →
	// Clinch.
	target := position.Clinch
	if attacker.CombatPosition == characters.PositionProne ||
		defender.CombatPosition == characters.PositionProne {
		target = position.SideControl
	}

	if err := position.TransitionPair(
		attacker, defender, target,
		state.TransitionReason{Trigger: position.TriggerGrappleEntry},
	); err != nil {
		mudlog.Warn("ApplyGrappleResult: TransitionPair failed",
			"attacker", attackerId, "target", target, "err", err)
		return
	}

	// Legacy parallel-write (deleted in S1). Derive from the FSM
	// target so the two views agree by construction even if
	// result.NewPosition is computed differently in the future.
	legacyPos := legacyMapPositionToCombatPosition(target)
	attacker.CombatPosition = legacyPos
	defender.CombatPosition = legacyPos
	attacker.GrappleControllerId = attackerId
	defender.GrappleControllerId = attackerId

	// Stage 8.3: Mark who is the controller
	attacker.AddCondition(characters.ConditionGrappleController, 0, 1.0, "grapple")
	defender.RemoveCondition(characters.ConditionGrappleController)
}

// legacyMapPositionToCombatPosition translates the new FSM State to
// the legacy CombatPosition enum during the migration window.
// Deleted in S1 alongside CombatPosition itself. Provided as a
// reference for future W-series tasks that need to translate an FSM
// state they don't already have a CombatPosition for.
func legacyMapPositionToCombatPosition(s position.State) characters.CombatPosition {
	switch s {
	case position.Standing:
		return characters.PositionStanding
	case position.Prone, position.Supine:
		return characters.PositionProne
	case position.Clinch, position.BackStanding:
		return characters.PositionClinched
	default: // Mount, SideControl, KOB, NS, Crucifix, BackGround, HalfGuard, Guard, Turtle
		return characters.PositionGrounded
	}
}

// IsThirdPartyAttack returns true if the attacker is not involved in the target's grapple.
// This identifies opportunistic attackers targeting grappling fighters.
// Stage 8.5: Third-party grapple vulnerability
func IsThirdPartyAttack(attacker *characters.Character, target *characters.Character) bool {
	// Target must be in a grapple position
	if !target.CombatPosition.IsGrapplePosition() {
		return false
	}

	// Target must have an active grapple (controller ID set)
	if target.GrappleControllerId == 0 {
		return false
	}

	// Attacker is third-party if they're not part of this grapple
	// (different controller ID or no grapple at all)
	return attacker.GrappleControllerId != target.GrappleControllerId
}

// AttemptSubmission performs a submission attempt from controller to controlled.
// Stage 8.6: High-risk finishing move requiring Grounded position.
//
// Submission calculation:
// attackScore = controller.Str + controller.CombatSkill
// defenseScore = controlled.Str + controlled.CombatSkill + (controlled.Dex / 2)
//
// Success threshold: z-score > 1.0 (harder than normal grapple)
//
// Note: Opponent choice logic (yield vs resist) is handled in the command handler
// where we have access to player/mob distinction.
func AttemptSubmission(controller *characters.Character, controlled *characters.Character) SubmissionResult {
	result := SubmissionResult{}

	// Base scores: Str + Combat Skill
	controllerScore := float64(controller.Stats.Strength.ValueAdj) + float64(controller.GetCombatSkillLevel())
	controlledScore := float64(controlled.Stats.Strength.ValueAdj) + float64(controlled.GetCombatSkillLevel()) +
		(float64(controlled.Stats.Dexterity.ValueAdj) * 0.5) // Half dex bonus for escape attempts

	// Opposed roll
	success, margin, attackRoll, _ := dice.OpposedRollStat(controllerScore, controlledScore)

	result.Margin = margin
	result.AttackZScore = attackRoll.ZScore

	// Success requires z-score > 1.0 (harder than normal grapple)
	result.Success = success && attackRoll.ZScore > 1.0

	return result
}

// ApplySubmissionFailure applies the consequences of a failed submission attempt.
// Stage 8.6: Controller falls prone, controlled escapes to standing.
func ApplySubmissionFailure(controller *characters.Character, controlled *characters.Character) {
	// Controlled escapes to standing
	controlled.CombatPosition = characters.PositionStanding

	// Controller falls prone (overcommitted)
	controller.CombatPosition = characters.PositionProne
	controller.PositionRoundsMin = 2 // Must spend 2 rounds recovering

	// Clear grapple state for both
	controller.GrappleControllerId = 0
	controlled.GrappleControllerId = 0
	controller.RemoveCondition(characters.ConditionGrappleController)
	controlled.RemoveCondition(characters.ConditionGrappleController)
}

// ApplySubmissionSuccess applies the consequences of a successful submission.
// Stage 8.6: If opponent yields, combat ends. If resists, takes 2x damage and attempts escape.
func ApplySubmissionSuccess(controller *characters.Character, controlled *characters.Character, choice string) {
	if choice == "yield" {
		// Opponent yields - combat ends, they are helpless
		// Note: Actual combat end logic is handled in submit.go command
		// This function just marks the state
		controlled.CombatPosition = characters.PositionProne
		controlled.PositionRoundsMin = 3 // Helpless on ground
	} else {
		// Opponent resists - takes damage based on controller's strength
		baseDamage := float64(controller.Stats.Strength.ValueAdj)
		damage := int(baseDamage * 2.0) // 2x strength as damage
		if damage < 1 {
			damage = 1
		}

		controlled.Health -= damage

		// Trigger automatic escape check (handled in position progression system)
		// The grounded escape will be attempted in the next round tick
	}
}

// CritFailureResult represents the outcome of a critical grapple failure (Stage 8.6)
type CritFailureResult struct {
	Message       string // For attacker
	TargetMessage string // For defender
	RoomMessage   string // For observers
}

// HandleGrappleCritFailure handles the consequences of a critical grapple failure.
// Stage 8.6: Attacker falls prone, defender gets reversal opportunity (+15% grapple bonus).
//
// Triggered when grapple fails with z-score < -2.0 (~2-3% chance)
func HandleGrappleCritFailure(attacker *characters.Character, defender *characters.Character) CritFailureResult {
	result := CritFailureResult{}

	// Attacker falls prone (badly overcommitted)
	attacker.CombatPosition = characters.PositionProne
	attacker.PositionRoundsMin = 2 // Must spend 2 rounds recovering

	// Defender gets grapple opportunity (reuse existing system from Stage 8.4)
	SetGrappleOpportunity(defender)

	// Generate dramatic messages
	result.Message = `<ansi fg="red-bold">You overextend badly and fall to the ground!</ansi>`
	result.TargetMessage = `<ansi fg="yellow-bold">Your opponent overextends and falls - you see an opening!</ansi>`
	result.RoomMessage = `<ansi fg="combat">The failed grapple sends them sprawling!</ansi>`

	return result
}
