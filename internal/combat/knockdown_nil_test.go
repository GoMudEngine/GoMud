package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These pin the nil-Position guards on the knockdown paths, the siblings of
// HandleGrappleCritFailure surfaced by the 2026-07-21 guard sweep. A
// pre-Validate() Character has no Position FSM; prod combatants are always
// Validated, so the guards only shield under-initialized test fixtures — but
// without them the (dice-gated) knockdown branches nil-panic the same way the
// grapple crit path used to.

func TestHandleDoubleFumble_NilPosition(t *testing.T) {
	src := &characters.Character{Name: "Src"} // Position intentionally nil
	tgt := &characters.Character{Name: "Tgt"} // Position intentionally nil
	require.NotPanics(t, func() {
		handleDoubleFumble(&AttackResult{}, src, tgt)
	})
}

func TestHandleDoubleFumble_BothGoProne(t *testing.T) {
	src := &characters.Character{Name: "Src", Position: position.NewMachine()}
	tgt := &characters.Character{Name: "Tgt", Position: position.NewMachine()}

	handleDoubleFumble(&AttackResult{}, src, tgt)

	assert.True(t, src.IsProne(), "source should be knocked prone by the double fumble")
	assert.True(t, tgt.IsProne(), "target should be knocked prone by the double fumble")
}

func TestExecuteSkillMove_NilDefenderPosition(t *testing.T) {
	// The knockdown branch derefs Defender.Position and is gated behind two
	// dice rolls. Loop enough to hit it essentially every run; with the guard
	// in place the call never panics regardless of whether knockdown triggers.
	require.NotPanics(t, func() {
		for i := 0; i < 200; i++ {
			def := &characters.Character{Name: "Def"} // Position intentionally nil
			def.HealthMax.Value = 100
			def.Health = 100
			ExecuteSkillMove(SkillMoveParams{
				Attacker:        &characters.Character{Name: "Atk", Position: position.NewMachine()},
				Defender:        def,
				AttackStat:      500, // overwhelming attacker → near-certain hit
				AttackSkill:     50,
				DefenseStat:     1,
				DefenseSkill:    0,
				DamagePercent:   0.8,
				KnockdownChance: 100, // guarantee the knockdown branch on a hit
				DamageStat:      100,
			})
		}
	})
}
