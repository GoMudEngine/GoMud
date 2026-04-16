package behaviortree

// actions_combat.go — combat actions:
// actAttack, actFlee, actCast, actAddBuff, actRemoveBuff

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func actAttack(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	targetUserId := ctx.Event.UserId
	// If no specific target, pick a random player in the room
	if targetUserId == 0 {
		room := rooms.LoadRoom(ctx.RoomId)
		if room == nil {
			return Failure
		}
		players := room.GetPlayers()
		if len(players) == 0 {
			return Failure
		}
		targetUserId = players[util.Rand(len(players))]
	}
	mob.Character.SetAggro(targetUserId, 0, characters.DefaultAttack)
	return Success
}

func actFlee(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	mob.Command("flee")
	return Success
}

func actCast(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	spell := getStringParam(params, "spell")
	if spell == "" {
		return Failure
	}
	mob.Command("cast " + spell)
	return Success
}

// actAddBuff applies a buff to the acting mob.
// params: buff_id (int)
func actAddBuff(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	buffId := getIntParam(params, "buff_id")
	if buffId == 0 {
		return Failure
	}
	mob.AddBuff(buffId, "behaviortree")
	return Success
}

// actRemoveBuff removes a buff from the triggering player by buff ID.
// params: buff_id (int)
func actRemoveBuff(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	buffId := getIntParam(params, "buff_id")
	user.Character.RemoveBuff(buffId)
	return Success
}
