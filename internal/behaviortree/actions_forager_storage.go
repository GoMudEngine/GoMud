package behaviortree

// actions_forager_storage.go — btree action primitive for the forager
// chest-deposit workflow (chunk 2.10-followups, Task 7).
//
// try_store_excess is a multi-tick action: each btree tick advances ONE
// step of the pathto → unlock → put items → lock pipeline. The btree
// re-evaluates on the next game tick, picking up where it left off.
//
// Required param:
//
//	chest_room (int) — room ID where the forager's lockbox lives.

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// actTryStoreExcess drives the forager through a chest deposit cycle:
//
//  1. chest_room param missing → log error, Failure.
//  2. Satchel is empty → Failure (nothing to deposit).
//  3. Mob not in chest_room → issue "pathto <chest_room>", Success.
//  4. Lockbox missing in chest_room → log warning, Failure.
//  5. Lockbox is locked → issue "unlock lockbox", Success.
//  6. Lockbox is unlocked → issue "put <item> in lockbox" for each item,
//     then "lock lockbox", Success.
//
// Steps 3–6 each return Success so the btree re-evaluates on the next
// tick and advances to the next step. Failed puts (chest full) are
// graceful no-ops: the engine leaves the item in the satchel and the
// forager will retry on the next deposit cycle.
func actTryStoreExcess(params map[string]any, ctx *EvalContext) Result {
	chestRoom := getIntParam(params, "chest_room")
	if chestRoom == 0 {
		mudlog.Error("try_store_excess",
			"error", "missing required `chest_room` parameter",
			"instance_id", ctx.InstanceId)
		return Failure
	}

	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}

	// 1. Nothing to deposit.
	if len(mob.Character.Items) == 0 {
		return Failure
	}

	// 2. Travel to chest room if not already there.
	if mob.Character.RoomId != chestRoom {
		mob.Command(fmt.Sprintf("pathto %d", chestRoom))
		return Success
	}

	// 3. Verify the lockbox container exists.
	room := rooms.LoadRoom(chestRoom)
	if room == nil {
		mudlog.Warn("try_store_excess",
			"warn", "chest_room not found",
			"chest_room", chestRoom, "instance_id", ctx.InstanceId)
		return Failure
	}

	containerKey := room.FindContainerByName("lockbox")
	if containerKey == "" {
		mudlog.Warn("try_store_excess",
			"warn", "chest_room has no lockbox container",
			"chest_room", chestRoom, "instance_id", ctx.InstanceId)
		return Failure
	}

	container := room.Containers[containerKey]

	// 4. Unlock if locked — one step this tick, next tick continues.
	if container.Lock.IsLocked() {
		mob.Command("unlock lockbox")
		return Success
	}

	// 5. Deposit all satchel items then lock.
	for _, item := range mob.Character.Items {
		mob.Command(fmt.Sprintf("put %s in lockbox", item.Name()))
	}
	mob.Command("lock lockbox")
	return Success
}
