package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Howl is a wolf conviction attack — a taunt reskin for mob use.
func Howl(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.Aggro == nil {
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
			`Something lets out a pitiful howl that trails off weakly.`,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> lets out a pitiful howl that trails off weakly.`, mob.Character.Name))

	case result.Hit:
		if targetPlayer != nil {
			if canSeeInDark(targetPlayer, room) {
				targetPlayer.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>'s menacing howl shakes your resolve! (<ansi fg="damage">%s</ansi>)`, mob.Character.Name, result.DmgDesc))
			} else {
				targetPlayer.SendText(fmt.Sprintf(`A menacing howl shakes your resolve! (<ansi fg="damage">%s</ansi>)`, result.DmgDesc))
			}
		}
		sendAudioRoomText(room, mob,
			fmt.Sprintf(`Something lets out a bone-chilling howl at <ansi fg="username">%s</ansi>!`, targetName),
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> throws back its head and lets out a bone-chilling howl at <ansi fg="username">%s</ansi>!`, mob.Character.Name, targetName))

	default: // miss
		if targetPlayer != nil {
			if canSeeInDark(targetPlayer, room) {
				targetPlayer.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> howls, but you steel yourself against the sound.`, mob.Character.Name))
			} else {
				targetPlayer.SendText(`Something howls, but you steel yourself against the sound.`)
			}
		}
		sendAudioRoomText(room, mob,
			fmt.Sprintf(`Something howls menacingly at <ansi fg="username">%s</ansi>, but it has no effect.`, targetName),
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> howls menacingly at <ansi fg="username">%s</ansi>, but it has no effect.`, mob.Character.Name, targetName))
	}

	return true, nil
}
