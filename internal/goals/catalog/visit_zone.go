// Package-level notes:
//
// # zoneGraphDistance is a 4.3 heuristic stub
//
// zoneGraphDistance always returns 1 (treat all zones as one hop away)
// in this implementation. The full BFS over zone adjacency — computed
// from inter-zone room exits and cached lazily — is deferred to chunk
// 4.4 where the planner integrates zone-aware pathfinding.
//
// Until then, ContextScore never returns 0.3 (the "4+ hops" branch);
// all out-of-zone goals score 0.8 ("2-3 hops") unless the mob is
// already standing in the target zone (returns 0). The simplification
// is intentional: it gives visit-zone goals a moderate, constant score
// that keeps them visible to the selector without over-weighting them.
// TODO-ADAPT in chunk 4.4: replace with real BFS.

package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("visit-zone", goals.GoalTypeMeta{
		Predicate:     visitZonePredicate,
		ContextScore:  visitZoneContextScore,
		AllowMultiple: true,
		DedupKey:      visitZoneDedupKey,
		Params: []goals.ParamSchema{
			{Key: "target_zone", Required: true, GoType: "string"},
		},
	})
}

func visitZoneDedupKey(g *goals.Goal) string {
	if z, ok := g.Params["target_zone"].(string); ok {
		return z
	}
	return ""
}

// visitZonePredicate: satisfied when mob.VisitedZones contains target_zone.
func visitZonePredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	zone, _ := g.Params["target_zone"].(string)
	if zone == "" {
		return false
	}
	if mob.VisitedZones == nil {
		return false
	}
	return mob.VisitedZones[zone]
}

// visitZoneContextScore scores urgency based on the mob's distance from
// the target zone:
//   - 0.0 if already in target_zone (predicate fires on next tick)
//   - 1.5 if adjacent (1 hop)
//   - 0.8 if 2-3 hops
//   - 0.3 if 4+ hops
//
// In 4.3 zoneGraphDistance always returns 1 (see file-level note), so
// the score is always 1.5 for any out-of-zone goal in this release.
func visitZoneContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	zone, _ := g.Params["target_zone"].(string)
	if zone == "" {
		return 0
	}
	if mob.Character.Zone == zone {
		return 0
	}
	hops := zoneGraphDistance(mob.Character.Zone, zone)
	switch {
	case hops < 0:
		return 0
	case hops == 1:
		return 1.5
	case hops <= 3:
		return 0.8
	}
	return 0.3
}

// ─── TODO-ADAPT helper ──────────────────────────────────────────────────────

// zoneGraphDistance returns the hop count between two zones in the zone
// adjacency graph, or -1 if no path is known. Zone adjacency can be
// computed from inter-zone room exits.
//
// 4.3 heuristic shortcut: same zone returns 0; any other pair returns 1
// (treat all zones as "adjacent" for scoring purposes). The full BFS
// over zone adjacency is deferred to chunk 4.4. See file-level note.
func zoneGraphDistance(from, to string) int {
	if from == to {
		return 0
	}
	return 1
}
