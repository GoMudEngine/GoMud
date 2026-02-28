package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
Stealth Skill
Level 1 - Sneak: enter a hidden state outside of combat
*/
func Sneak(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Stealth)

	// If they don't have the skill, act like it's not a valid command
	if skillLevel < 1 {
		return false, nil
	}

	// Already hidden
	if user.Character.HasBuffFlag(buffs.Hidden) {
		user.SendText("You're already hidden!")
		return true, nil
	}

	if user.Character.Aggro != nil {
		user.SendText("You can't do that while in combat!")
		return true, nil
	}

	if room := rooms.LoadRoom(user.Character.RoomId); room != nil && !room.IsCalm() {
		user.SendText("You can only do that in calm rooms!")
		return true, nil
	}

	user.AddBuff(9, `skill`)

	// Fire an event that a skill has been used
	events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: skills.Stealth, Details: `sneak`})

	return true, nil
}
