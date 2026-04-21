package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func resolvePurgeAffliction(user *users.UserRecord, target *users.UserRecord) {
	if user == nil || target == nil {
		mudlog.Error("resolvePurgeAffliction", "error", "nil user or target")
		return
	}
	room := rooms.LoadRoom(user.Character.RoomId)

	if user.UserId != target.UserId {
		user.SendText(fmt.Sprintf(
			`<ansi fg="green">You direct purging energy towards <ansi fg="username">%s</ansi>.</ansi>`,
			target.Character.Name))
		target.SendText(fmt.Sprintf(
			`<ansi fg="green"><ansi fg="username">%s</ansi> purges the afflictions from your body.</ansi>`,
			user.Character.Name))
		if room != nil {
			sendVisualRoomText(room, fmt.Sprintf(
				`<ansi fg="username">%s</ansi> directs purging energy towards <ansi fg="username">%s</ansi>.`,
				user.Character.Name, target.Character.Name), user.UserId, target.UserId)
		}
	} else {
		user.SendText(`<ansi fg="green">You purge the afflictions from your body.</ansi>`)
		if room != nil {
			sendVisualRoomText(room, fmt.Sprintf(
				`<ansi fg="username">%s</ansi> purges their afflictions.`,
				user.Character.Name), user.UserId)
		}
	}

	target.Character.CancelBuffsWithFlag(buffs.Poison)
	target.Character.RemoveCondition(characters.ConditionPoisoned)
}
