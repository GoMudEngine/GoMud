package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/conversationadapter"
	"github.com/GoMudEngine/GoMud/internal/conversations"
	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/relationships"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// unlockExit performs the shared tail of a successful exit unlock: it tells the
// actor, narrates to the room, plays the unlock sound, and clears the lock on
// both the caller's local exitInfo copy and the room's real exit.
//
// The three unlock paths in Go() (known lockpick sequence, key ring, and a
// loose key in the backpack) each repeated this exact sequence, differing only
// in the two messages — so a change to one, e.g. the sound or the SetExitLock
// call, silently diverged the others.
//
// exitInfo is a pointer because GetExitInfo returns a copy and Lock.SetUnlocked
// has a pointer receiver: the backpack path re-reads exitInfo.Lock.IsLocked()
// afterwards to decide whether to show the failure message, so the local copy
// must reflect the unlock.
func unlockExit(user *users.UserRecord, room *rooms.Room, exitName string, exitInfo *exit.RoomExit, playerMsg, roomMsg string) {
	user.SendText(messaging.CategorySystem, playerMsg)
	room.SendTextVisual(messaging.CategoryMobEmote, roomMsg, user.UserId)
	room.PlaySound(`change`, `other`)
	exitInfo.Lock.SetUnlocked()
	room.SetExitLock(exitName, false)
}

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

	if user.Character.IsInCombat() {
		// Always allow movement out of the death recovery room —
		// stale aggro must never trap a player in the Shadow Realm.
		// Use GetOriginalRoom() because the shadow realm is an ephemeral
		// copy — the actual room ID won't match the config value.
		deathRoom := int(configs.GetSpecialRoomsConfig().DeathRecoveryRoom)
		actualRoom := rooms.GetOriginalRoom(user.Character.RoomId)
		if actualRoom != deathRoom {
			user.SendText(messaging.CategorySystem, "You can't do that! You are in combat!")
			return true, nil
		}
		// Force-clear the stale aggro so it doesn't follow them out.
		user.Character.EndAggro()
	}

	// Block movement during quest sequences (e.g., Awakening Rite ceremony)
	if lockMsg, ok := user.GetTempData(`questSequenceLock`).(string); ok && lockMsg != "" {
		user.SendText(messaging.CategorySystem, lockMsg)
		return true, nil
	}

	// Movement cancels crafting and salvaging — route through Activity
	// machine for both. Casting is NOT interrupted by movement (policy).
	if user.Character.Activity != nil && !user.Character.Activity.IsFree() {
		switch user.Character.Activity.State() {
		case activity.Crafting:
			_ = user.Character.Activity.TransitionToFree(state.TransitionReason{
				Trigger: activity.TriggerMovementInterrupt,
				Actor:   state.ActorRef{UserId: user.UserId},
			})
			user.SendText(messaging.CategorySystem, `<ansi fg="red">Your movement interrupts your crafting.</ansi>`)
		case activity.Salvaging:
			_ = user.Character.Activity.TransitionToFree(state.TransitionReason{
				Trigger: activity.TriggerMovementInterrupt,
				Actor:   state.ActorRef{UserId: user.UserId},
			})
			user.SendText(messaging.CategorySystem, `<ansi fg="red">Your movement interrupts your salvaging.</ansi>`)
		}
	}
	// If has a buff that prevents combat, skip the player
	if user.Character.HasBuffFlag(buffs.NoMovement) {
		user.SendText(messaging.CategorySystem, "You can't do that!")
		return true, nil
	}

	c := configs.GetTextFormatsConfig()

	// Check both the buff flag (set by event queue on next tick) and the
	// misc-data flag (set synchronously by sneak command). This handles
	// the case where the player sneaks then immediately moves before the
	// buff event processes.
	isSneaking := user.Character.IsHidden()
	if !isSneaking {
		if sneakFlag, ok := user.Character.GetMiscData(`sneaking`).(bool); ok && sneakFlag {
			isSneaking = true
		}
	}

	handled := false

	if exitName != `` {

		if user.Character.IsDisabled() {
			user.SendText(messaging.CategorySystem, "You can't do that — you're dead. Hold on; you'll be pulled back to safety.")
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
				user.SendText(messaging.CategorySystem, "You're too encumbered to move (<ansi fg=\"command\">help encumbrance</ansi>)!")
			} else {
				user.SendText(messaging.CategorySystem, "You're too tired to move (slow down)!")
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
		if mutations.IsFlying(user.Character.Mutations) {
			// Winged Flight glides over terrain — movement barely tires you.
			staminaCost = int(float64(staminaCost) * float64(configs.GetBalanceConfig().FlightMoveStaminaMult))
			if staminaCost < 1 {
				staminaCost = 1
			}
		}
		if !user.Character.DeductStamina(staminaCost) {
			user.SendText(messaging.CategorySystem, "You're too exhausted to move! Rest and recover your stamina.")
			// Refund the action points since movement failed
			user.Character.ActionPoints += actionCost
			return true, nil
		}

		// Warn if stamina is getting low (< 25% of max)
		if user.Character.Stamina < user.Character.StaminaMax.Value/4 {
			user.SendText(messaging.CategorySystem, "<ansi fg=\"yellow\">You're feeling winded. Consider resting to recover your stamina.</ansi>")
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

				unlockExit(user, room, exitName, &exitInfo,
					`You know this lock well, you quickly pick it.`,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> quickly picks the lock on the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, exitName))

			} else if hasKey {

				unlockExit(user, room, exitName, &exitInfo,
					fmt.Sprintf(`You use the key on your key ring to unlock the <ansi fg="exit">%s</ansi> exit.`, exitName),
					fmt.Sprintf(`<ansi fg="username">%s</ansi> uses a key to unlock the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, exitName))

			} else {

				// check for a key item on their person
				if backpackKeyItm, hasBackpackKey := user.Character.FindKeyInBackpack(lockId); hasBackpackKey {

					itmSpec := backpackKeyItm.GetSpec()

					// Key entries look like:
					// "key-<roomid>-<exitname>": "<itemid>"
					user.Character.SetKey(`key-`+lockId, fmt.Sprintf(`%d`, backpackKeyItm.ItemId))
					user.Character.RemoveItem(backpackKeyItm)

					events.AddToQueue(events.ItemOwnership{
						UserId: user.UserId,
						Item:   backpackKeyItm,
						Gained: false,
					})

					unlockExit(user, room, exitName, &exitInfo,
						fmt.Sprintf(`You use your <ansi fg="item">%s</ansi> to unlock the <ansi fg="exit">%s</ansi> exit, and add it to your key ring for the future.`, itmSpec.Name, exitName),
						fmt.Sprintf(`<ansi fg="username">%s</ansi> uses a key to unlock the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, exitName))
				}

				if exitInfo.Lock.IsLocked() {
					user.SendText(messaging.CategorySystem, `There's a lock preventing you from going that way. You'll need a <ansi fg="item">Key</ansi> or to <ansi fg="command">pick</ansi> the lock with <ansi fg="item">lockpicks</ansi>.`)
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
			user.SendText(messaging.CategoryRoomDescription, exitInfo.ExitMessage)
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
			user.SendText(messaging.CategorySystem, "Oops, couldn't move there!")
		} else {

			// Quest engine: room_enter notification. For an ephemeral room
			// (tutorial antechamber, dungeons) the trigger must match the
			// TEMPLATE id -- the ephemeral id is generated and unauthorable.
			// OriginalRoomId returns the room's own id for non-ephemeral rooms,
			// so this is a no-op there. The bridge keeps the REAL room id so
			// npc_say and other actions resolve mobs/exits in the instance the
			// player is actually in.
			matchRoom, _ := rooms.OriginalRoomId(destRoom.RoomId)
			bridge := questengine.NewGameBridge(user, destRoom.RoomId)
			questengine.GetEngine().Notify("room_enter", questengine.EventDetails{
				UserId: user.UserId,
				RoomId: matchRoom,
			}, bridge, bridge)

			// Record this room as visited for fog-of-war web map. Use the
			// TEMPLATE id (matchRoom) for ephemeral rooms -- otherwise the raw,
			// unauthorable instance id (e.g. 1000000000) is permanently baked
			// into the saved VisitedRooms set and leaks onto the Zone.Map
			// snapshot. OriginalRoomId returns the room's own id for
			// non-ephemeral rooms, so this is a no-op there.
			user.Character.MarkRoomVisited(destRoom.Zone, matchRoom)

			// Tell the player they are moving
			if isSneaking {
				user.SendText(messaging.CategoryRoomExit,
					fmt.Sprintf(string(c.ExitRoomMessageWrapper),
						fmt.Sprintf(`You <ansi fg="black-bold">sneak</ansi> towards the <ansi fg="exit">%s</ansi> exit.`, exitName),
					))
			} else {
				user.SendText(messaging.CategoryRoomExit,
					fmt.Sprintf(string(c.ExitRoomMessageWrapper),
						fmt.Sprintf(`You head towards the <ansi fg="exit">%s</ansi> exit.`, exitName),
					))

				// Tell the old room they are leaving
				if user.Character.Pet.Exists() {

					room.SendTextVisual(messaging.CategoryRoomExit,
						fmt.Sprintf(string(c.ExitRoomMessageWrapper),
							fmt.Sprintf(`<ansi fg="username">%s</ansi> and %s leave towards the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, user.Character.Pet.DisplayName(), exitName),
						),
						user.UserId)

				} else {
					room.SendTextVisual(messaging.CategoryRoomExit,
						fmt.Sprintf(string(c.ExitRoomMessageWrapper),
							fmt.Sprintf(`<ansi fg="username">%s</ansi> leaves towards the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, exitName),
						),
						user.UserId)
				}

				// Tell everyone if the pet is following
				if user.Character.Pet.Exists() {

					user.SendText(messaging.CategorySystem, fmt.Sprintf(`%s follows you.`, user.Character.Pet.DisplayName()))

					destRoom.SendText(messaging.CategoryRoomEntry,
						fmt.Sprintf(string(c.ExitRoomMessageWrapper),
							fmt.Sprintf(`<ansi fg="username">%s</ansi> and %s enters from <ansi fg="exit">%s</ansi>.`, user.Character.Name, user.Character.Pet.DisplayName(), exitName),
						),
						user.UserId)

				} else {

					// Tell the new room they have arrived
					destRoom.SendText(messaging.CategoryRoomEntry,
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
								partyUser.SendText(messaging.CategorySystem, `You follow the party leader.`)
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
					// Companions interrupt casting to follow owner — Activity
					// cascade (Activity_Cascades.go movement interrupt) handles
					// the machine transition; no direct field manipulation needed.
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
				if !shadowP.Character.IsHidden() {
					continue
				}
				if !shadowIsTargetingUser(shadowP, user.UserId) {
					continue
				}
				// Buff-absent guard: misc data set but buff 87 gone means the
				// shadow expired or was cancelled out-of-band. Clear stale state
				// and skip the auto-follow so a dead/logged-off target can't drag
				// the player to an unexpected room.
				if !shadowP.Character.HasBuff(87) {
					shadowP.Character.SetMiscData("shadow-target-user", nil)
					shadowP.Character.SetMiscData("shadow-target-mob", nil)
					continue
				}
				// Shadower is in the old room and tracking the mover -- follow.
				shadowP.Command(rest)

				// After the move attempt, check if the shadower is still hidden.
				// The room-entry detection in go.go runs for the shadower's move,
				// so if they were spotted their hidden buff will already be gone.
				if !shadowP.Character.IsHidden() {
					endShadow(shadowP, "You've been spotted -- your shadow ends.")
					continue
				}

				// Target-specific detection roll: does the mover sense pursuit?
				if shadowDetectionRoll(shadowP, user, destRoom) {
					user.SendText(messaging.CategorySystem,
						"You sense someone following close behind you.")
				}
			}

			// Stealth detection: hidden player entering a room
			if isSneaking {
				// Build party exclusion set so allies don't expose the sneaker
				partyIds := make(map[int]bool)
				if p := parties.Get(user.UserId); p != nil {
					for _, uid := range p.GetMembers() {
						partyIds[uid] = true
					}
				}

				spotted := false

				// Check player observers. Sneak score is computed per-observer so
				// NightVision observers apply the correct light modifier.
				for _, pId := range destRoom.GetPlayers() {
					if pId == user.UserId || partyIds[pId] {
						continue
					}
					p := users.GetByUserId(pId)
					if p == nil {
						continue
					}
					sneakScore := actions.CalcSneakScoreVsObserver(user.Character, p.Character, destRoom)
					observerScore := actions.CalcSearchScore(p.Character)
					success, _, _, _ := dice.OpposedRollStat(sneakScore, observerScore)
					if !success {
						p.SendText(messaging.CategorySystem, fmt.Sprintf(
							`<ansi fg="username">%s</ansi> slips into the room but you notice them.`,
							user.Character.Name))
						spotted = true
						break
					}
				}

				// Check mob observers if not yet spotted
				if !spotted {
					for _, mId := range destRoom.GetMobs() {
						mob := mobs.GetInstance(mId)
						if mob == nil {
							continue
						}
						sneakScore := actions.CalcSneakScoreVsObserver(user.Character, &mob.Character, destRoom)
						observerScore := actions.CalcSearchScore(&mob.Character)
						success, _, _, _ := dice.OpposedRollStat(sneakScore, observerScore)
						if !success {
							spotted = true
							break
						}
					}
				}

				if spotted {
					// Drive the Awareness FSM out of Hidden — the mirror
					// cascade in Awareness_Cascades.go handles
					// CancelBuffsWithFlag(buffs.Hidden) and clears the
					// hidden state. Calling CancelBuffsWithFlag directly
					// here would expire buff 9 but leave the FSM in
					// Hidden, so IsHidden() would still return true and
					// the next attack would still surprise-strike.
					_ = user.Character.Awareness.TransitionToRevealing(
						state.TransitionReason{Trigger: awareness.TriggerObserverSearch})
					user.Character.SetMiscData(`sneaking`, nil)
					isSneaking = false
					// Intentionally silent — if the observer is itself hidden,
					// surfacing their name leaks information the player can't
					// see. The Hidden buff's end_user_text ("You no longer feel
					// sneaky.") on the next tick is sufficient signal that
					// stealth dropped.
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
					if hiddenP == nil || !hiddenP.Character.IsHidden() {
						continue
					}
					hiddenScore := actions.CalcSneakScoreVsObserver(hiddenP.Character, user.Character, destRoom)
					success, _, _, _ := dice.OpposedRollStat(observerScore, hiddenScore)
					if success {
						_ = hiddenP.Character.Awareness.TransitionToRevealing(
							state.TransitionReason{Trigger: awareness.TriggerObserverSearch})
						hiddenP.Character.SetMiscData(`sneaking`, nil)
						hiddenP.SendText(messaging.CategorySystem, fmt.Sprintf(
							"%s enters the room and notices you!", user.Character.Name))
						user.SendText(messaging.CategorySystem, fmt.Sprintf(
							`You notice <ansi fg="username">%s</ansi> lurking in the shadows.`,
							hiddenP.Character.Name))
					}
				}

				// Check hidden mobs
				for _, mId := range destRoom.GetMobs(rooms.FindAll) {
					mob := mobs.GetInstance(mId)
					if mob == nil || !mob.Character.IsHidden() {
						continue
					}
					hiddenScore := actions.CalcSneakScoreVsObserver(&mob.Character, user.Character, destRoom)
					success, _, _, _ := dice.OpposedRollStat(observerScore, hiddenScore)
					if success {
						_ = mob.Character.Awareness.TransitionToRevealing(
							state.TransitionReason{Trigger: awareness.TriggerObserverSearch})
						user.SendText(messaging.CategorySystem, fmt.Sprintf(
							`You notice <ansi fg="mobname">%s</ansi> lurking in the shadows!`,
							mob.Character.Name))
						destRoom.SendText(messaging.CategorySystem, fmt.Sprintf(
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

					if !mob.Character.IsInCombat() || mob.Character.EngagedTarget().UserId != user.UserId {
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
					if mob.Character.IsInCombat() {
						continue
					}
					if mob.Character.IsCharmed() {
						continue
					}
					// Non-combatants never aggro on entry even if their
					// group has been flagged hostile by prior attacks.
					if mob.IsNonCombatant() {
						continue
					}

					if !mob.AutoAggro { // Is it automatically hostile?
						continue
					}

					// Hidden mobs attack silently — no "notices you" message.
					// They still trigger lookfortrouble for the surprise attack.
					if !mob.Character.IsHidden() {
						if destRoom.GetVisibility() >= 1 || user.Character.HasFlagFromAnySource(buffs.NightVision) {
							user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> notices you as you enter!`, mob.Character.Name))
						} else {
							user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Something notices you in the darkness!</ansi>`)
						}
					}

					mob.Command(`lookfortrouble`, 4)

				}

			}

			// Chunk 3.3: a player arriving with a light source wakes any
			// sleepers in the destination room. False positives possible if
			// the room was already lit; acceptable for chunk 3.3 scope (most
			// NPC sleep rooms are dim/dark indoors).
			if user.Character.HasFlagFromAnySource(buffs.EmitsLight) {
				for _, otherUserId := range destRoom.GetPlayers() {
					if otherUserId == user.UserId {
						continue
					}
					if other := users.GetByUserId(otherUserId); other != nil &&
						other.Character.HasBuffFlag(buffs.Sleeping) {
						other.Character.CancelBuffsWithFlag(buffs.Sleeping)
						mobs.OnSleeperWoken(other.Character)
					}
				}
				for _, mobInstId := range destRoom.GetMobs() {
					if m := mobs.GetInstance(mobInstId); m != nil &&
						m.Character.HasBuffFlag(buffs.Sleeping) {
						m.Character.CancelBuffsWithFlag(buffs.Sleeping)
						mobs.OnSleeperWoken(&m.Character)
					}
				}
			}

			// 5a: NPC greetings. The first unoccupied NPC with an authored
			// greeting for its current mood welcomes the arriving player —
			// once per mob instance per player per boot, at most one greeting
			// per entry, and never for a hidden player: being hailed by name
			// would silently defeat stealth. IsHidden() is checked fresh here
			// rather than reusing move-time isSneaking, because a reveal
			// during the move (spotted by a hidden-mob check) should restore
			// the greeting. Runs before the conversation boost so the player
			// is welcomed before NPCs start talking amongst themselves.
			if !user.Character.IsHidden() {
				for _, greeterInstId := range destRoom.GetMobs() {
					gMob := mobs.GetInstance(greeterInstId)
					if gMob == nil {
						continue
					}
					if dialogue.HasGreeted(greeterInstId, user.UserId) {
						continue
					}
					if !conversations.IsFullyIdle(conversationadapter.AdaptMob(gMob)) {
						continue
					}
					df := dialogue.Load(int(gMob.MobId), gMob.Zone)
					if df == nil || len(df.Greetings) == 0 {
						continue
					}
					text, ok := dialogue.PickGreeting(df.Greetings, dialogue.GetMood(greeterInstId, df.DefaultMood))
					if !ok {
						continue
					}
					dialogue.MarkGreeted(greeterInstId, user.UserId)
					gMob.Command(`say ` + text)
					break // at most one greeting per entry (measured: 9 two-greeter rooms, none higher)
				}
			}

			// Chunk 3.6: player-arrival conversation boost. When the player
			// lands in a room with 2+ NPCs that are related and fully idle,
			// roll once at the boosted chance for an opening exchange so the
			// player is more likely to witness conversations rather than
			// always arriving between them.
			{
				cfg := configs.GetBalanceConfig()
				boostPct := int(cfg.ConversationPlayerArrivalBoostPct)
				if boostPct > 0 && util.Rand(100) < boostPct {
					if pairs := findRelateableEligiblePairsInRoom(destRoom); len(pairs) > 0 {
						p := pairs[util.Rand(len(pairs))]
						conversations.TryStartBetween(
							conversationadapter.AdaptMob(p.A),
							conversationadapter.AdaptMob(p.B),
						)
					}
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
			user.SendText(messaging.CategorySystem, "You're bumping into walls.")

			// Send GMCP message
			if f, ok := GetExportedFunction(`SendGMCPEvent`); ok {
				if gmcpSendFunc, ok := f.(func(int, string, any)); ok { // make sure the func definition is `func(int, string, any)`
					gmcpSendFunc(user.UserId, `Room.WrongDir`, fmt.Sprintf(`"%s"`, rest))
				}
			}

			if !user.Character.IsHidden() {

				room.SendTextVisual(messaging.CategoryMobEmote,
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

// relateableMobPair holds an unordered pair of mobs that share a
// relationship edge and are both eligible to start a conversation.
type relateableMobPair struct {
	A *mobs.Mob
	B *mobs.Mob
}

// findRelateableEligiblePairsInRoom enumerates all unordered (a, b) mob
// pairs in the room where both mobs:
//   - have a relationship edge per chunk 1.6 (relationships.AreRelated)
//   - are not in combat and have no pending aggro
//   - are not sleeping
//   - have no in-flight path step
//   - are not already in a conversation (partner_id MiscData check)
//   - are not on conversation cooldown
func findRelateableEligiblePairsInRoom(room *rooms.Room) []relateableMobPair {
	if room == nil {
		return nil
	}
	mobIds := room.GetMobs()
	if len(mobIds) < 2 {
		return nil
	}

	mobList := make([]*mobs.Mob, 0, len(mobIds))
	for _, id := range mobIds {
		m := mobs.GetInstance(id)
		if m == nil {
			continue
		}
		if m.Character.Aggro != nil || m.Character.IsInCombat() {
			continue
		}
		if m.Character.HasBuffFlag(buffs.Sleeping) {
			continue
		}
		if m.Path.Len() > 0 || m.Path.Current() != nil {
			continue
		}
		if pid, ok := m.Character.GetMiscData(conversations.MiscDataPartnerId).(int); ok && pid > 0 {
			continue
		}
		if until, ok := m.Character.GetMiscData(conversations.MiscDataCooldownUntilRound).(uint64); ok &&
			uint64(util.GetRoundCount()) < until {
			continue
		}
		mobList = append(mobList, m)
	}

	var out []relateableMobPair
	for i := 0; i < len(mobList); i++ {
		for j := i + 1; j < len(mobList); j++ {
			a, b := mobList[i], mobList[j]
			if relationships.AreRelated(int(a.MobId), int(b.MobId)) {
				out = append(out, relateableMobPair{A: a, B: b})
			}
		}
	}
	return out
}
