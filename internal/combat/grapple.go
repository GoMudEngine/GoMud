package combat

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
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

const (
	grappleStdDev = 20.0 // Standard deviation for grapple opposed rolls
)

// PositionProgressionResult represents the outcome of automatic position checks
type PositionProgressionResult struct {
	Changed          bool
	NewPosition      characters.CombatPosition
	ControllerWon    bool
	Margin           float64
	Message          string // For both participants
	RoomMessage      string // For room observers
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
	success, margin, attackRoll, defenseRoll := dice.OpposedRoll(result.AttackScore, result.DefenseScore, grappleStdDev)

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
func ApplyGrappleResult(attacker *characters.Character, defender *characters.Character, result GrappleResult, attackerId int) {
	if !result.Success {
		return
	}

	// Set both characters to the new position
	attacker.CombatPosition = result.NewPosition
	defender.CombatPosition = result.NewPosition

	// Track who initiated/controls the grapple
	attacker.GrappleControllerId = attackerId
	defender.GrappleControllerId = attackerId

	// Stage 8.3: Mark who is the controller
	attacker.IsGrappleController = true
	defender.IsGrappleController = false
}

// CheckClinchProgression performs automatic control check for clinched fighters
// Stage 8.3: Str + Combat Skill opposed roll
// Success: Transition to Grounded (controller maintains control)
// Failure: Break apart, both return to Standing
func CheckClinchProgression(controller *characters.Character, controlled *characters.Character) PositionProgressionResult {
	result := PositionProgressionResult{}

	// Opposed roll: Str + Combat Skill
	controllerScore := float64(controller.Stats.Strength.ValueAdj) + float64(controller.GetCombatSkillLevel())
	controlledScore := float64(controlled.Stats.Strength.ValueAdj) + float64(controlled.GetCombatSkillLevel())

	success, margin, _, _ := dice.OpposedRoll(controllerScore, controlledScore, grappleStdDev)

	result.Changed = true
	result.Margin = margin

	if success {
		// Controller advances to grounded
		result.NewPosition = characters.PositionGrounded
		result.ControllerWon = true
		result.Message = "The grapple intensifies as you're taken to the ground!"
		result.RoomMessage = "The grapple intensifies as they go to the ground!"
	} else {
		// Break apart, both return to standing
		result.NewPosition = characters.PositionStanding
		result.ControllerWon = false
		result.Message = "You break free from the grapple and return to standing!"
		result.RoomMessage = "They break apart and return to standing positions!"
	}

	return result
}

// CheckGroundedEscape performs automatic escape attempt for grounded fighters
// Stage 8.3: Controlled fighter attempts to escape
// Success: Both return to Standing
// Failure: Remain Grounded
func CheckGroundedEscape(controller *characters.Character, controlled *characters.Character) PositionProgressionResult {
	result := PositionProgressionResult{}

	// Opposed roll: controlled tries to escape
	// Controlled: Str + Combat Skill + Dex
	// Controller: Str + Combat Skill
	controlledScore := float64(controlled.Stats.Strength.ValueAdj) +
		float64(controlled.GetCombatSkillLevel()) +
		float64(controlled.Stats.Dexterity.ValueAdj) * 0.5 // Half dex bonus for scrambling

	// Stage 8.7: Apply armor escape modifier
	if controlled.Equipment.Body.ItemId != 0 {
		armorSpec := items.GetItemSpec(controlled.Equipment.Body.ItemId)
		if armorSpec != nil && armorSpec.EscapeModifier != 0.0 {
			// Multiplicative modifiers (compatible with future balance config YAML)
			if armorSpec.EscapeModifier > 0.0 {
				// Positive modifier: bonus to escape (e.g., +1.0 = doubles escape score for light armor)
				controlledScore *= (1.0 + armorSpec.EscapeModifier)
			} else {
				// Negative modifier: penalty to escape (e.g., -2.0 = reduces to 1/3 for heavy armor)
				controlledScore /= (1.0 + math.Abs(armorSpec.EscapeModifier))
			}
		}
	}

	controllerScore := float64(controller.Stats.Strength.ValueAdj) + float64(controller.GetCombatSkillLevel())

	success, margin, _, _ := dice.OpposedRoll(controlledScore, controllerScore, grappleStdDev)

	result.Changed = success
	result.Margin = margin

	if success {
		// Escape successful
		result.NewPosition = characters.PositionStanding
		result.ControllerWon = false
		result.Message = "You scramble free and return to standing!"
		result.RoomMessage = "They scramble apart and return to standing positions!"
	} else {
		// Remain grounded
		result.NewPosition = characters.PositionGrounded
		result.ControllerWon = true
		result.Message = "You struggle but remain pinned on the ground!"
		result.RoomMessage = "They struggle on the ground!"
	}

	return result
}

// ApplyPositionProgression applies the result of a position progression check
func ApplyPositionProgression(char1 *characters.Character, char2 *characters.Character, result PositionProgressionResult) {
	if !result.Changed {
		return
	}

	// Set new positions
	char1.CombatPosition = result.NewPosition
	char2.CombatPosition = result.NewPosition

	// If returning to standing, clear grapple tracking
	if result.NewPosition == characters.PositionStanding {
		char1.GrappleControllerId = 0
		char2.GrappleControllerId = 0
		char1.IsGrappleController = false
		char2.IsGrappleController = false
	}
	// If advancing to grounded, controller status stays the same (already set)
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
	success, margin, attackRoll, _ := dice.OpposedRoll(controllerScore, controlledScore, grappleStdDev)

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
	controller.IsGrappleController = false
	controlled.IsGrappleController = false
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
