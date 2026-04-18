package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* buff 				(All)
 */
func Buff(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// args should look like one of the following:
	// target buffId - put buff on target if in the room
	// buffId - put buff on self
	// search searchTerm - search for buff by name, display results
	args := util.SplitButRespectQuotes(rest)

	if len(args) > 0 {

		if (len(args) >= 2 && args[0] == "search") || (len(args) == 1 && args[0] == "list") {

			var foundBuffIds []int

			if args[0] == "list" {
				foundBuffIds = buffs.GetAllBuffIds()
			} else {
				foundBuffIds = buffs.SearchBuffs(args[1])
			}

			sort.Ints(foundBuffIds)

			headers := []string{"Id", "Description", "Flags"}
			rows := [][]string{}

			if len(foundBuffIds) == 0 {
				rows = append(rows, []string{"No Matches", "No Matches", "No Matches"})
			} else {
				for _, buffId := range foundBuffIds {
					if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil {
						flags := []string{}
						for _, flag := range buffSpec.Flags {
							flags = append(flags, string(flag))
						}
						rows = append(rows, []string{strconv.Itoa(buffSpec.BuffId), buffSpec.Name, strings.Join(flags, ", ")})
						rows = append(rows, []string{``, `-` + buffSpec.Description, ``})
					}
				}
			}

			searchResultsTable := templates.GetTable("Search Results", headers, rows)
			tplTxt, _ := templates.Process("tables/generic", searchResultsTable, user.UserId, user.UserId)
			user.SendText(tplTxt)
		} else {

			targetUserId := 0
			targetMobInstanceId := 0
			buffId := 0

			if len(args) >= 2 {

				room := rooms.LoadRoom(user.Character.RoomId)
				if room == nil {
					return false, fmt.Errorf(`room %d not found`, user.Character.RoomId)
				}

				if target, err := actions.ResolveTargetActor(room, args[0]); err == nil {
					if target.IsPlayer() {
						targetUserId = target.(*actions.UserActor).User.UserId
					} else {
						targetMobInstanceId = target.(*actions.MobActor).Mob.InstanceId
					}
				}
				// (on err, both IDs stay 0; downstream `if targetUserId > 0` /
				// `if targetMobInstanceId > 0` branches won't fire)

				buffId, _ = strconv.Atoi(args[1])
				if buffId == 0 {
					// Grab the first match
					foundBuffIds := buffs.SearchBuffs(args[1])
					if len(foundBuffIds) > 0 {
						buffId = foundBuffIds[0]
					}
				}

			} else if len(args) == 1 {
				targetUserId = user.UserId
				buffId, _ = strconv.Atoi(args[0])
				if buffId == 0 {
					// Grab the first match
					foundBuffIds := buffs.SearchBuffs(args[0])
					if len(foundBuffIds) > 0 {
						buffId = foundBuffIds[0]
					}
				}
			}

			if buffId == 0 {
				user.SendText("buffId must be an integer > 0.")
				return true, nil

			}

			if targetUserId > 0 {
				// get the user
				if targetUser := users.GetByUserId(targetUserId); targetUser != nil {
					// Get the buff
					if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil {
						targetUser.AddBuff(buffId, `admin`)
						user.SendText(fmt.Sprintf("Buff %d (%s) applied to %s.", buffId, buffSpec.Name, targetUser.Character.Name))

					} else {
						user.SendText(fmt.Sprintf("Buff Id %d not found.", buffId))
					}

					return true, nil
				}
			}

			if targetMobInstanceId > 0 {
				// get the user
				if targetMob := mobs.GetInstance(targetMobInstanceId); targetMob != nil {
					// Get the buff
					if buffSpec := buffs.GetBuffSpec(buffId); buffSpec != nil {
						targetMob.AddBuff(buffId, `admin`)
						user.SendText(fmt.Sprintf("Buff %d (%s) applied to %s.", buffSpec.BuffId, buffSpec.Name, targetMob.Character.Name))

					} else {
						user.SendText(fmt.Sprintf("Buff Id %d not found.", buffId))
					}

					return true, nil
				}
			}

		}
	}

	user.SendText("target not found.")

	// send some sort of help info?
	infoOutput, _ := templates.Process("admincommands/help/command.buff", nil, user.UserId, user.UserId)
	user.SendText(infoOutput)

	return true, nil
}
