package questengine

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestMain(m *testing.M) {
	// Initialize the slog logger so logging calls don't nil-panic during tests.
	mudlog.SetupLogger(nil, "debug", "", false)
	os.Exit(m.Run())
}
