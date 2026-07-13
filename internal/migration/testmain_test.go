package migration

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// TestMain initializes the mudlog logger once so migration functions that log
// (via mudlog.Info) don't panic on a nil logger during tests.
func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}
