package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Taunt is a generic conviction attack with non-wolf flavor text. Used by
// tank archetypes (golems, elementals) where wolf-themed "howl" messaging
// would be incongruous. Mechanically identical to Howl; purely a flavor wrapper.
func Taunt(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if !mob.Character.IsInCombat() {
		return true, nil
	}

	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.ExecuteTaunt(actor)

	if !result.Executed {
		return true, nil
	}

	targetName := result.Target.Name

	var targetPlayer *users.UserRecord
	if result.Target.UserId > 0 {
		targetPlayer = users.GetByUserId(result.Target.UserId)
	}

	switch {
	case result.Fumble:
		sendAudioRoomText(room, mob,
			`Something bellows a challenge that breaks into a strangled gasp.`,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> bellows a challenge that breaks into a strangled gasp.`, mob.Character.Name))

	case result.Hit:
		if targetPlayer != nil {
			if canSeeInDark(targetPlayer, room) {
				targetPlayer.SendTextLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s thunderous challenge rattles your nerve! (<ansi fg="damage">%s</ansi>)`, mob.Character.Name, result.DmgDesc))
			} else {
				targetPlayer.SendTextLegacy(fmt.Sprintf(`A thunderous challenge rattles your nerve! (<ansi fg="damage">%s</ansi>)`, result.DmgDesc))
			}
		}
		sendAudioRoomText(room, mob,
			fmt.Sprintf(`Something bellows a thunderous challenge at <ansi fg="username">%s</ansi>!`, targetName),
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> bellows a thunderous challenge at <ansi fg="username">%s</ansi>!`, mob.Character.Name, targetName))

		// Stoic resolve messaging
		if result.CritDeflected {
			if targetPlayer != nil {
				targetPlayer.SendTextLegacy(
					`<ansi fg="green">The challenge rolls off you like rain from stone — you are unmoved.</ansi>`)
			}
		} else if result.Deflected {
			if targetPlayer != nil {
				targetPlayer.SendTextLegacy(
					`<ansi fg="green">You set your jaw and weather the challenge.</ansi>`)
			}
		}

	default: // miss
		if targetPlayer != nil {
			if canSeeInDark(targetPlayer, room) {
				targetPlayer.SendTextLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> bellows a challenge, but you brush it off.`, mob.Character.Name))
			} else {
				targetPlayer.SendTextLegacy(`Something bellows a challenge, but you brush it off.`)
			}
		}
		sendAudioRoomText(room, mob,
			fmt.Sprintf(`Something bellows a challenge at <ansi fg="username">%s</ansi>, but they shrug it off.`, targetName),
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> bellows a challenge at <ansi fg="username">%s</ansi>, but they shrug it off.`, mob.Character.Name, targetName))
	}

	return true, nil
}
