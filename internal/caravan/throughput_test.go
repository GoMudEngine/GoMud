package caravan

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
