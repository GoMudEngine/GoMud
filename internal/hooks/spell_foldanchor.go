package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func resolveFoldAnchor(user *users.UserRecord) {
	roomId := user.Character.RoomId
	user.Character.SetMiscData("fold-anchor-room", roomId)
	user.SendText(`A Chrysalis anchor locks into place here. ` +
		`Cast <ansi fg="command">fold-recall</ansi> from elsewhere to return.`)
	if room := rooms.LoadRoom(roomId); room != nil {
		room.SendText(fmt.Sprintf(
			`A faint shimmer marks where <ansi fg="username">%s</ansi> has set an anchor.`,
			user.Character.Name), user.UserId)
	}
}
