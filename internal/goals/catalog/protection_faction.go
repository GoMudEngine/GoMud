package catalog

import (
	"github.com/GoMudEngine/GoMud/internal/factions"
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("protection-faction", goals.GoalTypeMeta{
		Predicate:     protectionFactionPredicate,
		ContextScore:  protectionFactionContextScore,
		AllowMultiple: true,
		DedupKey:      protectionFactionDedupKey,
		Params: []goals.ParamSchema{
			{Key: "faction_id", Required: true, GoType: "string"},
		},
	})
}

func protectionFactionDedupKey(g *goals.Goal) string {
	if fid, ok := g.Params["faction_id"].(string); ok {
		return fid
	}
	return ""
}

// protectionFactionPredicate: never satisfied (ongoing). The goal persists
// as long as the mob is alive; 4.6's pruning sweep is the removal path.
func protectionFactionPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	return false
}

// protectionFactionContextScore:
//   - 0   if no faction members are in the mob's zone
//   - 2.0 if any faction member is currently in combat in the zone
//   - 1.0 if hostile mobs are in the zone but no member is in combat
//   - 0.3 if zone is calm (members present but no active threat)
func protectionFactionContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	fid, _ := g.Params["faction_id"].(string)
	if fid == "" {
		return 0
	}
	if factionMembersInZone(mob, fid) == 0 {
		return 0
	}
	if factionMemberInCombatInZone(mob, fid) {
		return 2.0
	}
	if hostileMobsInZone(mob) {
		return 1.0
	}
	return 0.3
}

// factionMemberInCombatInZone returns true if any mob in the same zone as
// mob belongs to factionId and currently has an active Aggro (i.e., is in
// combat). The calling mob itself is excluded.
func factionMemberInCombatInZone(mob *mobs.Mob, factionId string) bool {
	zone := mob.Character.Zone
	for _, instId := range mobs.GetAllMobInstanceIds() {
		inst := mobs.GetInstance(instId)
		if inst == nil || inst.Character.Zone != zone {
			continue
		}
		if inst.InstanceId == mob.InstanceId {
			continue
		}
		if inst.Character.Aggro == nil {
			continue
		}
		for _, fid := range factions.FactionsForMob(inst) {
			if fid == factionId {
				return true
			}
		}
	}
	return false
}

// hostileMobsInZone returns true if any mob instance in the same zone as mob
// has AutoAggro set, indicating a standing threat. The calling mob is excluded.
func hostileMobsInZone(mob *mobs.Mob) bool {
	zone := mob.Character.Zone
	for _, instId := range mobs.GetAllMobInstanceIds() {
		inst := mobs.GetInstance(instId)
		if inst == nil || inst.Character.Zone != zone {
			continue
		}
		if inst.InstanceId == mob.InstanceId {
			continue
		}
		if inst.AutoAggro {
			return true
		}
	}
	return false
}
