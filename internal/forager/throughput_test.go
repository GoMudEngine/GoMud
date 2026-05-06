package forager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestThroughput_GetReturnsEmptyOnFirstCall(t *testing.T) {
	ClearThroughputCache()
	tp := GetThroughput("test_zone", 372)
	if tp == nil {
		t.Fatalf("expected non-nil throughput")
	}
	if len(tp.DeliveriesByTier) != 0 {
		t.Errorf("expected empty map, got %v", tp.DeliveriesByTier)
	}
}

func TestThroughput_IncrementDelivery(t *testing.T) {
	ClearThroughputCache()
	IncrementDelivery("test_zone", 372, 50)
	IncrementDelivery("test_zone", 372, 50)
	IncrementDelivery("test_zone", 372, 40)
	tp := GetThroughput("test_zone", 372)
	if tp.DeliveriesByTier[50] != 2 {
		t.Errorf("tier-50: got %d, want 2", tp.DeliveriesByTier[50])
	}
	if tp.DeliveriesByTier[40] != 1 {
		t.Errorf("tier-40: got %d, want 1", tp.DeliveriesByTier[40])
	}
}

func TestThroughput_IncrementDelivery_IgnoresZeroTier(t *testing.T) {
	ClearThroughputCache()
	IncrementDelivery("test_zone", 372, 0)
	IncrementDelivery("test_zone", 372, -1)
	tp := GetThroughput("test_zone", 372)
	if len(tp.DeliveriesByTier) != 0 {
		t.Errorf("zero/negative tiers should be ignored: %v", tp.DeliveriesByTier)
	}
}

func TestThroughput_SaveLoadRoundtrip(t *testing.T) {
	ClearThroughputCache()
	tmpDir := t.TempDir()
	SetThroughputBaseDirForTest(tmpDir)
	defer SetThroughputBaseDirForTest("")

	IncrementDelivery("test_zone", 372, 50)
	IncrementDelivery("test_zone", 372, 30)
	if err := SaveThroughput("test_zone", 372); err != nil {
		t.Fatalf("save: %v", err)
	}

	p := filepath.Join(tmpDir, "test_zone", "372.yaml")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected save file at %s: %v", p, err)
	}

	ClearThroughputCache()
	tp := GetThroughput("test_zone", 372)
	if tp.DeliveriesByTier[50] != 1 || tp.DeliveriesByTier[30] != 1 {
		t.Errorf("after reload: %v", tp.DeliveriesByTier)
	}
}

func TestForager_LbsDeliveredAccumulates(t *testing.T) {
	ClearThroughputCache()
	AddLbsDelivered("test_zone", 372, 150)
	AddLbsDelivered("test_zone", 372, 75)
	tp := GetThroughput("test_zone", 372)
	if tp.LbsDelivered != 225 {
		t.Errorf("LbsDelivered = %d, want 225", tp.LbsDelivered)
	}
}

func TestForager_LbsDelivered_IgnoresZero(t *testing.T) {
	ClearThroughputCache()
	AddLbsDelivered("test_zone", 372, 0)
	tp := GetThroughput("test_zone", 372)
	if tp.LbsDelivered != 0 {
		t.Errorf("zero lbs should not increment: got %d", tp.LbsDelivered)
	}
}

// TestPrewarm_KeysCacheByDisplayZoneNotDirName covers the regression
// where prewarmThroughputFrom keyed the cache by the snake_case
// directory name (e.g. "stillwater_marsh") while the YAML's zone
// field stored the display form ("Stillwater Marsh"). Result: cache
// key mismatch, SaveAllThroughputs failed with "no cached entry"
// errors at shutdown for every prewarmed entry.
func TestPrewarm_KeysCacheByDisplayZoneNotDirName(t *testing.T) {
	ClearThroughputCache()
	tmpDir := t.TempDir()
	SetThroughputBaseDirForTest(tmpDir)
	defer SetThroughputBaseDirForTest("")

	// Write a forager YAML at <tmp>/stillwater_marsh/371.yaml whose
	// zone field stores the display form. This mirrors what the live
	// engine produces when a runtime caller passes mob.Zone.
	zoneDir := filepath.Join(tmpDir, "stillwater_marsh")
	if err := os.MkdirAll(zoneDir, 0755); err != nil {
		t.Fatal(err)
	}
	yamlBody := []byte(
		"mob_id: 371\n" +
			"zone: Stillwater Marsh\n" +
			"deliveries_by_tier:\n  50: 7\n" +
			"last_updated_round: 100\n" +
			"lbs_delivered: 42\n",
	)
	if err := os.WriteFile(filepath.Join(zoneDir, "371.yaml"), yamlBody, 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := prewarmThroughputFrom(tmpDir)
	if err != nil {
		t.Fatalf("prewarm: %v", err)
	}
	if loaded != 1 {
		t.Errorf("loaded = %d, want 1", loaded)
	}

	// SaveAllThroughputs must not error — the cache key built from
	// tp.Zone (display form) must match the key the prewarm stored.
	// The prior bug surfaced as a "no cached entry" error here.
	if err := SaveThroughput("Stillwater Marsh", 371); err != nil {
		t.Errorf("SaveThroughput by display name: %v", err)
	}

	// Runtime callers using mob.Zone (display) should see the
	// prewarmed entry, not create a duplicate.
	tp := GetThroughput("Stillwater Marsh", 371)
	if tp.DeliveriesByTier[50] != 7 {
		t.Errorf("prewarmed tier-50 not surfaced: got %v", tp.DeliveriesByTier)
	}
	if tp.LbsDelivered != 42 {
		t.Errorf("prewarmed LbsDelivered not surfaced: got %d", tp.LbsDelivered)
	}
}

func TestThroughput_TwoDifferentMobs(t *testing.T) {
	ClearThroughputCache()
	IncrementDelivery("zone_a", 100, 50)
	IncrementDelivery("zone_b", 200, 40)
	tp1 := GetThroughput("zone_a", 100)
	tp2 := GetThroughput("zone_b", 200)
	if tp1.DeliveriesByTier[50] != 1 {
		t.Errorf("zone_a/100 tier-50: got %d, want 1", tp1.DeliveriesByTier[50])
	}
	if tp2.DeliveriesByTier[40] != 1 {
		t.Errorf("zone_b/200 tier-40: got %d, want 1", tp2.DeliveriesByTier[40])
	}
	// Cross-pollination check.
	if len(tp1.DeliveriesByTier) > 1 || len(tp2.DeliveriesByTier) > 1 {
		t.Errorf("counters should be per-mob; got tp1=%v, tp2=%v",
			tp1.DeliveriesByTier, tp2.DeliveriesByTier)
	}
}
