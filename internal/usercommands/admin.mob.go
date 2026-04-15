package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* mob 				(All)
* mob.spawn			(Spawn a mob in the room)
 */
func Mob(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) < 1 {
		infoOutput, _ := templates.Process("admincommands/help/command.mob", nil, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}

	// Spawn a mob instance
	if args[0] == `spawn` {

		if !user.HasRolePermission(`mob.spawn`) {
			user.SendText(`you do not have <ansi fg="command">mob.spawn</ansi> permission`)
			return true, nil
		}

		return mob_Spawn(strings.TrimSpace(rest[5:]), user, room, flags)
	}

	// List existing mobs
	if args[0] == `list` {

		if !user.HasRolePermission(`mob.spawn`) {
			user.SendText(`you do not have <ansi fg="command">mob.spawn</ansi> permission`)
			return true, nil
		}

		return mob_List(strings.TrimSpace(rest[4:]), user, room, flags)
	}

	return true, nil
}

func mob_List(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {

	mobList := []templates.NameDescription{}
	longestName := 0

	for _, mob := range mobs.GetAllMobInfo() {

		entry := templates.NameDescription{
			Id:   mob.MobId,
			Name: mob.Character.Name,
		}

		// If searching for matches
		if len(rest) > 0 {
			if !strings.Contains(rest, `*`) {
				rest += `*`
			}
			if !util.StringWildcardMatch(strings.ToLower(entry.Name), rest) {
				continue
			}
		}

		if len(entry.Name) > longestName {
			longestName = len(entry.Name)
		}

		mobList = append(mobList, entry)
	}

	sort.SliceStable(mobList, func(i, j int) bool {
		return strings.ToLower(mobList[i].Name) < strings.ToLower(mobList[j].Name)
	})

	numWidth := len(strconv.Itoa(len(mobList)))
	colWidth := 1 + numWidth + 2 + longestName + 1

	user.SendText(``)
	sw := user.ClientSettings().Display.GetScreenWidth()
	strOut := templates.DynamicList(mobList, colWidth, sw, numWidth, longestName)
	user.SendText(strOut)
	user.SendText(``)

	return true, nil
}

func mob_Spawn(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	c := configs.GetLootGoblinConfig()

	// special handling of loot goblin
	if rest == `loot goblin` && c.RoomId != 0 {
		if gRoom := rooms.LoadRoom(int(c.RoomId)); gRoom != nil { // loot goblin room
			user.SendText(`Somewhere in the realm, a <ansi fg="mobname">loot goblin</ansi> appears!`)
			mudlog.Info(`Loot Goblin Spawn`, `roundNumber`, util.GetRoundCount(), `forced`, true)
			gRoom.Prepare(false) // Make sure the loot goblin spawns.
		}
		return true, nil
	}

	mobId := mobs.MobIdByName(rest)

	if mobId < 1 {
		mobIdInt, _ := strconv.Atoi(rest)
		mobId = mobs.MobId(mobs.MobId(mobIdInt))
	}

	if mobId > 0 {
		if mob := mobs.NewMobById(mobId, room.RoomId); mob != nil {
			room.AddMob(mob.InstanceId)

			user.SendText(
				fmt.Sprintf(`You wave your hands around and <ansi fg="mobname">%s</ansi> appears in the air and falls to the ground.`, mob.Character.Name),
			)
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> waves their hands around and <ansi fg="mobname">%s</ansi> appears in the air and falls to the ground.`, user.Character.Name, mob.Character.Name),
				user.UserId,
			)

			return true, nil
		}
	}

	user.SendText(
		fmt.Sprintf(`Mob <ansi fg="mobname">%s</ansi> not found.`, rest),
	)

	return true, nil
}
