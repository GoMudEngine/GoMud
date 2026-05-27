package catalog

import (
	"strconv"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

// befriendDefaultThreshold is the opinion score a mob must hold for the
// target before the befriend goal is considered satisfied.
const befriendDefaultThreshold = 60

func init() {
	goals.RegisterGoalType("befriend", goals.GoalTypeMeta{
		Predicate:     befriendPredicate,
		ContextScore:  befriendContextScore,
		AllowMultiple: true,
		DedupKey:      befriendDedupKey,
		Params: []goals.ParamSchema{
			{Key: "target_kind", Required: true, GoType: "string"},
			{Key: "target_id", Required: true, GoType: "int"},
			{Key: "opinion_threshold", Required: false, GoType: "int"},
		},
	})
}

func befriendDedupKey(g *goals.Goal) string {
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return ""
	}
	return kind + ":" + strconv.Itoa(id)
}

// befriendPredicate: satisfied when mob's opinion of the target reaches
// the configured threshold. Only mob→player opinions are tracked (chunk
// 1.1 opinions package); mob→mob opinions are explicitly deferred to
// chunk 3.6. Passing target_kind="mob" will never satisfy this predicate.
func befriendPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	thr := paramIntOr(g, "opinion_threshold", befriendDefaultThreshold)
	if kind == "" || id == 0 {
		return false
	}
	return mobOpinionOf(mob, kind, id) >= thr
}

// befriendContextScore tiers by target proximity:
//
//	1.5 — target in the same room (highest urgency — interaction possible)
//	0.8 — target in the same zone but a different room
//	0   — target out of zone or unresolvable
func befriendContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return 0
	}
	switch targetProximityHops(mob, kind, id) {
	case 0:
		return 1.5 // same room
	case 1, 2, 3:
		return 0.8 // same zone, different room
	}
	return 0
}

// mobOpinionOf returns the score this mob holds for the given target.
//
// NOTE: The opinions package (chunk 1.1) only models mob→player opinions.
// Mob→mob opinions are explicitly deferred (chunk 3.6 spec rationale:
// "NPC↔NPC opinion store … is DEFERRED"). Passing kind="mob" returns 0,
// meaning the befriend predicate will never be satisfied for mob targets.
func mobOpinionOf(mob *mobs.Mob, kind string, id int) int {
	if kind != "player" {
		// mob→mob opinions deferred to chunk 3.6; predicate can never satisfy.
		return 0
	}
	return opinions.Get(int(mob.MobId), id)
}
