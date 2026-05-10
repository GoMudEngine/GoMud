package facts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestMain(m *testing.M) {
	// Use a temp dir so tests don't pollute real data files.
	tmp, err := os.MkdirTemp("", "facts-test-*")
	if err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	// Store the previous config to restore it after tests
	prev := configs.GetFilePathsConfig()
	configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles": tmp,
	})
	defer func() {
		configs.AddOverlayOverrides(map[string]any{
			"FilePaths.DataFiles": prev.DataFiles.String(),
		})
	}()

	// Create the facts directory structure
	factsDir := filepath.Join(tmp, "world", "dogmud", "facts.awareness")
	if err := os.MkdirAll(factsDir, 0o755); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// resetCaches wipes the in-memory caches between tests so each test starts clean.
func resetCaches() {
	registryMu.Lock()
	registry = nil
	registryMu.Unlock()

	awarenessCacheMu.Lock()
	awarenessCache = make(map[int]*Awareness)
	awarenessCacheMu.Unlock()
}
