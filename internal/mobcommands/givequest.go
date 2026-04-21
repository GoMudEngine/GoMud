package mobcommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Expected format is:
// givequest 1-start
// or
// givequest 1-start Say has anyone seen my locket?
// The second message will only be executed if the quest is successfully given.
func GiveQuest(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Don't bother if no players are present
	if room.PlayerCt() < 1 {
		return true, nil
	}

	parts := strings.SplitN(rest, " ", 2)

	questToken := parts[0]
	targetUser := ``

	if len(parts) > 1 {
		targetUser = parts[1]
	}

	if targetUser != `` {
		target, err := actions.ResolveTargetActor(room, targetUser)
		if err == nil && target.IsPlayer() {
			events.AddToQueue(events.Quest{
				UserId:     target.GetUserId(),
				QuestToken: questToken,
			})
		}
	} else {
		for _, pId := range room.GetPlayers() {

			events.AddToQueue(events.Quest{
				UserId:     pId,
				QuestToken: questToken,
			})

		}
	}

	return true, nil
}
