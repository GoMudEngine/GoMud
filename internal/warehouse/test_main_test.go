package warehouse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)

	// Redirect FilePaths.DataFiles to an OS-temp subdir so shops.SaveShop
	// calls triggered by ReleaseToVendorsInRoom tests write to a throwaway
	// location instead of polluting the repo (mirrors caravan/visit_test.go's
	// TestMain).
	tmpRoot := filepath.Join(os.TempDir(), "gomud-warehouse-test-datafiles")
	_ = os.RemoveAll(tmpRoot)
	_ = os.MkdirAll(tmpRoot, 0755)
	if err := configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles": tmpRoot,
	}); err != nil {
		panic(err)
	}

	code := m.Run()
	_ = os.RemoveAll(tmpRoot)
	os.Exit(code)
}
