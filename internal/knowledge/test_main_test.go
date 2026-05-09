package knowledge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)

	// Use a temp dir so tests don't pollute real data files.
	tmp, err := os.MkdirTemp("", "knowledge-test-*")
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

	// Create the knowledge directory structure
	knowledgeDir := filepath.Join(tmp, "world", "dogmud", "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// resetCache wipes the in-memory cache between tests so each test starts clean.
func resetCache() {
	knowledgeCacheMu.Lock()
	knowledgeCache = make(map[int]*ObserverFile)
	knowledgeCacheMu.Unlock()
}
