package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSpawnEntry_KindIsExclusive(t *testing.T) {
	known := spawnValidators{
		mobExists:  func(int) bool { return true },
		itemExists: func(int) bool { return true },
		buffExists: func(int) bool { return true },
		periodOK:   func(string) bool { return true },
		containers: map[string]struct{}{"chest": {}},
	}

	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{MobId: 1}, known))
	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{ItemId: 1}, known))
	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{Gold: 5}, known))

	// Exactly one kind.
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{}, known), "an entry must spawn something")
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, ItemId: 1}, known))
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, Gold: 5}, known))
}

func TestValidateSpawnEntry_ContainerRules(t *testing.T) {
	known := spawnValidators{
		mobExists:  func(int) bool { return true },
		itemExists: func(int) bool { return true },
		buffExists: func(int) bool { return true },
		periodOK:   func(string) bool { return true },
		containers: map[string]struct{}{"chest": {}},
	}

	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{ItemId: 1, Container: "chest"}, known))
	// A mob cannot spawn into a container.
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, Container: "chest"}, known))
	// The container must exist in THIS room.
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{ItemId: 1, Container: "barrel"}, known))
}

func TestValidateSpawnEntry_UnknownReferences(t *testing.T) {
	none := spawnValidators{
		mobExists:  func(int) bool { return false },
		itemExists: func(int) bool { return false },
		buffExists: func(int) bool { return false },
		periodOK:   func(string) bool { return true },
	}
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 999}, none))
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{ItemId: 999}, none))

	badBuff := spawnValidators{
		mobExists:  func(int) bool { return true },
		itemExists: func(int) bool { return true },
		buffExists: func(int) bool { return false },
		periodOK:   func(string) bool { return true },
	}
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, BuffIds: []int{404}}, badBuff))
}

// An unparseable respawn rate does not error at runtime — AddPeriod returns
// the caller's own round number, so the mob respawns IMMEDIATELY. Catch it
// here, where an author can still see it.
func TestValidateSpawnEntry_RespawnRateMustParse(t *testing.T) {
	v := spawnValidators{
		mobExists:  func(int) bool { return true },
		itemExists: func(int) bool { return true },
		buffExists: func(int) bool { return true },
		periodOK:   func(p string) bool { return p == "5 real minutes" },
	}
	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, RespawnRate: "5 real minutes"}, v))
	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, RespawnRate: ""}, v), "empty means the 15-minute default")
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, RespawnRate: "banana"}, v))
}
