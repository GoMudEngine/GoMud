package crawler

import (
	"fmt"
	"path"
	"sort"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// Options controls how the crawler interprets the world.
type Options struct {
	// IncludeSecretExits counts secret/hidden exits as adjacency.
	IncludeSecretExits bool
	// ExcludeZonePatterns lists glob patterns (path.Match syntax) for zones to
	// skip entirely, e.g. "instance_*".
	ExcludeZonePatterns []string
	// BuiltAtRound stamps the resulting graph with the round it was built on.
	BuiltAtRound uint64
}

// DefaultOptions returns the recommended crawler options: secret exits count
// toward adjacency, and transient instance/ephemeral zones are skipped.
func DefaultOptions() Options {
	return Options{
		IncludeSecretExits:  true,
		ExcludeZonePatterns: []string{"instance_*", "ephemeral_*"},
	}
}

// Build walks the world via r and produces a zone-adjacency Graph.
func Build(r WorldReader, opts Options) (*sim.Graph, error) {
	zones, err := includedZones(r, opts)
	if err != nil {
		return nil, err
	}
	roomZone := indexRoomZones(r, zones)
	nodes := buildNodes(r, zones)
	edges := buildEdges(r, zones, roomZone, opts)
	components := countComponents(zones, edges)

	return &sim.Graph{
		Version:      sim.GraphVersion,
		BuiltAtRound: opts.BuiltAtRound,
		Nodes:        nodes,
		Edges:        edges,
		Components:   components,
	}, nil
}

// includedZones returns the set of zones to crawl, skipping any whose name
// matches an ExcludeZonePatterns glob (path.Match syntax). It returns an error
// if any pattern is malformed.
func includedZones(r WorldReader, opts Options) (map[string]bool, error) {
	out := map[string]bool{}
	for _, z := range r.ZoneNames() {
		excluded, err := isExcluded(z, opts.ExcludeZonePatterns)
		if err != nil {
			return nil, err
		}
		if excluded {
			continue
		}
		out[z] = true
	}
	return out, nil
}

// isExcluded reports whether zone matches any of the glob patterns. It returns
// an error if a pattern is malformed (path.Match's ErrBadPattern).
func isExcluded(zone string, patterns []string) (bool, error) {
	for _, p := range patterns {
		ok, err := path.Match(p, zone)
		if err != nil {
			return false, fmt.Errorf("invalid ExcludeZonePatterns entry %q: %w", p, err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// indexRoomZones maps every room id in the included zones to its zone, so an
// exit (which only carries a destination room id) can be resolved to a zone.
func indexRoomZones(r WorldReader, zones map[string]bool) map[int]string {
	idx := map[int]string{}
	for zone := range zones {
		for _, id := range r.RoomIDs(zone) {
			idx[id] = zone
		}
	}
	return idx
}

// buildNodes creates a ZoneNode per included zone, populated with its biome,
// room count, and whether any of its rooms is outdoors.
func buildNodes(r WorldReader, zones map[string]bool) map[string]sim.ZoneNode {
	nodes := map[string]sim.ZoneNode{}
	for zone := range zones {
		ids := r.RoomIDs(zone)
		hasOutdoor := false
		for _, id := range ids {
			if room, ok := r.Room(id); ok && room.Outdoor {
				hasOutdoor = true
				break
			}
		}
		nodes[zone] = sim.ZoneNode{
			Zone:       zone,
			Biome:      r.ZoneBiome(zone),
			Rooms:      len(ids),
			HasOutdoor: hasOutdoor,
		}
	}
	return nodes
}

// buildEdges accumulates undirected, weighted adjacency from every cross-zone
// exit. Intra-zone exits and exits whose target resolves to no included zone
// are skipped.
func buildEdges(r WorldReader, zones map[string]bool, roomZone map[int]string, opts Options) []sim.Edge {
	weights := map[[2]string]int{}
	for zone := range zones {
		for _, id := range r.RoomIDs(zone) {
			room, ok := r.Room(id)
			if !ok {
				continue
			}
			for _, ex := range room.Exits {
				if ex.Secret && !opts.IncludeSecretExits {
					continue
				}
				dstZone, known := roomZone[ex.ToRoom]
				if !known || dstZone == zone {
					continue
				}
				weights[canonicalPair(zone, dstZone)]++
			}
		}
	}

	edges := make([]sim.Edge, 0, len(weights))
	for k, w := range weights {
		edges = append(edges, sim.Edge{A: k[0], B: k[1], Weight: w})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].A != edges[j].A {
			return edges[i].A < edges[j].A
		}
		return edges[i].B < edges[j].B
	})
	return edges
}

// canonicalPair orders two zone names so an edge has a single representation.
func canonicalPair(a, b string) [2]string {
	if a <= b {
		return [2]string{a, b}
	}
	return [2]string{b, a}
}

// countComponents returns the number of connected components in the zone graph
// using union-find. Every included zone is its own component until edges merge
// them, so isolated zones each count once.
//
// Precondition: every zone named in edges must also be present in zones; the
// crawler guarantees this because edges are only created between included zones.
func countComponents(zones map[string]bool, edges []sim.Edge) int {
	parent := make(map[string]string, len(zones))
	for z := range zones {
		parent[z] = z
	}

	var find func(string) string
	find = func(x string) string {
		for parent[x] != x {
			parent[x] = parent[parent[x]] // path halving
			x = parent[x]
		}
		return x
	}

	for _, e := range edges {
		ra, rb := find(e.A), find(e.B)
		if ra != rb {
			parent[ra] = rb
		}
	}

	roots := map[string]bool{}
	for z := range zones {
		roots[find(z)] = true
	}
	return len(roots)
}
