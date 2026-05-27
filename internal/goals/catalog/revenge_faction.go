package catalog

import (
	"github.com/GoMudEngine/GoMud/internal/factions"
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const factionKillsMiscPrefix = "faction_kills_inflicted:"

func init() {
	goals.RegisterGoalType("revenge-faction", goals.GoalTypeMeta{
		Predicate:     revengeFactionPredicate,
		ContextScore:  revengeFactionContextScore,
		AllowMultiple: true,
		DedupKey:      revengeFactionDedupKey,
		Params: []goals.ParamSchema{
			{Key: "faction_id", Required: true, GoType: "string"},
			{Key: "target_kill_count", Required: true, GoType: "int"},
		},
	})
}

func revengeFactionDedupKey(g *goals.Goal) string {
	if fid, ok := g.Params["faction_id"].(string); ok {
		return fid
	}
	return ""
}

// revengeFactionPredicate: satisfied when the mob's per-faction kill counter
// (stored in MiscData under "faction_kills_inflicted:<faction_id>") reaches
// the target_kill_count threshold.
//
// The counter is incremented by a 4.5 reactive hook on kill events; 4.3 only
// defines the read path. Until 4.5 ships the predicate always returns false.
func revengeFactionPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	fid, _ := g.Params["faction_id"].(string)
	target := paramIntOr(g, "target_kill_count", 0)
	if fid == "" || target == 0 {
		return false
	}
	current := mobMiscInt(mob, factionKillsMiscPrefix+fid)
	return current >= target
}

// revengeFactionContextScore: 0 if no faction members are in the mob's zone;
// otherwise 1.0 + 0.1 × member_count, capped at 2.0.
func revengeFactionContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	fid, _ := g.Params["faction_id"].(string)
	if fid == "" {
		return 0
	}
	count := factionMembersInZone(mob, fid)
	if count == 0 {
		return 0
	}
	score := 1.0 + 0.1*float64(count)
	if score > 2.0 {
		return 2.0
	}
	return score
}

// mobMiscInt reads an integer value from mob.Character.MiscData. Returns 0
// if the key is absent or the stored value is not a numeric type.
func mobMiscInt(mob *mobs.Mob, key string) int {
	raw := mob.Character.GetMiscData(key)
	if raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

// factionMembersInZone counts live mob instances in the same zone as mob
// that belong to factionId, using factions.FactionsForMob for membership.
// The calling mob itself is excluded from the count.
func factionMembersInZone(mob *mobs.Mob, factionId string) int {
	zone := mob.Character.Zone
	count := 0
	for _, instId := range mobs.GetAllMobInstanceIds() {
		inst := mobs.GetInstance(instId)
		if inst == nil || inst.Character.Zone != zone {
			continue
		}
		if inst.InstanceId == mob.InstanceId {
			continue // don't count self
		}
		for _, fid := range factions.FactionsForMob(inst) {
			if fid == factionId {
				count++
				break
			}
		}
	}
	return count
}
