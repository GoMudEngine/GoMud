package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func condMobInCombat(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.Aggro != nil {
		return Success
	}
	return Failure
}

func condMobHealthBelow(params map[string]any, ctx *EvalContext) Result {
	pct := getIntParam(params, "percent")
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	maxHP := mob.Character.HealthMax.Value
	if maxHP <= 0 {
		return Failure
	}
	currentPct := mob.Character.Health * 100 / maxHP
	if currentPct < pct {
		return Success
	}
	return Failure
}

func condMobAtHome(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.RoomId == mob.HomeRoomId {
		return Success
	}
	return Failure
}

func condMobHasBuff(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	buffId := getIntParam(params, "buff_id")
	if mob.Character.HasBuff(buffId) {
		return Success
	}
	return Failure
}

// condMobInRoom checks if a mob with the given template mob_id is present in
// the room.
func condMobInRoom(params map[string]any, ctx *EvalContext) Result {
	mobId := getIntParam(params, "mob_id")
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m != nil && int(m.MobId) == mobId {
			return Success
		}
	}
	return Failure
}
