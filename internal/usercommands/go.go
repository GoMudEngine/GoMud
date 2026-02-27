package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Go(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Character.Aggro != nil {
		user.SendText("You can't do that! You are in combat!")
		return true, nil
	}

	// If has a buff that prevents combat, skip the player
	if user.Character.HasBuffFlag(buffs.NoMovement) {
		user.SendText("You can't do that!")
		return true, nil
	}

	c := configs.GetTextFormatsConfig()

	isSneaking := user.Character.HasBuffFlag(buffs.Hidden)

	handled := false

	exitName, goRoomId := room.FindExitByName(rest)

	if exitName != `` {

		if user.Character.IsDisabled() {
			user.SendText("You are unable to do that while downed.")
			return true, nil
		}

		actionCost := 10
		encumbered := false
		if user.Character.GetCarriedWeight() > user.Character.CarryCapacity() {
			actionCost = 50
			encumbered = true
		}

		if !user.Character.DeductActionPoints(actionCost) {

			if encumbered {
				user.SendText("You're too encumbered to move (<ansi fg=\"command\">help encumbrance</ansi>)!")
			} else {
				user.SendText("You're too tired to move (slow down)!")
				mudlog.Debug("No ActionPoints", "AP", user.Character.ActionPoints, "Needed", actionCost)
			}

			return true, nil
		}

		// Calculate stamina cost for movement
		// Get destination room biome for terrain difficulty
		destRoom := rooms.LoadRoom(goRoomId)
		if destRoom == nil {
			return false, fmt.Errorf(`room %d not found`, goRoomId)
		}

		// Get biome movement cost
		biome, _ := rooms.GetBiome(destRoom.Biome)
		terrainMultiplier := 1.0
		if biome != nil {
			terrainMultiplier = biome.GetMovementCost()
		}

		// Calculate and check stamina cost
		staminaCost := user.Character.GetMovementStaminaCost(terrainMultiplier)
		if !user.Character.DeductStamina(staminaCost) {
			user.SendText("You're too exhausted to move! Rest and recover your stamina.")
			// Refund the action points since movement failed
			user.Character.ActionPoints += actionCost
			return true, nil
		}

		// Warn if stamina is getting low (< 25% of max)
		if user.Character.Stamina < user.Character.StaminaMax.Value/4 {
			user.SendText("<ansi fg=\"yellow\">You're feeling winded. Consider resting to recover your stamina.</ansi>")
		}

		originRoomId := user.Character.RoomId

		exitInfo, _ := room.GetExitInfo(exitName)

		if exitInfo.Lock.IsLocked() {

			lockId := fmt.Sprintf(`%d-%s`, room.RoomId, exitName)

			hasKey, hasSequence := user.Character.HasKey(lockId, int(room.Exits[exitName].Lock.Difficulty))

			lockpickItm := items.Item{}
			// Only look for a lockpick kit if they know the sequence
			if hasSequence {
				for _, itm := range user.Character.GetAllBackpackItems() {
					if itm.GetSpec().Type == items.Lockpicks {
						lockpickItm = itm
						break
					}
				}
			}

			if lockpickItm.ItemId > 0 && hasSequence {

				user.SendText(`You know this lock well, you quickly pick it.`)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> quickly picks the lock on the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, exitName),
					user.UserId)

				room.PlaySound(`change`, `other`)

				exitInfo.Lock.SetUnlocked()
				room.SetExitLock(exitName, false)

			} else if hasKey {
				user.SendText(fmt.Sprintf(`You use the key on your key ring to unlock the <ansi fg="exit">%s</ansi> exit.`, exitName))
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> uses a key to unlock the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, exitName),
					user.UserId)

				room.PlaySound(`change`, `other`)

				exitInfo.Lock.SetUnlocked()
				room.SetExitLock(exitName, false)

			} else {

				// check for a key item on their person
				if backpackKeyItm, hasBackpackKey := user.Character.FindKeyInBackpack(lockId); hasBackpackKey {

					itmSpec := backpackKeyItm.GetSpec()

					room.PlaySound(`change`, `other`)

					user.SendText(fmt.Sprintf(`You use your <ansi fg="item">%s</ansi> to unlock the <ansi fg="exit">%s</ansi> exit, and add it to your key ring for the future.`, itmSpec.Name, exitName))
					room.SendText(
						fmt.Sprintf(`<ansi fg="username">%s</ansi> uses a key to unlock the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, exitName),
						user.UserId)

					// Key entries look like:
					// "key-<roomid>-<exitname>": "<itemid>"
					user.Character.SetKey(`key-`+lockId, fmt.Sprintf(`%d`, backpackKeyItm.ItemId))
					user.Character.RemoveItem(backpackKeyItm)

					events.AddToQueue(events.ItemOwnership{
						UserId: user.UserId,
						Item:   backpackKeyItm,
						Gained: false,
					})

					exitInfo.Lock.SetUnlocked()
					room.SetExitLock(exitName, false)
				}

				if exitInfo.Lock.IsLocked() {
					user.SendText(`There's a lock preventing you from going that way. You'll need a <ansi fg="item">Key</ansi> or to <ansi fg="command">pick</ansi> the lock with <ansi fg="item">lockpicks</ansi>.`)
					// Send GMCP message
					if f, ok := GetExportedFunction(`SendGMCPEvent`); ok {
						if gmcpSendFunc, ok := f.(func(int, string, any)); ok { // make sure the func definition is `func(int, string, any)`
							gmcpSendFunc(user.UserId, `Room.WrongDir`, fmt.Sprintf(`"%s"`, exitName))
						}
					}

					return true, nil
				}
			}

		}

		if exitInfo.ExitMessage != `` && !flags.Has(events.CmdIsRequeue) {
			user.SendText(exitInfo.ExitMessage)
			user.CommandFlagged(rest, flags|events.CmdIsRequeue|events.CmdBlockInputUntilComplete, 1)
			return true, nil
		}

		// destRoom already loaded above for stamina calculation
		// Grab the exit in the target room that leads to this room (if any)
		enterFromExit := destRoom.FindExitTo(room.RoomId)

		if len(enterFromExit) < 1 {
			enterFromExit = "somewhere"
		} else {

			// Entering through the other side unlocks this side
			exitInfo := destRoom.Exits[enterFromExit]
			if exitInfo.Lock.IsLocked() {
				exitInfo.Lock.SetUnlocked()
				destRoom.SetExitLock(enterFromExit, false)
			}

			enterFromExit = fmt.Sprintf(`the <ansi fg="exit">%s</ansi>`, enterFromExit)
		}

		if err := rooms.MoveToRoom(user.UserId, destRoom.RoomId); err != nil {
			user.SendText("Oops, couldn't move there!")
		} else {

			scripting.TryRoomScriptEvent(`onExit`, user.UserId, originRoomId)

			// Tell the player they are moving
			if isSneaking {
				user.SendText(
					fmt.Sprintf(string(c.ExitRoomMessageWrapper),
						fmt.Sprintf(`You <ansi fg="black-bold">sneak</ansi> towards the <ansi fg="exit">%s</ansi> exit.`, exitName),
					))
			} else {
				user.SendText(
					fmt.Sprintf(string(c.ExitRoomMessageWrapper),
						fmt.Sprintf(`You head towards the <ansi fg="exit">%s</ansi> exit.`, exitName),
					))

				// Tell the old room they are leaving
				if user.Character.Pet.Exists() {

					room.SendText(
						fmt.Sprintf(string(c.ExitRoomMessageWrapper),
							fmt.Sprintf(`<ansi fg="username">%s</ansi> and %s leave towards the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, user.Character.Pet.DisplayName(), exitName),
						),
						user.UserId)

				} else {
					room.SendText(
						fmt.Sprintf(string(c.ExitRoomMessageWrapper),
							fmt.Sprintf(`<ansi fg="username">%s</ansi> leaves towards the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, exitName),
						),
						user.UserId)
				}

				// Tell everyone if the pet is following
				if user.Character.Pet.Exists() {

					user.SendText(fmt.Sprintf(`%s follows you.`, user.Character.Pet.DisplayName()))

					destRoom.SendText(
						fmt.Sprintf(string(c.ExitRoomMessageWrapper),
							fmt.Sprintf(`<ansi fg="username">%s</ansi> and %s enters from <ansi fg="exit">%s</ansi>.`, user.Character.Name, user.Character.Pet.DisplayName(), exitName),
						),
						user.UserId)

				} else {

					// Tell the new room they have arrived
					destRoom.SendText(
						fmt.Sprintf(string(c.EnterRoomMessageWrapper),
							fmt.Sprintf(`<ansi fg="username">%s</ansi> enters from <ansi fg="exit">%s</ansi>.`, user.Character.Name, enterFromExit),
						),
						user.UserId)

				}

				destRoom.SendTextToExits(`You hear someone moving around.`, true, room.GetPlayers(rooms.FindAll)...)
			}

			if currentParty := parties.Get(user.UserId); currentParty != nil {

				if currentParty.IsLeader(user.UserId) {

					for _, partyMemberId := range currentParty.UserIds {
						if partyMemberId == user.UserId {
							continue
						}
						if partyUser := users.GetByUserId(partyMemberId); partyUser != nil {
							if partyUser.Character.RoomId == room.RoomId {
								partyUser.SendText(`You follow the party leader.`)
								partyUser.Command(rest)
							}
						}
					}

				}
			}

			for _, instId := range room.GetMobs(rooms.FindCharmed) {
				mob := mobs.GetInstance(instId)
				if mob == nil {
					continue
				}
				// They only follow if they're in the same room as the player
				if mob.Character.RoomId != originRoomId {
					continue
				}
				if mob.Character.IsCharmed(user.UserId) { // Charmed mobs follow
					mob.Command(rest)
				}
			}

			if !isSneaking {
				//
				// When leaving a room, mobs who were attacking may follow
				//
				mobInstanceIds := room.GetMobs(rooms.FindFightingPlayer)
				for _, mobInstanceId := range mobInstanceIds {
					mob := mobs.GetInstance(mobInstanceId)
					if mob == nil {
						continue
					}

					if mob.Character.Aggro == nil || mob.Character.Aggro.UserId != user.UserId {
						continue
					}

					speedDelta := mob.Character.Stats.Dexterity.ValueAdj - user.Character.Stats.Dexterity.ValueAdj
					if speedDelta < 1 {
						speedDelta = 1
					}

					// Chance that a mob follows the player
					targetVal := 20 + mob.Character.Stats.Charisma.ValueAdj + speedDelta

					roll := util.Rand(100)

					util.LogRoll(`Mob Follow`, roll, targetVal)

					if roll >= targetVal {
						continue
					}

					mob.Command(rest)

				}

				//
				// When entering a room, mobs might be waiting to attack
				//
				mobInstanceIds = destRoom.GetMobs(rooms.FindAll)
				for _, mobInstanceId := range mobInstanceIds {
					mob := mobs.GetInstance(mobInstanceId)
					if mob == nil {
						continue
					}
					if mob.Character.Aggro != nil {
						continue
					}
					if mob.Character.IsCharmed() {
						continue
					}

					isHostile := mob.Hostile // Is it automatically hostile?
					if !isHostile {
						for _, groupName := range mob.Groups {
							if mobs.IsHostile(groupName, user.UserId) {
								isHostile = true
								break
							}
						}
						if !isHostile { // is it still not hostile?
							continue
						}
					}

					user.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> notices you as you enter!`, mob.Character.Name))

					mob.Command(`lookfortrouble`, 4)

				}

			}

			handled = true

			if doLook, err := scripting.TryRoomScriptEvent(`onEnter`, user.UserId, destRoom.RoomId); err != nil || doLook {
				Look(``, user, destRoom, events.CmdSecretly)
			}

			room.PlaySound(`room-exit`, `movement`, user.UserId)
			destRoom.PlaySound(`room-enter`, `movement`, user.UserId)

		}

	}

	if !handled {

		if rest == "north" || rest == "south" || rest == "east" || rest == "west" || rest == "up" || rest == "down" || rest == "northwest" || rest == "northeast" || rest == "southwest" || rest == "southeast" {
			user.SendText("You're bumping into walls.")

			// Send GMCP message
			if f, ok := GetExportedFunction(`SendGMCPEvent`); ok {
				if gmcpSendFunc, ok := f.(func(int, string, any)); ok { // make sure the func definition is `func(int, string, any)`
					gmcpSendFunc(user.UserId, `Room.WrongDir`, fmt.Sprintf(`"%s"`, rest))
				}
			}

			if !user.Character.HasBuffFlag(buffs.Hidden) {

				room.SendText(
					fmt.Sprintf(string(c.ExitRoomMessageWrapper),
						fmt.Sprintf(`<ansi fg="username">%s</ansi> is bumping into walls.`, user.Character.Name),
					),
					user.UserId)
			}
			handled = true
		}

	}

	return handled, nil
}
