package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZoneConfig_InstanceDefaults(t *testing.T) {
	zc := NewZoneConfig("test-zone")
	assert.False(t, zc.Instanced)
	assert.Equal(t, "rejoin", zc.DeathPolicy)
	assert.Equal(t, "", zc.PortalDuration)
	assert.Equal(t, 0, zc.EntryRoom)
	assert.True(t, zc.AllowRecall)
}

func TestZoneConfig_ValidateDefaults(t *testing.T) {
	zc := NewZoneConfig("test-zone")
	err := zc.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "rejoin", zc.DeathPolicy)
	assert.Equal(t, "", zc.PortalDuration)
}

func TestZoneConfig_ValidateInstancePortalDuration(t *testing.T) {
	zc := NewZoneConfig("instance-zone")
	zc.Instanced = true
	err := zc.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "30m", zc.PortalDuration)
}

func TestZoneConfig_ValidateInstanceCustomPortalDuration(t *testing.T) {
	zc := NewZoneConfig("instance-zone")
	zc.Instanced = true
	zc.PortalDuration = "1h"
	err := zc.Validate()
	assert.NoError(t, err)
	assert.Equal(t, "1h", zc.PortalDuration)
}
