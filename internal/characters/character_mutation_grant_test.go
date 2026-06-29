package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// GrantRandomMutation rolls one mutation from the weighted acquisition pool
// for the character's species and adds it at level 1. Returns the granted id
// ("" if none available).
func TestGrantRandomMutation_AddsOne(t *testing.T) {
	c := &Character{SpeciesId: 1, Mutations: map[string]int{}}
	got := c.GrantRandomMutation()
	if got == "" {
		t.Skip("no mutations registered in this test env; covered by integration")
	}
	assert.Equal(t, 1, c.Mutations[got], "granted mutation must be at level 1")
	assert.Len(t, c.Mutations, 1)
}
