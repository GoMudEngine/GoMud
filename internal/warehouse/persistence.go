package warehouse

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// warehouseBaseDir overrides the on-disk base directory in tests (via
// SetBaseDirForTest), mirroring caravan.throughputBaseDir.
var warehouseBaseDir string

// warehousePath returns the full filesystem path for a city's warehouse
// YAML save file.
func warehousePath(zone string) string {
	base := warehouseBaseDir
	if base == "" {
		base = util.FilePath(
			configs.GetFilePathsConfig().DataFiles.String(), `/`, `warehouses`,
		)
	}
	return filepath.Join(base, util.ConvertForFilename(zone)+".yaml")
}

// SetBaseDirForTest overrides the on-disk base directory (for use in
// tests with t.TempDir()). Pass "" to restore production behavior.
func SetBaseDirForTest(dir string) {
	warehouseBaseDir = dir
}

// saveOne writes the given city's warehouse to disk. Takes a snapshot of
// the in-memory pool under the package mutex before marshaling.
func saveOne(zone string) error {
	mu.Lock()
	w, ok := warehouses[zone]
	var snapshot Warehouse
	if ok {
		snapshot = *w
		snapshot.Stock = append([]Entry(nil), w.Stock...)
	}
	mu.Unlock()

	if !ok {
		snapshot = Warehouse{Zone: zone}
	}

	path := warehousePath(zone)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("warehouse.saveOne: mkdir: %w", err)
	}
	data, err := yaml.Marshal(&snapshot)
	if err != nil {
		return fmt.Errorf("warehouse.saveOne: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("warehouse.saveOne: write %s: %w", path, err)
	}

	mu.Lock()
	delete(dirty, zone)
	mu.Unlock()

	return nil
}

// SaveAll persists every registered city's warehouse to disk, regardless
// of dirty state. Intended for graceful server shutdown.
func SaveAll() {
	for zone := range cities {
		if err := saveOne(zone); err != nil {
			mudlog.Error("warehouse.SaveAll", "zone", zone, "error", err)
		}
	}
}

// SaveDirty persists only the cities whose in-memory pool has changed
// since the last save. Intended for the per-round tick.
func SaveDirty() {
	mu.Lock()
	toSave := make([]string, 0, len(dirty))
	for zone, isDirty := range dirty {
		if isDirty {
			toSave = append(toSave, zone)
		}
	}
	mu.Unlock()

	for _, zone := range toSave {
		if err := saveOne(zone); err != nil {
			mudlog.Error("warehouse.SaveDirty", "zone", zone, "error", err)
		}
	}
}

// LoadAll reads every registered city's warehouse YAML from disk, if
// present, into the in-memory cache. Missing files leave that city at
// its zero-value warehouse. Call once at boot, after ferry.LoadDataFiles().
func LoadAll() {
	loadedCount := 0
	for zone := range cities {
		path := warehousePath(zone)
		data, err := os.ReadFile(path)
		if err != nil {
			continue // no save file yet — fresh zero-value warehouse
		}
		var w Warehouse
		if err := yaml.Unmarshal(data, &w); err != nil {
			mudlog.Error("warehouse.LoadAll", "zone", zone, "path", path, "error", err)
			continue
		}
		w.Zone = zone

		mu.Lock()
		warehouses[zone] = &w
		delete(dirty, zone)
		mu.Unlock()

		loadedCount++
	}
	mudlog.Info("warehouse.LoadAll()", "loadedCount", loadedCount)
}
