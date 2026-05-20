package relationships

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// graph is the in-memory adjacency map: mob template id → outgoing
// edges. Edges are stored on both sides (auto-mirrored at load time
// or via Add) so callers always see a complete picture from
// whichever side they query.
var (
	graph   = make(map[int][]Relation)
	graphMu sync.RWMutex
)

// MobEdges is the flattened input for LoadFromMobs: one entry per
// mob template that has authored relationships.
type MobEdges struct {
	MobId int
	Edges []EdgeInput
}

// EdgeInput is the per-edge authoring shape. Mirrors the YAML on
// the mob template (To, Type, Subtype) but pre-typed.
type EdgeInput struct {
	To      int
	Type    Type
	Subtype string
}

// LoadFromMobs populates the in-memory graph from authored mob
// edges. `validateMobId` is a callback the caller provides to
// confirm a mob template id exists (the relationships package
// doesn't import mobs directly to avoid coupling). Validation:
//   - to: == declaring mobId  → warn + skip (self-edge)
//   - to: not validateMobId   → warn + skip (unknown target)
//   - type: not in known enum → warn + skip
//   - duplicate edges (same from/to/type) → warn + first wins
func LoadFromMobs(input []MobEdges, validateMobId func(mobId int) bool) {
	graphMu.Lock()
	defer graphMu.Unlock()

	graph = make(map[int][]Relation)

	knownTypes := map[Type]bool{
		TypeFamily: true, TypeFriend: true, TypeRival: true,
		TypeLover: true, TypeEmployer: true, TypeEmployee: true,
	}

	for _, me := range input {
		for _, ei := range me.Edges {
			if ei.To == me.MobId {
				mudlog.Warn("relationships: self-edge skipped",
					"mobId", me.MobId)
				continue
			}
			if !validateMobId(ei.To) {
				mudlog.Warn("relationships: edge to unknown mob skipped",
					"from", me.MobId, "to", ei.To)
				continue
			}
			if !knownTypes[ei.Type] {
				mudlog.Warn("relationships: unknown type skipped",
					"from", me.MobId, "to", ei.To, "type", ei.Type)
				continue
			}
			// Forward edge with dedup.
			if hasEdge(me.MobId, ei.To, ei.Type) {
				mudlog.Warn("relationships: duplicate edge skipped",
					"from", me.MobId, "to", ei.To, "type", ei.Type)
				continue
			}
			graph[me.MobId] = append(graph[me.MobId],
				Relation{Other: ei.To, Type: ei.Type, Subtype: ei.Subtype})
			// Mirror edge (no subtype on the mirror; per-side).
			mirrorType := InverseType(ei.Type)
			if !hasEdge(ei.To, me.MobId, mirrorType) {
				graph[ei.To] = append(graph[ei.To],
					Relation{Other: me.MobId, Type: mirrorType})
			}
		}
	}
}

// hasEdge checks if from→to with the given type already exists.
// Caller must hold graphMu (write lock).
func hasEdge(from, to int, t Type) bool {
	for _, r := range graph[from] {
		if r.Other == to && r.Type == t {
			return true
		}
	}
	return false
}
