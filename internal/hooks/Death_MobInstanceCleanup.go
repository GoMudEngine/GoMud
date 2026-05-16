package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

// wireMobInstanceCleanup schedules mob instance cleanup when a
// mob's Life transitions Alive→Dead. Deletes the saved instance
// file, destroys the in-memory instance, updates the home-room
// spawn slot, and removes the mob from the current room.
//
// NOTE (Chunk-2): This observer is wired but dormant until
// Task 10 routes mob death paths through Life.TransitionToDead.
// The existing inline cleanup block in
// internal/mobcommands/suicide.go (lines 225-237) is the live
// path today; it will be removed when Task 10 lands.
func wireMobInstanceCleanup(c *characters.Character) {
	c.Life.Inner().AfterTransition("mob_instance_cleanup",
		func(from, to life.State, r state.TransitionReason) {
			// [DESPAWN_DIAG] Temporary — log every fire of this
			// cascade so we can confirm it's running for dying mobs.
			mudlog.Warn("[DESPAWN_DIAG] mob_instance_cleanup observer fired",
				"name", c.Name, "mob_inst_id", c.MobInstanceId,
				"from", from, "to", to, "trigger", r.Trigger)
			if from != life.Alive || to != life.Dead {
				return
			}
			// Only fire for mob characters.
			if c.MobInstanceId == 0 {
				mudlog.Warn("[DESPAWN_DIAG] skip: MobInstanceId==0",
					"name", c.Name)
				return
			}
			m := mobs.GetInstance(c.MobInstanceId)
			if m == nil {
				mudlog.Warn("[DESPAWN_DIAG] skip: mob already destroyed",
					"name", c.Name, "mob_inst_id", c.MobInstanceId)
				return
			}
			mudlog.Warn("[DESPAWN_DIAG] calling scheduleMobDespawnFromLife",
				"name", c.Name, "mob_inst_id", c.MobInstanceId,
				"room_id", m.Character.RoomId)
			scheduleMobDespawnFromLife(m)
		})
}

// scheduleMobDespawnFromLife is the observer-side port of the
// instance-cleanup block from suicide.go (lines 225-237). When
// Task 10 thins suicide.go, the inline block there is removed
// and this function becomes the sole call site.
//
// Order mirrors suicide.go exactly:
//  1. Delete persisted instance file so respawn starts fresh.
//  2. Destroy the in-memory instance record.
//  3. Clean up the home-room spawn slot (no cooldown skip).
//  4. Remove mob from its current room.
func scheduleMobDespawnFromLife(m *mobs.Mob) {
	// 1. Delete saved instance so respawn starts fresh from template.
	mobs.DeleteMobInstance(m.MobId, m.Zone, m.Character.Name, m.HomeRoomId)

	// 2. Destroy any in-memory record of this mob instance.
	mobs.DestroyInstance(m.InstanceId)

	// 3. Clean up the home-room spawn slot.
	if r := rooms.LoadRoom(m.HomeRoomId); r != nil {
		r.CleanupMobSpawns(false)
	}

	// 4. Remove from current room.
	if currentRoom := rooms.LoadRoom(m.Character.RoomId); currentRoom != nil {
		currentRoom.RemoveMob(m.InstanceId)
	}
}

func init() {
	characters.OnCharacterCreated(wireMobInstanceCleanup)
}
