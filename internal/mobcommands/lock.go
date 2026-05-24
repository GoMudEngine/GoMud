package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Lock locks a container in the mob's current room. The mob must carry a
// matching key item in its backpack. Keyring bookkeeping (user.Character.SetKey)
// is player-only and deliberately omitted here.
func Lock(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 1 {
		return true, nil
	}

	containerName := room.FindContainerByName(args[0])
	if containerName == `` {
		return true, nil
	}

	container := room.Containers[containerName]

	if container.Lock.IsLocked() {
		// Already locked — nothing to do, return silently.
		return true, nil
	}

	lockId := fmt.Sprintf(`%d-%s`, room.RoomId, containerName)
	keyItm, hasKey := mob.Character.FindKeyInBackpack(lockId)

	if !hasKey {
		return true, nil
	}

	_ = keyItm // key is kept in inventory; mobs don't consume keys

	container.Lock.SetLocked()
	room.Containers[containerName] = container

	room.PlaySound(`change`, `other`)

	room.SendTextVisual(messaging.CategoryMobEmote,
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> uses a key to lock the <ansi fg="container">%s</ansi>.`,
			mob.Character.Name, containerName),
	)

	return true, nil
}
