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
}

const (
	grappleStdDev = 20.0 // Standard deviation for grapple opposed rolls
)

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

	result.AttackScore = float64(attacker.Stats.Dexterity.ValueAdj) + attackerCombatSkill
	result.DefenseScore = float64(defender.Stats.Dexterity.ValueAdj) + defenderCombatSkill

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
}
