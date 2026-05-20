package crimes

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

var (
	crimeCacheMu sync.RWMutex
	crimeCache   = map[string]*FactionCrimes{}

	// saveMu serializes file I/O so concurrent Record calls on the
	// same faction don't trigger Windows ERROR_SHARING_VIOLATION.
	// Mirrors internal/factions and internal/opinions patterns.
	saveMu sync.Mutex
)

// crimesBaseDir returns the directory holding faction crime logs.
// Honors DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE for tests.
func crimesBaseDir() string {
	if override := os.Getenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE"); override != "" {
		return override
	}
	return util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `factions.crimes`,
	)
}

func crimePath(factionId string) string {
	return filepath.Join(crimesBaseDir(), factionId+".yaml")
}

// loadCrimesFromDisk reads the YAML for factionId. Returns nil on
// missing file or parse error. Computes nextId from max existing
// id + 1 so subsequent Records don't collide.
func loadCrimesFromDisk(factionId string) *FactionCrimes {
	path := crimePath(factionId)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var fc FactionCrimes
	if err := yaml.Unmarshal(bytes, &fc); err != nil {
		mudlog.Error("crimes.loadCrimesFromDisk", "path", path, "error", err)
		return nil
	}
	if fc.Crimes == nil {
		fc.Crimes = []*Crime{}
	}
	fc.nextId = computeNextId(fc.Crimes)
	return &fc
}

// computeNextId returns max(crimes[].id) + 1, or 1 if empty.
func computeNextId(crimes []*Crime) int {
	max := 0
	for _, c := range crimes {
		if c.Id > max {
			max = c.Id
		}
	}
	return max + 1
}

// saveCrimesToDisk persists the cached FactionCrimes for factionId.
// Serialized via saveMu. Re-acquires the cache RLock inside the
// critical section so the marshal sees a consistent snapshot.
func saveCrimesToDisk(factionId string) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	crimeCacheMu.RLock()
	fc, ok := crimeCache[factionId]
	if !ok {
		crimeCacheMu.RUnlock()
		return fmt.Errorf("crimes.saveCrimesToDisk: no cached entry for %q", factionId)
	}
	bytes, err := yaml.Marshal(fc)
	crimeCacheMu.RUnlock()
	if err != nil {
		return fmt.Errorf("crimes.saveCrimesToDisk: marshal: %w", err)
	}

	path := crimePath(factionId)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("crimes.saveCrimesToDisk: mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("crimes.saveCrimesToDisk: write %s: %w", path, err)
	}
	return nil
}

// SaveAllCrimes flushes every cached FactionCrimes to disk. Defined
// for parity with factions.SaveAllRep; not currently wired into
// shutdown.
func SaveAllCrimes() {
	crimeCacheMu.RLock()
	ids := make([]string, 0, len(crimeCache))
	for id := range crimeCache {
		ids = append(ids, id)
	}
	crimeCacheMu.RUnlock()
	for _, id := range ids {
		if err := saveCrimesToDisk(id); err != nil {
			mudlog.Error("crimes.SaveAllCrimes", "error", err)
		}
	}
}

// ClearCache wipes the crime cache. Test-only helper.
func ClearCache() {
	crimeCacheMu.Lock()
	crimeCache = map[string]*FactionCrimes{}
	crimeCacheMu.Unlock()
}

// Test-only seams.
func clearCrimeCacheForTest() { ClearCache() }
func crimeCacheStoreForTest(fc *FactionCrimes) {
	crimeCacheMu.Lock()
	crimeCache[fc.FactionId] = fc
	crimeCacheMu.Unlock()
}
