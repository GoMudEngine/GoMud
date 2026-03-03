package characters

import "testing"

func TestNextPowerOfTwo(t *testing.T) {
	tests := []struct {
		input    int
		expected int
	}{
		{1, 2},
		{2, 2},
		{3, 4},
		{4, 4},
		{5, 8},
		{8, 8},
		{9, 16},
	}

	for _, tt := range tests {
		got := NextPowerOfTwo(tt.input)
		if got != tt.expected {
			t.Errorf("NextPowerOfTwo(%d) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestCalcFoldsPerRound(t *testing.T) {
	tests := []struct {
		perception      int
		spellcastingLvl int
		expected        int
	}{
		// max(1, round((perception + weightedSkill*25) / 100)), SkillWeight=2.0
		{0, 0, 1},   // round(0/100)=0 → clamped to 1
		{50, 2, 2},  // round((50+4*25)/100) = round(1.5) = 2
		{100, 4, 3}, // round((100+8*25)/100) = round(3.0) = 3
		{75, 3, 2},  // round((75+6*25)/100) = round(2.25) = 2
		{25, 1, 1},  // round((25+2*25)/100) = round(0.75) = 1
	}

	for _, tt := range tests {
		got := CalcFoldsPerRound(tt.perception, tt.spellcastingLvl)
		if got != tt.expected {
			t.Errorf("CalcFoldsPerRound(%d, %d) = %d, want %d",
				tt.perception, tt.spellcastingLvl, got, tt.expected)
		}
	}
}

func TestCalcInitiationChance(t *testing.T) {
	tests := []struct {
		willpower       int
		spellcastingLvl int
		expected        int
	}{
		// clamp(60 + willpower/4 + weightedSkill*5, 10, 95), SkillWeight=2.0
		{0, 0, 60},    // 60 + 0 + 0 = 60
		{40, 1, 80},   // 60 + 10 + 2*5 = 80
		{100, 7, 95},  // 60 + 25 + 14*5 = 155 → clamped to 95
		{0, 7, 95},    // 60 + 0 + 14*5 = 130 → clamped to 95
		{-200, 0, 10}, // negative willpower → clamped to 10
	}

	for _, tt := range tests {
		got := CalcInitiationChance(tt.willpower, tt.spellcastingLvl)
		if got != tt.expected {
			t.Errorf("CalcInitiationChance(%d, %d) = %d, want %d",
				tt.willpower, tt.spellcastingLvl, got, tt.expected)
		}
	}
}

func TestCalcConcentrationChance(t *testing.T) {
	tests := []struct {
		willpower int
		damagePct int
		expected  int
	}{
		// clamp(50 + willpower/4 - damagePct, 5, 95)
		{100, 5, 70},  // 50+25-5=70  (average caster, minor hit)
		{100, 20, 55}, // 50+25-20=55 (average caster, moderate hit)
		{100, 50, 25}, // 50+25-50=25 (average caster, heavy hit)
		{0, 50, 5},    // 50+0-50=0  → clamped to 5
		{180, 5, 90},  // 50+45-5=90 (high willpower caster)
		{200, 1, 95},  // 50+50-1=99 → clamped to 95
	}
	for _, tt := range tests {
		got := CalcConcentrationChance(tt.willpower, tt.damagePct)
		if got != tt.expected {
			t.Errorf("CalcConcentrationChance(%d, %d) = %d, want %d",
				tt.willpower, tt.damagePct, got, tt.expected)
		}
	}
}

func TestCalcSpellAttack(t *testing.T) {
	tests := []struct {
		will, skill int
		expected    float64
	}{
		// willpower + weightedSkill*3, SkillWeight=2.0
		{100, 0, 100.0},
		{100, 5, 130.0},  // 100 + 10*3 = 130
		{150, 10, 210.0}, // 150 + 20*3 = 210
		{0, 0, 0.0},
	}
	for _, tt := range tests {
		if got := CalcSpellAttack(tt.will, tt.skill); got != tt.expected {
			t.Errorf("CalcSpellAttack(%d,%d)=%v want %v", tt.will, tt.skill, got, tt.expected)
		}
	}
}
