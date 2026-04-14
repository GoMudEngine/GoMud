package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Picklock(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	skillLevel := user.Character.GetSkillLevel(skills.Skullduggery)
	if skillLevel < 1 {
		return false, nil
	}

	lockpickItm := items.Item{}
	for _, itm := range user.Character.GetAllBackpackItems() {
		if itm.GetSpec().Type == items.Lockpicks {
			lockpickItm = itm
			break
		}
	}

	if lockpickItm.ItemId < 1 {
		user.SendText(`You need <ansi fg="item">lockpicks</ansi> to pick a lock.`)
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 1 {
		user.SendText("You wanna pock a lock? Specify where it is.")
		return true, nil
	}

	lockId := ``
	lockStrength := 0
	lockTrap := []int{}

	containerName := room.FindContainerByName(args[0])
	if containerName != `` {
		if c, exists := room.Containers[containerName]; exists && c.Hidden {
			if user == nil || !user.Character.HasDiscovery(room.RoomId, containerName) {
				containerName = ``
			}
		}
	}
	exitName, _ := room.FindExitByName(args[0])

	if containerName != `` {

		container := room.Containers[containerName]

		if !container.HasLock() {
			user.SendText("There is no lock there.")
			return true, nil
		}

		if !container.Lock.IsLocked() {
			user.SendText("It's already unlocked.")
			return true, nil
		}

		args = args[1:]
		lockStrength = int(container.Lock.Difficulty)
		lockTrap = container.Lock.TrapBuffIds
		lockId = fmt.Sprintf(`%d-%s`, room.RoomId, containerName)

	} else if exitName != `` {

		// get the first entry int he slice and shorten the slice
		args = args[1:]

		exitInfo, _ := room.GetExitInfo(exitName)

		if !exitInfo.HasLock() {
			user.SendText("There is no lock there.")
			return true, nil
		}

		if !exitInfo.Lock.IsLocked() {
			user.SendText("It's already unlocked.")
			return true, nil
		}

		lockStrength = int(exitInfo.Lock.Difficulty)
		lockTrap = exitInfo.Lock.TrapBuffIds
		lockId = fmt.Sprintf(`%d-%s`, room.RoomId, exitName)

	} else {

		user.SendText("There is no such exit or container.")
		return true, nil
	}

	//
	// Most of what follows shouldn't reference an exit or a chest, but rather lock details.
	//
	keyring_sequence := user.Character.GetKey(lockId)

	sequence := util.GetLockSequence(lockId, lockStrength, string(configs.GetServerConfig().Seed))

	// Calculate any presolve from buffs, gear, pet perks, etc.
	if len(keyring_sequence) == 0 {
		if presolve := user.Character.StatMod(string(statmods.Picklock)); presolve > 0 {
			// All locks bottom out at 3 pins
			if presolve > lockStrength-3 {
				presolve = lockStrength - 3
				if presolve < 0 {
					presolve = 0
				}
			}

			if len(keyring_sequence) < presolve {
				keyring_sequence = strings.Repeat(`*`, presolve)
			}
		}
	}

	if sequenceMatches(keyring_sequence, sequence) {
		user.SendText("")
		user.SendText("Your keyring already has this lock on it.")

		user.ClearPrompt()

		user.SendText(``)
		user.SendText(`<ansi fg="yellow-bold">***</ansi> <ansi fg="green-bold">You Successfully picked the lock!</ansi> <ansi fg="yellow-bold">***</ansi>`)
		user.SendText(`<ansi fg="yellow-bold">***</ansi> <ansi fg="green-bold">You can automatically pick this lock any time as long as you carry <ansi fg="item">lockpicks</ansi>!</ansi> <ansi fg="yellow-bold">***</ansi>`)
		user.SendText(``)

		user.Character.CheckSkillProgression(string(skills.Skullduggery), user.UserId, 1.0)

		room.PlaySound(`change`, `other`)

		if containerName != `` {

			room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> picks the <ansi fg="container">%s</ansi> lock`, user.Character.Name, containerName), user.UserId)

			container := room.Containers[containerName]
			container.Lock.SetUnlocked()
			room.Containers[containerName] = container
		} else {

			room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> picks the <ansi fg="exit">%s</ansi> lock`, user.Character.Name, exitName), user.UserId)

			room.SetExitLock(exitName, false)

		}
		return true, nil
	}

	// Get if already exists, otherwise create new
	cmdPrompt, isNew := user.StartPrompt(`picklock`, rest)

	if isNew {
		user.SendText(GetLockRender(sequence, keyring_sequence, user.UserId))
	}

	entered := ``
	if len(keyring_sequence) > 0 {
		entered = keyring_sequence
	}

	question := cmdPrompt.Ask(`Move your lockpick?`, []string{`UP`, `DOWN`, `quit`})
	if !question.Done {
		return true, nil
	}

	if question.Response == `quit` {
		user.ClearPrompt()
		user.SendText(`Type '<ansi fg="command">help picklock</ansi>' for more information on picking locks.`)
		return true, nil
	}

	direction := question.Response

	question.RejectResponse() // Always reset this question, since we want to keep reusing it.

	r := strings.ToUpper(direction)
	r = string(r[0])

	if r != "U" && r != "D" {
		return true, nil
	}

	entered += r

	for i := 0; i < len(entered); i++ {
		if entered[i] == '*' {
			continue
		}
		if entered[i] != sequence[i] {
			// Mismatch! BREAKS!
			entered = ``
			user.Character.UseItem(lockpickItm)

			user.SendText(``)
			user.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">***</ansi> <ansi fg="red-bold">Oops! Your <ansi fg="item">%s</ansi> break off in the lock, resetting the lock. You'll have to start all over.</ansi> <ansi fg="yellow-bold">***</ansi>`, lockpickItm.GetSpec().NameSimple))
			user.SendText(``)

			room.SendTextVisual(fmt.Sprintf(`<ansi fg="alert-2"><ansi fg="username">%s</ansi> broke their lockpicks trying to pick a lock!</ansi>`, user.Character.Name), user.UserId)

			if len(lockTrap) > 0 {

				user.SendText(`<ansi fg="yellow-bold">***</ansi> <ansi fg="alert-5">A trap was triggered!</ansi> <ansi fg="yellow-bold">***</ansi>`)
				user.SendText(``)
				room.SendTextVisual(fmt.Sprintf(`<ansi fg="alert-3"><ansi fg="username">%s</ansi> triggered a trap!</ansi>`, user.Character.Name), user.UserId)

				for _, buffId := range lockTrap {
					user.AddBuff(buffId, `trap`)
				}
			}
		}
	}

	user.Character.SetKey(lockId, entered)

	if len(entered) > 0 {
		user.SendText(``)
		user.SendText(`<ansi fg="green-bold">A satisfying *click* tells you that you're making progress...</ansi>`)
	} else {
		user.ClearPrompt()
		return true, nil
	}

	user.SendText(GetLockRender(sequence, entered, user.UserId))

	if sequenceMatches(entered, sequence) {

		if entered != sequence {
			entered = sequence
			user.Character.SetKey(lockId, entered)
		}

		user.SendText(``)
		user.SendText(`<ansi fg="yellow-bold">***</ansi> <ansi fg="green-bold">You Successfully picked the lock!</ansi> <ansi fg="yellow-bold">***</ansi>`)
		user.SendText(`<ansi fg="yellow-bold">***</ansi> <ansi fg="green-bold">You can automatically pick this lock any time as long as you carry <ansi fg="item">lockpicks</ansi>!</ansi> <ansi fg="yellow-bold">***</ansi>`)
		user.SendText(``)

		user.Character.CheckSkillProgression(string(skills.Skullduggery), user.UserId, 1.0)

		room.PlaySound(`change`, `other`)

		if containerName != `` {

			room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> picks the <ansi fg="container">%s</ansi> lock`, user.Character.Name, containerName), user.UserId)

			container := room.Containers[containerName]
			container.Lock.SetUnlocked()
			room.Containers[containerName] = container
		} else {
			room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> picks the <ansi fg="exit">%s</ansi> lock`, user.Character.Name, exitName), user.UserId)
			room.SetExitLock(exitName, false)
		}

		user.ClearPrompt()

		return true, nil

	} else {
		if containerName != `` {
			room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to pick the <ansi fg="container">%s</ansi> lock`, user.Character.Name, containerName), user.UserId)
		} else {
			room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to pick the <ansi fg="exit">%s</ansi> lock`, user.Character.Name, exitName), user.UserId)
		}
	}

	return true, nil
}

func GetLockRender(sequence string, entered string, viewingUserId int) string {

	rows := [][]string{}

	if len(entered) > len(sequence) {
		entered = entered[:len(sequence)]
	}

	formatting := make([]string, len(sequence))

	row := []string{}
	for i := 0; i < len(sequence); i++ {
		if len(entered) > i && entered[i] == sequence[i] && entered[i] == 'U' {
			row = append(row, `  U  `)
			formatting[i] = `<ansi fg="green-bold">%s</ansi>`
		} else {
			row = append(row, `     `)
		}
	}
	rows = append(rows, row)

	row = []string{}
	for i := 0; i < len(sequence); i++ {
		if i < len(entered) && entered[i] == '*' {
			row = append(row, `FREE!`)
			formatting[i] = `<ansi fg="green-bold">%s</ansi>`
		} else if i >= len(entered) || entered[i] != sequence[i] {

			row = append(row, `  ?  `)

			formatting[i] = `<ansi fg="red-bold">%s</ansi>`
		} else {
			if entered[i] == 'U' {
				row = append(row, `  ↑  `)
			} else if entered[i] == 'D' {
				row = append(row, `  ↓  `)
			} else {
				row = append(row, `     `)
			}
		}
	}
	rows = append(rows, row)

	row = []string{}
	for i := 0; i < len(sequence); i++ {
		if len(entered) > i && entered[i] == sequence[i] && entered[i] == 'D' {
			row = append(row, `  D  `)
			formatting[i] = `<ansi fg="green-bold">%s</ansi>`
		} else {
			row = append(row, `     `)
		}
	}
	rows = append(rows, row)

	picklockTable := templates.GetTable(`The Lock Sequence Looks like:`, rows[0], rows, formatting)
	tplTxt, _ := templates.Process("tables/lockpicking", picklockTable, viewingUserId)

	return tplTxt

}

func sequenceMatches(input string, correctSequence string) bool {

	if len(input) != len(correctSequence) {
		return false
	}

	for i := 0; i < len(input); i++ {
		if input[i] != '*' && input[i] != correctSequence[i] {
			return false
		}
	}

	return true
}
