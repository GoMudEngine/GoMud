package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Flee makes a mob disengage from combat and move to a random adjacent room.
func Flee(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// If in combat, clear aggro
	if mob.Character.Aggro != nil {
		mob.Character.EndAggro()
	}

	// Get a random exit (skips secret and locked exits)
	exitName, _ := room.GetRandomExit()
	if exitName == "" {
		// Cornered — no exits available
		return true, nil
	}

	// Send flee message before moving
	sendRoomText(room,
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> flees!`, mob.Character.Name))

	// Move via the existing Go command
	Go(exitName, mob, room)

	return true, nil
}
