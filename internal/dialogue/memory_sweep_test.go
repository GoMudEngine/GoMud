package dialogue

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestSweepMemories_DropsIdleAndKeepsRecent(t *testing.T) {
	defer util.ResetRoundCountForTest()
	for k := range memoryCache {
		delete(memoryCache, k)
	}

	util.SetRoundCountForTest(memorySweepIdleRounds + 500)

	// Idle: last visited at round 1, far beyond the window.
	idle := GetMemory(101, 1)
	idle.LastVisitRound = 1

	// Recent: visited just now.
	recent := GetMemory(102, 1)
	recent.LastVisitRound = util.GetRoundCount()

	// Never visited: created but no node reached yet.
	_ = GetMemory(103, 1)

	removed := SweepMemories()

	if removed != 1 {
		t.Errorf("expected 1 memory swept, got %d", removed)
	}
	if _, ok := memoryCache[memKey(101, 1)]; ok {
		t.Error("idle memory survived the sweep")
	}
	if _, ok := memoryCache[memKey(102, 1)]; !ok {
		t.Error("recently-used memory was swept")
	}
	if _, ok := memoryCache[memKey(103, 1)]; !ok {
		t.Error("never-visited memory was swept; it is at most one round old")
	}
}

func TestForgetMobInstance_OnlyDropsThatInstance(t *testing.T) {
	for k := range memoryCache {
		delete(memoryCache, k)
	}

	// Two players remember mob instance 200; one remembers 201.
	GetMemory(200, 1)
	GetMemory(200, 2)
	GetMemory(201, 1)

	removed := ForgetMobInstance(200)

	if removed != 2 {
		t.Errorf("expected 2 memories forgotten, got %d", removed)
	}
	if _, ok := memoryCache[memKey(201, 1)]; !ok {
		t.Error("ForgetMobInstance dropped a different mob instance's memory")
	}
	if _, ok := memoryCache[memKey(200, 1)]; ok {
		t.Error("target instance memory survived")
	}
}

// The key packs two ints into one uint64; a high user id must not bleed into
// the mob-instance half and make ForgetMobInstance drop the wrong entries.
func TestForgetMobInstance_HighUserIdDoesNotBleedIntoKey(t *testing.T) {
	for k := range memoryCache {
		delete(memoryCache, k)
	}

	GetMemory(1, 0x7FFFFFFF)
	GetMemory(2, 1)

	if removed := ForgetMobInstance(2); removed != 1 {
		t.Errorf("expected 1 forgotten, got %d", removed)
	}
	if _, ok := memoryCache[memKey(1, 0x7FFFFFFF)]; !ok {
		t.Error("a large user id bled into the mob-instance half of the key")
	}
}
