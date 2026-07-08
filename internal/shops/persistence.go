package shops

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
		if inv.CraftSupport == "" && template.CraftSupport != "" {
			inv.CraftSupport = template.CraftSupport
			if err := SaveShop(zone, mobId, roomId); err != nil {
				mudlog.Warn("RegisterShop CraftSupport migration save", "key", key, "error", err)
			}
		}
		return inv
	}
	shopCacheMu.RUnlock()

	// Try loading persisted state first.
	inv := loadFromDisk(zone, mobId, roomId)
	needsCraftMigration := false
	if inv == nil {
		// Seed from template at the ABUNDANCE level, not the restock-batch
		// level. A newly opened merchant is fully provisioned: floor-priced
		// (0.25x) abundant stock is the long-run steady state of every
		// established shop (restock ticks accumulate toward MaxStock), so a
		// fresh shop should start there too.
		//
		// The old behaviour seeded Current = RestockQty, which left the
		// scarcity ratio (Current/RestockQty) at exactly 1.0 — the curve's
		// ~2.36x price. That made every fresh shop open at ~2.36x list prices
		// AND pay ~1.18x value when players sold to it (the fresh-shop
		// arbitrage). Seeding at abundance skips that misleading transient.
		//
		// Current = min(MaxStock, ceil(RestockQty × AbundanceThreshold)) for
		// stocked entries; 0 for crafted entries (RestockQty == 0), which the
		// NPC produces over time.
		cfg := PricingConfigFromBalance()
		seeded := template // copy
		seeded.Stock = make([]StockEntry, len(template.Stock))
		copy(seeded.Stock, template.Stock)
		for i := range seeded.Stock {
			rq := seeded.Stock[i].RestockQty
			if rq <= 0 {
				seeded.Stock[i].Current = 0
				continue
			}
			abundant := int(math.Ceil(float64(rq) * cfg.AbundanceThreshold))
			if ms := seeded.Stock[i].MaxStock; ms > 0 && abundant > ms {
				abundant = ms
			}
			seeded.Stock[i].Current = abundant
		}
		inv = &seeded
	} else if inv.CraftSupport == "" && template.CraftSupport != "" {
		// Flag for migration after cache insert so SaveShop can find the entry.
		needsCraftMigration = true
	}

	inv.Zone = zone
	inv.MobId = mobId
	inv.RoomId = roomId

	shopCacheMu.Lock()
	shopCache[key] = inv
	shopCacheMu.Unlock()

	if needsCraftMigration {
		inv.CraftSupport = template.CraftSupport
		if err := SaveShop(zone, mobId, roomId); err != nil {
			mudlog.Warn("RegisterShop CraftSupport migration save", "key", key, "error", err)
		}
	}

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
// RemoveShopFile deletes a shop's persisted YAML (if present). Used for test
// isolation and admin resets; a missing file is not an error.
func RemoveShopFile(zone string, mobId int, roomId int) error {
	err := os.Remove(shopPath(zone, mobId, roomId))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

func ClearCache() {
	shopCacheMu.Lock()
	shopCache = map[string]*ShopInventory{}
	shopCacheMu.Unlock()
}

// AllShops returns a snapshot of every registered ShopInventory in
// the cache. The returned slice contains pointers to the cached
// inventories — callers must not mutate them. Used by the
// economy/health dashboard for hourly capture.
func AllShops() []*ShopInventory {
	shopCacheMu.RLock()
	defer shopCacheMu.RUnlock()
	out := make([]*ShopInventory, 0, len(shopCache))
	for _, inv := range shopCache {
		out = append(out, inv)
	}
	return out
}

// shopFileRe matches filenames of the form "{mobid}-room{roomid}.yaml" and
// captures the two integer parts.
var shopFileRe = regexp.MustCompile(`^(\d+)-room(\d+)\.yaml$`)

// PrewarmFromPersistedFilesIn loads every persisted shop YAML found under
// baseDir/{zone}/*.yaml into the shop cache. It calls zoneLookup(mobId) to
// resolve the canonical (un-sanitized) zone string — the YAML files do not
// store Zone because that field is tagged yaml:"-". Entries whose mobId is
// not in zoneLookup are skipped (logged as a warning).
//
// This variant is the testable entrypoint; production code calls
// PrewarmFromPersistedFiles which binds the real paths and lookup.
func PrewarmFromPersistedFilesIn(baseDir string, zoneLookup func(mobId int) (string, bool)) (int, error) {
	zoneDirs, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // no shops persisted yet — not an error
		}
		return 0, fmt.Errorf("shops.PrewarmFromPersistedFilesIn: readdir %q: %w", baseDir, err)
	}

	loaded := 0
	for _, zd := range zoneDirs {
		if !zd.IsDir() {
			continue
		}
		zoneDir := filepath.Join(baseDir, zd.Name())
		files, err := os.ReadDir(zoneDir)
		if err != nil {
			mudlog.Warn("shops.PrewarmFromPersistedFilesIn: readdir zone", "zone", zd.Name(), "error", err)
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			m := shopFileRe.FindStringSubmatch(f.Name())
			if m == nil {
				continue
			}
			mobId, _ := strconv.Atoi(m[1])
			roomId, _ := strconv.Atoi(m[2])

			zone, ok := zoneLookup(mobId)
			if !ok {
				mudlog.Warn("shops.PrewarmFromPersistedFilesIn: no mob template for shop file",
					"file", f.Name(), "mobId", mobId)
				continue
			}

			key := shopKey(zone, mobId, roomId)
			shopCacheMu.RLock()
			_, already := shopCache[key]
			shopCacheMu.RUnlock()
			if already {
				continue
			}

			// Read directly from baseDir — avoids the configs dependency
			// in loadFromDisk (which uses the production shopPath).
			filePath := filepath.Join(zoneDir, f.Name())
			raw, readErr := os.ReadFile(filePath)
			if readErr != nil {
				mudlog.Warn("shops.PrewarmFromPersistedFilesIn: read file", "path", filePath, "error", readErr)
				continue
			}
			var inv ShopInventory
			if unmarshalErr := yaml.Unmarshal(raw, &inv); unmarshalErr != nil {
				mudlog.Warn("shops.PrewarmFromPersistedFilesIn: unmarshal", "path", filePath, "error", unmarshalErr)
				continue
			}
			inv.Zone = zone
			inv.MobId = mobId
			inv.RoomId = roomId

			shopCacheMu.Lock()
			shopCache[key] = &inv
			shopCacheMu.Unlock()
			loaded++
		}
	}
	return loaded, nil
}

// PrewarmFromPersistedFiles is the production wrapper for
// PrewarmFromPersistedFilesIn. It resolves the data-files path and builds a
// mob-template zone lookup from mobs.AllMobTemplates(). Call after
// mobs.LoadDataFiles().
func PrewarmFromPersistedFiles(mobZoneLookup func(mobId int) (string, bool)) (int, error) {
	baseDir := util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `shops`,
	)
	return PrewarmFromPersistedFilesIn(baseDir, mobZoneLookup)
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
