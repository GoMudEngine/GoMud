package relationships

import "sync"

// graph is the in-memory adjacency map: mob template id → outgoing
// edges. Edges are stored on both sides (auto-mirrored at load time
// or via Add) so callers always see a complete picture from
// whichever side they query.
var (
	graph   = make(map[int][]Relation)
	graphMu sync.RWMutex
)
