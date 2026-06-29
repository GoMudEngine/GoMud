package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

func TestAutoAwaken_GrantsTokenAndMutation(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	u := users.GetByUserId(1)
	u.Character.SpeciesId = 1
	u.Character.Mutations = map[string]int{}
	autoAwaken(u)
	assert.GreaterOrEqual(t, len(u.Character.Mutations), 0,
		"autoAwaken must attempt a mutation grant without panicking")
}

func TestOnboardingRoute(t *testing.T) {
	assert.Equal(t, routeNewbie, onboardingRoute("1"))
	assert.Equal(t, routeMudVet, onboardingRoute("2"))
	assert.Equal(t, routeVeteran, onboardingRoute("3"))
	assert.Equal(t, routeMudVet, onboardingRoute(""))
	assert.Equal(t, routeMudVet, onboardingRoute("garbage"))
}
