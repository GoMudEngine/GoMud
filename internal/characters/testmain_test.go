package characters

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// TestMain initializes the mudlog logger once for all tests in this
// package. Several diagnostics (chunk 4b grapple-investigation Warns
// in Die/SetAggro/EndAggro) write to mudlog and panic on a nil
// logger if a test exercises those paths without the engine's normal
// boot init.
func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}
