package splash

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

func TestSplashEventType(t *testing.T) {
	assert.Equal(t, "Splash", Splash{}.Type())
}

func TestTargetConstants(t *testing.T) {
	// Distinct values; Global is the zero value so a bare Splash{} is global.
	assert.Equal(t, SplashTarget(0), TargetGlobal)
	assert.NotEqual(t, TargetGlobal, TargetZone)
	assert.NotEqual(t, TargetZone, TargetUser)
}

func TestFilterByZone(t *testing.T) {
	u := func(zone string) *users.UserRecord {
		return &users.UserRecord{Character: &characters.Character{Zone: zone}}
	}
	all := []*users.UserRecord{u("Stillwater"), u("Thornwall City"), u("Stillwater"), nil}

	got := filterByZone(all, "Stillwater")
	assert.Len(t, got, 2, "both Stillwater users match; other zone + nil skipped")

	assert.Len(t, filterByZone(all, "Nowhere"), 0)
	assert.Len(t, filterByZone(nil, "Stillwater"), 0)
}
