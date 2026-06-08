package playtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigDefaults(t *testing.T) {
	// fakeGetter simulates plug.Config.Get returning nil for unset keys.
	c := buildConfig(func(string) any { return nil })
	assert.False(t, c.Enabled) // nil -> zero value; real defaults come from overlay yaml
	assert.Equal(t, "", c.SandboxZoneTag)
}

func TestConfigReadsValues(t *testing.T) {
	vals := map[string]any{
		"Enabled":         true,
		"SafeMode":        true,
		"SandboxZoneTag":  "playtest-sandbox",
		"DeathProtection": true,
		"Beacons":         true,
	}
	c := buildConfig(func(k string) any { return vals[k] })
	assert.True(t, c.Enabled)
	assert.True(t, c.SafeMode)
	assert.Equal(t, "playtest-sandbox", c.SandboxZoneTag)
	assert.True(t, c.DeathProtection)
	assert.True(t, c.Beacons)
}
