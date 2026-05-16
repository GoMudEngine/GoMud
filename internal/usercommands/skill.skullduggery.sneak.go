package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
Skullduggery Skill
Level 1 - Sneak: attempt to enter a hidden state outside of combat.
Uses an opposed roll against each observer in the room.
*/
func Sneak(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)

	// If they don't have the skill, act like it's not a valid command
	if skillLevel < 1 {
		return false, nil
	}

	// Can't sneak while crafting or otherwise occupied
	if !user.Character.IsFree() {
		user.SendText(`You are busy with something else.`)
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	sneakCooldownKey := skills.Skullduggery.String(`sneak`)

	// Check cooldown — only block if on cooldown from a prior failure
	if user.Character.GetCooldown(sneakCooldownKey) > 0 {
		user.SendText(fmt.Sprintf(
			"You need to wait %d more rounds before you can try that again.",
			user.Character.GetCooldown(sneakCooldownKey)))
		return true, nil
	}

	result := actions.Sneak(&actions.UserActor{User: user, Room: room})

	switch {
	case result.AlreadyHidden:
		user.SendText("You're already hidden!")
		return true, nil

	case result.InCombat:
		user.SendText("You can't do that while in combat!")
		return true, nil

	case result.SpottedByName != "":
		// Apply failure cooldown so the player can't spam sneak
		user.Character.TryCooldown(sneakCooldownKey,
			fmt.Sprintf(`%d rounds`, cfg.SneakFailCooldown))
		user.SendText(fmt.Sprintf(
			`You try to blend into the shadows but <ansi fg="mobname">%s</ansi> notices you.`,
			result.SpottedByName))

		// Progress the skill only when a roll actually happened
		if result.RollHappened {
			user.Character.CheckSkillProgression(string(skills.Skullduggery), user.UserId, 1.0)
		}
		return true, nil
	}

	// Success
	user.SendText(`You slip into the shadows.`)

	// Progress the skill only when a roll actually happened
	if result.RollHappened {
		user.Character.CheckSkillProgression(string(skills.Skullduggery), user.UserId, 1.0)
	}

	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.Skullduggery,
		Details: `sneak`,
	})

	return true, nil
}
