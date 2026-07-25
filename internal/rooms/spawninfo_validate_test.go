package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/stretchr/testify/assert"
)

func TestValidateSpawnEntry_KindIsExclusive(t *testing.T) {
	known := SpawnValidators{
		MobExists:  func(int) bool { return true },
		ItemExists: func(int) bool { return true },
		BuffExists: func(int) bool { return true },
		PeriodOK:   func(string) bool { return true },
		Containers: map[string]struct{}{"chest": {}},
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
	known := SpawnValidators{
		MobExists:  func(int) bool { return true },
		ItemExists: func(int) bool { return true },
		BuffExists: func(int) bool { return true },
		PeriodOK:   func(string) bool { return true },
		Containers: map[string]struct{}{"chest": {}},
	}

	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{ItemId: 1, Container: "chest"}, known))
	// A mob cannot spawn into a container.
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, Container: "chest"}, known))
	// The container must exist in THIS room.
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{ItemId: 1, Container: "barrel"}, known))
}

func TestValidateSpawnEntry_UnknownReferences(t *testing.T) {
	none := SpawnValidators{
		MobExists:  func(int) bool { return false },
		ItemExists: func(int) bool { return false },
		BuffExists: func(int) bool { return false },
		PeriodOK:   func(string) bool { return true },
	}
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 999}, none))
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{ItemId: 999}, none))

	badBuff := SpawnValidators{
		MobExists:  func(int) bool { return true },
		ItemExists: func(int) bool { return true },
		BuffExists: func(int) bool { return false },
		PeriodOK:   func(string) bool { return true },
	}
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, BuffIds: []int{404}}, badBuff))
}

// An unparseable respawn rate does not error at runtime — AddPeriod returns
// the caller's own round number, so the mob respawns IMMEDIATELY. Catch it
// here, where an author can still see it.
func TestValidateSpawnEntry_RespawnRateMustParse(t *testing.T) {
	v := SpawnValidators{
		MobExists:  func(int) bool { return true },
		ItemExists: func(int) bool { return true },
		BuffExists: func(int) bool { return true },
		PeriodOK:   func(p string) bool { return p == "5 real minutes" },
	}
	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, RespawnRate: "5 real minutes"}, v))
	assert.NoError(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, RespawnRate: ""}, v), "empty means the 15-minute default")
	assert.Error(t, ValidateSpawnEntry(SpawnInfo{MobId: 1, RespawnRate: "banana"}, v))
}

// AddPeriod cannot be asked whether it understood a string: it never errors,
// and it never leaves the round unchanged. Unrecognised units take a generic
// failover that treats the quantity as ROUNDS, so "banana" yields one round —
// about four seconds. A typo'd respawn rate is therefore not a loud failure
// but a spawn that returns almost instantly, which is exactly what this
// validator exists to catch.
func TestRealPeriodOK(t *testing.T) {
	for _, ok := range []string{
		"", // empty is legal — it means the engine default
		"5 real minutes", "10 real minutes", "2 real hours", "2 real days",
		"600 rounds", // the failover path, and the idiom used across the world
		"1 game day", "daily", "hourly", "2 sunrises", "sunset",
	} {
		if !RealPeriodOK(ok) {
			t.Errorf("RealPeriodOK(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{
		"banana", "soon", "5 bananas", "every so often",
		"0 real minutes", "-3 rounds", "later on today please",
	} {
		if RealPeriodOK(bad) {
			t.Errorf("RealPeriodOK(%q) = true, want false", bad)
		}
	}
}

// Anti-drift guard. RealPeriodOK maintains its own copy of AddPeriod's unit
// vocabulary because the parser cannot be interrogated. If AddPeriod ever
// grows a real branch for a unit we treat as failover-only, or loses one we
// accept, this test fails and the vocabulary must be re-checked against it.
func TestRealPeriodOK_VocabularyMatchesParser(t *testing.T) {
	gd := gametime.GetDate(1)
	// A word with no branch takes the failover: exactly qty rounds.
	if got, want := gd.AddPeriod("7 flurbles"), gd.RoundNumber+7; got != want {
		t.Errorf("AddPeriod failover changed: got %d want %d — re-check periodUnitPrefixes", got, want)
	}
	// Every prefix we accept as a REAL unit must advance by more than the
	// failover would, i.e. it must hit a genuine branch. "rou" is excluded:
	// it IS the failover.
	for _, u := range periodUnitPrefixes {
		if u == "rou" {
			continue
		}
		p := "2 real " + u + "s"
		if got := gd.AddPeriod(p); got <= gd.RoundNumber+2 {
			t.Errorf("AddPeriod(%q) = %d, only failover-far from %d — %q may no longer be a real unit",
				p, got, gd.RoundNumber, u)
		}
	}
}
