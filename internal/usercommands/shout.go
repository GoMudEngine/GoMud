package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Shout(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Muted {
		user.SendText(messaging.CategoryWarning, `You are <ansi fg="alert-5">MUTED</ansi>. You can only send <ansi fg="command">whisper</ansi>'s to Admins and Moderators.`)
		return true, nil
	}

	// Shouting is a noisy action — reveal if hidden.
	if user.Character.IsHidden() {
		user.Character.Awareness.TransitionToRevealing(state.TransitionReason{
			Trigger:  awareness.TriggerNoisyAction,
			Metadata: map[string]any{"command": "shout"},
		})
	}

	isSneaking := user.Character.IsHidden()
	isDrunk := user.Character.HasBuffFlag(buffs.Drunk)

	rest = strings.ToUpper(rest)

	if isDrunk {
		// modify the text to look like it's the speech of a drunk person
		rest = drunkify(rest)
	}

	if isSneaking {
		msg := fmt.Sprintf(`someone shouts, "<ansi fg="yellow">%s</ansi>"`, rest)
		room.SendTextCommunication(util.SplitStringNL(msg, 80), user.UserId)
	} else {
		msg := fmt.Sprintf(`<ansi fg="username">%s</ansi> shouts, "<ansi fg="yellow">%s</ansi>"`, user.Character.Name, rest)
		room.SendTextCommunication(util.SplitStringNL(msg, 80), user.UserId)
	}

	// Standard, temporary, and mutator-added exits, each neighbour once.
	room.ForEachAdjacentRoom(func(otherRoom *rooms.Room, sourceExit string) {
		otherRoom.SendTextCommunication(fmt.Sprintf(`Someone shouts from the <ansi fg="exit">%s</ansi> direction, "<ansi fg="yellow">%s</ansi>"`, sourceExit, rest), user.UserId)
	})

	selfMsg := fmt.Sprintf(`You shout, "<ansi fg="yellow">%s</ansi>"`, rest)
	user.SendText(messaging.CategoryShout, util.SplitStringNL(selfMsg, 80))

	// Chunk 3.3: shout wakes sleepers in the same room. Same-room only —
	// adjacent-room sound propagation is out of scope.
	for _, otherUserId := range room.GetPlayers() {
		if otherUserId == user.UserId {
			continue
		}
		if other := users.GetByUserId(otherUserId); other != nil {
			if other.Character.HasBuffFlag(buffs.Sleeping) {
				other.Character.CancelBuffsWithFlag(buffs.Sleeping)
				mobs.OnSleeperWoken(other.Character)
			}
		}
	}
	for _, mobInstanceId := range room.GetMobs() {
		if m := mobs.GetInstance(mobInstanceId); m != nil {
			if m.Character.HasBuffFlag(buffs.Sleeping) {
				m.Character.CancelBuffsWithFlag(buffs.Sleeping)
				mobs.OnSleeperWoken(&m.Character)
			}
		}
	}

	return true, nil
}
