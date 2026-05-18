package combat

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

// SubmissionTier classifies the outcome of a per-round submission
// attempt roll into one of four bands. See spec
// "Per-round submission tick mechanics".
type SubmissionTier int

const (
	SubTierBad     SubmissionTier = iota // Attempter overcommits; falls Prone, pair breaks to Standing
	SubTierNeutral                       // Failed but no consequence; pair stays
	SubTierSuccess                       // Sub locks; outcome resolves via attempter's SubmissionPolicy
	SubTierCrit                          // Sub locks AND recipient is Stunned next round (T10 buff)
)

func (t SubmissionTier) String() string {
	switch t {
	case SubTierBad:
		return "bad"
	case SubTierNeutral:
		return "neutral"
	case SubTierSuccess:
		return "success"
	case SubTierCrit:
		return "crit"
	default:
		return "unknown"
	}
}

// SubmissionAttemptResult is the output of RollSubmissionAttempt.
// All fields are useful downstream: SubType drives narration +
// body-part mapping, Tier drives the outcome branch, AttackerScore /
// DefenderScore are exposed for analytics + debugging, and
// AttackerZScore is the basis for the tier classification.
type SubmissionAttemptResult struct {
	SubType        position.SubmissionType
	Tier           SubmissionTier
	AttackerScore  float64
	DefenderScore  float64
	AttackerZScore float64
	DefenderZScore float64
	Margin         float64 // attacker margin (positive = attacker won)
}

// RollSubmissionAttempt rolls a fresh opposed Strength + Unarmed-
// combat-skill check between attempter and recipient. This is a
// SEPARATE roll from the chunk-4b drift roll — drift gates the
// opportunity, this roll resolves the attempt.
//
// Formula:
//
//	attackerScore = attempter.Strength
//	              + attempter.UnarmedCombatSkill * SubSkillWeight
//	defenderScore = recipient.Strength
//	              + recipient.Vitality
//	              + recipient.UnarmedCombatSkill * SubSkillWeight
func RollSubmissionAttempt(
	attempter *characters.Character,
	recipient *characters.Character,
	subType position.SubmissionType,
) SubmissionAttemptResult {
	cfg := configs.GetBalanceConfig()
	skillWeight := float64(cfg.SubSkillWeight)

	atkScore := float64(attempter.Stats.Strength.ValueAdj) +
		float64(attempter.GetSkillLevel(skills.UnarmedCombat))*skillWeight
	defScore := float64(recipient.Stats.Strength.ValueAdj) +
		float64(recipient.Stats.Vitality.ValueAdj) +
		float64(recipient.GetSkillLevel(skills.UnarmedCombat))*skillWeight

	success, margin, atkRoll, defRoll := dice.OpposedRollStat(atkScore, defScore)

	return SubmissionAttemptResult{
		SubType:        subType,
		Tier:           ClassifySubmissionTier(success, atkRoll.ZScore),
		AttackerScore:  atkScore,
		DefenderScore:  defScore,
		AttackerZScore: atkRoll.ZScore,
		DefenderZScore: defRoll.ZScore,
		Margin:         margin,
	}
}

// ClassifySubmissionTier maps (success, attacker z-score) to a tier
// per the spec table. Exposed for unit testing of the boundary
// conditions independently from the dice roll.
func ClassifySubmissionTier(success bool, attackerZ float64) SubmissionTier {
	cfg := configs.GetBalanceConfig()
	bad := float64(cfg.SubBadZThreshold)
	crit := float64(cfg.SubCritZThreshold)

	if success {
		if attackerZ >= crit {
			return SubTierCrit
		}
		return SubTierSuccess
	}
	if attackerZ < bad {
		return SubTierBad
	}
	return SubTierNeutral
}
