package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// validateFoldRecall is called during the onCast phase. Returns false to abort.
func validateFoldRecall(user *users.UserRecord) bool {
	currentRoomId := user.Character.RoomId

	// Check if recall is blocked in this room (instanced zones with allow_recall: false)
	if currentRoom := rooms.LoadRoom(currentRoomId); currentRoom != nil {
		if blocked, ok := currentRoom.GetTempData("allow_recall").(bool); ok && !blocked {
			user.SendText("Something about this place prevents you from recalling.")
			return false
		}
	}

	anchorRoom := getMiscDataInt(user, "fold-anchor-room")
	if anchorRoom <= 0 {
		user.SendText(`You reach for the Veil, but there is no anchor to ` +
			`pull you. Set one first with ` +
			`<ansi fg="command">cast fold-anchor</ansi>.`)
		return false
	}

	if anchorRoom == currentRoomId {
		user.SendText("You are already standing on your anchor.")
		return false
	}

	return true
}

// resolveFoldRecall is called during the onMagic phase.
func resolveFoldRecall(user *users.UserRecord) {
	anchorRoom := getMiscDataInt(user, "fold-anchor-room")
	currentRoomId := user.Character.RoomId

	if anchorRoom <= 0 || anchorRoom == currentRoomId {
		user.SendText("The fold collapses — no valid anchor found.")
		return
	}

	// Clear combat state before teleporting
	user.Character.EndAggro()

	// Send departure text to current room
	if room := rooms.LoadRoom(currentRoomId); room != nil {
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> folds through the Veil and vanishes!`,
			user.Character.Name), user.UserId)
	}

	// Move to anchor room
	rooms.MoveToRoom(user.UserId, anchorRoom)

	user.SendText("You fold through the Veil and arrive at your anchor point!")

	// Send arrival text to destination room (user is now there)
	if room := rooms.LoadRoom(anchorRoom); room != nil {
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> folds through the Veil and appears!`,
			user.Character.Name), user.UserId)
	}
}

// getMiscDataInt retrieves an integer stored in MiscData, handling both int
// and float64 (the latter can occur after YAML round-trips).
func getMiscDataInt(user *users.UserRecord, key string) int {
	val := user.Character.GetMiscData(key)
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
