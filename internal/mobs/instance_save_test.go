package mobs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
)

// TestSaveMobInstance_CharmedMobSkipsWrite verifies the guard added to
// SaveMobInstance: any mob charmed to a user must not write to
// mobs.instances/ because its progression lives on CompanionInfo on the
// owner's user YAML.
func TestSaveMobInstance_CharmedMobSkipsWrite(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Enable mob progression for this test
	prevBal := configs.GetBalanceConfig()
	t.Cleanup(func() {
		configs.AddOverlayOverrides(map[string]any{
			"Balance.MobProgressionEnabled": prevBal.MobProgressionEnabled,
		})
	})
	configs.AddOverlayOverrides(map[string]any{
		"Balance.MobProgressionEnabled": true,
	})

	mob := NewMobById(1, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil")
	}

	// Clean up any stale files from previous test runs
	path := instancePath(mob.MobId, mob.Zone, mob.Character.Name, mob.HomeRoomId)
	_ = os.Remove(path)
	_ = os.Remove(filepath.Dir(path))

	// Give the mob some progression so it WOULD normally persist.
	mob.Character.Stats.Strength.Training = 10
	// Charm it to a user — this is the signal that it's a companion.
	mob.Character.Charm(42, 99999, "")

	err := SaveMobInstance(mob)
	assert.NoError(t, err)

	// Assert no file was written.
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr),
		"expected no file at %s for charmed mob, got stat err %v",
		path, statErr)

	// Cleanup in case the test fails and a file was written.
	_ = os.Remove(path)
	_ = os.Remove(filepath.Dir(path))
}

// TestSaveMobInstance_UncharmedMobWritesFile verifies the inverse: an
// uncharmed mob with progression still gets its file written, so genuine
// world-mob persistence is unaffected by the guard.
func TestSaveMobInstance_UncharmedMobWritesFile(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Enable mob progression for this test
	prevBal := configs.GetBalanceConfig()
	t.Cleanup(func() {
		configs.AddOverlayOverrides(map[string]any{
			"Balance.MobProgressionEnabled": prevBal.MobProgressionEnabled,
		})
	})
	configs.AddOverlayOverrides(map[string]any{
		"Balance.MobProgressionEnabled": true,
	})

	mob := NewMobById(1, 100)
	if mob == nil {
		t.Fatal("NewMobById returned nil")
	}

	// Clean up any stale files from previous test runs
	path := instancePath(mob.MobId, mob.Zone, mob.Character.Name, mob.HomeRoomId)
	_ = os.Remove(path)
	_ = os.Remove(filepath.Dir(path))

	mob.Character.Stats.Strength.Training = 10
	// NOT charming — this is an organic world mob.

	err := SaveMobInstance(mob)
	assert.NoError(t, err)

	// Verify the file was written
	_, statErr := os.Stat(path)
	assert.NoError(t, statErr, "expected file at %s for uncharmed mob", path)

	// Cleanup.
	_ = os.Remove(path)
	_ = os.Remove(filepath.Dir(path))
}
