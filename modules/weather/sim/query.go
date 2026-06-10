package sim

import "sort"

// Coverage describes one front's projection onto a queried zone.
type Coverage struct {
	Front     Front
	Effective float64 // projected intensity at the queried zone
	Hops      int     // graph distance from the front's center
}

// Covering returns the fronts whose area projection reaches zone, strongest
// effective intensity first (ties broken by lowest front id). It mirrors
// resolveWeather's coverage rule exactly: projection = Intensity *
// CoverageFalloff^hops within MaxFrontRadius, covered while >= MinProjected.
func Covering(g *Graph, fronts []Front, cfg Config, zone ZoneId) []Coverage {
	var out []Coverage
	for i := range fronts {
		f := fronts[i]
		hops, ok := zonesWithin(g, f.Zone, cfg.MaxFrontRadius)[zone]
		if !ok {
			continue
		}
		eff := f.Intensity * pow(cfg.CoverageFalloff, hops)
		if eff < cfg.MinProjected {
			continue
		}
		out = append(out, Coverage{Front: f, Effective: eff, Hops: hops})
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Effective != out[b].Effective {
			return out[a].Effective > out[b].Effective
		}
		return out[a].Front.Id < out[b].Front.Id
	})
	return out
}
