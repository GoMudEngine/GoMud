package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
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

// CompanionSweepCallback adapts PushCompanionsToRoom to the
// behaviortree.SetCompanionSweep signature (which takes only userId so
// behaviortree doesn't need to import users).
func CompanionSweepCallback(userId, destRoomId int) {
	user := users.GetByUserId(userId)
	if user == nil {
		return
	}
	PushCompanionsToRoom(user, destRoomId)
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
		if mob.Character.Activity != nil && mob.Character.Activity.IsCasting() {
			mob.Character.Activity.ForceFree(state.TransitionReason{
				Trigger: "companion_follow",
				Actor:   mob.Character.Activity.Self(),
			})
		}

		// Remove from current room.
		curRoom := rooms.LoadRoom(mob.Character.RoomId)
		if curRoom != nil {
			curRoom.RemoveMob(mob.InstanceId)
			// COMPANION-NAME-LEAK FIX (T11c): named companion mention is
			// visual content — route through SendTextVisual so infrared
			// observers see the anonymized form, and blind observers don't
			// see a free identification.
			curRoom.SendTextVisual(messaging.CategoryMobEmote,
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

		// Inform owner. Owner-private channel — owner-knowable companion
		// name is fine over audio (no infrared / blind observer to leak to).
		owner.SendText(messaging.CategorySystem, fmt.Sprintf("Your %s rejoins you.", mob.Character.Name))

		// End aggro if the current target is no longer in the destination room.
		clearAggroIfTargetAbsent(mob, destRoom)
	}
}

// clearAggroIfTargetAbsent ends a mob's aggro if its current target (player
// or mob) is not present in destRoom. Shared by TransportCompanions and
// PushCompanionsToRoom.
func clearAggroIfTargetAbsent(mob *mobs.Mob, destRoom *rooms.Room) {
	if !mob.Character.IsInCombat() {
		return
	}
	aggro := mob.Character.Aggro
	// Only strip aggro when it has a concrete target to check for;
	// an Aggro struct with both IDs zero is mid-setup state and
	// must not be clobbered by the move (see code review d6b9fc19).
	if aggro.UserId == 0 && aggro.MobInstanceId == 0 {
		return
	}
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

// PushCompanionsToRoom forcibly relocates every live companion of owner into
// destRoomId, regardless of the companion's current room. Used by boss adds
// (e.g. the Hull Sweeper) to shove conjured allies out of a fight into a
// holding room.
//
// GEAR-SAFETY GUARANTEE: this moves the companion mob instance the same way
// TransportCompanions does — RemoveMob/AddMob + a RoomId reassignment on the
// existing *mobs.Mob instance. The instance (and its Character.Equipment /
// Character.Items, i.e. its worn gear and carried items) is never destroyed
// or recreated, so equipped/carried gear survives untouched. Companions are
// relocated, never destroyed.
//
// No-op when owner is nil.
func PushCompanionsToRoom(owner *users.UserRecord, destRoomId int) {
	if owner == nil {
		return
	}

	for _, c := range owner.Character.Companions {
		mob := mobs.GetInstance(c.InstanceId)
		if mob == nil {
			// Companion has been reaped; skip silently.
			continue
		}
		if mob.Character.RoomId == destRoomId {
			// Already in destination room.
			continue
		}

		// Interrupt any in-progress cast (spent conviction is forfeit).
		if mob.Character.Activity != nil && mob.Character.Activity.IsCasting() {
			mob.Character.Activity.ForceFree(state.TransitionReason{
				Trigger: "companion_sweep",
				Actor:   mob.Character.Activity.Self(),
			})
		}

		// Remove from current room.
		curRoomId := mob.Character.RoomId
		curRoom := rooms.LoadRoom(curRoomId)
		if curRoom != nil {
			curRoom.RemoveMob(mob.InstanceId)
			// COMPANION-NAME-LEAK FIX (T11c): named companion mention is
			// visual content — route through SendTextVisual so infrared
			// observers see the anonymized form, and blind observers don't
			// see a free identification.
			curRoom.SendTextVisual(messaging.CategoryMobEmote,
				fmt.Sprintf("%s is swept out of the room!", mob.Character.Name),
				owner.UserId,
			)
		}

		// Add to destination room.
		destRoom := rooms.LoadRoom(destRoomId)
		if destRoom == nil {
			mudlog.Error("PushCompanionsToRoom", "error",
				fmt.Sprintf("destination room %d not found for companion %d",
					destRoomId, mob.InstanceId))
			continue
		}
		destRoom.AddMob(mob.InstanceId)
		mob.Character.RoomId = destRoomId

		// Inform owner. Owner-private channel — owner-knowable companion
		// name is fine over audio (no infrared / blind observer to leak to).
		owner.SendText(messaging.CategorySystem, fmt.Sprintf("Your %s is swept away!", mob.Character.Name))

		// End aggro if the current target is no longer in the destination room.
		clearAggroIfTargetAbsent(mob, destRoom)
	}
}
