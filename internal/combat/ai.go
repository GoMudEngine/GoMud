package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// AI profile definitions with move preference weights
var aiProfiles = map[string]map[string]int{
	"caster": {
		"kick": 10,
		"trip": 10,
		// prefers casting; ChooseCastAction() handles spell selection separately
	},
	"default": {
		"bash":    25,
		"trip":    20,
		"kick":    15,
		"grapple": 40,
	},
	"aggressive": {
		"bash":    40,
		"kick":    30,
		"trip":    20,
		"grapple": 10,
	},
	"defensive": {
		"trip":    35,
		"grapple": 35,
		"bash":    20,
		"kick":    10,
	},
	"grappler": {
		"grapple": 60,
		"submit":  30,
		"trip":    10,
	},
	"brawler": {
		"kick":    40,
		"trip":    30,
		"grapple": 20,
		"bash":    10,
	},
	"tactical": {
		// Dynamic weights based on context (uses default for now)
		"bash":    25,
		"trip":    20,
		"kick":    15,
		"grapple": 40,
	},
}

// ChooseSpecialMove is the main AI entry point
// Returns the name of a special move to use ("bash", "trip", etc.) or "" for none
func ChooseSpecialMove(mob *mobs.Mob, target *characters.Character) string {

	// Check if mob should attempt special move (based on SpecialMoveChance)
	if util.Rand(100) >= mob.SpecialMoveChance {
		return ""
	}

	// Build list of viable moves with their scores
	moveScores := make(map[string]int)

	if CanUseBash(&mob.Character) {
		moveScores["bash"] = ScoreBash(mob, target)
	}

	if CanUseTrip(&mob.Character) {
		moveScores["trip"] = ScoreTrip(mob, target)
	}

	if CanUseKick(&mob.Character) {
		moveScores["kick"] = ScoreKick(mob, target)
	}

	if CanUseGrapple(&mob.Character) {
		moveScores["grapple"] = ScoreGrapple(mob, target)
	}

	if CanUseSubmit(&mob.Character) {
		moveScores["submit"] = ScoreSubmit(mob, target)
	}

	// No viable moves
	if len(moveScores) == 0 {
		return ""
	}

	// Get AI profile preferences
	profile := GetAIProfile(mob.AIProfile, mob.MovePreferences)

	// Weight scores by mob preferences
	weightedScores := make(map[string]int)
	for move, score := range moveScores {
		if score > 0 {
			weight := profile[move]
			if weight == 0 {
				weight = 10 // Default weight if not in profile
			}
			weightedScores[move] = score * weight / 100
		}
	}

	// No viable moves after weighting
	if len(weightedScores) == 0 {
		return ""
	}

	// Find highest-scoring move
	bestMove := ""
	bestScore := 0
	for move, score := range weightedScores {
		if score > bestScore {
			bestScore = score
			bestMove = move
		}
	}

	return bestMove
}

// GetAIProfile returns move preference weights for a profile
// Custom preferences override profile defaults
func GetAIProfile(profileName string, customPreferences map[string]int) map[string]int {
	// Start with profile defaults
	profile, exists := aiProfiles[profileName]
	if !exists {
		profile = aiProfiles["default"]
	}

	// Make a copy to avoid modifying the original
	result := make(map[string]int)
	for k, v := range profile {
		result[k] = v
	}

	// Override with custom preferences
	for k, v := range customPreferences {
		result[k] = v
	}

	return result
}

// --- Viability checks ---

func CanUseBash(char *characters.Character) bool {
	// Must have a shield
	if !char.HasShield() {
		return false
	}
	// Must not be on cooldown
	if _, exists := char.Cooldowns["special-move"]; exists {
		return false
	}
	return true
}

func CanUseTrip(char *characters.Character) bool {
	// Must not be on cooldown
	if _, exists := char.Cooldowns["special-move"]; exists {
		return false
	}
	// Can't trip if in grapple
	if char.CombatPosition.IsGrapplePosition() {
		return false
	}
	return true
}

func CanUseKick(char *characters.Character) bool {
	// Must not be on cooldown
	if _, exists := char.Cooldowns["special-move"]; exists {
		return false
	}
	return true
}

func CanUseGrapple(char *characters.Character) bool {
	// Must not be on cooldown
	if _, exists := char.Cooldowns["special-move"]; exists {
		return false
	}
	// Can't grapple if already in grapple
	if char.CombatPosition.IsGrapplePosition() {
		return false
	}
	// Grapple-immune species can't initiate grapple either
	if sp := species.GetSpecies(char.SpeciesId); sp != nil && sp.GrappleImmune {
		return false
	}
	return true
}

func CanUseSubmit(char *characters.Character) bool {
	// Must be in grounded position
	if char.CombatPosition != characters.PositionGrounded {
		return false
	}
	// Must be grapple controller
	if !char.HasCondition(characters.ConditionGrappleController) {
		return false
	}
	// Must not be on cooldown
	if _, exists := char.Cooldowns["special-move"]; exists {
		return false
	}
	return true
}

// --- Scoring functions ---

func ScoreBash(mob *mobs.Mob, target *characters.Character) int {
	score := 50 // Base score

	// Must have shield (already checked in viability)
	if !mob.Character.HasShield() {
		return 0
	}

	// Bonus for high target health (bash is good for damage early)
	targetHealthPercent := float64(target.Health) * 100.0 / float64(target.HealthMax.Value)
	if targetHealthPercent > 60 {
		score += 20
	}

	// Bonus for high combat skill
	if mob.Character.GetCombatSkillLevel() > 50 {
		score += 15
	}

	// Penalty if target is already on the ground (prone or grounded)
	if target.CombatPosition.IsGroundPosition() {
		score -= 50
	}

	if score < 0 {
		score = 0
	}
	return score
}

func ScoreTrip(mob *mobs.Mob, target *characters.Character) int {
	score := 40 // Base score

	// Bonus for high dexterity
	if mob.Character.Stats.Dexterity.ValueAdj > 14 {
		score += 25
	}

	// Bonus if target has low dexterity
	if target.Stats.Dexterity.ValueAdj < 10 {
		score += 20
	}

	// Bonus for high unarmed combat skill
	if mob.Character.GetSkillLevel(skills.UnarmedCombat) > 40 {
		score += 15
	}

	// Can't trip someone already on the ground
	if target.CombatPosition.IsGroundPosition() {
		return 0
	}

	// Penalty if in grapple
	if mob.Character.CombatPosition.IsGrapplePosition() {
		score -= 50
	}

	if score < 0 {
		score = 0
	}
	return score
}

func ScoreKick(mob *mobs.Mob, target *characters.Character) int {
	score := 45 // Base score

	// Bonus for high strength
	if mob.Character.Stats.Strength.ValueAdj > 14 {
		score += 20
	}

	// Bonus for high unarmed combat skill
	if mob.Character.GetSkillLevel(skills.UnarmedCombat) > 40 {
		score += 15
	}

	// Small bonus if target is low health (finish them)
	targetHealthPercent := float64(target.Health) * 100.0 / float64(target.HealthMax.Value)
	if targetHealthPercent < 30 {
		score += 10
	}

	if score < 0 {
		score = 0
	}
	return score
}

func ScoreGrapple(mob *mobs.Mob, target *characters.Character) int {
	score := 50 // Base score

	// Bonus for high unarmed combat skill
	if mob.Character.GetSkillLevel(skills.UnarmedCombat) > 40 {
		score += 30
	}

	// Bonus if stronger than target
	if mob.Character.Stats.Strength.ValueAdj > target.Stats.Strength.ValueAdj {
		score += 20
	}

	// Bonus if target is low health (finish them with grapple)
	targetHealthPercent := float64(target.Health) * 100.0 / float64(target.HealthMax.Value)
	if targetHealthPercent < 30 {
		score += 15
	}

	// Can't grapple if already in grapple
	if mob.Character.CombatPosition.IsGrapplePosition() {
		return 0
	}

	// Penalty if mob is low health (risky)
	mobHealthPercent := float64(mob.Character.Health) * 100.0 / float64(mob.Character.HealthMax.Value)
	if mobHealthPercent < 20 {
		score -= 50
	}

	if score < 0 {
		score = 0
	}
	return score
}

// CanUseCast returns true if the mob has spells, is not already casting, and has conviction.
func CanUseCast(char *characters.Character) bool {
	if char.CastingState != nil {
		return false
	}
	if len(char.SpellBook) == 0 {
		return false
	}
	return char.Conviction >= 3
}

// preferredSpell returns the spell ID the mob should cast this round.
// Priority: (1) minor-shield if unshielded, (2) heal-self if < 30% HP, (3) harm spells.
func preferredSpell(mob *mobs.Mob) string {
	// Shield self if not already shielded
	if !mob.Character.HasCondition(characters.ConditionShield) {
		if _, has := mob.Character.SpellBook["conviction-ward"]; has {
			if sd := spells.GetSpell("conviction-ward"); sd != nil && mob.Character.Conviction >= sd.Cost {
				return "conviction-ward"
			}
		}
	}
	// Heal when critically low
	selfPct := float64(mob.Character.Health) * 100 / float64(mob.Character.HealthMax.Value)
	if selfPct < 30 {
		if _, has := mob.Character.SpellBook["heal"]; has {
			if sd := spells.GetSpell("heal"); sd != nil && mob.Character.Conviction >= sd.Cost {
				return "heal"
			}
		}
	}
	// Harm spells by magnitude
	for _, id := range []string{"hemorrhagic-burst", "pyretic-surge", "conviction-spike", "mind-spike", "nerve-disruption", "sparks", "neural-stun", "sensory-veil"} {
		if _, has := mob.Character.SpellBook[id]; has {
			if sd := spells.GetSpell(id); sd != nil && mob.Character.Conviction >= sd.Cost {
				return id
			}
		}
	}
	return ""
}

// ChooseCastAction returns "cast <spellId>" for caster-profile mobs, or "" otherwise.
func ChooseCastAction(mob *mobs.Mob) string {
	if mob.AIProfile != "caster" {
		return ""
	}
	if !CanUseCast(&mob.Character) {
		return ""
	}
	if id := preferredSpell(mob); id != "" {
		return "cast " + id
	}
	return ""
}

func ScoreSubmit(mob *mobs.Mob, target *characters.Character) int {
	// Only viable if in grounded position and controlling
	if !mob.Character.HasCondition(characters.ConditionGrappleController) {
		return 0
	}
	if mob.Character.CombatPosition != characters.PositionGrounded {
		return 0
	}

	score := 40 // Base score

	// In dominant position
	score += 40

	// Bonus for high unarmed combat skill
	if mob.Character.GetSkillLevel(skills.UnarmedCombat) > 60 {
		score += 20
	}

	// Bonus if target is low health
	targetHealthPercent := float64(target.Health) * 100.0 / float64(target.HealthMax.Value)
	if targetHealthPercent < 40 {
		score += 15
	}

	return score
}
