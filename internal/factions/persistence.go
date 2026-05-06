package factions

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
	repCacheMu sync.RWMutex
	repCache   = map[string]*FactionRep{}

	// saveMu serializes file I/O so concurrent BumpRep on the same
	// faction don't trigger Windows ERROR_SHARING_VIOLATION on
	// overlapping os.WriteFile calls. Mirrors internal/opinions.
	saveMu sync.Mutex
)

func repBaseDir() string {
	if override := os.Getenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE"); override != "" {
		return override
	}
	return util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `factions.rep`,
	)
}

func repPath(factionId string) string {
	return filepath.Join(repBaseDir(), factionId+".yaml")
}

func loadRepFromDisk(factionId string) *FactionRep {
	path := repPath(factionId)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var fr FactionRep
	if err := yaml.Unmarshal(bytes, &fr); err != nil {
		mudlog.Error("factions.loadRepFromDisk", "path", path, "error", err)
		return nil
	}
	if fr.Players == nil {
		fr.Players = map[int]*RepEntry{}
	}
	return &fr
}

// saveRepToDisk persists the cached FactionRep for factionId. File
// I/O is serialized via saveMu so concurrent callers don't race
// (Windows ERROR_SHARING_VIOLATION). Re-acquires the cache RLock
// inside the saveMu critical section so the marshal sees a
// consistent snapshot of any Bumps that completed after the
// caller released the cache lock.
func saveRepToDisk(factionId string) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	repCacheMu.RLock()
	fr, ok := repCache[factionId]
	if !ok {
		repCacheMu.RUnlock()
		return fmt.Errorf("factions.saveRepToDisk: no cached entry for %q", factionId)
	}
	bytes, err := yaml.Marshal(fr)
	repCacheMu.RUnlock()
	if err != nil {
		return fmt.Errorf("factions.saveRepToDisk: marshal: %w", err)
	}

	path := repPath(factionId)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("factions.saveRepToDisk: mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("factions.saveRepToDisk: write %s: %w", path, err)
	}
	return nil
}

// SaveAllRep flushes every cached FactionRep to disk. Defined for
// parity with opinions.SaveAllOpinions; not currently wired into
// shutdown (synchronous-on-mutation save covers it).
func SaveAllRep() {
	repCacheMu.RLock()
	ids := make([]string, 0, len(repCache))
	for id := range repCache {
		ids = append(ids, id)
	}
	repCacheMu.RUnlock()
	for _, id := range ids {
		if err := saveRepToDisk(id); err != nil {
			mudlog.Error("factions.SaveAllRep", "error", err)
		}
	}
}

// ClearCache wipes the rep cache. Tests use this for isolation.
func ClearCache() {
	repCacheMu.Lock()
	repCache = map[string]*FactionRep{}
	repCacheMu.Unlock()
}

// Test-only seams.

func clearRepCacheForTest() { ClearCache() }
func repCacheStoreForTest(fr *FactionRep) {
	repCacheMu.Lock()
	repCache[fr.FactionId] = fr
	repCacheMu.Unlock()
}
