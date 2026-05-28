package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/relationships"
)

const ruleNameFriendKilledToRevenge = "friend_killed_to_revenge"
const friendKilledRevengePriority = 85

func init() {
	Register(ruleNameFriendKilledToRevenge, friendKilledToRevenge, "MobDeath")
}

// friendKilledToRevenge: on MobDeath, walk the victim's relationship
// edges (keyed by template id); for each friend/family/lover edge,
// find all loaded instances of the friend mob template and seed a
// revenge-mob goal on each, targeting the killer.
//
// Priority 85 (above survival's 80 — grief outweighs baseline
// self-preservation). seedRevengeGoalIfAbsent dedups against an
// existing revenge goal targeting the same killer.
//
// Player-as-killer events (KillerMobInstanceId == 0) are skipped:
// the relationships API links template ids, not player ids, so
// mob-vs-mob attribution is the only well-defined path here.
func friendKilledToRevenge(event events.Event) {
	md, ok := event.(events.MobDeath)
	if !ok {
		return
	}
	killerKind, killerId := resolveKillerFromMobDeath(event)
	if killerKind == "" || killerId == 0 {
		return // unresolvable killer (player kill or unclear)
	}

	// RelationsOf takes the mob TEMPLATE id, not the instance id.
	relations := relationships.RelationsOf(md.MobId)
	if len(relations) == 0 {
		return
	}

	// For each friendly relation, walk all loaded instances looking
	// for mobs whose template id matches the relation's Other field.
	for _, rel := range relations {
		if !isFriendlyRelationType(rel.Type) {
			continue
		}
		for _, instId := range mobs.GetAllMobInstanceIds() {
			inst := mobs.GetInstance(instId)
			if inst == nil || int(inst.MobId) != rel.Other {
				continue
			}
			// Don't seed revenge onto the victim itself — defensive
			// guard against authoring errors that put a mob in its
			// own relation list.
			if inst.InstanceId == md.InstanceId {
				continue
			}
			seedRevengeGoalIfAbsent(inst, killerKind, killerId, friendKilledRevengePriority)
		}
	}
}

// isFriendlyRelationType returns true for the relationship types
// where the surviving NPC should mourn and potentially seek revenge:
// friend, family, and lover. Rival, employer, and employee are
// excluded (rival's killer may be welcome; workplace relations are
// neutral with respect to grief).
func isFriendlyRelationType(t relationships.Type) bool {
	switch t {
	case relationships.TypeFriend, relationships.TypeFamily, relationships.TypeLover:
		return true
	}
	return false
}
