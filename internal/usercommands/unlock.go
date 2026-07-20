package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Unlock(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 1 {
		user.SendText(messaging.CategorySystem, "Unlock what?")
		return true, nil
	}

	containerName := room.FindContainerByName(args[0])
	if containerName != `` {
		if c, exists := room.Containers[containerName]; exists && c.Hidden {
			if user == nil || !user.Character.HasDiscovery(room.RoomId, containerName) {
				containerName = ``
			}
		}
	}
	exitName, _ := room.FindExitByName(args[0])

	if containerName != `` {

		container := room.Containers[containerName]

		if !container.Lock.IsLocked() {
			user.SendText(messaging.CategorySystem, "That's not locked.")
			return true, nil
		}

		lockId := fmt.Sprintf(`%d-%s`, room.RoomId, containerName)
		hasKey, _ := user.Character.HasKey(lockId, int(container.Lock.Difficulty))

		var backpackKeyItm items.Item = items.Item{}
		var hasBackpackKey bool = false
		if !hasKey {
			backpackKeyItm, hasBackpackKey = user.Character.FindKeyInBackpack(lockId)
		}

		if hasKey {
			container.Lock.SetUnlocked()
			room.Containers[containerName] = container

			room.PlaySound(`change`, `other`)

			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You use a key to unlock the <ansi fg="container">%s</ansi>.`, containerName))
			room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> uses a key to unlock the <ansi fg="container">%s</ansi>.`, user.Character.Name, containerName), user.UserId)
		} else if hasBackpackKey {

			itmSpec := backpackKeyItm.GetSpec()

			container.Lock.SetUnlocked()
			room.Containers[containerName] = container

			// Key entries look like:
			// "key-<roomid>-<exitname>": "<itemid>"
			user.Character.SetKey(`key-`+lockId, fmt.Sprintf(`%d`, backpackKeyItm.ItemId))
			user.Character.RemoveItem(backpackKeyItm)

			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   backpackKeyItm,
				Gained: false,
			})

			room.PlaySound(`change`, `other`)

			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You use your <ansi fg="item">%s</ansi> to unlock the <ansi fg="container">%s</ansi>, and add it to your key ring for the future.`, itmSpec.Name, containerName))
			room.SendTextVisual(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> uses a key to unlock the <ansi fg="container">%s</ansi>.`, user.Character.Name, containerName),
				user.UserId)
		} else {
			user.SendText(messaging.CategorySystem, `You do not have the key for that. Maybe you could <ansi fg="command">picklock</ansi> the lock.`)
		}

		return true, nil

	} else if exitName != `` {

		exitInfo, _ := room.GetExitInfo(exitName)

		if !exitInfo.Lock.IsLocked() {
			user.SendText(messaging.CategorySystem, "That's not locked.")
			return true, nil
		}

		lockId := fmt.Sprintf(`%d-%s`, room.RoomId, exitName)
		hasKey, _ := user.Character.HasKey(lockId, int(exitInfo.Lock.Difficulty))

		var backpackKeyItm items.Item = items.Item{}
		var hasBackpackKey bool = false
		if !hasKey {
			backpackKeyItm, hasBackpackKey = user.Character.FindKeyInBackpack(lockId)
		}

		if hasKey {
			exitInfo.Lock.SetUnlocked()
			room.SetExitLock(exitName, false)

			room.PlaySound(`change`, `other`)

			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You use a key to unlock the <ansi fg="exit">%s</ansi> lock.`, exitName))
			room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="username">%s</ansi> uses a key to unlock the <ansi fg="exit">%s</ansi> lock`, user.Character.Name, exitName), user.UserId)
		} else if hasBackpackKey {

			itmSpec := backpackKeyItm.GetSpec()

			exitInfo.Lock.SetUnlocked()
			room.SetExitLock(exitName, false)

			// Key entries look like:
			// "key-<roomid>-<exitname>": "<itemid>"
			user.Character.SetKey(`key-`+lockId, fmt.Sprintf(`%d`, backpackKeyItm.ItemId))
			user.Character.RemoveItem(backpackKeyItm)

			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   backpackKeyItm,
				Gained: false,
			})

			room.PlaySound(`change`, `other`)

			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You use your <ansi fg="item">%s</ansi> to unlock the <ansi fg="exit">%s</ansi> exit, and add it to your key ring for the future.`, itmSpec.Name, exitName))
			room.SendTextVisual(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> uses a key to unlock the <ansi fg="exit">%s</ansi> lock`, user.Character.Name, exitName),
				user.UserId)
		} else {
			user.SendText(messaging.CategorySystem, `You do not have the key for that. Maybe you could <ansi fg="command">picklock</ansi> the lock.`)
		}

		return true, nil

	}

	user.SendText(messaging.CategorySystem, "There is no such exit or container.")
	return true, nil

}
