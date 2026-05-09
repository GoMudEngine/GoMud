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

	out, err := yaml.Marshal(fc)
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
