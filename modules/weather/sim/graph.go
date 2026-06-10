package sim

import (
	"encoding/json"
	"sort"
	"strings"
)

// GraphVersion is bumped whenever the on-disk cache format changes, so a
// loader can detect and rebuild a stale cache.
const GraphVersion = 1

// ZoneNode is one node in the geography graph: a single GoMud zone, plus the
// metadata the weather simulation needs about it.
type ZoneNode struct {
	Zone       string `json:"zone"`
	Biome      string `json:"biome"`
	Rooms      int    `json:"rooms"`
	HasOutdoor bool   `json:"hasOutdoor"`
}

// Edge is an undirected adjacency between two zones, weighted by the number of
// distinct room-exits that cross the border between them. A and B are stored in
// a canonical order (A <= B) so an edge has one representation.
type Edge struct {
	A      string `json:"a"`
	B      string `json:"b"`
	Weight int    `json:"weight"`
}

// Graph is the zone-adjacency graph: the crawler's output and the weather
// simulation's input. It is pure data and carries no engine types.
type Graph struct {
	Version      int                 `json:"version"`
	BuiltAtRound uint64              `json:"builtAtRound"`
	Nodes        map[string]ZoneNode `json:"nodes"`
	Edges        []Edge              `json:"edges"`
	Components   int                 `json:"components"`

	adj map[string][]Edge // lazy adjacency index; rebuilt after FromJSON (nil there)
}

// ToJSON serializes the graph to the on-disk cache format (indented for
// human inspection).
func (g *Graph) ToJSON() ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}

// FromJSON parses a graph from its cache format. Callers should compare the
// returned Graph's Version against GraphVersion and rebuild the graph if they
// differ, since an older cache may use an incompatible layout.
func FromJSON(b []byte) (*Graph, error) {
	var g Graph
	if err := json.Unmarshal(b, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// Zones returns all zone names in the graph, sorted for deterministic iteration.
func (g *Graph) Zones() []string {
	out := make([]string, 0, len(g.Nodes))
	for z := range g.Nodes {
		out = append(out, z)
	}
	sort.Strings(out)
	return out
}

// Neighbors returns the zones adjacent to z, each as an Edge oriented from z
// (Edge.A == z). The result is a shared slice from a lazily-built index —
// callers MUST NOT mutate it (copy before sorting). Returns nil for unknown
// or isolated zones.
func (g *Graph) Neighbors(z string) []Edge {
	if g.adj == nil {
		g.buildAdjacency()
	}
	return g.adj[z]
}

// FindZone resolves a zone name case-insensitively to its canonical graph key.
// An exact match wins; otherwise the first case-insensitive match is returned.
func (g *Graph) FindZone(name string) (string, bool) {
	if _, ok := g.Nodes[name]; ok {
		return name, true
	}
	for _, z := range g.Zones() {
		if strings.EqualFold(z, name) {
			return z, true
		}
	}
	return "", false
}

// buildAdjacency indexes Edges by zone, both orientations. Called lazily from
// Neighbors; the module runs on GoMud's single game-loop goroutine, so the
// unsynchronized lazy build is safe.
func (g *Graph) buildAdjacency() {
	g.adj = make(map[string][]Edge, len(g.Nodes))
	for _, e := range g.Edges {
		g.adj[e.A] = append(g.adj[e.A], e)
		if e.B != e.A {
			g.adj[e.B] = append(g.adj[e.B], Edge{A: e.B, B: e.A, Weight: e.Weight})
		}
	}
}
