package devtools

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// LinkRooms adds a bidirectional exit between two rooms that may be in different zones.
// direction is the exit direction from roomA to roomB (e.g. "north").
// The reciprocal exit is added automatically on roomB.
func LinkRooms(zoneA string, roomIdA int, direction string, zoneB string, roomIdB int) error {

	roomA := rooms.LoadRoomTemplate(roomIdA)
	if roomA == nil {
		return fmt.Errorf("room %d (zone %q) not found", roomIdA, zoneA)
	}

	roomB := rooms.LoadRoomTemplate(roomIdB)
	if roomB == nil {
		return fmt.Errorf("room %d (zone %q) not found", roomIdB, zoneB)
	}

	rev := mapper.GetReciprocalExit(direction)
	if rev == "" {
		return fmt.Errorf("direction %q has no reciprocal exit", direction)
	}

	if roomA.Exits == nil {
		roomA.Exits = make(map[string]exit.RoomExit)
	}
	if roomB.Exits == nil {
		roomB.Exits = make(map[string]exit.RoomExit)
	}

	roomA.Exits[direction] = exit.RoomExit{RoomId: roomIdB}
	roomB.Exits[rev] = exit.RoomExit{RoomId: roomIdA}

	if err := rooms.SaveRoomTemplate(*roomA); err != nil {
		return fmt.Errorf("failed to save room %d: %w", roomIdA, err)
	}

	if err := rooms.SaveRoomTemplate(*roomB); err != nil {
		return fmt.Errorf("failed to save room %d: %w", roomIdB, err)
	}

	return nil
}
