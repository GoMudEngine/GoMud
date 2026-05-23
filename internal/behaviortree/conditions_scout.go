package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// condRoomHasHiddenEntity returns Success when at least one hidden mob
// or hidden player is in the room. Cheap pre-check for archetypes
// before paying for try_search's cooldown + per-discovery rolls.
func condRoomHasHiddenEntity(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	for _, pId := range room.GetPlayers() {
		if p := users.GetByUserId(pId); p != nil && p.Character.IsHidden() {
			return Success
		}
	}
	for _, mId := range room.GetMobs() {
		if m := mobs.GetInstance(mId); m != nil && m.Character.IsHidden() && m.InstanceId != mob.InstanceId {
			return Success
		}
	}
	return Failure
}

// condMobIsTracking returns Success when the self mob carries buff 86
// (Active Tracking).
func condMobIsTracking(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.HasBuff(86) {
		return Success
	}
	return Failure
}
