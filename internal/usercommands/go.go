package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Go(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	exitResult := actions.FindExit(room, rest)
	exitName, goRoomId := exitResult.ExitName, exitResult.RoomId

	// If no valid exit, check if it's a recognized cardinal direction (handled below
	// as "bumping into walls"). Otherwise return false so the dispatcher shows
	// "not recognized" instead of a confusing "can't do that in combat" message.
	isCardinal := false
	if exitName == `` {
		switch rest {
		case "north", "south", "east", "west", "up", "down",
			"northwest", "northeast", "southwest", "southeast":
			isCardinal = true
		}
		if !isCardinal {
			return false, nil
		}
	}

	if user.Character.Aggro != nil {
		// Always allow movement out of the death recovery room —
		// stale aggro must never trap a player in the Shadow Realm.
		// Use GetOriginalRoom() because the shadow realm is an ephemeral
		// copy — the actual room ID won't match the config value.
		deathRoom := int(configs.GetSpecialRoomsConfig().DeathRecoveryRoom)
		actualRoom := rooms.GetOriginalRoom(user.Character.RoomId)
		if actualRoom != deathRoom {
			user.SendText("You can't do that! You are in combat!")
			return true, nil
		}
		// Force-clear the stale aggro so it doesn't follow them out.
		user.Character.EndAggro()
	}

	// Block movement during quest sequences (e.g., Awakening Rite ceremony)
	if lockMsg, ok := user.GetTempData(`questSequenceLock`).(string); ok && lockMsg != "" {
		user.SendText(lockMsg)
		return true, nil
	}

	// Movement cancels crafting
	if user.Character.CraftingState != nil {
		user.Character.CraftingState = nil
		user.SendText(`<ansi fg="red">Your movement interrupts your crafting.</ansi>`)
	}

	// If has a buff that prevents combat, skip the player
	if user.Character.HasBuffFlag(buffs.NoMovement) {
		user.SendText("You can't do that!")
		return true, nil
	}

	c := configs.GetTextFormatsConfig()

	// Check both the buff flag (set by event queue on next tick) and the
	// misc-data flag (set synchronously by sneak command). This handles
	// the case where the player sneaks then immediately moves before the
	// buff event processes.
	isSneaking := user.Character.HasBuffFlag(buffs.Hidden)
	if !isSneaking {
		if sneakFlag, ok := user.Character.GetMiscData(`sneaking`).(bool); ok && sneakFlag {
			isSneaking = true
		}
	}

	handled := false

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
				room.SendTextVisual(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> quickly picks the lock on the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, exitName),
					user.UserId)

				room.PlaySound(`change`, `other`)

				exitInfo.Lock.SetUnlocked()
				room.SetExitLock(exitName, false)

			} else if hasKey {
				user.SendText(fmt.Sprintf(`You use the key on your key ring to unlock the <ansi fg="exit">%s</ansi> exit.`, exitName))
				room.SendTextVisual(
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
					room.SendTextVisual(
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

		behaviortree.TryRoomBehavior(room.RoomId, behaviortree.EventContext{
			EventType: "room_exit",
			UserId:    user.UserId,
			RoomId:    room.RoomId,
			Direction: exitName,
		})

		if err := rooms.MoveToRoom(user.UserId, destRoom.RoomId); err != nil {
			user.SendText("Oops, couldn't move there!")
		} else {


			// Quest engine: room_enter notification
			bridge := questengine.NewGameBridge(user, destRoom.RoomId)
			questengine.GetEngine().Notify("room_enter", questengine.EventDetails{
				UserId: user.UserId,
				RoomId: destRoom.RoomId,
			}, bridge, bridge)

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

					room.SendTextVisual(
						fmt.Sprintf(string(c.ExitRoomMessageWrapper),
							fmt.Sprintf(`<ansi fg="username">%s</ansi> and %s leave towards the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, user.Character.Pet.DisplayName(), exitName),
						),
						user.UserId)

				} else {
					room.SendTextVisual(
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
					// Companions interrupt casting to follow owner
					if mob.Character.CastingState != nil {
						mob.Character.CastingState = nil
					}
					mob.Command(rest)
				}
			}

			// Shadow follow -- check if any hidden player in the OLD room was
			// shadowing the mover (user). Auto-move them to the destination.
			for _, pId := range room.GetPlayers(rooms.FindAll) {
				if pId == user.UserId {
					continue
				}
				shadowP := users.GetByUserId(pId)
				if shadowP == nil {
					continue
				}
				if !shadowP.Character.HasBuffFlag(buffs.Hidden) {
					continue
				}
				if !shadowIsTargetingUser(shadowP, user.UserId) {
					continue
				}
				// Shadower is in the old room and tracking the mover -- follow.
				shadowP.Command(rest)

				// After the move attempt, check if the shadower is still hidden.
				// The room-entry detection in go.go runs for the shadower's move,
				// so if they were spotted their hidden buff will already be gone.
				if !shadowP.Character.HasBuffFlag(buffs.Hidden) {
					endShadow(shadowP, "You've been spotted -- your shadow ends.")
					continue
				}

				// Target-specific detection roll: does the mover sense pursuit?
				if shadowDetectionRoll(shadowP, user) {
					user.SendText(
						"You sense someone following close behind you.")
				}
			}

			// Stealth detection: hidden player entering a room
			if isSneaking {
				if spotted, spotterName := checkStealthDetection(user, destRoom); spotted {
					user.Character.CancelBuffsWithFlag(buffs.Hidden)
					user.Character.SetMiscData(`sneaking`, nil)
					isSneaking = false
					user.SendText(fmt.Sprintf(
						"You slip into the room but %s notices you.", spotterName))
				}
			}

			// Newcomer tries to spot hidden occupants (players and mobs)
			if !isSneaking {
				observerScore := actions.CalcSearchScore(user.Character)

				// Check hidden players
				for _, pId := range destRoom.GetPlayers() {
					if pId == user.UserId {
						continue
					}
					hiddenP := users.GetByUserId(pId)
					if hiddenP == nil || !hiddenP.Character.HasBuffFlag(buffs.Hidden) {
						continue
					}
					hiddenScore := calcSneakScore(hiddenP.Character)
					success, _, _, _ := dice.OpposedRollStat(observerScore, hiddenScore)
					if success {
						hiddenP.Character.CancelBuffsWithFlag(buffs.Hidden)
						hiddenP.Character.SetMiscData(`sneaking`, nil)
						hiddenP.SendText(fmt.Sprintf(
							"%s enters the room and notices you!", user.Character.Name))
						user.SendText(fmt.Sprintf(
							`You notice <ansi fg="username">%s</ansi> lurking in the shadows.`,
							hiddenP.Character.Name))
					}
				}

				// Check hidden mobs
				for _, mId := range destRoom.GetMobs(rooms.FindAll) {
					mob := mobs.GetInstance(mId)
					if mob == nil || !mob.Character.HasBuffFlag(buffs.Hidden) {
						continue
					}
					hiddenScore := calcSneakScore(&mob.Character)
					success, _, _, _ := dice.OpposedRollStat(observerScore, hiddenScore)
					if success {
						mob.Character.RemovePermaBuff(9)
						mob.Character.CancelBuffsWithFlag(buffs.Hidden)
						mob.Character.Buffs.RemoveBuff(9)
						mob.Character.Validate(true)
						user.SendText(fmt.Sprintf(
							`You notice <ansi fg="mobname">%s</ansi> lurking in the shadows!`,
							mob.Character.Name))
						destRoom.SendText(fmt.Sprintf(
							`<ansi fg="username">%s</ansi> spots <ansi fg="mobname">%s</ansi> hiding in the shadows!`,
							user.Character.Name, mob.Character.Name),
							user.UserId)
					}
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

				// Room behavior tree: fire room_enter for the destination room
				behaviortree.TryRoomBehavior(destRoom.RoomId, behaviortree.EventContext{
					EventType: "room_enter",
					UserId:    user.UserId,
					RoomId:    destRoom.RoomId,
					Direction: exitName,
				})

				// Behavior tree: notify mobs that a player entered
				if !isSneaking {
					for _, mobInstId := range destRoom.GetMobs(rooms.FindAll) {
						mob := mobs.GetInstance(mobInstId)
						if mob == nil || mob.Character.IsCharmed() {
							continue
						}
						behaviortree.TryMobBehavior(mobInstId, behaviortree.EventContext{
							EventType: "player_enter",
							UserId:    user.UserId,
							RoomId:    destRoom.RoomId,
						})
					}
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

					// Hidden mobs attack silently — no "notices you" message.
					// They still trigger lookfortrouble for the surprise attack.
					if !mob.Character.HasBuffFlag(buffs.Hidden) {
						if destRoom.GetVisibility() >= 1 || user.Character.HasFlagFromAnySource(buffs.NightVision) {
							user.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> notices you as you enter!`, mob.Character.Name))
						} else {
							user.SendText(`<ansi fg="yellow">Something notices you in the darkness!</ansi>`)
						}
					}

					mob.Command(`lookfortrouble`, 4)

				}

			}

			handled = true

			// Skip onEnter scripts when hidden — NPCs shouldn't react
			// to a player they can't see. Still show the room via Look.
			if isSneaking {
				Look(``, user, destRoom, events.CmdSecretly)
			} else {
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

				room.SendTextVisual(
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
