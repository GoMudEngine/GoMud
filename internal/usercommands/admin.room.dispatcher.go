package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// adminRoom_Edit dispatches the `room edit <subtype>` subcommands.
// Returns (handled, nil) for the help fallthrough so the caller can stop.
func adminRoom_Edit(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if !user.HasRolePermission(`room.edit`) {
		user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">room.edit</ansi> permission`)
		return true, nil
	}

	if rest == `edit container` || rest == `edit containers` {
		if !user.HasRolePermission(`room.edit.container`) {
			user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">room.edit.container</ansi> permission`)
			return true, nil
		}
		return room_Edit_Containers(``, user, room, flags)
	}

	if rest == `edit exit` || rest == `edit exits` {
		if !user.HasRolePermission(`room.edit.exits`) {
			user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">room.edit.exits</ansi> permission`)
			return true, nil
		}
		return room_Edit_Exits(``, user, room, flags)
	}

	if rest == `edit mutator` || rest == `edit mutators` {
		if !user.HasRolePermission(`room.edit.mutators`) {
			user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">room.edit.mutators</ansi> permission`)
			return true, nil
		}
		return room_Edit_Mutators(``, user, room, flags)
	}

	user.SendText(messaging.CategorySystem, `<ansi fg="red">edit WHAT?</ansi> Try:`)
	user.SendText(messaging.CategorySystem, `    <ansi fg="command">room edit containers</ansi>`)
	user.SendText(messaging.CategorySystem, `    <ansi fg="command">room edit exits</ansi>`)
	user.SendText(messaging.CategorySystem, `    <ansi fg="command">room edit mutators</ansi>`)
	return true, nil
}

// adminRoom_Noun handles `room noun`, `room nouns`, `room noun <name>`,
// and `room noun <name> <description>`.
func adminRoom_Noun(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.nouns`) {
		user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">room.noun</ansi> permission`)
		return true, nil
	}

	if len(args) > 2 {
		noun := args[1]
		description := strings.Join(args[2:], ` `)

		if room.Nouns == nil {
			room.Nouns = map[string]string{}
		}
		room.Nouns[noun] = description

		user.SendText(messaging.CategorySystem, `Noun Added:`)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="noun">%s</ansi> - %s`, strings.Repeat(` `, 20-len(noun))+noun, description))

		rooms.SaveRoomTemplate(*room)
		return true, nil
	}

	if len(args) == 2 || (len(args) == 3 && len(args[2]) == 0) {
		if _, ok := room.Nouns[args[1]]; ok {
			delete(room.Nouns, args[1])
			user.SendText(messaging.CategorySystem, `Noun deleted.`)
		} else {
			user.SendText(messaging.CategorySystem, `Noun not found.`)
		}
		return true, nil
	}

	user.SendText(messaging.CategorySystem, `Room Nouns:`)
	for noun, description := range room.Nouns {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="noun">%s</ansi> - %s`, strings.Repeat(` `, 20-len(noun))+noun, description))
	}
	return true, nil
}

// adminRoom_Copy handles `room copy <property> <source-room-id>`.
func adminRoom_Copy(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.copy`) {
		user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">room.copy</ansi> permission`)
		return true, nil
	}

	property := args[1]

	if property == "spawninfo" {
		sourceRoom, _ := strconv.Atoi(args[2])
		if sourceRoom := rooms.LoadRoom(sourceRoom); sourceRoom != nil {
			room.SpawnInfo = sourceRoom.SpawnInfo
			rooms.SaveRoomTemplate(*room)
			user.SendText(messaging.CategorySystem, "Spawn info copied/overwritten.")
		}
	}

	if property == "idlemessages" {
		sourceRoom, _ := strconv.Atoi(args[2])
		if sourceRoom := rooms.LoadRoom(sourceRoom); sourceRoom != nil {
			room.IdleMessages = append(room.IdleMessages, sourceRoom.IdleMessages...)
			rooms.SaveRoomTemplate(*room)
			user.SendText(messaging.CategorySystem, "IdleMessages copied/overwritten.")
		}
	}

	if property == "mutator" || property == "mutators" {
		sourceRoom, _ := strconv.Atoi(args[2])
		if sourceRoom := rooms.LoadRoom(sourceRoom); sourceRoom != nil {
			room.Mutators = append(room.Mutators, sourceRoom.Mutators...)
			rooms.SaveRoomTemplate(*room)
			user.SendText(messaging.CategorySystem, "Mutators copied/overwritten.")
		}
	}

	return true, nil
}

// adminRoom_Info handles `room info [roomId]`.
func adminRoom_Info(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.info`) {
		user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">room.info</ansi> permission`)
		return true, nil
	}

	roomId := 0
	if len(args) == 1 {
		roomId = room.RoomId
	} else {
		roomId, _ = strconv.Atoi(args[1])
	}

	targetRoom := rooms.LoadRoom(roomId)
	if targetRoom == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Room %d not found.", roomId))
		return false, fmt.Errorf("room %d not found", roomId)
	}

	roomInfo := map[string]any{
		`room`: targetRoom,
		`zone`: rooms.GetZoneConfig(targetRoom.Zone),
	}

	infoOutput, _ := templates.Process("admincommands/ingame/roominfo", roomInfo, user.UserId)
	user.SendText(messaging.CategorySystem, infoOutput)
	return true, nil
}

// adminRoom_Exit handles `room exit <direction> <roomId-or-rename>`.
func adminRoom_Exit(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.exits`) {
		user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">room.exit</ansi> permission`)
		return true, nil
	}

	direction := strings.ToLower(args[1])
	roomId := 0
	var numError error = nil
	exitRename := ``

	if len(args) > 2 {
		roomId, numError = strconv.Atoi(args[2])
		if numError != nil {
			exitRename = args[2]
		}
	}

	if len(args) < 3 {
		if _, ok := room.Exits[direction]; !ok {
			user.SendText(messaging.CategorySystem, fmt.Sprintf("Exit %s does not exist.", direction))
			return true, nil
		}
		delete(room.Exits, direction)
		return true, nil
	}

	if currentExit, ok := room.Exits[direction]; ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Exit %s already exists (overwriting).", direction))

		if exitRename != `` {
			delete(room.Exits, direction)
			room.Exits[exitRename] = currentExit
			user.SendText(messaging.CategorySystem, fmt.Sprintf("Exit %s renamed to %s.", direction, exitRename))
			return true, nil
		}
	}

	targetRoom := rooms.LoadRoom(roomId)
	if targetRoom == nil {
		err := fmt.Errorf(`room %d not found`, roomId)
		user.SendText(messaging.CategorySystem, err.Error())
		return true, nil
	}

	rooms.ConnectRoom(room.RoomId, targetRoom.RoomId, direction)
	user.SendText(messaging.CategorySystem, fmt.Sprintf("Exit %s added.", direction))
	return true, nil
}

// adminRoom_SecretExit handles `room secretexit <direction>`.
func adminRoom_SecretExit(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.exits`) {
		user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">room.exit</ansi> permission`)
		return true, nil
	}

	direction := args[1]
	if exit, ok := room.Exits[direction]; ok {
		if exit.Secret {
			exit.Secret = false
			room.Exits[direction] = exit
			rooms.SaveRoomTemplate(*room)
			user.SendText(messaging.CategorySystem, fmt.Sprintf("Exit %s secrecy REMOVED.", direction))
		} else {
			exit.Secret = true
			room.Exits[direction] = exit
			rooms.SaveRoomTemplate(*room)
			user.SendText(messaging.CategorySystem, fmt.Sprintf("Exit %s secrecy ADDED.", direction))
		}
	} else {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Exit %s not found.", direction))
	}
	return true, nil
}

// adminRoom_Set handles `room set <property> [value]`.
func adminRoom_Set(args []string, user *users.UserRecord, room *rooms.Room) (bool, error) {
	if !user.HasRolePermission(`room.set`) {
		user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">room.set</ansi> permission`)
		return true, nil
	}

	propertyName := args[1]
	propertyValue := ``
	if len(args) > 2 {
		propertyValue = strings.Join(args[2:], ` `)
	}
	propertyValue = strings.Trim(propertyValue, `"`)

	if propertyName == "mutator" || propertyName == "mutators" {
		if propertyValue == `` {
			user.SendText(messaging.CategorySystem, `<ansi fg="table-title">Mutators:</ansi>`)
			if len(room.Mutators) == 0 {
				user.SendText(messaging.CategorySystem, `  None.`)
			}
			for _, mut := range room.Mutators {
				user.SendText(messaging.CategorySystem, `  <ansi fg="mutator">` + mut.MutatorId + `</ansi>`)
			}
			user.SendText(messaging.CategorySystem, ``)
		} else {
			user.SendText(messaging.CategorySystem, ``)
			if !mutators.IsMutator(propertyValue) {
				user.SendText(messaging.CategorySystem, `<ansi fg="table-title"><ansi fg="mutator">` + propertyValue + `</ansi> is an invalid mutator id.</ansi>`)
				user.SendText(messaging.CategorySystem, `<ansi fg="table-title">  Here is a list of valid mutator id's:</ansi>`)
				for _, name := range mutators.GetAllMutatorIds() {
					user.SendText(messaging.CategorySystem, `    <ansi fg="mutator">` + name + `</ansi>`)
				}
			} else if room.Mutators.Remove(propertyValue) {
				user.SendText(messaging.CategorySystem, `<ansi fg="table-title">Mutator <ansi fg="mutator">` + propertyValue + `</ansi> Removed.</ansi>`)
			} else if room.Mutators.Add(propertyValue) {
				user.SendText(messaging.CategorySystem, `<ansi fg="table-title">Mutator <ansi fg="mutator">` + propertyValue + `</ansi> Added.</ansi>`)
			}
			user.SendText(messaging.CategorySystem, ``)
		}
		return true, nil
	}

	if propertyName == "spawninfo" {
		if propertyValue == `clear` {
			room.SpawnInfo = room.SpawnInfo[:0]
			rooms.SaveRoomTemplate(*room)
		}
	} else if propertyName == "title" {
		if propertyValue == `` {
			propertyValue = `[no title]`
		}
		room.Title = propertyValue
		rooms.SaveRoomTemplate(*room)
	} else if propertyName == "description" {
		if propertyValue == `` {
			propertyValue = `[no description]`
		}
		propertyValue = strings.ReplaceAll(propertyValue, `\n`, "\n")
		room.Description = propertyValue
		rooms.SaveRoomTemplate(*room)
	} else if propertyName == "idlemessages" {
		room.IdleMessages = []string{}
		for _, idleMsg := range strings.Split(propertyValue, ";") {
			idleMsg = strings.TrimSpace(idleMsg)
			if len(idleMsg) < 1 {
				continue
			}
			room.IdleMessages = append(room.IdleMessages, idleMsg)
		}
		rooms.SaveRoomTemplate(*room)
	} else if propertyName == "symbol" || propertyName == "mapsymbol" {
		room.MapSymbol = propertyValue
		rooms.SaveRoomTemplate(*room)
	} else if propertyName == "legend" || propertyName == "maplegend" {
		room.MapLegend = propertyValue
		rooms.SaveRoomTemplate(*room)
	} else if propertyName == "zone" {
		if err := rooms.MoveToZone(room.RoomId, propertyValue); err != nil {
			user.SendText(messaging.CategorySystem, err.Error())
			return true, nil
		}
	} else if propertyName == "biome" {
		room.Biome = strings.ToLower(propertyValue)
	} else {
		user.SendText(messaging.CategorySystem, `Invalid property provided to <ansi fg="command">room set</ansi>.`)
		return false, fmt.Errorf("unknown room set property %q", propertyName)
	}

	user.SendText(messaging.CategorySystem, fmt.Sprintf("Room %s set to %s.", propertyName, propertyValue))
	return true, nil
}
