package relationships

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestMain(m *testing.M) {
	// In-memory only; no temp dir needed (no v1 persistence).
	// Initialize mudlog to suppress nil pointer errors during tests.
	mudlog.SetupLogger(nil, "L", "", false)
	os.Exit(m.Run())
}

// resetGraph wipes the in-memory graph between tests.
func resetGraph() {
	graphMu.Lock()
	graph = make(map[int][]Relation)
	graphMu.Unlock()
}
