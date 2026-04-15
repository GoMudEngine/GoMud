package combat

import (
	"github.com/GoMudEngine/GoMud/internal/util"
)

// GetDamageDescription converts numeric damage into a descriptive phrase based on
// the damage as a percentage of the target's maximum HP.
//
// This creates immersive combat text instead of showing raw numbers:
//   - "Your slash causes moderate injuries to the guard"
// instead of:
//   - "Your slash causes 12 damage to the guard"
//
// The description is scaled relative to the target's max HP, so the same
// numeric damage feels different against weak vs. strong opponents.
func GetDamageDescription(damageAmount int, targetMaxHP int) string {
	// Fallback for edge cases (no target HP data)
	if targetMaxHP <= 0 {
		return "moderate injuries"
	}

	// Calculate damage as percentage of target's max HP
	pct := float64(damageAmount) / float64(targetMaxHP) * 100

	// Return descriptive text based on percentage thresholds
	// These phrases work grammatically with existing YAML messages:
	//   "causing {damage}" → "causing moderate injuries"
	//   "dealing {damage}" → "dealing serious wounds"
	//   "inflicting {damage}" → "inflicting critical injuries"
	switch {
	case pct < 5:
		return "negligible damage"
	case pct < 15:
		return "light wounds"
	case pct < 30:
		return "moderate injuries"
	case pct < 50:
		return "serious wounds"
	case pct < 75:
		return "critical injuries"
	default:
		return "devastating wounds"
	}
}

// GetHealDescription converts a numeric healing amount into a descriptive phrase,
// scaled relative to the target's maximum HP. Mirrors GetDamageDescription so that
// neither function leaks raw numbers into player-facing messages.
func GetHealDescription(healAmount int, targetMaxHP int) string {
	if targetMaxHP <= 0 {
		return "light mending"
	}

	pct := float64(healAmount) / float64(targetMaxHP) * 100

	switch {
	case pct < 5:
		return "negligible mending"
	case pct < 15:
		return "light mending"
	case pct < 30:
		return "moderate restoration"
	case pct < 50:
		return "substantial healing"
	case pct < 75:
		return "significant healing"
	default:
		return "extraordinary healing"
	}
}

// GetConvictionCostDescription returns a qualitative tier for conviction cost.
func GetConvictionCostDescription(cost int) string {
	switch {
	case cost <= 10:
		return "trivial"
	case cost <= 30:
		return "moderate"
	case cost <= 60:
		return "significant"
	case cost <= 100:
		return "substantial"
	default:
		return "demanding"
	}
}

// GetWaitRoundsDescription returns a qualitative label for spell wait rounds.
func GetWaitRoundsDescription(rounds int) string {
	switch {
	case rounds <= 0:
		return "instant"
	case rounds <= 1:
		return "quick"
	case rounds <= 3:
		return "moderate"
	case rounds <= 6:
		return "slow"
	default:
		return "lengthy"
	}
}

// GetCastCountDescription returns a qualitative label for cast count (familiarity).
func GetCastCountDescription(count int) string {
	switch {
	case count <= 0:
		return "untested"
	case count <= 10:
		return "novice"
	case count <= 50:
		return "practiced"
	case count <= 200:
		return "experienced"
	default:
		return "mastered"
	}
}

// GetSuccessChanceDescription returns a qualitative label for cast success percentage.
func GetSuccessChanceDescription(pct int) string {
	switch {
	case pct < 40:
		return "unreliable"
	case pct < 60:
		return "risky"
	case pct < 75:
		return "average"
	case pct < 90:
		return "reliable"
	default:
		return "near-certain"
	}
}

// GetDifficultyDescription returns a qualitative label for spell difficulty.
func GetDifficultyDescription(difficulty int) string {
	switch {
	case difficulty <= 0:
		return "trivial"
	case difficulty <= 10:
		return "simple"
	case difficulty <= 20:
		return "moderate"
	case difficulty <= 35:
		return "challenging"
	case difficulty <= 50:
		return "demanding"
	case difficulty <= 65:
		return "formidable"
	default:
		return "masterwork"
	}
}

// bodyParts lists humanoid body locations for hit flavor text.
var bodyParts = []string{
	"head", "face", "neck", "throat",
	"chest", "ribs", "stomach", "back",
	"left shoulder", "right shoulder",
	"left arm", "right arm",
	"left hand", "right hand",
	"left leg", "right leg",
	"left knee", "right knee",
	"left side", "right side",
}

// GetRandomBodyPart returns a random body part string for combat flavor.
func GetRandomBodyPart() string {
	return bodyParts[util.Rand(len(bodyParts))]
}
