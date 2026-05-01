package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
)

func resolveFoldAnchor(actor actions.Actor) {
	char := actor.GetCharacter()
	if char == nil {
		return
	}
	roomId := char.RoomId
	char.SetMiscData("fold-anchor-room", roomId)

	actor.SendText(`A Chrysalis anchor locks into place here. ` +
		`Cast <ansi fg="command">fold-recall</ansi> from elsewhere to return.`)

	actor.SendRoomText(fmt.Sprintf(
		`A faint shimmer marks where <ansi fg="username">%s</ansi> has set an anchor.`,
		actor.GetName()), true)
}
