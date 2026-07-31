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
			//
			// An existing edge here is usually the MIRROR created when the
			// other side declared this relationship: mirrors are written
			// without a subtype (subtypes are per-side), so if this explicit
			// declaration carries one, it fills the gap rather than being
			// discarded. Without this, a pair that authors the relationship
			// from both sides — which is how the friendships in the dogmud
			// world are written — silently lost one side's subtype, and the
			// conversation layer fell back to generic exchanges in that
			// direction.
			//
			// A genuine duplicate (the same side declaring the same edge
			// twice) still warns and still lets the first declaration win.
			if hasEdge(me.MobId, ei.To, ei.Type) {
				if ei.Subtype != "" && fillEdgeSubtype(me.MobId, ei.To, ei.Type, ei.Subtype) {
					continue
				}
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

// fillEdgeSubtype sets the subtype on an existing from→to edge of the given
// type, but only when that edge currently has none. It reports whether it
// filled one.
//
// This exists for the mirror case: mirrors are created without a subtype, so an
// explicit declaration from the other side should be allowed to supply one.
// It deliberately will NOT overwrite a subtype that is already set, so a
// genuine duplicate still lets the first declaration win.
//
// Caller must hold graphMu (write lock).
func fillEdgeSubtype(from, to int, t Type, subtype string) bool {
	for i, r := range graph[from] {
		if r.Other == to && r.Type == t {
			if r.Subtype != "" {
				return false
			}
			graph[from][i].Subtype = subtype
			return true
		}
	}
	return false
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
