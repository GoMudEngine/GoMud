package characters

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// CastingState tracks active fold-based spellcasting progress.
// Not persisted — the Character field uses yaml:"-".
type CastingState struct {
	SpellId              string
	FoldsNeeded          int
	FoldsAccumulated     int
	FoldsPerRound        int
	TotalConvictionCost  int // conviction owed for full cast
	ConvictionSpent      int // conviction paid so far (non-refundable on cancel)
	TargetUserIds        []int
	TargetMobInstanceIds []int
	SpellRest            string // for Neutral-type spells
}

// NextPowerOfTwo returns the smallest power of 2 >= n (minimum 2).
func NextPowerOfTwo(n int) int {
	if n <= 2 {
		return 2
	}
	p := 2
	for p < n {
		p <<= 1
	}
	return p
}

// CalcFoldsPerRound returns max(1, round((primaryStat + skillLevel*SpellFoldsSkillFactor) / 100)).
// Example: stat=50, skill=2 → round(100/100)=1; stat=100, skill=4 → round(200/100)=2
func CalcFoldsPerRound(primaryStat, skillLevel int) int {
	b := configs.GetBalanceConfig()
	skillFactor := int(b.SpellFoldsSkillFactor)
	weightedSkill := int(math.Round(float64(skillLevel) * float64(b.SkillWeight)))
	result := int(math.Round(float64(primaryStat+weightedSkill*skillFactor) / 100.0))
	if result < 1 {
		return 1
	}
	return result
}

// CalcInitiationChance returns clamp(base + willpower/divisor + level*factor, 10, 95).
// Beginners ~65%, skilled casters ~90%+.
func CalcInitiationChance(willpower, spellcastingLevel int) int {
	b := configs.GetBalanceConfig()
	base := int(b.SpellInitiationBase)
	divisor := int(b.SpellInitiationWillpowerDivisor)
	factor := int(b.SpellInitiationSkillFactor)
	weightedSkill := int(math.Round(float64(spellcastingLevel) * float64(b.SkillWeight)))
	chance := base + willpower/divisor + weightedSkill*factor
	if chance < 10 {
		return 10
	}
	if chance > 95 {
		return 95
	}
	return chance
}

// CalcConcentrationChance returns the % chance to maintain concentration
// when struck for damagePct percent of max health.
// Formula: clamp(base + willpower/divisor - damagePct, 5, 95)
func CalcConcentrationChance(willpower, damagePct int) int {
	b := configs.GetBalanceConfig()
	base := int(b.SpellConcentrationBase)
	divisor := int(b.SpellInitiationWillpowerDivisor)
	chance := base + willpower/divisor - damagePct
	if chance < 5 {
		return 5
	}
	if chance > 95 {
		return 95
	}
	return chance
}

// CalcSpellAttack returns the float64 mean used as the spell attack roll's mean
// in dice.OpposedRoll. Higher willpower and spellcasting level increase spell offense.
// Formula: float64(willpower + spellcastingLevel * SpellAttackSkillFactor)
func CalcSpellAttack(willpower, spellcastingLevel int) float64 {
	b := configs.GetBalanceConfig()
	factor := int(b.SpellAttackSkillFactor)
	weightedSkill := int(math.Round(float64(spellcastingLevel) * float64(b.SkillWeight)))
	return float64(willpower + weightedSkill*factor)
}
