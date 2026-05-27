package catalog

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// resolveTargetRoomId returns the room id of the given target, or 0 if
// the target cannot be located. Shared by revenge-mob and protection-mob.
func resolveTargetRoomId(kind string, id int) int {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			if inst := mobs.GetInstance(instId); inst != nil && inst.MobId == mobs.MobId(id) {
				return inst.Character.RoomId
			}
		}
	case "player":
		if u := users.GetByUserId(id); u != nil {
			return u.Character.RoomId
		}
	}
	return 0
}

// targetAlive reports whether the given target is alive and present in
// the live entity maps.
//
//   - "mob" kind: at least one live instance with matching template MobId.
//   - "player" kind: user present in the live users map with Health > 0.
func targetAlive(kind string, id int) bool {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			if inst := mobs.GetInstance(instId); inst != nil && inst.MobId == mobs.MobId(id) {
				return true
			}
		}
		return false
	case "player":
		u := users.GetByUserId(id)
		return u != nil && u.Character.Health > 0
	}
	return false
}

// targetProximityHops returns the hop distance between mob and its target:
//
//	 0  — same room
//	 1  — same zone but different room (simplified: no full BFS; single bucket)
//	-1  — target unresolvable, or in a different zone
//
// Used by befriend's ContextScore to bucket: 1.5 (same room), 0.8 (same
// zone), 0 (other). Shared helper extracted from Task 15.
func targetProximityHops(mob *mobs.Mob, kind string, id int) int {
	targetRoom := resolveTargetRoomId(kind, id)
	if targetRoom == 0 {
		return -1
	}
	if targetRoom == mob.Character.RoomId {
		return 0
	}
	r := rooms.LoadRoom(targetRoom)
	if r == nil || r.Zone != mob.Character.Zone {
		return -1
	}
	// Same zone, different room — collapse to bucket 1.
	return 1
}

// targetInCombat reports whether the given target is currently in combat
// (has a non-nil Aggro pointer).
func targetInCombat(kind string, id int) bool {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			inst := mobs.GetInstance(instId)
			if inst != nil && inst.MobId == mobs.MobId(id) {
				return inst.Character.Aggro != nil
			}
		}
		return false
	case "player":
		u := users.GetByUserId(id)
		return u != nil && u.Character.Aggro != nil
	}
	return false
}
