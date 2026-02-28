package usercommands

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type trackingInfo struct {
	Name            string
	Type            string // mob / user
	Strength        string
	NumericStrength float64
	ExitName        string
}

/*
Skill Track
Level 1 - Display the last player or mob to walk through here (not the currently player or current mobs)
Level 2 - Display all players and mobs to recently walk through here
Level 3 - Shows exit information for all tracked players or mobs
Level 4 - Specify a mob or username and every room you enter will tell you what exit they took.
*/
func Track(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Tracking)

	if skillLevel == 0 {
		user.SendText("You don't know how to track.")
		return true, errors.New(`you don't know how to track`)
	}

	currentMobs := room.GetMobs()
	currentUsers := room.GetPlayers()

	//
	// If no argument supplied
	// Handle skill level 1 and 2
	//
	if rest == `` {

		if !user.Character.TryCooldown(skills.Tracking.String(), "1 round") {
			user.SendText(
				fmt.Sprintf("You need to wait %d more rounds to use that skill again.", user.Character.GetCooldown(skills.Tracking.String())))
			return true, errors.New(`you're doing that too often`)
		}

		// Fire an event that a skill has been used
		events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: skills.Tracking, Details: ``})

		visitorData := make([]trackingInfo, 0)

		for mId, timeLeft := range room.Visitors(rooms.VisitorMob) {

			skip := false
			for _, currentRoomMobId := range currentMobs {
				if mId == currentRoomMobId {
					skip = true
					break
				}
			}

			checkMob := mobs.GetInstance(mId)
			if checkMob == nil {
				skip = true
			}

			if skip {
				continue
			}

			newTrackInfo := trackingInfo{
				Name:            checkMob.Character.Name,
				Type:            `mob`,
				Strength:        trailStrengthToString(timeLeft),
				NumericStrength: timeLeft,
			}

			if skillLevel >= 3 {
				newTrackInfo.ExitName = findExited(room, mId, rooms.VisitorMob)
			}

			if skillLevel == 1 {

				if len(visitorData) == 0 {
					visitorData = append(visitorData, newTrackInfo)
				} else if visitorData[0].NumericStrength < timeLeft {
					visitorData[0] = newTrackInfo
				}

				continue
			}

			visitorData = append(visitorData, newTrackInfo)
		}

		for uId, timeLeft := range room.Visitors(rooms.VisitorUser) {

			if uId == user.UserId {
				continue
			}

			skip := false
			for _, currentRoomUserId := range currentUsers {
				if uId == currentRoomUserId {
					skip = true
					break
				}
			}

			checkUser := users.GetByUserId(uId)
			if checkUser == nil {
				skip = true
			}

			if skip {
				continue
			}

			newTrackInfo := trackingInfo{
				Name:            checkUser.Character.Name,
				Type:            `user`,
				Strength:        trailStrengthToString(timeLeft),
				NumericStrength: timeLeft,
			}

			if skillLevel >= 3 {
				newTrackInfo.ExitName = findExited(room, uId, rooms.VisitorUser)
			}

			if skillLevel == 1 {

				if len(visitorData) == 0 {
					visitorData = append(visitorData, newTrackInfo)
				} else if visitorData[0].NumericStrength < timeLeft {
					visitorData[0] = newTrackInfo
				}

				continue
			}

			visitorData = append(visitorData, newTrackInfo)
		}

		//
		// If a any visitors are revealed...
		//
		if len(visitorData) > 0 {
			trackTxt, _ := templates.Process("descriptions/track", visitorData, user.UserId)
			user.SendText(trackTxt)
		} else {
			user.SendText("You don't see any tracks.")
		}

		return true, nil

	}

	// only level 3 and 4 can specify a target
	if skillLevel < 3 {

		user.SendText("You can't track a specific person or mob... yet.")
		return true, errors.New(`you can't track a specific person or mob yet`)

	}

	if !user.Character.TryCooldown(skills.Tracking.String(), "1 round") {

		user.SendText(
			fmt.Sprintf("You need to wait %d more rounds to use that skill again.", user.Character.GetCooldown(skills.Tracking.String())))

		return true, errors.New(`you're doing that too often`)

	}

	// Fire an event that a skill has been used
	events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: skills.Tracking, Details: ``})

	//
	// At skill level 3, search the room and adjacent rooms for quarry
	//
	if skillLevel >= 3 {

		foundPlayerId, foundMobId := room.FindByName(rest, rooms.FindAll)

		if foundPlayerId > 0 {
			foundUser := users.GetByUserId(foundPlayerId)
			if foundUser != nil {
				user.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> is in the room with you!`, foundUser.Character.Name))
				return true, nil

			}

		}

		if foundMobId > 0 {
			foundMob := mobs.GetInstance(foundMobId)
			if foundMob != nil {
				user.SendText(
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is in the room with you!`, foundMob.Character.Name))
				return true, nil

			}
		}

		// at skill level 4, becomes an active tracking skill
		if skillLevel >= 4 {

			allNames := []string{}

			for uId, _ := range room.Visitors(rooms.VisitorUser) {

				if uId == user.UserId {
					continue
				}

				if visitorUser := users.GetByUserId(uId); visitorUser != nil {
					allNames = append(allNames, visitorUser.Character.Name)
				}

			}

			match, closeMatch := util.FindMatchIn(rest, allNames...)
			if match != `` {

				user.Character.SetMiscData("tracking-user", match)
				user.Character.SetMiscData("tracking-mob", nil)

				user.AddBuff(26, `skill`) // 26 is the buff for active tracking

				return true, nil

			} else if closeMatch != `` {

				user.Character.SetMiscData("tracking-user", closeMatch)
				user.Character.SetMiscData("tracking-mob", nil)

				user.AddBuff(26, `skill`) // 26 is the buff for active tracking

				return true, nil

			}

			allNames = []string{}

			for mId, _ := range room.Visitors(rooms.VisitorMob) {
				if visitorMob := mobs.GetInstance(mId); visitorMob != nil {
					allNames = append(allNames, visitorMob.Character.Name)
				}
			}

			match, closeMatch = util.FindMatchIn(rest, allNames...)
			if match != `` {

				user.Character.SetMiscData("tracking-user", nil)
				user.Character.SetMiscData("tracking-mob", match)

				user.AddBuff(26, `skill`) // 26 is the buff for active tracking

				return true, nil

			} else if closeMatch != `` {

				user.Character.SetMiscData("tracking-user", nil)
				user.Character.SetMiscData("tracking-mob", closeMatch)

				user.AddBuff(26, `skill`) // 26 is the buff for active tracking

				return true, nil

			}

			user.SendText("You don't see any tracks.")

			return true, nil
		}

		/*
			type TrackInfo struct {
				Name            string
				Type            string // mob / user
				Strength        string
				NumericStrength float64
				ExitName        string
			}
		*/

		allUsersAndMobs := make(map[string]trackingInfo)

		for exitName, exitInfo := range room.Exits {

			// Skip secret exits
			if exitInfo.Secret {
				continue
			}

			exitRoom := rooms.LoadRoom(exitInfo.RoomId)
			if exitRoom == nil {
				continue
			}

			for uId, timeLeft := range room.Visitors(rooms.VisitorUser) {

				if uId == user.UserId {
					continue
				}

				if visitorUser := users.GetByUserId(uId); visitorUser != nil {

					if !strings.HasPrefix(visitorUser.Character.Name, rest) {
						continue
					}

					userTrackInfo, ok := allUsersAndMobs[visitorUser.Character.Name]

					if ok {

						if userTrackInfo.NumericStrength < timeLeft {
							allUsersAndMobs[visitorUser.Character.Name] = trackingInfo{
								Name:            visitorUser.Character.Name,
								Type:            `user`,
								Strength:        trailStrengthToString(timeLeft),
								NumericStrength: timeLeft,
								ExitName:        exitName,
							}
						}

					} else {
						allUsersAndMobs[visitorUser.Character.Name] = trackingInfo{
							Name:            visitorUser.Character.Name,
							Type:            `user`,
							Strength:        trailStrengthToString(timeLeft),
							NumericStrength: timeLeft,
							ExitName:        exitName,
						}
					}

				}

			}

			for mId, timeLeft := range room.Visitors(rooms.VisitorMob) {

				if visitorMob := mobs.GetInstance(mId); visitorMob != nil {

					if !strings.HasPrefix(visitorMob.Character.Name, rest) {
						continue
					}

					mobTrackInfo, ok := allUsersAndMobs[visitorMob.Character.Name]

					if ok {

						if mobTrackInfo.NumericStrength < timeLeft {
							allUsersAndMobs[visitorMob.Character.Name] = trackingInfo{
								Name:            visitorMob.Character.Name,
								Type:            `mob`,
								Strength:        trailStrengthToString(timeLeft),
								NumericStrength: timeLeft,
								ExitName:        exitName,
							}
						}

					} else {
						allUsersAndMobs[visitorMob.Character.Name] = trackingInfo{
							Name:            visitorMob.Character.Name,
							Type:            `mob`,
							Strength:        trailStrengthToString(timeLeft),
							NumericStrength: timeLeft,
							ExitName:        exitName,
						}
					}

				}
			}

			// Search for the strongest tracking in adjacent room

		}

		visitorData := make([]trackingInfo, len(allUsersAndMobs))
		for _, vInfo := range allUsersAndMobs {
			visitorData = append(visitorData, vInfo)
		}

		//
		// If a any visitors are revealed...
		//
		if len(visitorData) > 0 {
			trackTxt, _ := templates.Process("descriptions/track", visitorData, user.UserId)
			user.SendText(trackTxt)
		} else {
			user.SendText("You don't see any tracks.")
		}

		return true, nil
	}

	return true, nil
}

func trailStrengthToString(trailStrength float64) string {
	strengthStr := ""
	strength := int(math.Round(trailStrength * 100))
	if strength < 15 {
		strengthStr = "Dead"
	} else if strength < 50 {
		strengthStr = "Weak"
	} else if strength < 70 {
		strengthStr = "Good"
	} else if strength < 90 {
		strengthStr = "Warm"
	} else {
		strengthStr = "Hot"
	}
	return strengthStr
}

func findExited(room *rooms.Room, targetId int, targetType string) string {

	var bestExit string = ``
	var bestStrength float64 = 0

	for exitName, exitInfo := range room.Exits {

		if exitInfo.Secret {
			continue
		}

		if testRoom := rooms.LoadRoom(exitInfo.RoomId); testRoom != nil {

			for vId, vStr := range testRoom.Visitors(rooms.VisitorType(targetType)) {
				if vId != targetId {
					continue
				}
				if vStr < bestStrength {
					continue
				}
				bestExit = exitName
				bestStrength = vStr
			}
		}

	}

	return bestExit
}
