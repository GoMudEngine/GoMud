package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Train is disabled in Stage 3.5 — skills now improve only through use.
// Trainers give flavor advice instead.
func Train(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if len(room.SkillTraining) == 0 {
		user.SendText(`There is no trainer here.`)
		return false, nil
	}

	user.SendText(`The trainer studies you for a moment and says, "True mastery comes through practice, not instruction. Go use your skills in the world, and they will grow."`)

	return true, nil
}
