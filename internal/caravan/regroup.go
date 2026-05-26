package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// ForceRegroupCrew force-moves every in-world caravan crew member
// (every mob whose template id is in caravanMobIds, except the
// leader itself) to the leader's current room. Skips already
// co-located members. Idempotent — safe to call when no regroup is
// needed.
//
// Used by:
//   - CaravanArrivalListener (handleDepotArrival) when the leader
//     arrives at wp0 carrying the patrol_fresh_respawn marker — this
//     pulls stragglers home after a leader respawn.
//   - ResetAllCaravanStates / ResetCaravanStateByInstanceId in the
//     admin caravan reset command — pulls the whole crew back to
//     the leader's room as part of a hard reset.
//
// Does nothing if the leader is nil or the leader's room can't be
// loaded.
func ForceRegroupCrew(leader *mobs.Mob) {
	if leader == nil {
		return
	}
	leaderRoomId := leader.Character.RoomId
	newRoom := rooms.LoadRoom(leaderRoomId)
	if newRoom == nil {
		return
	}
	for templateId := range caravanMobIds {
		if templateId == int(leader.MobId) {
			continue // skip the leader itself
		}
		for _, instId := range mobs.GetAllMobInstanceIds() {
			m := mobs.GetInstance(instId)
			if m == nil || int(m.MobId) != templateId {
				continue
			}
			if m.Character.RoomId == leaderRoomId {
				continue // already co-located
			}
			oldRoomId := m.Character.RoomId
			oldRoom := rooms.LoadRoom(oldRoomId)
			if oldRoom != nil {
				oldRoom.RemoveMob(m.InstanceId)
			}
			newRoom.AddMob(m.InstanceId) // sets m.Character.RoomId internally
			mudlog.Info("caravan crew regroup",
				"leader", leader.Character.Name,
				"crew_template", templateId,
				"crew_instance", m.InstanceId,
				"from_room", oldRoomId,
				"to_room", leaderRoomId,
			)
		}
	}
}
