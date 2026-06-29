package behaviortree

// actions_room.go — room-manipulation actions:
// actSetRoomLocked, actSpawnItemInRoom, actAddTempExit, actMovePlayer, actIntercept

import (
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// actSetRoomLocked locks or unlocks a named exit in the current room.
// params: direction (string), locked ("true"/"false", default true)
func actSetRoomLocked(params map[string]any, ctx *EvalContext) Result {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	direction := getStringParam(params, "direction")
	if direction == "" {
		return Failure
	}
	locked := getStringParam(params, "locked") != "false"
	room.SetExitLock(direction, locked)
	return Success
}

// actSpawnItemInRoom drops an item on the floor of a room.
// params: item_id (int), room_id (int, default ctx.RoomId), chance (int 1-100, default 100)
func actSpawnItemInRoom(params map[string]any, ctx *EvalContext) Result {
	itemId := getIntParam(params, "item_id")
	if itemId == 0 {
		return Failure
	}
	roomId := getIntParam(params, "room_id")
	if roomId == 0 {
		roomId = ctx.RoomId
	}
	chance := getIntParam(params, "chance")
	if chance <= 0 {
		chance = 100
	}
	if util.Rand(100) >= chance {
		return Success // skipped by chance — not a failure
	}
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return Failure
	}
	item := items.New(itemId)
	room.AddItem(item, false)
	return Success
}

func actAddTempExit(params map[string]any, ctx *EvalContext) Result {
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	exitName := getStringParam(params, "exit_name")
	if exitName == "" {
		return Failure
	}
	roomId := getIntParam(params, "room_id")
	// If room_id is not a static param, fall back to reading from BehaviorState.
	// This lets create_instance store the result and add_temp_exit consume it.
	if roomId == 0 && ctx.MobState != nil {
		stateKey := getStringParam(params, "state_key")
		if stateKey == "" {
			stateKey = "instance_entry_room_id"
		}
		roomId = ctx.MobState.GetInt(stateKey)
	}
	if roomId == 0 {
		return Failure
	}
	tempExit := exit.TemporaryRoomExit{
		RoomId:  roomId,
		Title:   getStringParam(params, "title"),
		UserId:  ctx.Event.UserId,
		Expires: getStringParam(params, "expires"),
	}
	room.AddTemporaryExit(exitName, tempExit)
	return Success
}

// actMovePlayer teleports the triggering player to a target room.
// Failure (not Success) on a bad destination or missing player keeps
// btree sequences honest: portal NPCs sequence say-lines before this
// action, and a Failure stops anything wired after the teleport.
// params: room_id (int)
func actMovePlayer(params map[string]any, ctx *EvalContext) Result {
	if ctx.Event.UserId <= 0 {
		return Failure
	}
	roomId := getIntParam(params, "room_id")
	if roomId == 0 {
		return Failure
	}
	if err := rooms.MoveToRoom(ctx.Event.UserId, roomId); err != nil {
		return Failure
	}
	// Show the destination room. MoveToRoom does not auto-describe, so a
	// portal teleport (e.g. Warden Esk's arch out of the tutorial) otherwise
	// drops the player into the new room with NO output until they manually
	// look (2026-06-29 veteran smoke). Queue a look so arrival is never silent.
	if u := users.GetByUserId(ctx.Event.UserId); u != nil {
		u.Command("look")
	}
	return Success
}

// actIntercept marks the event as intercepted, preventing the default command
// handler from processing it.
func actIntercept(params map[string]any, ctx *EvalContext) Result {
	ctx.Intercepted = true
	return Success
}
