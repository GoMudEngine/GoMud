package bounties

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
)

// cleanBountiesFile removes the persisted registry file so the test
// starts from a clean slate (no entries from previous test runs).
func cleanBountiesFile() {
	p := filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "bounties.yaml")
	os.Remove(p)
}

// TestSeedFromEntries_Idempotency verifies that seedFromEntries declares
// each entry on the first call and skips them (returns 0) on a second
// call because OpenForTarget already returns results.
func TestSeedFromEntries_Idempotency(t *testing.T) {
	cleanBountiesFile()
	resetCache()
	defer func() {
		roundForTest = nil
		goldMultiplierForTest = nil
		goldFloorForTest = nil
		statpoolForTest = nil
	}()

	// Stub round + statpool so no real config/mob fixtures are needed.
	roundForTest = func() uint64 { return 100 }
	goldMultiplierForTest = func() float64 { return 0.5 }
	goldFloorForTest = func() int { return 50 }
	statpoolForTest = func(_ knowledge.Subject) int { return 200 }

	entries := []standingEntry{
		{
			TargetMobId:   272,
			IssuerFaction: "thornwall_guards",
			Reason:        "Standing contract: the Chrysalis Phantom",
		},
		{
			TargetMobId:   286,
			IssuerFaction: "thornwall_guards",
			Reason:        "Standing contract: the bandit Soren",
		},
	}

	// First call: both bounties are declared (OpenForTarget returns empty).
	count1 := seedFromEntries(entries)
	if count1 != 2 {
		t.Errorf("first seedFromEntries: expected 2 declared, got %d", count1)
	}

	// Verify both bounties are now open.
	for _, e := range entries {
		tgt := knowledge.MobSubject(e.TargetMobId)
		open := OpenForTarget(tgt)
		if len(open) != 1 {
			t.Errorf("mobId=%d: expected 1 open bounty after seed, got %d",
				e.TargetMobId, len(open))
		}
	}

	// Second call: all already open — should declare 0.
	count2 := seedFromEntries(entries)
	if count2 != 0 {
		t.Errorf("second seedFromEntries (idempotency): expected 0 declared, got %d", count2)
	}

	// Total open bounties unchanged.
	allOpen := AllOpen()
	if len(allOpen) != 2 {
		t.Errorf("total open bounties should still be 2, got %d", len(allOpen))
	}
}

// TestSeedFromEntries_SkipsInvalidEntries verifies that entries with
// zero TargetMobId or empty IssuerFaction are silently skipped.
func TestSeedFromEntries_SkipsInvalidEntries(t *testing.T) {
	cleanBountiesFile()
	resetCache()
	defer func() {
		roundForTest = nil
		goldMultiplierForTest = nil
		goldFloorForTest = nil
		statpoolForTest = nil
	}()

	roundForTest = func() uint64 { return 100 }
	goldMultiplierForTest = func() float64 { return 0.5 }
	goldFloorForTest = func() int { return 50 }
	statpoolForTest = func(_ knowledge.Subject) int { return 200 }

	entries := []standingEntry{
		{TargetMobId: 0, IssuerFaction: "thornwall_guards", Reason: "zero mob id — skip"},
		{TargetMobId: 272, IssuerFaction: "", Reason: "empty faction — skip"},
		{TargetMobId: 286, IssuerFaction: "thornwall_guards", Reason: "valid entry"},
	}

	count := seedFromEntries(entries)
	if count != 1 {
		t.Errorf("expected 1 valid entry declared, got %d", count)
	}
}
