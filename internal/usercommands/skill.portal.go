package usercommands

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
Portal Skill
Level 1 - Teleport back to town square
Level 2 - Teleport back to the root of the area you are in
Level 3 - Set a new destination for your portal teleportation
Level 4 - Create a physical portal that you can share with players, or return through.
*/
func Portal(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	c := configs.GetSpecialRoomsConfig()

	if user.Character.RoomId == int(c.DeathRecoveryRoom) {
		return false, errors.New(`portal command ignored in death recovery`)
	}

	// This is a hack because using "portal" to enter an existing portal is very common
	if rest == `` {
		if handled, err := Go(`portal`, user, room, flags); handled {
			return handled, err
		}
	}

	skillLevel := user.Character.GetSkillLevel(skills.Portal)

	if skillLevel == 0 {
		user.SendText("You don't know how to portal.")
		return true, errors.New(`you don't know how to portal`)
	}

	if !user.Character.TimerExpired(`jail`) {
		user.SendText("Your portal is blocked here.")
		return true, errors.New(`Your portal is blocked here`)
	}

	// Establish the default portal location
	portalTargetRoomId := rooms.StartRoomIdAlias // Defaults to Start Room

	if skillLevel >= 2 { // Defaults to root of current zone
		portalTargetRoomId, _ = rooms.GetZoneRoot(user.Character.Zone)
		if portalTargetRoomId == int(c.DeathRecoveryRoom) {
			portalTargetRoomId = int(c.StartRoom) // If they are in the holding zone, send htem back to start
		}
	}

	if skillLevel >= 3 { // Defaults to wherever the player last set it
		portalSetting := user.Character.GetSetting("portal")
		if portalSetting != "" {
			if settingRoomId, err := strconv.Atoi(portalSetting); err == nil {
				portalTargetRoomId = settingRoomId
			}
		}
	}

	portalLifeInSeconds := user.Character.Stats.Willpower.ValueAdj * 10 // 0 willpower = 60 seconds, 100 willpower = 1060 seconds
	if portalLifeInSeconds < 0 {
		portalLifeInSeconds = 0
	}
	portalLifeInSeconds += 60

	portalLifeInMinutes := int(math.Floor(float64(portalLifeInSeconds) / 60))
	if portalLifeInMinutes < 1 {
		portalLifeInMinutes = 1
	}

	// If no argument supplied, is a direct teleport.
	if rest == "" {

		if user.Character.Aggro != nil {
			user.SendText("You can't do that! You are in combat!")
			return true, nil
		}

		if !user.Character.TryCooldown(skills.Portal.String(), "1 real minute") {
			user.SendText(
				fmt.Sprintf("You need to wait %d more rounds to use that skill again.", user.Character.GetCooldown(skills.Portal.String())),
			)
			return true, errors.New(`you're doing that too often`)
		}

		// Fire an event that a skill has been used
		events.AddToQueue(events.SkillUsed{user.UserId, skills.Portal, ``})

		// move to portalTargetRoomId
		if err := rooms.MoveToRoom(user.UserId, portalTargetRoomId); err == nil {
			user.SendText("You draw a quick symbol in the air with your finger, and the world warps around you. You seem to have moved.")
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> draws a quick symbol in the air, and is sucked into a portal!`, user.Character.Name),
				user.UserId,
			)
			newRoom := rooms.LoadRoom(portalTargetRoomId)
			newRoom.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> suddenly pops into existence!`, user.Character.Name),
				user.UserId,
			)
		} else {
			user.SendText("Oops, portal sad!")
		}
		return true, nil
	}

	if skillLevel >= 3 {

		if rest == "set" {

			// Fire an event that a skill has been used
			events.AddToQueue(events.SkillUsed{user.UserId, skills.Portal, `set`})

			user.Character.SetSetting("portal", strconv.Itoa(user.Character.RoomId))

			user.SendText("You enscribe a glowing pentagram on the ground with your finger, which then fades away. Your portals now lead to this area.")
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> enscribes a glowing pentagram on the ground with their finger. It quickly fades away.`, user.Character.Name),
				user.UserId,
			)
		}

		if rest == "unset" || rest == "clear" {

			// Fire an event that a skill has been used
			events.AddToQueue(events.SkillUsed{user.UserId, skills.Portal, `unset`})

			user.Character.SetSetting("portal", "")

			user.SendText("You draw an arcane symbol in the air with your finger. Your portals now lead to their default location.")
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> draws a shape in the air with their finger. The floor trembles mildly, but then returns to normal.`, user.Character.Name),
				user.UserId,
			)
		}
	}

	if skillLevel == 4 {

		if rest == "open" {

			// Load current room details
			targetRoom := rooms.LoadRoom(portalTargetRoomId)
			if targetRoom == nil {
				return false, fmt.Errorf(`room %d not found`, portalTargetRoomId)
			}

			if portalTargetRoomId == user.Character.RoomId {
				user.SendText("You can't open a portal to the room you're already in!")
				return true, nil
			}

			if !user.Character.TryCooldown(skills.Portal.String(), "1 real minute") {
				user.SendText(
					fmt.Sprintf("You need to wait %d more rounds to use that skill again.", user.Character.GetCooldown(skills.Portal.String())),
				)

				return true, errors.New(`you're doing that too often`)
			}

			glowingPortalColorized := colorpatterns.ApplyColorPattern(`glowing portal`, `glowing`)

			// Check whether they already have a portal open, and if so, shut it down.
			if currentPortal := user.Character.GetSetting("portal:open"); len(currentPortal) > 0 {

				// Data looks like {roomId1}:{roomId2}
				// Extract the two room id's
				portalRooms := strings.Split(currentPortal, ":")
				portalRoomId1, _ := strconv.Atoi(portalRooms[0])
				portalRoomId2, _ := strconv.Atoi(portalRooms[1])

				var oldPortalsRemoved bool = false

				if r1 := rooms.LoadRoom(portalRoomId1); r1 != nil {

					if tmpExit, found := r1.FindTemporaryExitByUserId(user.UserId); found {
						if r1.RemoveTemporaryExit(tmpExit) {
							r1.SendText(
								fmt.Sprintf("Suddenly, the %s before you snaps closed, leaving no evidence it ever existed.", glowingPortalColorized),
							)
							oldPortalsRemoved = true
						}
					}
				}

				if r2 := rooms.LoadRoom(portalRoomId2); r2 != nil {

					if tmpExit, found := r2.FindTemporaryExitByUserId(user.UserId); found {
						if r2.RemoveTemporaryExit(tmpExit) {
							r2.SendText(
								fmt.Sprintf("Suddenly, the %s before you snaps closed, leaving no evidence it ever existed.", glowingPortalColorized),
							)
							oldPortalsRemoved = true
						}
					}
				}

				if oldPortalsRemoved {
					user.SendText("Your old portals snap shut.")
				}

				user.Character.SetSetting("portal:open", "")
			}

			// Target = portalTargetRoomId
			// Current = user.Character.RoomId
			// At this point we have no open portals, we can create a new one.

			newPortalExitName := fmt.Sprintf("glowing portal from %s", user.Character.Name)
			newPortal := exit.TemporaryRoomExit{
				RoomId:  portalTargetRoomId,
				Title:   fmt.Sprintf(`%s from <ansi fg="username">%s</ansi>`, glowingPortalColorized, user.Character.Name),
				UserId:  user.UserId,
				Expires: fmt.Sprintf(`%d real minutes`, portalLifeInMinutes),
			}

			// Spawn a portal in the room that leads to the portal location
			if !room.AddTemporaryExit(newPortalExitName, newPortal) {
				user.SendText("Something went wrong. That's the problem with portal!")
				return true, fmt.Errorf("failed to add temporary exit to room")
			}
			user.SendText(
				fmt.Sprintf("You trace the shape of a doorway in front of you with your finger, which becomes a %s to another area.", glowingPortalColorized),
			)
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> traces the shape of a doorway with their finger, and a %s appears!`, user.Character.Name, glowingPortalColorized),
				user.UserId,
			)

			// Modify it for this room
			newPortal.RoomId = user.Character.RoomId

			if !targetRoom.AddTemporaryExit(newPortalExitName, newPortal) {
				user.SendText("Something went wrong. That's the problem with portal!")
				return true, fmt.Errorf("failed to add temporary exit to room")
			}

			targetRoom.SendText(
				fmt.Sprintf("A %s suddenly appears!", glowingPortalColorized),
			)

			// Fire an event that a skill has been used
			events.AddToQueue(events.SkillUsed{user.UserId, skills.Portal, `open`})

			user.Character.SetSetting("portal:open", fmt.Sprintf("%d:%d", user.Character.RoomId, portalTargetRoomId))
		}
	}

	return true, nil
}
