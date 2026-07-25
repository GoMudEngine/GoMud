package devtools

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// reciprocalDir maps each standard exit direction to its opposite.
//
// mapper.GetReciprocalExit is deterministic as of the 2026-07-25 fix, but this
// table stays deliberately narrower: LinkRooms accepts only plain cardinal
// directions and reports "not a supported cardinal direction" for anything
// else, and the mapper's full vocabulary ("north-x2", "east-gap") would
// silently widen that contract.
var reciprocalDir = map[string]string{
	"north":     "south",
	"south":     "north",
	"east":      "west",
	"west":      "east",
	"northeast": "southwest",
	"southwest": "northeast",
	"northwest": "southeast",
	"southeast": "northwest",
	"up":        "down",
	"down":      "up",
}

// LinkRooms adds a bidirectional exit between two rooms that may be in different zones.
// direction is the exit direction from roomA to roomB (e.g. "north").
// The reciprocal exit is added automatically on roomB.
func LinkRooms(zoneA string, roomIdA int, direction string, zoneB string, roomIdB int) error {

	rev, ok := reciprocalDir[direction]
	if !ok {
		return fmt.Errorf("direction %q is not a supported cardinal direction", direction)
	}

	// LoadRoom (not LoadRoomTemplate) ensures the room is added to the
	// in-memory cache. SaveRoomTemplate reads roomManager.rooms[id] and
	// panics with a nil dereference if the room isn't cached first.
	roomA := rooms.LoadRoom(roomIdA)
	if roomA == nil {
		return fmt.Errorf("room %d (zone %q) not found", roomIdA, zoneA)
	}

	roomB := rooms.LoadRoom(roomIdB)
	if roomB == nil {
		return fmt.Errorf("room %d (zone %q) not found", roomIdB, zoneB)
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
