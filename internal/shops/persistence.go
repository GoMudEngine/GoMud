package shops

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// shopCache is the in-memory store of all registered shop inventories.
var (
	shopCacheMu sync.RWMutex
	shopCache   = map[string]*ShopInventory{}
)

// shopKey returns a unique cache key for a shop NPC.
func shopKey(zone string, mobId int, roomId int) string {
	return fmt.Sprintf("%s/%d/room%d", zone, mobId, roomId)
}

// shopPath returns the full filesystem path for a shop's YAML save file.
func shopPath(zone string, mobId int, roomId int) string {
	zoneSanitized := util.ConvertForFilename(zone)
	filename := fmt.Sprintf("%d-room%d.yaml", mobId, roomId)
	return util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `shops`, `/`, zoneSanitized, `/`, filename,
	)
}

// GetShopInventory returns the ShopInventory for the given location. It checks
// the in-memory cache first, then falls back to loading from disk. Returns nil
// if the shop has not been registered and no save file exists.
func GetShopInventory(zone string, mobId int, roomId int) *ShopInventory {
	key := shopKey(zone, mobId, roomId)

	shopCacheMu.RLock()
	if inv, ok := shopCache[key]; ok {
		shopCacheMu.RUnlock()
		return inv
	}
	shopCacheMu.RUnlock()

	// Not cached — try disk.
	inv := loadFromDisk(zone, mobId, roomId)
	if inv == nil {
		return nil
	}

	inv.Zone = zone
	inv.MobId = mobId
	inv.RoomId = roomId

	shopCacheMu.Lock()
	shopCache[key] = inv
	shopCacheMu.Unlock()

	return inv
}

// RegisterShop ensures a ShopInventory exists for the given NPC. If a save
// file is present on disk it is loaded; otherwise the template is used to
// seed a fresh inventory. The returned pointer is always non-nil and cached.
//
// Call RegisterShop once per shop NPC at server startup (or mob spawn).
func RegisterShop(zone string, mobId int, roomId int, template ShopInventory) *ShopInventory {
	key := shopKey(zone, mobId, roomId)

	shopCacheMu.RLock()
	if inv, ok := shopCache[key]; ok {
		shopCacheMu.RUnlock()
		return inv
	}
	shopCacheMu.RUnlock()

	// Try loading persisted state first.
	inv := loadFromDisk(zone, mobId, roomId)
	if inv == nil {
		// Seed from template: set Current to RestockQty for stocked items.
		seeded := template // copy
		seeded.Stock = make([]StockEntry, len(template.Stock))
		copy(seeded.Stock, template.Stock)
		for i := range seeded.Stock {
			if seeded.Stock[i].RestockQty > 0 {
				seeded.Stock[i].Current = seeded.Stock[i].RestockQty
			} else {
				seeded.Stock[i].Current = 0
			}
		}
		inv = &seeded
	}

	inv.Zone = zone
	inv.MobId = mobId
	inv.RoomId = roomId

	shopCacheMu.Lock()
	shopCache[key] = inv
	shopCacheMu.Unlock()

	return inv
}

// SaveShop writes the cached ShopInventory for the given location to disk.
// Returns an error if the shop is not in the cache or the write fails.
func SaveShop(zone string, mobId int, roomId int) error {
	key := shopKey(zone, mobId, roomId)

	shopCacheMu.RLock()
	inv, ok := shopCache[key]
	shopCacheMu.RUnlock()
	if !ok {
		return fmt.Errorf("shops.SaveShop: no cached inventory for %s", key)
	}

	savePath := shopPath(zone, mobId, roomId)

	dir := filepath.Dir(savePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("shops.SaveShop: mkdir %s: %w", dir, err)
	}

	bytes, err := yaml.Marshal(inv)
	if err != nil {
		return fmt.Errorf("shops.SaveShop: marshal: %w", err)
	}

	if err := os.WriteFile(savePath, bytes, 0644); err != nil {
		return fmt.Errorf("shops.SaveShop: write %s: %w", savePath, err)
	}

	return nil
}

// SaveAllShops persists every cached ShopInventory to disk. Intended for
// graceful server shutdown.
func SaveAllShops() {
	shopCacheMu.RLock()
	// Snapshot keys and location fields while holding the read lock.
	type entry struct {
		zone   string
		mobId  int
		roomId int
	}
	entries := make([]entry, 0, len(shopCache))
	for _, inv := range shopCache {
		entries = append(entries, entry{inv.Zone, inv.MobId, inv.RoomId})
	}
	shopCacheMu.RUnlock()

	for _, e := range entries {
		if err := SaveShop(e.zone, e.mobId, e.roomId); err != nil {
			mudlog.Error("shops.SaveAllShops", "error", err)
		}
	}
}

// ClearCache drops all cached shop inventories. Used in tests to ensure
// isolation between test cases.
func ClearCache() {
	shopCacheMu.Lock()
	shopCache = map[string]*ShopInventory{}
	shopCacheMu.Unlock()
}

// loadFromDisk reads a shop's YAML save file and returns the parsed
// ShopInventory. Returns nil if the file does not exist or cannot be parsed.
func loadFromDisk(zone string, mobId int, roomId int) *ShopInventory {
	savePath := shopPath(zone, mobId, roomId)

	bytes, err := os.ReadFile(savePath)
	if err != nil {
		return nil // File doesn't exist = fresh shop
	}

	var inv ShopInventory
	if err := yaml.Unmarshal(bytes, &inv); err != nil {
		mudlog.Error("shops.loadFromDisk", "path", savePath, "error", err)
		return nil
	}

	return &inv
}
