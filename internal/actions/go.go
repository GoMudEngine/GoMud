package actions

import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// FindExitResult holds the outcome of an exit lookup.
type FindExitResult struct {
	ExitName string
	RoomId   int
	Found    bool
}

// FindExit looks up an exit on the given room by name (supports aliases and
// fuzzy matching via Room.FindExitByName). Returns Found=false when no
// matching exit exists.
func FindExit(room *rooms.Room, exitName string) FindExitResult {
	name, roomId := room.FindExitByName(exitName)
	if name == `` {
		return FindExitResult{Found: false}
	}
	return FindExitResult{
		ExitName: name,
		RoomId:   roomId,
		Found:    true,
	}
}
