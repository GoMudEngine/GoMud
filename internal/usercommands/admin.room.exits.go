package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// room_Edit_Exits walks an admin through the multi-step prompt to create,
// rename, reconfigure, or delete a room exit. The lock questionnaire is
// delegated to editLockAndTrap (lives in admin.room.go; shared with
// room_Edit_Containers).
func room_Edit_Exits(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// This basic struct will be used to keep track of what we're editing
	type ExitEdit struct {
		Name    string
		NameNew string
		Exit    exit.RoomExit
		Exists  bool
	}

	exitOptions := []templates.NameDescription{}

	for name, c := range room.Exits {

		exitOpt := templates.NameDescription{Name: name}

		if c.Lock.Difficulty > 0 {
			exitOpt.Description += fmt.Sprintf(`[Lvl %d Lock] `, c.Lock.Difficulty)
		}

		if c.Secret {
			exitOpt.Description += `[hidden] `
		}

		exitOptions = append(exitOptions, exitOpt)

	}

	// Must sort since maps will often change between iterations
	sort.SliceStable(exitOptions, func(i, j int) bool {
		return exitOptions[i].Name < exitOptions[j].Name
	})

	//
	// Create a holder for exit editing data
	//
	currentlyEditing := ExitEdit{}

	cmdPrompt, _ := user.StartPrompt(`room edit exits`, rest)

	question := cmdPrompt.Ask(`Choose one:`, []string{`new`}, `new`)
	if !question.Done {
		tplTxt, _ := templates.Process("tables/numbered-list", exitOptions, user.UserId)
		user.SendText(tplTxt)
		return true, nil
	}

	currentlyEditing.Name = question.Response

	if restNum, err := strconv.Atoi(currentlyEditing.Name); err == nil {
		if restNum > 0 && restNum <= len(exitOptions) {
			currentlyEditing.Name = exitOptions[restNum-1].Name
		}
	}

	for _, o := range exitOptions {
		if strings.EqualFold(o.Name, currentlyEditing.Name) {
			currentlyEditing.Name = o.Name
			break
		}
	}

	// Load the (possible) existing exit
	currentlyEditing.Exit, currentlyEditing.Exists = room.Exits[currentlyEditing.Name]

	// If they entered a exit name...
	if currentlyEditing.Name != `new` {

		// Does the exit name they entered not exist? Failure!
		if !currentlyEditing.Exists {
			user.SendText("Invalid option selected.")
			user.SendText("Aborting...")
			user.ClearPrompt()
			return true, nil
		}

		// Since they picked a exit that exists, lets get the question of delete out of the way immediately.
		question := cmdPrompt.Ask(`Delete this exit?`, []string{`yes`, `no`}, `no`)
		if !question.Done {
			return true, nil
		}

		// Delete the exit if that's what they want!
		if question.Response == `yes` {

			delete(room.Exits, currentlyEditing.Name)
			rooms.SaveRoomTemplate(*room)

			user.SendText(``)
			user.SendText(fmt.Sprintf(`<ansi fg="exit">%s</ansi> deleted from the room.`, currentlyEditing.Name))
			user.SendText(``)

			user.ClearPrompt()
			return true, nil
		}

	}

	//
	// Name Selection
	//
	{
		// If they are creating a new exit, we don't want that to become a viable exit name, lets empty it
		if currentlyEditing.Name == `new` {
			currentlyEditing.Name = ``
		}

		// allow them to name/rename the exit.
		question := cmdPrompt.Ask(`Choose a name for this exit:`, []string{currentlyEditing.Name}, currentlyEditing.Name)
		if !question.Done {
			return true, nil
		}
		currentlyEditing.NameNew = question.Response

		// Make sure they aren't using any reserved names.
		if currentlyEditing.NameNew == `quit` || currentlyEditing.NameNew == `new` {
			user.SendText("Invalid new name selected.")
			user.SendText("Aborting...")
			user.ClearPrompt()
			return true, nil
		}

		// Make sure the new name isn't a duplicate
		if currentlyEditing.Name != currentlyEditing.NameNew {
			if _, ok := room.Exits[currentlyEditing.NameNew]; ok {

				user.SendText(`<ansi fg="red">An exit with that name already exists!</ansi>`)
				question.RejectResponse()
				return true, nil

			}
		}

	}

	//
	// Target RoomId
	//
	{
		// allow them to name/rename the exit.
		question := cmdPrompt.Ask(`What RoomId will this exit lead to?`, []string{strconv.Itoa(currentlyEditing.Exit.RoomId)}, strconv.Itoa(currentlyEditing.Exit.RoomId))
		if !question.Done {
			return true, nil
		}

		currentlyEditing.Exit.RoomId, _ = strconv.Atoi(question.Response)

		// Make sure they aren't using any reserved names.
		if rooms.LoadRoom(currentlyEditing.Exit.RoomId) == nil {
			user.SendText("Invalid RoomId provided.")
			question.RejectResponse()
			return true, nil
		}

	}

	//
	// Exit message?
	//
	{
		secretExitDefault := `no`
		if currentlyEditing.Exit.Secret {
			secretExitDefault = `yes`
		}

		// allow them to name/rename the exit.
		question := cmdPrompt.Ask(`Is this a hidden exit?`, []string{`yes`, `no`}, secretExitDefault)
		if !question.Done {
			return true, nil
		}

		currentlyEditing.Exit.Secret = question.Response == `yes`
	}

	//
	// Special message when using the exit?
	//
	{
		defaultMessage := currentlyEditing.Exit.ExitMessage
		if defaultMessage == `` {
			defaultMessage = `none`
		}
		// allow them to name/rename the exit.
		question := cmdPrompt.Ask(`Special message when using the exit?`, []string{defaultMessage}, defaultMessage)
		if !question.Done {
			return true, nil
		}

		if question.Response != `none` {
			currentlyEditing.Exit.ExitMessage = question.Response
		}

	}

	//
	// Lock Options
	//
	{
		var pending bool
		currentlyEditing.Exit.Lock, pending = editLockAndTrap(cmdPrompt, user, currentlyEditing.Exit.Lock, `exit`)
		if pending {
			return true, nil
		}
	}

	//
	// Done editing. Save results
	//
	if currentlyEditing.Name != `` {
		delete(room.Exits, currentlyEditing.Name)
	}

	room.Exits[currentlyEditing.NameNew] = currentlyEditing.Exit
	rooms.SaveRoomTemplate(*room)

	user.SendText(``)

	if currentlyEditing.Exit.Lock.Difficulty > 0 {
		lockId := fmt.Sprintf(`%d-%s`, room.RoomId, currentlyEditing.NameNew)
		user.SendText(fmt.Sprintf(`<ansi fg="red">To Create Key -  LockId: <ansi fg="231" bg="5">%s</ansi></ansi>`, lockId))

		seqString := ``
		for _, dir := range util.GetLockSequence(lockId, int(currentlyEditing.Exit.Lock.Difficulty), string(configs.GetServerConfig().Seed), currentlyEditing.Exit.Lock.RotationSeed) {
			seqString += string(dir) + " "
		}
		user.SendText(fmt.Sprintf(`<ansi fg="red">To pick lock - Sequence: <ansi fg="green">%s</ansi></ansi>`, seqString))
	}

	user.SendText(``)
	user.SendText(`Changes saved.`)
	user.SendText(``)

	user.ClearPrompt()

	return true, nil
}
