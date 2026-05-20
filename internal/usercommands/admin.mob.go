package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
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
		user.SendText(messaging.CategorySystem, infoOutput)
		return true, nil
	}

	// Spawn a mob instance
	if args[0] == `spawn` {

		if !user.HasRolePermission(`mob.spawn`) {
			user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">mob.spawn</ansi> permission`)
			return true, nil
		}

		return mob_Spawn(strings.TrimSpace(rest[5:]), user, room, flags)
	}

	// List existing mobs
	if args[0] == `list` {

		if !user.HasRolePermission(`mob.spawn`) {
			user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">mob.spawn</ansi> permission`)
			return true, nil
		}

		return mob_List(strings.TrimSpace(rest[4:]), user, room, flags)
	}

	// Heal a mob (refills HP/SP/CP to max)
	if args[0] == `heal` {

		if !user.HasRolePermission(`mob.spawn`) {
			user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">mob.spawn</ansi> permission`)
			return true, nil
		}

		return mob_Heal(args[1:], user, room, flags)
	}

	return true, nil
}

// mob_Heal restores a mob's Health / Stamina / Conviction to their
// respective max values. Intended for sustained combat testing where
// the tester wants a non-lethal sparring partner that won't die in
// 4-6 rounds. Targets a mob instance id from the args.
//
// Usage: mob heal <instId>
// Or:    mob heal             (heals every mob in the current room)
func mob_Heal(args []string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {

	// No id → heal every mob in the room.
	if len(args) == 0 {
		healed := 0
		for _, mobInstId := range room.GetMobs() {
			if m := mobs.GetInstance(mobInstId); m != nil {
				m.Character.Health = m.Character.HealthMax.Value
				m.Character.Stamina = m.Character.StaminaMax.Value
				m.Character.Conviction = m.Character.ConvictionMax.Value
				healed++
			}
		}
		if healed == 0 {
			user.SendText(messaging.CategorySystem, `No mobs in this room to heal.`)
			return true, nil
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Healed %d mob(s) in the room to full.`, healed))
		return true, nil
	}

	// Specific instance id.
	instId, err := strconv.Atoi(args[0])
	if err != nil || instId < 1 {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">mob heal [instId]</ansi> — omit instId to heal every mob in the room.`)
		return true, nil
	}

	m := mobs.GetInstance(instId)
	if m == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`No mob instance with id %d.`, instId))
		return true, nil
	}

	m.Character.Health = m.Character.HealthMax.Value
	m.Character.Stamina = m.Character.StaminaMax.Value
	m.Character.Conviction = m.Character.ConvictionMax.Value

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`Healed <ansi fg="mobname">%s</ansi> (instance %d) to full.`, m.Character.Name, instId))
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

	user.SendText(messaging.CategorySystem, ``)
	sw := user.ClientSettings().Display.GetScreenWidth()
	strOut := templates.DynamicList(mobList, colWidth, sw, numWidth, longestName)
	user.SendText(messaging.CategorySystem, strOut)
	user.SendText(messaging.CategorySystem, ``)

	return true, nil
}

func mob_Spawn(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	c := configs.GetLootGoblinConfig()

	// special handling of loot goblin
	if rest == `loot goblin` && c.RoomId != 0 {
		if gRoom := rooms.LoadRoom(int(c.RoomId)); gRoom != nil { // loot goblin room
			user.SendText(messaging.CategorySystem, `Somewhere in the realm, a <ansi fg="mobname">loot goblin</ansi> appears!`)
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

			user.SendText(messaging.CategorySystem, 
				fmt.Sprintf(`You wave your hands around and <ansi fg="mobname">%s</ansi> appears in the air and falls to the ground.`, mob.Character.Name),
			)
			room.SendTextVisual(messaging.CategoryMobEmote, 
				fmt.Sprintf(`<ansi fg="username">%s</ansi> waves their hands around and <ansi fg="mobname">%s</ansi> appears in the air and falls to the ground.`, user.Character.Name, mob.Character.Name),
				user.UserId,
			)

			return true, nil
		}
	}

	user.SendText(messaging.CategorySystem, 
		fmt.Sprintf(`Mob <ansi fg="mobname">%s</ansi> not found.`, rest),
	)

	return true, nil
}
