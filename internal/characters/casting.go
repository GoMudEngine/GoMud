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
