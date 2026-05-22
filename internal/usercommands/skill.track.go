package usercommands

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/questengine"
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
Uses a gaussian roll based on Perception + Search skill to determine
how much tracking information the player receives:
  - roll < 125: failure
  - roll >= 125: most recent visitor only
  - roll >= 135: all recent visitors
  - roll >= 175: all visitors + exit directions + targeted/active tracking
*/
func Track(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Compute search score: Perception + skill-scaled bonus
	searchRank := user.Character.GetSkillLevel(skills.Search)
	searchScore := float64(user.Character.Stats.Perception.ValueAdj) + combat.SkillMultiplier(searchRank)*25.0

	// Single roll determines info depth
	roll := dice.RollStat(searchScore)

	// Skill progression check
	user.Character.CheckSkillProgression(string(skills.Search), user.UserId, 1.0)

	// Failure case
	if roll.Value < 125 {
		user.SendText(messaging.CategorySystem, "You don't see any tracks.")
		return true, nil
	}

	currentMobs := room.GetMobs()
	currentUsers := room.GetPlayers()

	//
	// If no argument supplied
	// Handle skill level 1 and 2
	//
	if rest == `` {

		if !user.Character.TryCooldown(skills.Search.String(), "1 round") {
			user.SendText(messaging.CategorySystem, 
				fmt.Sprintf("You need to wait %d more rounds to use that skill again.", user.Character.GetCooldown(skills.Search.String())))
			return true, errors.New(`you're doing that too often`)
		}

		// Fire an event that a skill has been used
		events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: skills.Search, Details: ``})

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

			if roll.Value >= 175 {
				newTrackInfo.ExitName = findExited(room, mId, rooms.VisitorMob)
			}

			if roll.Value < 135 {

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

			if roll.Value >= 175 {
				newTrackInfo.ExitName = findExited(room, uId, rooms.VisitorUser)
			}

			if roll.Value < 135 {

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
			user.SendText(messaging.CategorySystem, trackTxt)
		} else {
			user.SendText(messaging.CategorySystem, "You don't see any tracks.")
		}

		// Quest engine: command notification
		bridge := questengine.NewGameBridge(user, room.RoomId)
		questengine.GetEngine().Notify("command", questengine.EventDetails{
			UserId:  user.UserId,
			RoomId:  room.RoomId,
			Command: "track",
		}, bridge, bridge)

		return true, nil

	}

	// Handle "track stop" / "track clear" — cancel active tracking
	if rest == "stop" || rest == "clear" {
		user.Character.SetMiscData("tracking-mob", nil)
		user.Character.SetMiscData("tracking-user", nil)
		user.Character.SetMiscData("tracking-display-count", nil)
		user.Character.RemoveBuff(26)
		user.SendText(messaging.CategorySystem, `You stop tracking.`)
		return true, nil
	}

	// Targeted tracking requires a high roll
	if roll.Value < 175 {

		user.SendText(messaging.CategorySystem, "Your tracking skills aren't sharp enough right now.")
		return true, errors.New(`your tracking skills aren't sharp enough right now`)

	}

	if !user.Character.TryCooldown(skills.Search.String(), "1 round") {

		user.SendText(messaging.CategorySystem, 
			fmt.Sprintf("You need to wait %d more rounds to use that skill again.", user.Character.GetCooldown(skills.Search.String())))

		return true, errors.New(`you're doing that too often`)

	}

	// Fire an event that a skill has been used
	events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: skills.Search, Details: ``})

	//
	// Search the room and adjacent rooms for quarry
	//
	if roll.Value >= 175 {

		target, err := actions.ResolveTargetActor(room, rest, actions.ResolveTargetOptions{
			FindFlags: []rooms.FindFlag{rooms.FindAll},
		})
		if err == nil {
			if target.IsPlayer() {
				u := target.(*actions.UserActor).User
				user.SendText(messaging.CategorySystem, 
					fmt.Sprintf(`<ansi fg="username">%s</ansi> is in the room with you!`, u.Character.Name))
			} else {
				m := target.(*actions.MobActor).Mob
				user.SendText(messaging.CategorySystem, 
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is in the room with you!`, m.Character.Name))
			}
			return true, nil
		}
		// fall through to active-tracking / nearby-room scan below

		// active tracking when roll is high enough
		if roll.Value >= 175 {

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

				user.AddBuff(86, `skill`) // 86 is the Active Tracking buff

				// Quest engine: command notification
				bridge := questengine.NewGameBridge(user, room.RoomId)
				questengine.GetEngine().Notify("command", questengine.EventDetails{
					UserId:  user.UserId,
					RoomId:  room.RoomId,
					Command: "track",
				}, bridge, bridge)

				return true, nil

			} else if closeMatch != `` {

				user.Character.SetMiscData("tracking-user", closeMatch)
				user.Character.SetMiscData("tracking-mob", nil)

				user.AddBuff(86, `skill`) // 86 is the Active Tracking buff

				// Quest engine: command notification
				bridge := questengine.NewGameBridge(user, room.RoomId)
				questengine.GetEngine().Notify("command", questengine.EventDetails{
					UserId:  user.UserId,
					RoomId:  room.RoomId,
					Command: "track",
				}, bridge, bridge)

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

				user.AddBuff(86, `skill`) // 86 is the Active Tracking buff

				// Quest engine: command notification
				bridge := questengine.NewGameBridge(user, room.RoomId)
				questengine.GetEngine().Notify("command", questengine.EventDetails{
					UserId:  user.UserId,
					RoomId:  room.RoomId,
					Command: "track",
				}, bridge, bridge)

				return true, nil

			} else if closeMatch != `` {

				user.Character.SetMiscData("tracking-user", nil)
				user.Character.SetMiscData("tracking-mob", closeMatch)

				user.AddBuff(86, `skill`) // 86 is the Active Tracking buff

				// Quest engine: command notification
				bridge := questengine.NewGameBridge(user, room.RoomId)
				questengine.GetEngine().Notify("command", questengine.EventDetails{
					UserId:  user.UserId,
					RoomId:  room.RoomId,
					Command: "track",
				}, bridge, bridge)

				return true, nil

			}

			user.SendText(messaging.CategorySystem, "You don't see any tracks.")

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
			user.SendText(messaging.CategorySystem, trackTxt)
		} else {
			user.SendText(messaging.CategorySystem, "You don't see any tracks.")
		}

		// Quest engine: command notification
		bridge := questengine.NewGameBridge(user, room.RoomId)
		questengine.GetEngine().Notify("command", questengine.EventDetails{
			UserId:  user.UserId,
			RoomId:  room.RoomId,
			Command: "track",
		}, bridge, bridge)

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
