package combat

import (
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
