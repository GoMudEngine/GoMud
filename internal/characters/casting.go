package characters

import "math"

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

// CalcFoldsPerRound returns max(1, round((perception + spellcastingLevel*25) / 100)).
// Example: per=50, skill=2 → round(100/100)=1; per=100, skill=4 → round(200/100)=2
func CalcFoldsPerRound(perception, spellcastingLevel int) int {
	result := int(math.Round(float64(perception+spellcastingLevel*25) / 100.0))
	if result < 1 {
		return 1
	}
	return result
}

// CalcInitiationChance returns clamp(60 + willpower/4 + spellcastingLevel*5, 10, 95).
// Beginners ~65%, skilled casters ~90%+.
func CalcInitiationChance(willpower, spellcastingLevel int) int {
	chance := 60 + willpower/4 + spellcastingLevel*5
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
// Formula: clamp(50 + willpower/4 - damagePct, 5, 95)
// Examples (baseline human willpower ~100):
//
//	willpower=100, damagePct=5  → 50+25-5  = 70%
//	willpower=100, damagePct=20 → 50+25-20 = 55%
//	willpower=100, damagePct=50 → 50+25-50 = 25%
//	willpower=0,   damagePct=50 → 50+0-50  = 0  → clamped to 5%
//	willpower=180, damagePct=5  → 50+45-5  = 90%
func CalcConcentrationChance(willpower, damagePct int) int {
	chance := 50 + willpower/4 - damagePct
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
// Formula: float64(willpower + spellcastingLevel*3)
// Examples:
//
//	will=100, skill=0  → 100.0
//	will=100, skill=5  → 115.0
//	will=150, skill=10 → 180.0
func CalcSpellAttack(willpower, spellcastingLevel int) float64 {
	return float64(willpower + spellcastingLevel*3)
}
