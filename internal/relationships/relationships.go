package relationships

import "sort"

// RelationsOf returns all outgoing edges for a mob template.
func RelationsOf(mobId int) []Relation {
	graphMu.RLock()
	defer graphMu.RUnlock()
	out := make([]Relation, len(graph[mobId]))
	copy(out, graph[mobId])
	return out
}

// RelationsOfType filters by type. Empty types slice → all types.
func RelationsOfType(mobId int, types ...Type) []Relation {
	graphMu.RLock()
	defer graphMu.RUnlock()
	if len(types) == 0 {
		out := make([]Relation, len(graph[mobId]))
		copy(out, graph[mobId])
		return out
	}
	want := make(map[Type]bool, len(types))
	for _, t := range types {
		want[t] = true
	}
	out := make([]Relation, 0)
	for _, r := range graph[mobId] {
		if want[r.Type] {
			out = append(out, r)
		}
	}
	return out
}

// KinOf returns family + lover edges.
func KinOf(mobId int) []Relation {
	return RelationsOfType(mobId, TypeFamily, TypeLover)
}

// AlliesOf returns family + friend + lover edges.
func AlliesOf(mobId int) []Relation {
	return RelationsOfType(mobId, TypeFamily, TypeFriend, TypeLover)
}

// RivalsOf returns rival edges.
func RivalsOf(mobId int) []Relation {
	return RelationsOfType(mobId, TypeRival)
}

// RelationsBetween returns edges from a to b (a's view of the
// relationship). Empty if not connected.
func RelationsBetween(a, b int) []Relation {
	graphMu.RLock()
	defer graphMu.RUnlock()
	out := make([]Relation, 0)
	for _, r := range graph[a] {
		if r.Other == b {
			out = append(out, r)
		}
	}
	return out
}

// AreRelated returns true if any edge connects a to b.
func AreRelated(a, b int) bool {
	graphMu.RLock()
	defer graphMu.RUnlock()
	for _, r := range graph[a] {
		if r.Other == b {
			return true
		}
	}
	return false
}

// EmployerOf returns mob ids this NPC employs.
func EmployerOf(mobId int) []int {
	rels := RelationsOfType(mobId, TypeEmployer)
	out := make([]int, len(rels))
	for i, r := range rels {
		out[i] = r.Other
	}
	return out
}

// EmployedBy returns mob ids that employ this NPC.
func EmployedBy(mobId int) []int {
	rels := RelationsOfType(mobId, TypeEmployee)
	out := make([]int, len(rels))
	for i, r := range rels {
		out[i] = r.Other
	}
	return out
}

// OwnedRelation pairs an edge with the mob that owns it (used by
// AllRelations for debug/admin display).
type OwnedRelation struct {
	Owner int
	Relation
}

// AllRelations returns a flat snapshot of every edge in the graph.
// Each forward + mirror edge is represented separately. Sorted by
// owner mob id, then by Other.
func AllRelations() []OwnedRelation {
	graphMu.RLock()
	defer graphMu.RUnlock()
	out := make([]OwnedRelation, 0)
	for owner, edges := range graph {
		for _, r := range edges {
			out = append(out, OwnedRelation{Owner: owner, Relation: r})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Owner != out[j].Owner {
			return out[i].Owner < out[j].Owner
		}
		return out[i].Other < out[j].Other
	})
	return out
}

// Add appends an edge with auto-mirror. No-op if identical edge
// already exists. In-memory only — no persistence v1.
func Add(a, b int, t Type, subtype string) {
	graphMu.Lock()
	defer graphMu.Unlock()
	if hasEdge(a, b, t) {
		return
	}
	graph[a] = append(graph[a], Relation{Other: b, Type: t, Subtype: subtype})
	mirrorType := InverseType(t)
	if !hasEdge(b, a, mirrorType) {
		graph[b] = append(graph[b], Relation{Other: a, Type: mirrorType})
	}
}

// Remove drops an edge and its mirror.
func Remove(a, b int, t Type) {
	graphMu.Lock()
	defer graphMu.Unlock()
	graph[a] = removeEdge(graph[a], b, t)
	graph[b] = removeEdge(graph[b], a, InverseType(t))
}

// removeEdge filters out matching (Other, Type) entries. Caller
// must hold graphMu (write lock).
func removeEdge(edges []Relation, other int, t Type) []Relation {
	out := edges[:0]
	for _, r := range edges {
		if r.Other == other && r.Type == t {
			continue
		}
		out = append(out, r)
	}
	return out
}

// ChangeType atomically removes (a→b oldType) and adds (a→b newType
// + subtype). Mirror also flips.
func ChangeType(a, b int, oldType, newType Type, newSubtype string) {
	graphMu.Lock()
	defer graphMu.Unlock()
	graph[a] = removeEdge(graph[a], b, oldType)
	graph[b] = removeEdge(graph[b], a, InverseType(oldType))
	graph[a] = append(graph[a], Relation{Other: b, Type: newType, Subtype: newSubtype})
	mirrorType := InverseType(newType)
	if !hasEdge(b, a, mirrorType) {
		graph[b] = append(graph[b], Relation{Other: a, Type: mirrorType})
	}
}
