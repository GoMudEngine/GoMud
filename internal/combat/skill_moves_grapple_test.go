package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// makeGrappleTestParams builds SkillMoveParams that reliably HIT (huge attacker
// advantage) and roll a knockdown (KnockdownChance 100), so the only variable
// under test is whether the FSM knockdown transition succeeds.
func makeGrappleTestParams(defender *characters.Character) SkillMoveParams {
	atk := characters.New()
	return SkillMoveParams{
		Attacker: atk, Defender: defender,
		AttackStat: 1000, AttackSkill: 50, DefenseStat: 1, DefenseSkill: 0,
		DamagePercent: 1.0, KnockdownChance: 100, SkillRank: 0,
		DamageStat: 100, MitigationMultiplier: 1.0,
	}
}

// A standing target: the knockdown transition succeeds → KnockedDown true.
func TestExecuteSkillMove_KnockdownStanding(t *testing.T) {
	d := characters.New()
	d.HealthMax.Value = 1000
	d.Health = 1000
	setCombatPositionParallel(d, position.Standing)

	res := ExecuteSkillMove(makeGrappleTestParams(d))
	assert.True(t, res.Hit, "huge attacker advantage should hit")
	assert.True(t, res.KnockedDown, "standing target should be knocked down")
}

// Grapple move-collision fix: a target already in a controlled grapple position
// can't be knocked down (TransitionToProne fails), so KnockedDown must be false
// — no misleading "you knock them down!" narration. Damage still applies.
func TestExecuteSkillMove_KnockdownGrappledIsSuppressed(t *testing.T) {
	d := characters.New()
	d.HealthMax.Value = 1000
	d.Health = 1000
	setCombatPositionParallel(d, position.Clinch) // controlled grapple, not Standing

	res := ExecuteSkillMove(makeGrappleTestParams(d))
	assert.True(t, res.Hit, "the move still connects")
	assert.False(t, res.KnockedDown, "knockdown must be suppressed when the FSM transition fails")
	assert.Greater(t, res.Damage, 0, "damage still applies even without knockdown")
}
