package goals

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// TestMain points the goals-dir override at a per-process temp dir
// before any test runs, so every test starts with a clean slate and
// no test needs a configs fixture. Mirrors opinions/test_main_test.go.
func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	dir, err := os.MkdirTemp("", "goals-test-*")
	if err != nil {
		panic("goals test: mkdirtemp: " + err.Error())
	}
	os.Setenv("DOGMUD_GOALS_DIR_OVERRIDE", filepath.Join(dir, "goals"))
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
