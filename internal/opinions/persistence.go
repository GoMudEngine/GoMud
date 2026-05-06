package opinions

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

// opinionCache is the in-memory store of all loaded MobOpinions
// keyed by mobId.
var (
	opinionCacheMu sync.RWMutex
	opinionCache   = map[int]*MobOpinions{}
	// nameByMobId remembers the namesimple used at file write time,
	// so we can reconstruct the path when persisting again without
	// re-reading the mob template.
	nameByMobId = map[int]string{}
)

// opinionsBaseDir returns the directory that holds opinion files.
// Honors the DOGMUD_OPINIONS_DIR_OVERRIDE env var so tests can
// redirect to a temp dir without a real DataFiles config.
func opinionsBaseDir() string {
	if override := os.Getenv("DOGMUD_OPINIONS_DIR_OVERRIDE"); override != "" {
		return override
	}
	return util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `world`, `/`, `dogmud`, `/`, `opinions`,
	)
}

// opinionPath returns the absolute path to a mob's opinion file.
func opinionPath(mobId int, namesimple string) string {
	filename := fmt.Sprintf("%d-%s.yaml", mobId, namesimple)
	return filepath.Join(opinionsBaseDir(), filename)
}

// loadFromDisk reads the YAML file for mobId and returns the parsed
// MobOpinions. Returns nil if the file is missing or malformed.
func loadFromDisk(mobId int, namesimple string) *MobOpinions {
	path := opinionPath(mobId, namesimple)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil // missing file — fresh slate
	}
	var mo MobOpinions
	if err := yaml.Unmarshal(bytes, &mo); err != nil {
		mudlog.Error("opinions.loadFromDisk", "path", path, "error", err)
		return nil
	}
	if mo.Opinions == nil {
		mo.Opinions = map[int]*Opinion{}
	}
	return &mo
}

// saveToDisk persists the cached MobOpinions for mobId to the
// configured opinions directory. Returns an error if the cache is
// missing the entry or the write fails.
func saveToDisk(mobId int, namesimple string) error {
	opinionCacheMu.RLock()
	mo, ok := opinionCache[mobId]
	opinionCacheMu.RUnlock()
	if !ok {
		return fmt.Errorf("opinions.saveToDisk: no cached entry for mobId=%d", mobId)
	}

	path := opinionPath(mobId, namesimple)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("opinions.saveToDisk: mkdir %s: %w", filepath.Dir(path), err)
	}
	bytes, err := yaml.Marshal(mo)
	if err != nil {
		return fmt.Errorf("opinions.saveToDisk: marshal: %w", err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("opinions.saveToDisk: write %s: %w", path, err)
	}
	return nil
}

// SaveAllOpinions writes every cached MobOpinions to disk. Defined
// for parity with shops.SaveAllShops; not currently wired into
// shutdown (synchronous-on-mutation save covers it). Useful for
// admin commands and future graceful-shutdown work.
func SaveAllOpinions() {
	opinionCacheMu.RLock()
	type entry struct {
		mobId int
		name  string
	}
	entries := make([]entry, 0, len(opinionCache))
	for mobId := range opinionCache {
		name := nameByMobId[mobId]
		entries = append(entries, entry{mobId, name})
	}
	opinionCacheMu.RUnlock()
	for _, e := range entries {
		if err := saveToDisk(e.mobId, e.name); err != nil {
			mudlog.Error("opinions.SaveAllOpinions", "error", err)
		}
	}
}

// ClearCache drops every cached MobOpinions. Tests use this to
// isolate cases; production code should not call it.
func ClearCache() {
	opinionCacheMu.Lock()
	opinionCache = map[int]*MobOpinions{}
	nameByMobId = map[int]string{}
	opinionCacheMu.Unlock()
}

// cacheStoreForTest seeds the cache directly. Test-only seam used
// by persistence_test.go before Get/Bump/Set exist.
func cacheStoreForTest(namesimple string, mo *MobOpinions) {
	opinionCacheMu.Lock()
	opinionCache[mo.MobId] = mo
	nameByMobId[mo.MobId] = namesimple
	opinionCacheMu.Unlock()
}
