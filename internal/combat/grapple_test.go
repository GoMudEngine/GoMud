package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// ─── IsThirdPartyAttack ─────────────────────────────────────────────────────

func TestIsThirdPartyAttack(t *testing.T) {
	tests := []struct {
		name              string
		atkPos            characters.CombatPosition
		tgtPos            characters.CombatPosition
		atkGrappleCtrlId  int
		tgtGrappleCtrlId  int
		want              bool
	}{
		{
			name:   "neither in grapple → not third party",
			atkPos: characters.PositionStanding,
			tgtPos: characters.PositionStanding,
			want:   false,
		},
		{
			name:             "target in grapple but no controller ID → not third party",
			atkPos:           characters.PositionStanding,
			tgtPos:           characters.PositionClinched,
			tgtGrappleCtrlId: 0, // no controller set
			want:             false,
		},
		{
			name:             "both in same grapple → not third party",
			atkPos:           characters.PositionGrounded,
			tgtPos:           characters.PositionGrounded,
			atkGrappleCtrlId: 42,
			tgtGrappleCtrlId: 42,
			want:             false,
		},
		{
			name:             "attacker standing, target in grapple with different controller → third party",
			atkPos:           characters.PositionStanding,
			tgtPos:           characters.PositionClinched,
			atkGrappleCtrlId: 0,
			tgtGrappleCtrlId: 42,
			want:             true,
		},
		{
			name:             "attacker in different grapple than target → third party",
			atkPos:           characters.PositionGrounded,
			tgtPos:           characters.PositionGrounded,
			atkGrappleCtrlId: 10,
			tgtGrappleCtrlId: 42,
			want:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			atk := characters.New()
			atk.CombatPosition = tt.atkPos
			atk.GrappleControllerId = tt.atkGrappleCtrlId

			tgt := characters.New()
			tgt.CombatPosition = tt.tgtPos
			tgt.GrappleControllerId = tt.tgtGrappleCtrlId

			got := IsThirdPartyAttack(atk, tgt)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ─── AttemptGrapple ─────────────────────────────────────────────────────────

func TestAttemptGrapple_Statistical(t *testing.T) {
	iterations := 10000

	t.Run("higher dex attacker wins more often", func(t *testing.T) {
		wins := 0
		for i := 0; i < iterations; i++ {
			atk := characters.New()
			atk.Stats.Dexterity.ValueAdj = 150
			atk.Skills["unarmed-combat"] = 50

			def := characters.New()
			def.Stats.Dexterity.ValueAdj = 80
			def.Skills["unarmed-combat"] = 10

			result := AttemptGrapple(atk, def)
			if result.Success {
				wins++
			}
		}
		winRate := float64(wins) / float64(iterations)
		assert.Greater(t, winRate, 0.55, "stronger attacker should win > 55%% of grapples")
	})

	t.Run("equal characters → ~50/50", func(t *testing.T) {
		wins := 0
		for i := 0; i < iterations; i++ {
			atk := characters.New()
			atk.Stats.Dexterity.ValueAdj = 100
			atk.Skills["unarmed-combat"] = 25

			def := characters.New()
			def.Stats.Dexterity.ValueAdj = 100
			def.Skills["unarmed-combat"] = 25

			result := AttemptGrapple(atk, def)
			if result.Success {
				wins++
			}
		}
		winRate := float64(wins) / float64(iterations)
		assert.InDelta(t, 0.50, winRate, 0.10, "equal characters should be near 50/50")
	})

	t.Run("prone defender gets penalized", func(t *testing.T) {
		wins := 0
		for i := 0; i < iterations; i++ {
			atk := characters.New()
			atk.Stats.Dexterity.ValueAdj = 100
			atk.Skills["unarmed-combat"] = 25

			def := characters.New()
			def.Stats.Dexterity.ValueAdj = 100
			def.Skills["unarmed-combat"] = 25
			def.CombatPosition = characters.PositionProne

			result := AttemptGrapple(atk, def)
			if result.Success {
				wins++
			}
		}
		winRate := float64(wins) / float64(iterations)
		assert.Greater(t, winRate, 0.70, "attacker should win > 70%% when defender is prone")
	})
}

func TestAttemptGrapple_PositionTransition(t *testing.T) {
	// Successful grapple against standing → Clinched
	for i := 0; i < 100; i++ {
		atk := characters.New()
		atk.Stats.Dexterity.ValueAdj = 200
		atk.Skills["unarmed-combat"] = 50

		def := characters.New()
		def.Stats.Dexterity.ValueAdj = 50
		def.Skills["unarmed-combat"] = 5

		result := AttemptGrapple(atk, def)
		if result.Success {
			assert.Equal(t, characters.PositionClinched, result.NewPosition,
				"successful grapple on standing defender → Clinched")
			return
		}
	}
	t.Log("Warning: did not get a successful grapple in 100 tries (statistical fluke)")
}
