package facts

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
	registry      *Registry
	registryMu    sync.RWMutex
	registrySaveMu sync.Mutex // serializes registry disk writes

	awarenessCache    = make(map[int]*Awareness)
	awarenessCacheMu  sync.RWMutex
	awarenessSaveMu   sync.Mutex // serializes awareness disk writes
)

func registryFilePath() string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "world", "dogmud", "facts.yaml")
}

func awarenessFilePath(mobId int, mobName string) string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles),
		"world", "dogmud", "facts.awareness",
		fmt.Sprintf("%d-%s.yaml", mobId, util.ConvertForFilename(mobName)))
}

// saveRegistry serializes under cache RLock for snapshot consistency,
// writes via tmp-rename for atomicity. Mirrors chunk 1.4 review fix.
func saveRegistry(r *Registry) error {
	registrySaveMu.Lock()
	defer registrySaveMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(registryFilePath()), 0o755); err != nil {
		return fmt.Errorf("mkdir registry dir: %w", err)
	}

	registryMu.RLock()
	out, err := yaml.Marshal(r)
	registryMu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	path := registryFilePath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

func loadRegistryFromDisk() *Registry {
	data, err := os.ReadFile(registryFilePath())
	if err != nil {
		return nil
	}
	r := &Registry{}
	if err := yaml.Unmarshal(data, r); err != nil {
		return nil
	}
	return r
}
