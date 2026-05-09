package relationships

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// In-memory only; no temp dir needed (no v1 persistence).
	os.Exit(m.Run())
}

// resetGraph wipes the in-memory graph between tests.
func resetGraph() {
	graphMu.Lock()
	graph = make(map[int][]Relation)
	graphMu.Unlock()
}
