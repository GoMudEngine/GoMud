package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Defuse(rest string, user *users.UserRecord,
	room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)
	if skillLevel < 3 {
		return false, nil
	}

	if rest == "" {
		user.SendText("Defuse what?")
		return true, nil
	}

	user.SendText("You don't detect any traps here.")
	return true, nil
}
