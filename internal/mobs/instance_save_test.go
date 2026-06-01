package mobs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

// withMobProgressionEnabled turns on Balance.MobProgressionEnabled for
// the duration of a test and restores the prior value via t.Cleanup.
// Several tests in this package need progression active so that
// SaveMobInstance actually writes a file.
func withMobProgressionEnabled(t *testing.T) {
	t.Helper()
	prev := configs.GetBalanceConfig()
	configs.AddOverlayOverrides(map[string]any{"Balance.MobProgressionEnabled": true})
	t.Cleanup(func() {
		configs.AddOverlayOverrides(map[string]any{"Balance.MobProgressionEnabled": bool(prev.MobProgressionEnabled)})
	})
}

// TestSaveMobInstance_CharmedMobSkipsWrite verifies the guard added to
// SaveMobInstance: any mob charmed to a user must not write to
// mobs.instances/ because its progression lives on CompanionInfo on the
// owner's user YAML.
func TestSaveMobInstance_CharmedMobSkipsWrite(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	withMobProgressionEnabled(t)

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

	withMobProgressionEnabled(t)

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

// TestNukeSummonsInstances_RemovesAllFiles verifies the boot-cleanup
// nuke — every file under mobs.instances/summons/ is removed, and the
// count is returned for logging.
func TestNukeSummonsInstances_RemovesAllFiles(t *testing.T) {
	baseDir := filepath.Join(
		configs.GetFilePathsConfig().DataFiles.String(),
		"mobs.instances", "summons")

	// Seed three fake files under summons/.
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.RemoveAll(baseDir)

	for _, name := range []string{"1-foo-room1.yaml", "2-bar-room2.yaml", "3-baz-room3.yaml"} {
		if err := os.WriteFile(filepath.Join(baseDir, name), []byte("x"), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	pruned := NukeSummonsInstances()
	assert.Equal(t, 3, pruned, "expected 3 files nuked")

	// Directory should be empty (may still exist).
	entries, err := os.ReadDir(baseDir)
	if err == nil {
		assert.Empty(t, entries, "summons/ should have no files remaining")
	}
}

// TestNukeSummonsInstances_IgnoresOtherZones verifies the nuke only
// targets summons/ — legitimate world-mob instance files under other
// zones are untouched.
func TestNukeSummonsInstances_IgnoresOtherZones(t *testing.T) {
	base := filepath.Join(
		configs.GetFilePathsConfig().DataFiles.String(),
		"mobs.instances")

	summonsDir := filepath.Join(base, "summons")
	worldDir := filepath.Join(base, "thornwall_city")

	if err := os.MkdirAll(summonsDir, 0755); err != nil {
		t.Fatalf("mkdir summons: %v", err)
	}
	if err := os.MkdirAll(worldDir, 0755); err != nil {
		t.Fatalf("mkdir world: %v", err)
	}
	defer os.RemoveAll(summonsDir)
	defer os.RemoveAll(worldDir)

	_ = os.WriteFile(filepath.Join(summonsDir, "1-foo-room1.yaml"), []byte("x"), 0644)
	worldFile := filepath.Join(worldDir, "2-wolf-room200.yaml")
	_ = os.WriteFile(worldFile, []byte("x"), 0644)

	pruned := NukeSummonsInstances()
	assert.Equal(t, 1, pruned)

	// World-zone file must still exist.
	_, err := os.Stat(worldFile)
	assert.NoError(t, err, "world-mob instance file must not be touched")
}

// TestNukeSummonsInstances_NoDirectory verifies the nuke is a no-op
// (no panic, returns 0) when the summons/ directory doesn't exist.
func TestNukeSummonsInstances_NoDirectory(t *testing.T) {
	base := filepath.Join(
		configs.GetFilePathsConfig().DataFiles.String(),
		"mobs.instances", "summons")
	_ = os.RemoveAll(base)

	pruned := NukeSummonsInstances()
	assert.Equal(t, 0, pruned)
}

func TestMobInstanceData_GoalProgressFields_RoundTrip(t *testing.T) {
	gold := 999
	in := MobInstanceData{
		Gold: &gold,
		Equipment: &characters.Worn{
			Body: items.Item{ItemId: 1, EnchantTier: 3, EnchantType: "frost"},
		},
		PlanState: map[string]any{"plan:wealth-gold:target_shop_room": 4101},
	}

	bytes, err := yaml.Marshal(&in)
	assert.NoError(t, err)

	var out MobInstanceData
	assert.NoError(t, yaml.Unmarshal(bytes, &out))

	assert.NotNil(t, out.Gold)
	assert.Equal(t, 999, *out.Gold)
	assert.NotNil(t, out.Equipment)
	assert.Equal(t, 1, out.Equipment.Body.ItemId)
	assert.Equal(t, 3, out.Equipment.Body.EnchantTier)
	assert.Equal(t, "frost", out.Equipment.Body.EnchantType)
	assert.Equal(t, 4101, out.PlanState["plan:wealth-gold:target_shop_room"])
}

func TestMobInstanceData_GoldZero_RoundTrips(t *testing.T) {
	zero := 0
	in := MobInstanceData{Gold: &zero}
	b, err := yaml.Marshal(&in)
	assert.NoError(t, err)

	var out MobInstanceData
	assert.NoError(t, yaml.Unmarshal(b, &out))
	assert.NotNil(t, out.Gold, "non-nil *int(0) must survive marshal (presence semantics)")
	assert.Equal(t, 0, *out.Gold)
}

func TestCollectPlanState_OnlyPlanPrefixedKeys(t *testing.T) {
	mob := &Mob{}
	mob.Character.MiscData = map[string]any{
		"plan:wealth-gold:target_shop_room": 4101,
		"plan:upgrade-gear:worst_slot":      "body",
		"conversation_line_idx":             2,
		"faction_kills:bandits":             3,
	}

	got := collectPlanState(mob)
	assert.Len(t, got, 2)
	assert.Equal(t, 4101, got["plan:wealth-gold:target_shop_room"])
	assert.Equal(t, "body", got["plan:upgrade-gear:worst_slot"])
	_, hasNonPlan := got["conversation_line_idx"]
	assert.False(t, hasNonPlan)
}

func TestCollectPlanState_NilMiscData(t *testing.T) {
	mob := &Mob{}
	assert.Nil(t, collectPlanState(mob))
}
