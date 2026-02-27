package devtools

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// reciprocalDir maps each standard exit direction to its opposite.
// We use a hard-coded table rather than mapper.GetReciprocalExit because that
// function iterates over a map whose entries share duplicate (x,y,z) deltas
// (e.g. "east" and "east-gap" both have delta {1,0,0}), producing a
// non-deterministic result.
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
