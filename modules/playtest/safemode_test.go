package playtest

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

func TestShouldSnapBack(t *testing.T) {
	// AI-port tester leaving the sandbox -> snap back.
	assert.True(t, shouldSnapBack(true, "sandbox", []string{"town"}))
	// AI-port tester staying in the sandbox -> no snap back.
	assert.False(t, shouldSnapBack(true, "sandbox", []string{"sandbox", "quiet"}))
	// No sandbox tag configured -> never snap back.
	assert.False(t, shouldSnapBack(true, "", []string{"town"}))
	// Not an AI-port session -> never snap back.
	assert.False(t, shouldSnapBack(false, "sandbox", []string{"town"}))
}

func TestContainsTag(t *testing.T) {
	assert.True(t, containsTag([]string{"a", "sandbox"}, "sandbox"))
	assert.False(t, containsTag([]string{"a", "b"}, "sandbox"))
	assert.False(t, containsTag(nil, "sandbox"))
}

// TestApplyDeathProtection verifies the DOGMud adaptation:
// DOGMud has no ExtraLives / permadeath mechanic (death routes through
// justice/jail; bleedout/downed were removed), so applyDeathProtection is a
// no-op. We only verify it is safe to call and does not panic.
func TestApplyDeathProtection(t *testing.T) {
	m := &PlaytestModule{cfg: Config{DeathProtection: true}}
	u := &users.UserRecord{Character: &characters.Character{}}
	// Must not panic; no field is mutated (no-op in DOGMud).
	assert.NotPanics(t, func() { m.applyDeathProtection(u) })

	// Nil character — must not panic.
	uNil := &users.UserRecord{Character: nil}
	assert.NotPanics(t, func() { m.applyDeathProtection(uNil) })

	// Disabled — must not panic.
	mOff := &PlaytestModule{cfg: Config{DeathProtection: false}}
	assert.NotPanics(t, func() { mOff.applyDeathProtection(u) })
}
