package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// CompanionTransportCallback adapts TransportCompanions to the
// rooms.SetCompanionTransport signature (which takes only userId so
// rooms doesn't need to import users).
func CompanionTransportCallback(userId, oldRoomId, newRoomId int) {
	user := users.GetByUserId(userId)
	if user == nil {
		return
	}
	TransportCompanions(user, oldRoomId, newRoomId)
}

// TransportCompanions moves every live companion of owner into the owner's
// new room. Called after a successful owner room change (go, recall, portal,
// fold-recall, etc). Aborts any in-progress cast (conviction already spent is
// forfeit). Ends aggro on targets no longer in the new room.
//
// No-op when newRoomId == oldRoomId or owner is nil.
func TransportCompanions(owner *users.UserRecord, oldRoomId, newRoomId int) {
	if owner == nil || newRoomId == oldRoomId {
		return
	}

	for _, c := range owner.Character.Companions {
		mob := mobs.GetInstance(c.InstanceId)
		if mob == nil {
			// Companion has been reaped; skip silently.
			continue
		}
		if mob.Character.RoomId == newRoomId {
			// Already in destination room (e.g. summoned directly there).
			continue
		}

		// Interrupt any in-progress cast (spent conviction is forfeit).
		if mob.Character.CastingState != nil {
			mob.Character.CastingState = nil
		}

		// Remove from current room.
		curRoom := rooms.LoadRoom(mob.Character.RoomId)
		if curRoom != nil {
			curRoom.RemoveMob(mob.InstanceId)
			curRoom.SendText(
				fmt.Sprintf("%s follows %s.", mob.Character.Name, owner.Character.Name),
				owner.UserId,
			)
		}

		// Add to destination room.
		destRoom := rooms.LoadRoom(newRoomId)
		if destRoom == nil {
			mudlog.Error("TransportCompanions", "error",
				fmt.Sprintf("destination room %d not found for companion %d",
					newRoomId, mob.InstanceId))
			continue
		}
		destRoom.AddMob(mob.InstanceId)
		mob.Character.RoomId = newRoomId

		// Inform owner.
		owner.SendText(fmt.Sprintf("Your %s rejoins you.", mob.Character.Name))

		// End aggro if the current target is no longer in the destination room.
		if mob.Character.IsInCombat() {
			aggro := mob.Character.Aggro
			// Only strip aggro when it has a concrete target to check for;
			// an Aggro struct with both IDs zero is mid-setup state and
			// must not be clobbered by the move (see code review d6b9fc19).
			if aggro.UserId != 0 || aggro.MobInstanceId != 0 {
				targetStillPresent := false
				if aggro.UserId != 0 {
					for _, pId := range destRoom.GetPlayers() {
						if pId == aggro.UserId {
							targetStillPresent = true
							break
						}
					}
				}
				if !targetStillPresent && aggro.MobInstanceId != 0 {
					for _, mId := range destRoom.GetMobs() {
						if mId == aggro.MobInstanceId {
							targetStillPresent = true
							break
						}
					}
				}
				if !targetStillPresent {
					mob.Character.EndAggro()
				}
			}
		}
	}
}
