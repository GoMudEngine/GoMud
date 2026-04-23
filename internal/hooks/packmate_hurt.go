package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// dispatchEventFn is a package-level indirection so tests can spy on the
// behaviortree call without firing a real tree. Production value is
// behaviortree.TryMobBehavior.
var dispatchEventFn = behaviortree.TryMobBehavior

// dispatchPackmateHurt fires a `packmate_hurt` btree event on every mob
// that shares a pack with the victim (same room, matching routine/link,
// alive, not charmed).
//
// Called from handleAggroAndAssist after the defender-side aggro has
// been established. Replaces the former mobs.MakeHostile call.
//
// Spec: docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md
// (§ Event dispatch).
func dispatchPackmateHurt(victim *mobs.Mob, attackerUserId int, attackerMobInstanceId int) {
	if victim == nil {
		return
	}
	packmates := mobs.FindPackmatesInRoom(victim)
	for _, pm := range packmates {
		ctx := behaviortree.EventContext{
			EventType: "packmate_hurt",
			UserId:    attackerUserId,
			MobId:     attackerMobInstanceId,
		}
		dispatchEventFn(pm.InstanceId, ctx)
	}
}
