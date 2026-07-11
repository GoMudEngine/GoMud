package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanAffordCompanion_Budget(t *testing.T) {
	// Newbie: ConvictionMax ~440, wolf reserves 333 -> affords 1, not a 2nd.
	c := New()
	c.ConvictionMax.Value = 440
	assert.True(t, c.CanAffordCompanion(333))
	c.AddCompanion(CompanionInfo{InstanceId: 1, Name: "Wolf", ConvictionReserve: 333})
	assert.False(t, c.CanAffordCompanion(333)) // 333+333=666 > 440

	// Meirok: ConvictionMax ~547, golem reserves 229 -> affords 2, not 3.
	m := New()
	m.ConvictionMax.Value = 547
	m.AddCompanion(CompanionInfo{InstanceId: 1, Name: "G1", ConvictionReserve: 229})
	assert.True(t, m.CanAffordCompanion(229)) // 229+229=458 <= 547
	m.AddCompanion(CompanionInfo{InstanceId: 2, Name: "G2", ConvictionReserve: 229})
	assert.False(t, m.CanAffordCompanion(229)) // 458+229=687 > 547

	// Soft backstop still bites even if the budget allows (cheap pets).
	s := New()
	s.ConvictionMax.Value = 100000
	for i := 0; i < 5; i++ {
		s.AddCompanion(CompanionInfo{InstanceId: i + 1, Name: "x", ConvictionReserve: 1})
	}
	assert.False(t, s.CanAffordCompanion(1)) // at soft cap 5
}
