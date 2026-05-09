package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v3"
)

var (
	knowledgeCache   = make(map[int]*ObserverFile)
	knowledgeCacheMu sync.RWMutex
	saveMu           sync.Mutex // serializes disk writes (Windows file-lock safety)
)

func knowledgeBaseDir() string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "world", "dogmud", "knowledge")
}

// observerFilePath returns the absolute path for the given observer mob template id.
// The mobName is used in the filename for human readability; mismatch is
// tolerated (filename is not the lookup key).
func observerFilePath(mobId int, mobName string) string {
	return filepath.Join(knowledgeBaseDir(),
		fmt.Sprintf("%d-%s.yaml", mobId, util.ConvertForFilename(mobName)))
}

func saveObserverFile(fc *ObserverFile) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	if err := os.MkdirAll(knowledgeBaseDir(), 0o755); err != nil {
		return fmt.Errorf("mkdir knowledge dir: %w", err)
	}
	path := observerFilePath(fc.ObserverMobId, fc.ObserverName)

	// Hold the cache read-lock around Marshal so another goroutine mutating
	// fc.Records under knowledgeCacheMu doesn't race the serialization.
	// Mirrors internal/crimes/persistence.go saveCrimesToDisk pattern.
	knowledgeCacheMu.RLock()
	out, err := yaml.Marshal(fc)
	knowledgeCacheMu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal observer file %d: %w", fc.ObserverMobId, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write tmp file %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename tmp -> final %q: %w", path, err)
	}
	return nil
}

func loadObserverFileFromDisk(mobId int, mobName string) *ObserverFile {
	path := observerFilePath(mobId, mobName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	fc := &ObserverFile{}
	if err := yaml.Unmarshal(data, fc); err != nil {
		return nil
	}
	return fc
}

// loadOrLazyInit returns the cached *ObserverFile for the given observer
// mob id, loading from disk on first access. If neither cache nor disk
// has data, an empty ObserverFile is created and cached. Mirrors the
// chunk 1.3 double-check-lock pattern.
func loadOrLazyInit(mobId int, mobName string) *ObserverFile {
	knowledgeCacheMu.RLock()
	if fc, ok := knowledgeCache[mobId]; ok {
		knowledgeCacheMu.RUnlock()
		return fc
	}
	knowledgeCacheMu.RUnlock()

	if fc := loadObserverFileFromDisk(mobId, mobName); fc != nil {
		knowledgeCacheMu.Lock()
		if cached, ok := knowledgeCache[mobId]; ok {
			knowledgeCacheMu.Unlock()
			return cached
		}
		knowledgeCache[mobId] = fc
		knowledgeCacheMu.Unlock()
		return fc
	}

	fc := &ObserverFile{
		ObserverMobId: mobId,
		ObserverName:  mobName,
		Records:       []*Record{},
	}
	knowledgeCacheMu.Lock()
	if cached, ok := knowledgeCache[mobId]; ok {
		knowledgeCacheMu.Unlock()
		return cached
	}
	knowledgeCache[mobId] = fc
	knowledgeCacheMu.Unlock()
	return fc
}
