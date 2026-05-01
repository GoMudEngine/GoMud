package shops

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

// writeZoneShopFile writes a ShopInventory YAML to baseDir/{zone}/{mobid}-room{roomid}.yaml.
func writeZoneShopFile(t *testing.T, baseDir, zone string, mobId, roomId int, inv ShopInventory) {
	t.Helper()
	zoneDir := filepath.Join(baseDir, zone)
	if err := os.MkdirAll(zoneDir, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", zoneDir, err)
	}
	filename := fmt.Sprintf("%d-room%d.yaml", mobId, roomId)
	data, err := yaml.Marshal(&inv)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(zoneDir, filename), data, 0644); err != nil {
		t.Fatalf("writefile: %v", err)
	}
}

func TestPrewarmFromPersistedFilesIn_LoadsAllZoneFiles(t *testing.T) {
	ClearCache()
	t.Cleanup(ClearCache)

	baseDir := t.TempDir()

	// Zone A: 2 shops
	zoneAInv1 := ShopInventory{Gold: 400, StartingGold: 500, Stock: []StockEntry{{ItemId: 10, Current: 3}}}
	zoneAInv2 := ShopInventory{Gold: 250, StartingGold: 500, Stock: []StockEntry{{ItemId: 20, Current: 7}}}
	// Zone B: 1 shop
	zoneBInv := ShopInventory{Gold: 100, StartingGold: 500, Stock: []StockEntry{{ItemId: 30, Current: 2}}}

	writeZoneShopFile(t, baseDir, "stillwater", 101, 4105, zoneAInv1)
	writeZoneShopFile(t, baseDir, "stillwater", 102, 4106, zoneAInv2)
	writeZoneShopFile(t, baseDir, "sanctum_basin", 201, 1001, zoneBInv)

	// Lookup maps sanitized dir name back to canonical zone string.
	lookup := map[int]string{
		101: "Stillwater",
		102: "Stillwater",
		201: "Sanctum Basin",
	}
	zoneLookup := func(mobId int) (string, bool) {
		z, ok := lookup[mobId]
		return z, ok
	}

	n, err := PrewarmFromPersistedFilesIn(baseDir, zoneLookup)
	if err != nil {
		t.Fatalf("PrewarmFromPersistedFilesIn: %v", err)
	}
	if n != 3 {
		t.Errorf("loaded count: got %d, want 3", n)
	}

	// Verify each shop is in the cache with the canonical zone string.
	cases := []struct {
		zone   string
		mobId  int
		roomId int
		gold   int
	}{
		{"Stillwater", 101, 4105, 400},
		{"Stillwater", 102, 4106, 250},
		{"Sanctum Basin", 201, 1001, 100},
	}
	for _, tc := range cases {
		inv := GetShopInventory(tc.zone, tc.mobId, tc.roomId)
		if inv == nil {
			t.Errorf("zone=%q mob=%d room=%d: got nil, want loaded inventory", tc.zone, tc.mobId, tc.roomId)
			continue
		}
		if inv.Gold != tc.gold {
			t.Errorf("zone=%q mob=%d room=%d: gold=%d, want %d", tc.zone, tc.mobId, tc.roomId, inv.Gold, tc.gold)
		}
		if inv.Zone != tc.zone {
			t.Errorf("zone=%q mob=%d room=%d: inv.Zone=%q, want canonical zone", tc.zone, tc.mobId, tc.roomId, inv.Zone)
		}
	}
}

func TestPrewarmFromPersistedFilesIn_SkipsUnknownMobs(t *testing.T) {
	ClearCache()
	t.Cleanup(ClearCache)

	baseDir := t.TempDir()
	inv := ShopInventory{Gold: 123, StartingGold: 500, Stock: []StockEntry{}}
	writeZoneShopFile(t, baseDir, "testzone", 999, 1, inv)

	// lookup returns false for mob 999 — simulate missing template
	n, err := PrewarmFromPersistedFilesIn(baseDir, func(mobId int) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("loaded count: got %d, want 0 (mob template missing)", n)
	}
}

func TestPrewarmFromPersistedFilesIn_EmptyDir(t *testing.T) {
	ClearCache()
	t.Cleanup(ClearCache)

	baseDir := t.TempDir()
	n, err := PrewarmFromPersistedFilesIn(baseDir, func(mobId int) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("loaded count: got %d, want 0", n)
	}
}

func TestPrewarmFromPersistedFilesIn_NonexistentDir(t *testing.T) {
	ClearCache()
	t.Cleanup(ClearCache)

	n, err := PrewarmFromPersistedFilesIn("/nonexistent/path/to/shops", func(mobId int) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("nonexistent dir should return nil error (treated as no shops): %v", err)
	}
	if n != 0 {
		t.Errorf("loaded count: got %d, want 0", n)
	}
}

func TestPrewarmFromPersistedFilesIn_DoesNotOverwriteExistingCache(t *testing.T) {
	ClearCache()
	t.Cleanup(ClearCache)

	baseDir := t.TempDir()
	diskInv := ShopInventory{Gold: 50, StartingGold: 500, Stock: []StockEntry{}}
	writeZoneShopFile(t, baseDir, "testzone", 55, 10, diskInv)

	// Pre-load a different value into the cache for the same key.
	cacheInv := ShopInventory{Gold: 999, StartingGold: 500, Stock: []StockEntry{}}
	RegisterShop("TestZone", 55, 10, cacheInv)

	n, err := PrewarmFromPersistedFilesIn(baseDir, func(mobId int) (string, bool) {
		return "TestZone", true
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("loaded count: got %d, want 0 (already cached)", n)
	}
	// Cache value should be unchanged.
	got := GetShopInventory("TestZone", 55, 10)
	if got == nil || got.Gold != 999 {
		t.Errorf("cache overwritten: got gold=%v, want 999", got)
	}
}
