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
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "facts.yaml")
}

func awarenessFilePath(mobId int, mobName string) string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles),
		"facts.awareness",
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

func saveAwareness(a *Awareness) error {
	awarenessSaveMu.Lock()
	defer awarenessSaveMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(awarenessFilePath(a.ObserverMobId, a.ObserverName)), 0o755); err != nil {
		return fmt.Errorf("mkdir awareness dir: %w", err)
	}

	awarenessCacheMu.RLock()
	out, err := yaml.Marshal(a)
	awarenessCacheMu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal awareness: %w", err)
	}

	path := awarenessFilePath(a.ObserverMobId, a.ObserverName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

func loadAwarenessFromDisk(mobId int, mobName string) *Awareness {
	data, err := os.ReadFile(awarenessFilePath(mobId, mobName))
	if err != nil {
		return nil
	}
	a := &Awareness{}
	if err := yaml.Unmarshal(data, a); err != nil {
		return nil
	}
	return a
}

// loadOrLazyInitAwareness returns the cached *Awareness for the given
// observer mob template id. Mirrors the chunk-1.3/1.4 double-check-
// lock pattern.
func loadOrLazyInitAwareness(mobId int, mobName string) *Awareness {
	awarenessCacheMu.RLock()
	if a, ok := awarenessCache[mobId]; ok {
		awarenessCacheMu.RUnlock()
		return a
	}
	awarenessCacheMu.RUnlock()

	if a := loadAwarenessFromDisk(mobId, mobName); a != nil {
		awarenessCacheMu.Lock()
		if cached, ok := awarenessCache[mobId]; ok {
			awarenessCacheMu.Unlock()
			return cached
		}
		awarenessCache[mobId] = a
		awarenessCacheMu.Unlock()
		return a
	}

	a := &Awareness{ObserverMobId: mobId, ObserverName: mobName}
	awarenessCacheMu.Lock()
	if cached, ok := awarenessCache[mobId]; ok {
		awarenessCacheMu.Unlock()
		return cached
	}
	awarenessCache[mobId] = a
	awarenessCacheMu.Unlock()
	return a
}

// loadOrLazyInitRegistry mirrors the awareness pattern but for the
// single-instance registry.
func loadOrLazyInitRegistry() *Registry {
	registryMu.RLock()
	if registry != nil {
		r := registry
		registryMu.RUnlock()
		return r
	}
	registryMu.RUnlock()

	if r := loadRegistryFromDisk(); r != nil {
		// Normalize hand-edited entries: empty status defaults to active,
		// zero declared_round filled in with current round at first load.
		now := currentRound()
		for _, f := range r.Facts {
			if f.Status == "" {
				f.Status = StatusActive
			}
			if f.DeclaredRound == 0 {
				f.DeclaredRound = now
			}
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

	r := &Registry{Facts: []*Fact{}}
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
