package factions

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

func TestRepConstants(t *testing.T) {
	if RepMin != -100 {
		t.Errorf("RepMin = %d, want -100", RepMin)
	}
	if RepMax != 100 {
		t.Errorf("RepMax = %d, want 100", RepMax)
	}
}

func setupTestFaction(t *testing.T, slug string, defaultRep int) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	body := "faction_id: " + slug + "\n" +
		"display_name: \"" + slug + "\"\n" +
		"description: \"x\"\n" +
		"default_rep: " + intToStr(defaultRep) + "\n" +
		"allies: []\n" +
		"enemies: []\n"
	if err := os.WriteFile(filepath.Join(dir, slug+".yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	clearRegistryForTest()
	clearRepCacheForTest()
	if err := LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	roundForTest = func() uint64 { return 1000 }
	t.Cleanup(func() { roundForTest = nil })
}

func intToStr(n int) string {
	return fmt.Sprintf("%d", n)
}

func TestGetRep_ReturnsDefaultWhenNoRow(t *testing.T) {
	setupTestFaction(t, "warren", -25)
	if got := GetRep("warren", 17); got != -25 {
		t.Errorf("GetRep with no row = %d, want -25 (default_rep)", got)
	}
}

func TestGetRep_UnknownFactionReturnsZero(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	if got := GetRep("nonexistent", 17); got != 0 {
		t.Errorf("GetRep on unknown faction = %d, want 0", got)
	}
}

func TestSetRep_PersistsAndClamps(t *testing.T) {
	setupTestFaction(t, "warren", 0)

	SetRep("warren", 17, 50)
	if got := GetRep("warren", 17); got != 50 {
		t.Errorf("after Set 50, GetRep = %d, want 50", got)
	}

	SetRep("warren", 17, 500)
	if got := GetRep("warren", 17); got != 100 {
		t.Errorf("after Set 500 (clamped), GetRep = %d, want 100", got)
	}

	SetRep("warren", 17, -500)
	if got := GetRep("warren", 17); got != -100 {
		t.Errorf("after Set -500 (clamped), GetRep = %d, want -100", got)
	}

	// Drop cache, reload from disk.
	ClearCache()
	if got := GetRep("warren", 17); got != -100 {
		t.Errorf("after restart-equivalent, GetRep = %d, want -100", got)
	}
}

func TestBumpRep_StartsFromDefaultWhenNoRow(t *testing.T) {
	setupTestFaction(t, "warren", -25)
	BumpRep("warren", 17, 30)
	// -25 + 30 = +5
	if got := GetRep("warren", 17); got != 5 {
		t.Errorf("after first Bump from default, GetRep = %d, want 5", got)
	}
}

func TestBumpRep_AccumulatesAndClamps(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	for i := 0; i < 12; i++ {
		BumpRep("warren", 17, 10)
	}
	// 12 * 10 = 120, clamped to 100
	if got := GetRep("warren", 17); got != 100 {
		t.Errorf("after 12 bumps of 10, GetRep = %d, want 100 (clamped)", got)
	}
}

func TestBumpRep_UnknownFactionIsNoop(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	BumpRep("nonexistent", 17, 50)
	// No panic; nothing else to assert (no rep file should be written).
}

func TestTierFor(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	SetRep("warren", 17, -60)
	if got := TierFor("warren", 17); got != opinions.TierHostile {
		t.Errorf("after Set -60, TierFor = %v, want Hostile", got)
	}
	SetRep("warren", 17, 25)
	if got := TierFor("warren", 17); got != opinions.TierWarm {
		t.Errorf("after Set 25, TierFor = %v, want Warm", got)
	}
}

func TestFactionsForMob_OnlyReturnsDefinedFactions(t *testing.T) {
	setupTestFaction(t, "warren", 0)

	mob := &mobs.Mob{
		Groups: []string{"humanoid", "warren", "undocumented_group"},
	}
	got := FactionsForMob(mob)
	if len(got) != 1 || got[0] != "warren" {
		t.Errorf("FactionsForMob = %v, want [warren]", got)
	}
}

func TestFactionsForMob_EmptyForNoMatches(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	mob := &mobs.Mob{Groups: []string{"humanoid", "undead"}}
	if got := FactionsForMob(mob); len(got) != 0 {
		t.Errorf("FactionsForMob = %v, want []", got)
	}
}

func TestIsPeacefulToward_FalseAtNeutral(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	mob := &mobs.Mob{Groups: []string{"warren"}}
	if IsPeacefulToward(mob, 17) {
		t.Errorf("IsPeacefulToward at default 0 should be false (Neutral tier)")
	}
}

func TestIsPeacefulToward_TrueAtWarmTier(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	SetRep("warren", 17, 30)
	mob := &mobs.Mob{Groups: []string{"warren"}}
	if !IsPeacefulToward(mob, 17) {
		t.Errorf("IsPeacefulToward at rep 30 (Warm) should be true")
	}
}

func TestIsPeacefulToward_AnyMatchingFactionTriggers(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	// Set up a second faction inline.
	dir := os.Getenv("DOGMUD_FACTIONS_DIR_OVERRIDE")
	if err := os.WriteFile(filepath.Join(dir, "thornwall_guards.yaml"),
		[]byte(`faction_id: thornwall_guards
display_name: "Thornwall Guards"
description: "x"
default_rep: 0
allies: []
enemies: []
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	SetRep("warren", 17, 30) // Warm with warren only
	mob := &mobs.Mob{Groups: []string{"warren", "thornwall_guards"}}
	if !IsPeacefulToward(mob, 17) {
		t.Errorf("IsPeacefulToward should be true if any defined faction is Warm+")
	}
}

func TestIsPeacefulToward_FalseWhenMobHasNoDefinedFactions(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	mob := &mobs.Mob{Groups: []string{"humanoid"}}
	if IsPeacefulToward(mob, 17) {
		t.Errorf("IsPeacefulToward with no defined factions should be false")
	}
	_ = characters.Character{} // keep import alive
}

func TestParallelBumpsConverge(t *testing.T) {
	setupTestFaction(t, "warren", 0)

	const goroutines = 10
	const bumpsPer = 100
	const delta = 1

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < bumpsPer; j++ {
				BumpRep("warren", 17, delta)
			}
		}()
	}
	wg.Wait()

	want := goroutines * bumpsPer
	if want > RepMax {
		want = RepMax
	}
	if got := GetRep("warren", 17); got != want {
		t.Errorf("after %d parallel bumps: got %d, want %d", goroutines*bumpsPer, got, want)
	}
}
