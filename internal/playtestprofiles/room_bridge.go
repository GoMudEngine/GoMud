package playtestprofiles

import "github.com/GoMudEngine/GoMud/internal/rooms"

func loadRoomExists(roomID int) bool {
	return rooms.LoadRoom(roomID) != nil
}
