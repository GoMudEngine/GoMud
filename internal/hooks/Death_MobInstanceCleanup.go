package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
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
			if from != life.Alive || to != life.Dead {
				return
			}
			if c.MobInstanceId == 0 {
				return
			}
			m := mobs.GetInstance(c.MobInstanceId)
			if m == nil {
				return
			}
			scheduleMobDespawnFromLife(m)
		})
}

// scheduleMobDespawnFromLife handles the full death pipeline for a
// mob instance: drop loot + corpse FIRST (while m is still valid),
// then despawn the instance.
//
// Consolidated into a single observer because the chunk-2 split
// between Death_MobLoot.go and this file produced a registration-
// order bug: AfterTransition cascades run in init() order, which
// is alphabetical by filename, so Death_MobInstanceCleanup.go ('I')
// ran BEFORE Death_MobLoot.go ('L') and destroyed the instance
// before the loot drop could read it. Symptom: mobs despawn
// instantly with no corpse / no loot drop.
//
// Order:
//  1. Drop loot + add corpse (uses live mob instance).
//  2. Delete persisted instance file so respawn starts fresh.
//  3. Destroy the in-memory instance record.
//  4. Clean up the home-room spawn slot (no cooldown skip).
//  5. Remove mob from its current room.
func scheduleMobDespawnFromLife(m *mobs.Mob) {
	// 1. Drop loot + corpse BEFORE destroying the instance.
	if room := rooms.LoadRoom(m.Character.RoomId); room != nil {
		dropMobLootAndSetCorpse(m, room)
	}

	// 2. Delete saved instance so respawn starts fresh from template.
	mobs.DeleteMobInstance(m.MobId, m.Zone, m.Character.Name, m.HomeRoomId)

	// 3. Destroy any in-memory record of this mob instance.
	mobs.DestroyInstance(m.InstanceId)

	// 4. Clean up the home-room spawn slot.
	if r := rooms.LoadRoom(m.HomeRoomId); r != nil {
		r.CleanupMobSpawns(false)
	}

	// 5. Remove from current room.
	if currentRoom := rooms.LoadRoom(m.Character.RoomId); currentRoom != nil {
		currentRoom.RemoveMob(m.InstanceId)
	}
}

func init() {
	characters.OnCharacterCreated(wireMobInstanceCleanup)
}
