package bounties

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"gopkg.in/yaml.v3"
)

var (
	registry   *Registry
	registryMu sync.RWMutex
	saveMu     sync.Mutex // serializes disk writes (Windows file-lock safety)
)

func registryFilePath() string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "bounties.yaml")
}

// saveRegistry serializes the registry under cache RLock to ensure
// a consistent snapshot, then writes via tmp-rename for atomicity.
// (The chunk-1.4 review caught the missing RLock-around-marshal
// pattern; bake it in v1 here.)
func saveRegistry(r *Registry) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(registryFilePath()), 0o755); err != nil {
		return fmt.Errorf("mkdir bounties dir: %w", err)
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
		return fmt.Errorf("write tmp file %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename tmp -> final %q: %w", path, err)
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

// loadOrLazyInit returns the cached *Registry, loading from disk on
// first access. If neither cache nor disk has data, an empty
// Registry is created and cached. Mirrors the chunk 1.3 / 1.4
// double-check-lock pattern.
func loadOrLazyInit() *Registry {
	registryMu.RLock()
	if registry != nil {
		r := registry
		registryMu.RUnlock()
		return r
	}
	registryMu.RUnlock()

	if r := loadRegistryFromDisk(); r != nil {
		registryMu.Lock()
		if registry != nil {
			cached := registry
			registryMu.Unlock()
			return cached
		}
		registry = r
		registryMu.Unlock()
		return r
	}

	r := &Registry{
		NextId:   1,
		Bounties: []*Bounty{},
	}
	registryMu.Lock()
	if registry != nil {
		cached := registry
		registryMu.Unlock()
		return cached
	}
	registry = r
	registryMu.Unlock()
	return r
}
