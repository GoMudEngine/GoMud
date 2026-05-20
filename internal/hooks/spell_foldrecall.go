package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// validateFoldRecall is called during the onCast phase. Returns false to
// abort the spell.
func validateFoldRecall(actor actions.Actor) bool {
	char := actor.GetCharacter()
	if char == nil {
		return false
	}
	currentRoomId := char.RoomId

	// Check if recall is blocked in the current room (instanced zones with
	// allow_recall: false).
	if currentRoom := rooms.LoadRoom(currentRoomId); currentRoom != nil {
		if blocked, ok := currentRoom.GetTempData("allow_recall").(bool); ok && !blocked {
			actor.SendTextLegacy("Something about this place prevents you from recalling.")
			return false
		}
	}

	anchorRoom := getMiscDataInt(char, "fold-anchor-room")
	if anchorRoom <= 0 {
		actor.SendTextLegacy(`You reach for the Veil, but there is no anchor to ` +
			`pull you. Set one first with ` +
			`<ansi fg="command">cast fold-anchor</ansi>.`)
		return false
	}

	if anchorRoom == currentRoomId {
		actor.SendTextLegacy("You are already standing on your anchor.")
		return false
	}

	return true
}

// resolveFoldRecall is called during the onMagic phase.
func resolveFoldRecall(actor actions.Actor) {
	char := actor.GetCharacter()
	if char == nil {
		return
	}
	anchorRoom := getMiscDataInt(char, "fold-anchor-room")
	currentRoomId := char.RoomId

	if anchorRoom <= 0 || anchorRoom == currentRoomId {
		actor.SendTextLegacy("The fold collapses — no valid anchor found.")
		return
	}

	// Clear combat state before teleporting.
	char.EndAggro()

	// Move the actor first; only broadcast on success so a failed teleport
	// doesn't leave the departure room thinking the actor vanished.
	if !teleportActor(actor, anchorRoom) {
		actor.SendTextLegacy("The fold collapses — no valid anchor found.")
		return
	}

	// Departure broadcast on the room the actor LEFT (use the snapshotted
	// currentRoomId — char.RoomId has been updated by teleport).
	if oldRoom := rooms.LoadRoom(currentRoomId); oldRoom != nil {
		oldRoom.SendText(messaging.CategorySpellManifestation, fmt.Sprintf(
			`<ansi fg="username">%s</ansi> folds through the Veil and vanishes!`,
			actor.GetName()), actor.GetUserId())
	}

	actor.SendTextLegacy("You fold through the Veil and arrive at your anchor point!")

	// Arrival broadcast on the new room.
	if newRoom := rooms.LoadRoom(anchorRoom); newRoom != nil {
		newRoom.SendText(messaging.CategorySpellManifestation, fmt.Sprintf(
			`<ansi fg="username">%s</ansi> folds through the Veil and appears!`,
			actor.GetName()), actor.GetUserId())
	}
}

// teleportActor moves the actor to the destination room. For players this
// goes through rooms.MoveToRoom (handles cross-zone bookkeeping). For mobs
// it manipulates room membership directly. Returns false if the destination
// room can't be loaded.
func teleportActor(actor actions.Actor, toRoomId int) bool {
	if actor.IsPlayer() {
		// Players: existing helper handles the cross-zone case.
		if err := rooms.MoveToRoom(actor.GetUserId(), toRoomId); err != nil {
			return false
		}
		return true
	}

	// Mobs: manual room membership update.
	char := actor.GetCharacter()
	fromRoom := rooms.LoadRoom(char.RoomId)
	toRoom := rooms.LoadRoom(toRoomId)
	if toRoom == nil {
		return false
	}
	instId := actor.GetMobInstanceId()
	if fromRoom != nil {
		fromRoom.RemoveMob(instId)
	}
	toRoom.AddMob(instId) // AddMob sets mob.Character.RoomId internally (rooms.go:827)
	return true
}

// getMiscDataInt retrieves an integer stored in MiscData, handling both int
// and float64 (the latter can occur after YAML round-trips).
func getMiscDataInt(char *characters.Character, key string) int {
	val := char.GetMiscData(key)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}
