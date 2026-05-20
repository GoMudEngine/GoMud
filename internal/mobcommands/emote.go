package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Emote(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Don't bother if no players are present
	if room.PlayerCt() < 1 {
		return true, nil
	}

	if len(rest) == 0 {
		sendRoomText(room, messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> emotes.`, mob.Character.Name))
		return true, nil
	}

	result := actions.Emote(rest)
	emoteText := rest
	if result.IsAlias {
		emoteText = result.AliasText
	}

	sendRoomText(room, messaging.CategoryMobEmote, actions.FormatEmoteText(mob.Character.Name, emoteText, "mobname"))

	return true, nil
}
