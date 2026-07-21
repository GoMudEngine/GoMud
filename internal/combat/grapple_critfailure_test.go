package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleGrappleCritFailure_Outcome pins the crit-failure success path
// deterministically.
//
// The behaviour was previously exercised only via the ~2.3% random fumble in
// TestAttackInCombat/grapple_in_combat, which usually skipped this path and
// asserted nothing — so a regression here was invisible until it panicked in CI
// on an unlucky roll (the "flaky CI panic" that motivated the nil-guard). This
// forces the path every run and asserts the full outcome.
//
// The nil-Position guard itself is covered by
// TestHandleGrappleCritFailure_NilAttackerPosition in grapple_test.go.
func TestHandleGrappleCritFailure_Outcome(t *testing.T) {
	attacker := &characters.Character{Name: "Attacker", Position: position.NewMachine()}
	defender := &characters.Character{Name: "Defender", Position: position.NewMachine()}
	require.True(t, attacker.IsStanding(), "precondition: attacker starts standing")

	res := HandleGrappleCritFailure(attacker, defender)

	assert.True(t, attacker.IsProne(), "attacker should be knocked prone by the crit failure")
	assert.True(t, HasGrappleOpportunity(defender), "defender should gain a grapple reversal opportunity")
	assert.NotEmpty(t, res.Message, "attacker message should be populated")
	assert.NotEmpty(t, res.TargetMessage, "defender message should be populated")
	assert.NotEmpty(t, res.RoomMessage, "room message should be populated")
}
