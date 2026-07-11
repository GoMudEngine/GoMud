package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPoolReservation_IncludesCompanions(t *testing.T) {
	c := New()
	c.Skills[string(skills.Manifestation)] = 38 // old cap 2 — keeps this test independent of Task 6
	c.RecalculateStats()
	base := c.GetPoolReservation("conviction", c.ConvictionMax.Value)

	require.True(t, c.AddCompanion(CompanionInfo{InstanceId: 1, Name: "A", ConvictionReserve: 100}))
	require.True(t, c.AddCompanion(CompanionInfo{InstanceId: 2, Name: "B", ConvictionReserve: 60}))

	got := c.GetPoolReservation("conviction", c.ConvictionMax.Value)
	assert.Equal(t, base+160, got)

	// Companions never affect non-conviction pools.
	assert.Equal(t, 0, c.GetPoolReservation("stamina", c.StaminaMax.Value))
}
