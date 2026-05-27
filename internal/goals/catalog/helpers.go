package catalog

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
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
