package engine

import "github.com/GoMudEngine/GoMud/modules/weather/sim"

// CacheIdentifier is the plugin-storage key for the geography graph cache
// (written via plugin.WriteBytes / read via plugin.ReadBytes).
const CacheIdentifier = "geography"

// DecodeCache parses cached graph bytes and reports whether they are usable.
// It returns ok=false (without an error) for absent, empty, unparseable, or
// stale-version data, signaling the caller to rebuild the graph.
func DecodeCache(b []byte) (*sim.Graph, bool) {
	if len(b) == 0 {
		return nil, false
	}
	g, err := sim.FromJSON(b)
	if err != nil {
		return nil, false
	}
	if g.Version != sim.GraphVersion {
		return nil, false
	}
	return g, true
}
