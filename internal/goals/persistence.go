package goals

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v3"
)

var (
	cacheMu     sync.RWMutex
	cache       = map[int]*MobGoals{}
	nameByMobId = map[int]string{}

	// saveMu serializes disk writes to avoid Windows ERROR_SHARING_VIOLATION
	// when two goroutines write the same path. Held only during marshal +
	// write, never across cache mutations.
	saveMu sync.Mutex
)

func goalsBaseDir() string {
	if override := os.Getenv("DOGMUD_GOALS_DIR_OVERRIDE"); override != "" {
		return override
	}
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "goals")
}

func goalPath(mobId int, namesimple string) string {
	return filepath.Join(goalsBaseDir(),
		fmt.Sprintf("%d-%s.yaml", mobId, util.ConvertForFilename(namesimple)))
}

func loadFromDisk(mobId int, namesimple string) *MobGoals {
	path := goalPath(mobId, namesimple)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	mg := &MobGoals{}
	if err := yaml.Unmarshal(data, mg); err != nil {
		mudlog.Warn("goals.loadFromDisk", "path", path, "error", err)
		return nil
	}
	// Stamp OwnerMobId on every loaded goal — the field is unmarshal-
	// skipped (yaml:"-") so we set it here from the parent.
	for _, g := range mg.Goals {
		g.OwnerMobId = mg.MobId
	}
	return mg
}

// saveToDisk writes the cached MobGoals for mobId. Returns an error if
// the cache is missing the entry or the write fails. Atomic via
// .tmp + os.Rename.
func saveToDisk(mobId int, namesimple string) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	cacheMu.RLock()
	mg, ok := cache[mobId]
	if !ok {
		cacheMu.RUnlock()
		return fmt.Errorf("goals.saveToDisk: no cached entry for mobId=%d", mobId)
	}
	out, err := yaml.Marshal(mg)
	cacheMu.RUnlock()
	if err != nil {
		return fmt.Errorf("goals.saveToDisk: marshal: %w", err)
	}

	path := goalPath(mobId, namesimple)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("goals.saveToDisk: mkdir %s: %w", filepath.Dir(path), err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("goals.saveToDisk: write tmp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("goals.saveToDisk: rename %s: %w", path, err)
	}
	return nil
}

// cacheStoreForTest seeds the cache directly. Test-only seam used
// before Add() exists (Task 4).
func cacheStoreForTest(namesimple string, mg *MobGoals) {
	cacheMu.Lock()
	cache[mg.MobId] = mg
	nameByMobId[mg.MobId] = namesimple
	cacheMu.Unlock()
}

// ClearCache drops every cached entry. Tests use this to isolate
// cases; production code should not call it.
func ClearCache() {
	cacheMu.Lock()
	cache = map[int]*MobGoals{}
	nameByMobId = map[int]string{}
	cacheMu.Unlock()
}
