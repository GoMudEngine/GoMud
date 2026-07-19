package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// ─── IsThirdPartyAttack ─────────────────────────────────────────────────────

// Chunk 4b R2: IsThirdPartyAttack now reads the Position FSM
// (target.IsGrappling + GrappleData.Partner) instead of the legacy
// CombatPosition + GrappleControllerId fields. Fixtures use FSM
// transitions so the test covers the new code path.
func TestIsThirdPartyAttack(t *testing.T) {
	newChar := func(uid int) *characters.Character {
		c := characters.New()
		c.SetUserId(uid)
		return c
	}

	t.Run("neither in grapple → not third party", func(t *testing.T) {
		atk := newChar(1)
		tgt := newChar(2)
		assert.False(t, IsThirdPartyAttack(atk, tgt))
	})

	t.Run("both in same Clinch (partner refs match) → not third party", func(t *testing.T) {
		atk := newChar(10)
		tgt := newChar(20)
		err := atk.Position.TransitionToClinch(
			position.GrappleData{Partner: state.ActorRef{UserId: tgt.GetUserId()}},
			state.TransitionReason{Trigger: position.TriggerGrappleEntry},
		)
		assert.NoError(t, err)
		err = tgt.Position.TransitionToClinch(
			position.GrappleData{Partner: state.ActorRef{UserId: atk.GetUserId()}},
			state.TransitionReason{Trigger: position.TriggerGrappleEntry},
		)
		assert.NoError(t, err)
		assert.False(t, IsThirdPartyAttack(atk, tgt))
	})

	t.Run("attacker standing, target in grapple with someone else → third party", func(t *testing.T) {
		atk := newChar(1)
		other := newChar(42)
		tgt := newChar(2)
		err := tgt.Position.TransitionToClinch(
			position.GrappleData{Partner: state.ActorRef{UserId: other.GetUserId()}},
			state.TransitionReason{Trigger: position.TriggerGrappleEntry},
		)
		assert.NoError(t, err)
		assert.True(t, IsThirdPartyAttack(atk, tgt))
	})

	t.Run("attacker in a different grapple than target → third party", func(t *testing.T) {
		atk := newChar(10)
		atkPartner := newChar(11)
		tgt := newChar(20)
		tgtPartner := newChar(21)
		err := atk.Position.TransitionToClinch(
			position.GrappleData{Partner: state.ActorRef{UserId: atkPartner.GetUserId()}},
			state.TransitionReason{Trigger: position.TriggerGrappleEntry},
		)
		assert.NoError(t, err)
		err = tgt.Position.TransitionToClinch(
			position.GrappleData{Partner: state.ActorRef{UserId: tgtPartner.GetUserId()}},
			state.TransitionReason{Trigger: position.TriggerGrappleEntry},
		)
		assert.NoError(t, err)
		assert.True(t, IsThirdPartyAttack(atk, tgt))
	})
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
			setCombatPositionParallel(def, position.Prone)

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
			assert.False(t, result.IsGroundGrapple,
				"successful grapple on standing defender → clinch (not ground grapple)")
			return
		}
	}
	t.Log("Warning: did not get a successful grapple in 100 tries (statistical fluke)")
}

// Regression: HandleGrappleCritFailure must not panic when the attacker's
// Position machine is nil. ResetForMobInstance() clears Position on a mob
// instance until it is re-wired, so a not-yet-wired mob that rolls the ~2.3%
// grapple crit-failure previously dereferenced a nil *position.Machine and
// crashed (a flaky panic in TestAttackInCombat/grapple_in_combat on CI).
func TestHandleGrappleCritFailure_NilAttackerPosition(t *testing.T) {
	attacker := &characters.Character{} // Position is nil (zero-value pointer)
	defender := &characters.Character{}
	assert.NotPanics(t, func() {
		res := HandleGrappleCritFailure(attacker, defender)
		assert.NotEmpty(t, res.Message, "crit-failure messaging still produced without a Position machine")
	})
}
