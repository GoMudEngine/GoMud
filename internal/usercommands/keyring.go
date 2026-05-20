package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func KeyRing(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	headers := []string{`Type`, `Location`, `Where`, `Sequence`}
	allFormatting := [][]string{}

	for lockId, _ := range user.Character.KeyRing {
		if !strings.HasPrefix(lockId, `key-`) {
			break
		}
	}

	containerKeyFormatting := []string{
		`<ansi fg="white-bold">%s</ansi>`,
		`<ansi fg="room-title">%s</ansi>`,
		`<ansi fg="container">%s</ansi>`,
		`<ansi fg="black-bold">%s</ansi>`,
	}
	containerPickFormatting := []string{
		`<ansi fg="white-bold">%s</ansi>`,
		`<ansi fg="room-title">%s</ansi>`,
		`<ansi fg="container">%s</ansi>`,
		`<ansi fg="green-bold">%s</ansi>`,
	}

	exitKeyFormatting := []string{
		`<ansi fg="white-bold">%s</ansi>`,
		`<ansi fg="room-title">%s</ansi>`,
		`<ansi fg="exit">%s</ansi>`,
		`<ansi fg="black-bold">%s</ansi>`,
	}

	exitPickFormatting := []string{
		`<ansi fg="white-bold">%s</ansi>`,
		`<ansi fg="room-title">%s</ansi>`,
		`<ansi fg="exit">%s</ansi>`,
		`<ansi fg="green-bold">%s</ansi>`,
	}

	// Different row entries lets us easily sort them by type
	keyRows := [][]string{}
	keyFormatting := [][]string{}
	pickRows := [][]string{}
	pickFormatting := [][]string{}
	pickRowsIncomplete := [][]string{}

	cfgSeed := string(configs.GetServerConfig().Seed)

	for lockId, seq := range user.Character.KeyRing {

		complete := true

		row := []string{}

		keyType := `Lockpick`
		sequence := ``
		if strings.HasPrefix(lockId, `key-`) {
			lockId = lockId[4:]
			keyType = `Key`
		} else {

			for _, c := range seq {
				sequence += string(c) + ` `
			}
		}

		row = append(row, keyType)

		roomIdStr := strings.Split(lockId, `-`)[0]
		lockId = lockId[len(roomIdStr)+1:]

		roomId, _ := strconv.Atoi(roomIdStr)
		room := rooms.LoadRoom(roomId)
		if room == nil {
			continue
		}

		if keyType != `Key` {

			exitName, _ := room.FindExitByName(lockId)
			exitInfo := room.Exits[exitName]

			actualSequence := util.GetLockSequence(lockId, int(exitInfo.Lock.Difficulty), cfgSeed, exitInfo.Lock.RotationSeed)
			diff := len(actualSequence) - len(sequence)/2
			if diff > 0 {
				complete = false
				for i := 0; i < diff; i++ {
					sequence += `? `
				}
			}

		}

		row = append(row, fmt.Sprintf(`#%d %s`, room.RoomId, room.Title))

		var formatting []string

		if containerName := room.FindContainerByName(lockId); containerName != `` {
			row = append(row, containerName)

			if keyType == `Key` {
				formatting = containerKeyFormatting
			} else {
				formatting = containerPickFormatting
			}

		} else {

			if keyType == `Key` {
				formatting = exitKeyFormatting
			} else {
				formatting = exitPickFormatting
			}

			exitName, _ := room.FindExitByName(lockId)
			row = append(row, exitName)
		}

		if keyType == `Key` {
			keyFormatting = append(keyFormatting, formatting)
		} else {
			pickFormatting = append(pickFormatting, formatting)
		}

		// Different row entries lets us easily sort them by type
		if keyType == `Lockpick` {

			row = append(row, sequence)

			if complete {
				pickRows = append(pickRows, row)
			} else {
				pickRowsIncomplete = append(pickRowsIncomplete, row)
			}
		} else {
			row = append(row, `-`)
			keyRows = append(keyRows, row)
		}
	}

	rows := [][]string{}
	rows = append(rows, keyRows...)
	rows = append(rows, pickRows...)
	rows = append(rows, pickRowsIncomplete...)

	allFormatting = append(allFormatting, keyFormatting...)
	allFormatting = append(allFormatting, pickFormatting...)

	keyRingTable := templates.GetTable(`Your Keyring:`, headers, rows, allFormatting...)
	tplTxt, _ := templates.Process("tables/generic", keyRingTable, user.UserId)
	user.SendText(messaging.CategorySystem, tplTxt)

	return true, nil
}
