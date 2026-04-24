package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupDiscoveryTestBalance installs a balance config with the documented
// default offset knobs so the test cases below match the spec tables.
func setupDiscoveryTestBalance(t *testing.T) {
	t.Helper()
	b := &Balance{
		DiscoveryPerceptionScale: 200.0,
		DiscoverySkillScale:      100.0,
		DiscoveryMaxDecayOffset:  0.8,
	}
	setBalanceForTest(b)
}

func TestDiscoveryChance(t *testing.T) {
	setupDiscoveryTestBalance(t)

	const tolerance = 0.05 // 0.05 percentage points

	cases := []struct {
		name                     string
		base, decay              float64
		known, perception, skill int
		wantChance               float64
	}{
		// Spec table — spells (base 5%, decay 0.1)
		{"spells: newbie", 5.0, 0.1, 3, 100, 0, 3.85},
		{"spells: early", 5.0, 0.1, 8, 110, 10, 2.97},
		{"spells: mid", 5.0, 0.1, 12, 130, 25, 2.83},
		{"spells: late", 5.0, 0.1, 18, 160, 50, 3.07},
		{"spells: gm (caps at 0.8)", 5.0, 0.1, 20, 200, 100, 3.57},

		// Spec table — recipes (base 10%, decay 0.1)
		{"recipes: newbie", 10.0, 0.1, 3, 100, 0, 7.69},
		{"recipes: gm (caps at 0.8)", 10.0, 0.1, 20, 200, 100, 7.14},

		// Edge cases
		{"known=0 returns base", 5.0, 0.1, 0, 100, 0, 5.00},
		{"per below baseline clamps to 0", 5.0, 0.1, 10, 50, 50, 3.33},
		{"skill above scale clamps to 1", 5.0, 0.1, 10, 100, 200, 4.17},
		{"offset caps at 0.8 (very high per+skill)", 5.0, 0.1, 20, 300, 200, 3.57},
		{"pure per build (skill=0)", 5.0, 0.1, 10, 200, 0, 3.33},
		{"pure skill build (per=100)", 5.0, 0.1, 10, 100, 100, 4.17},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DiscoveryChance(DiscoveryParams{
				Base:       tc.base,
				Decay:      tc.decay,
				Known:      tc.known,
				Perception: tc.perception,
				Skill:      tc.skill,
			})
			assert.InDelta(t, tc.wantChance, got, tolerance,
				"expected %.2f%%, got %.4f%%", tc.wantChance, got)
		})
	}
}

func TestDiscoveryChance_NegativeKnownClampsToZero(t *testing.T) {
	setupDiscoveryTestBalance(t)
	got := DiscoveryChance(DiscoveryParams{
		Base: 5.0, Decay: 0.1, Known: -3, Perception: 100, Skill: 0,
	})
	// Negative known would produce > Base if not handled; we treat it as 0.
	assert.InDelta(t, 5.0, got, 0.01)
}

func TestDiscoveryChance_DefaultsApplied(t *testing.T) {
	// When Balance has not been set, DiscoveryChance should still work
	// using the validator-applied defaults.
	b := &Balance{}
	b.validateDiscovery()
	assert.InDelta(t, 200.0, float64(b.DiscoveryPerceptionScale), 0.001)
	assert.InDelta(t, 100.0, float64(b.DiscoverySkillScale), 0.001)
	assert.InDelta(t, 0.8, float64(b.DiscoveryMaxDecayOffset), 0.001)
}

// confirm that setBalanceForTest unsafely overwrites the singleton so the
// helper isn't reading an unconfigured struct.
func TestDiscoveryChance_ReadsConfiguredBalance(t *testing.T) {
	b := &Balance{
		DiscoveryPerceptionScale: 200.0,
		DiscoverySkillScale:      100.0,
		DiscoveryMaxDecayOffset:  0.0, // cap at 0 → no offset ever applied
	}
	setBalanceForTest(b)

	// With MaxOffset=0, Per+Skill irrelevant; pure baseline formula.
	got := DiscoveryChance(DiscoveryParams{
		Base: 5.0, Decay: 0.1, Known: 10, Perception: 200, Skill: 100,
	})
	want := 5.0 / (1.0 + 10.0*0.1) // 2.5
	assert.InDelta(t, want, got, 0.01)
}
