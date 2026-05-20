package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/prompt"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
* Role Permissions:
* room	 				(All)
* room.edit				(All edit commands)
* room.edit.container	(Edit containers)
* room.edit.exits		(Edit exits)
* room.edit.mutators	(Edit mutators)
* room.edit.nouns		(Edit nouns)
* room.copy				(Copy room properties from one room to another)
* room.info				(See a room summary)
* room.set				(Set properties of the room)
 */
func Room(rest string, user *users.UserRecord, liveRoom *rooms.Room, flags events.EventFlag) (bool, error) {

	handled := true

	args := util.SplitButRespectQuotes(rest)

	if len(args) == 0 {
		infoOutput, _ := templates.Process("admincommands/help/command.room", nil, user.UserId)
		user.SendTextLegacy(infoOutput)
		return handled, nil
	}

	var room *rooms.Room
	if liveRoom.IsEphemeral() {
		room = liveRoom
	} else {
		room = rooms.LoadRoomTemplate(liveRoom.RoomId)
	}

	if room == nil {
		err := fmt.Errorf(`Something went wrong for RoomId: %d`, liveRoom.RoomId)
		user.SendTextLegacy(err.Error())
		return true, err
	}

	roomCmd := strings.ToLower(args[0])

	switch roomCmd {
	case `edit`:
		return adminRoom_Edit(rest, user, room, flags)
	case `noun`, `nouns`:
		return adminRoom_Noun(args, user, room)
	case `info`:
		return adminRoom_Info(args, user, room)
	case `secretexit`:
		if len(args) < 2 {
			user.SendTextLegacy(fmt.Sprintf(`Invalid room command: <ansi fg="command">%s</ansi>`, roomCmd))
			return handled, nil
		}
		return adminRoom_SecretExit(args, user, room)
	case `copy`:
		if len(args) < 3 {
			user.SendTextLegacy(fmt.Sprintf(`Invalid room command: <ansi fg="command">%s</ansi>`, roomCmd))
			return handled, nil
		}
		return adminRoom_Copy(args, user, room)
	case `exit`:
		if len(args) < 2 {
			user.SendTextLegacy(fmt.Sprintf(`Invalid room command: <ansi fg="command">%s</ansi>`, roomCmd))
			return handled, nil
		}
		return adminRoom_Exit(args, user, room)
	case `set`:
		if len(args) < 2 {
			user.SendTextLegacy(fmt.Sprintf(`Invalid room command: <ansi fg="command">%s</ansi>`, roomCmd))
			return handled, nil
		}
		return adminRoom_Set(args, user, room)
	default:
		user.SendTextLegacy(fmt.Sprintf(`Invalid room command: <ansi fg="command">%s</ansi>`, roomCmd))
	}

	return handled, nil
}

func room_Edit_Containers(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// This basic struct will be used to keep track of what we're editing
	type ContainerEdit struct {
		Name      string
		NameNew   string
		Container rooms.Container
		Exists    bool
	}

	containerOptions := []templates.NameDescription{}

	for name, c := range room.Containers {

		// If it's ephemeral, don't bother.
		if c.DespawnRound != 0 {
			continue
		}

		containerOption := templates.NameDescription{Name: name}

		if c.Lock.Difficulty > 0 {
			containerOption.Description += fmt.Sprintf(`[Lvl %d Lock] `, c.Lock.Difficulty)
		}

		if len(c.Recipes) > 0 {
			containerOption.Description += fmt.Sprintf(`[%d Recipe(s)] `, len(c.Recipes))
		}

		containerOptions = append(containerOptions, containerOption)

	}

	// Must sort since maps will often change between iterations
	sort.SliceStable(containerOptions, func(i, j int) bool {
		return containerOptions[i].Name < containerOptions[j].Name
	})

	//
	// Create a holder for container editing data
	//
	currentlyEditing := ContainerEdit{}

	cmdPrompt, _ := user.StartPrompt(`room edit containers`, rest)

	question := cmdPrompt.Ask(`Choose one:`, []string{`new`}, `new`)
	if !question.Done {
		tplTxt, _ := templates.Process("tables/numbered-list", containerOptions, user.UserId)
		user.SendTextLegacy(tplTxt)
		return true, nil
	}

	currentlyEditing.Name = question.Response

	if restNum, err := strconv.Atoi(currentlyEditing.Name); err == nil {
		if restNum > 0 && restNum <= len(containerOptions) {
			currentlyEditing.Name = containerOptions[restNum-1].Name
		}
	}

	for _, o := range containerOptions {
		if strings.EqualFold(o.Name, currentlyEditing.Name) {
			currentlyEditing.Name = o.Name
			break
		}
	}

	// Load the (possible) existing container
	currentlyEditing.Container, currentlyEditing.Exists = room.Containers[currentlyEditing.Name]

	// If they entered a container name...
	if currentlyEditing.Name != `new` {

		// Does the container name they entered not exist? Failure!
		if !currentlyEditing.Exists {
			user.SendTextLegacy("Invalid option selected.")
			user.SendTextLegacy("Aborting...")
			user.ClearPrompt()
			return true, nil
		}

		// Since they picked a container that exists, lets get the question of delete out of the way immediately.
		question := cmdPrompt.Ask(`Delete this container?`, []string{`yes`, `no`}, `no`)
		if !question.Done {
			return true, nil
		}

		// Delete the container if that's what they want!
		if question.Response == `yes` {

			delete(room.Containers, currentlyEditing.Name)
			rooms.SaveRoomTemplate(*room)

			user.SendTextLegacy(``)
			user.SendTextLegacy(fmt.Sprintf(`<ansi fg="container">%s</ansi> deleted from the room.`, currentlyEditing.Name))
			user.SendTextLegacy(``)

			user.ClearPrompt()
			return true, nil
		}

	}

	//
	// Name Selection
	//
	{
		// If they are creating a new container, we don't want that to become a viable container name, lets empty it
		if currentlyEditing.Name == `new` {
			currentlyEditing.Name = ``
		}

		// allow them to name/rename the container.
		question := cmdPrompt.Ask(`Choose a name for this container:`, []string{currentlyEditing.Name}, currentlyEditing.Name)
		if !question.Done {
			return true, nil
		}
		currentlyEditing.NameNew = question.Response

		// Make sure they aren't using any reserved names.
		if currentlyEditing.NameNew == `quit` || currentlyEditing.NameNew == `new` {
			user.SendTextLegacy("Invalid new name selected.")
			user.SendTextLegacy("Aborting...")
			user.ClearPrompt()
			return true, nil
		}

		// Make sure the new name isn't a duplicate
		if currentlyEditing.Name != currentlyEditing.NameNew {
			if _, ok := room.Containers[currentlyEditing.NameNew]; ok {

				user.SendTextLegacy(`<ansi fg="red">A container with that name already exists!</ansi>`)
				question.RejectResponse()
				return true, nil

			}
		}

	}

	//
	// Lock Options
	//
	{
		var pending bool
		currentlyEditing.Container.Lock, pending = editLockAndTrap(cmdPrompt, user, currentlyEditing.Container.Lock, `container`)
		if pending {
			return true, nil
		}
	}

	//
	// Recipe Options
	//
	{
		var pending bool
		currentlyEditing.Container.Recipes, pending = editContainerRecipes(cmdPrompt, user, currentlyEditing.Container.Recipes)
		if pending {
			return true, nil
		}
	}

	//
	// Done editing. Save results
	//
	if currentlyEditing.Name != `` {
		delete(room.Containers, currentlyEditing.Name)
	}

	if room.Containers == nil {
		room.Containers = map[string]rooms.Container{}
	}

	room.Containers[currentlyEditing.NameNew] = currentlyEditing.Container
	rooms.SaveRoomTemplate(*room)

	user.SendTextLegacy(``)

	if currentlyEditing.Container.Lock.Difficulty > 0 {
		lockId := fmt.Sprintf(`%d-%s`, room.RoomId, currentlyEditing.NameNew)
		user.SendTextLegacy(fmt.Sprintf(`<ansi fg="red">To Create Key -  LockId: <ansi fg="231" bg="5">%s</ansi></ansi>`, lockId))

		seqString := ``
		for _, dir := range util.GetLockSequence(lockId, int(currentlyEditing.Container.Lock.Difficulty), string(configs.GetServerConfig().Seed), currentlyEditing.Container.Lock.RotationSeed) {
			seqString += string(dir) + " "
		}
		user.SendTextLegacy(fmt.Sprintf(`<ansi fg="red">To pick lock - Sequence: <ansi fg="green">%s</ansi></ansi>`, seqString))
	}

	user.SendTextLegacy(``)
	user.SendTextLegacy(`Changes saved.`)
	user.SendTextLegacy(``)

	user.ClearPrompt()

	return true, nil
}

func room_Edit_Containers_SendRecipes(user *users.UserRecord, recipeResultItemId int, recipeItems map[int]int) {

	itm := items.New(recipeResultItemId)

	user.SendTextLegacy(``)
	user.SendTextLegacy(fmt.Sprintf(`    Current Recipe for %d (<ansi fg="itemname">%s</ansi>):`, recipeResultItemId, itm.DisplayName()))

	itemsList := []string{}
	for itemId, qty := range recipeItems {
		itm := items.New(itemId)
		itemsList = append(itemsList, fmt.Sprintf(`        <ansi fg="red">[x%d]</ansi> %d (<ansi fg="itemname">%s</ansi>)`, qty, itemId, itm.DisplayName()))
	}

	// Must sort since maps will often change between iterations
	sort.SliceStable(itemsList, func(i, j int) bool {
		return itemsList[i] < itemsList[j]
	})

	for _, txt := range itemsList {
		user.SendTextLegacy(txt)
	}

	user.SendTextLegacy(``)
}

// editLockAndTrap handles the lock/trap questionnaire shared by containers and exits.
// Returns the updated lock and whether the caller should early-return (pending=true means a question is still in progress).
func editLockAndTrap(cmdPrompt *prompt.Prompt, user *users.UserRecord, lock gamelock.Lock, itemType string) (gamelock.Lock, bool) {

	question := cmdPrompt.Ask(`Will this `+itemType+` be locked?`, []string{`yes`, `no`}, util.BoolYN(lock.Difficulty > 0))
	if !question.Done {
		return lock, true
	}

	if question.Response == `yes` {

		defaultDifficultyAnswer := ``
		if lock.Difficulty > 0 {
			defaultDifficultyAnswer = strconv.Itoa(int(lock.Difficulty))
		}

		question := cmdPrompt.Ask(`What difficulty will the lock be (2-32)?`, []string{defaultDifficultyAnswer}, defaultDifficultyAnswer)
		if !question.Done {
			return lock, true
		}

		difficultyInt, _ := strconv.Atoi(question.Response)

		// Make sure the provided difficulty is within acceptable range.
		if difficultyInt < 2 || difficultyInt > 32 {
			user.SendTextLegacy("Difficulty must between 2 and 32, inclusive.")
			question.RejectResponse()
			return lock, true
		}

		lock.Difficulty = uint8(difficultyInt)

	} else {
		// reset the lock state if there is no lock.
		lock = gamelock.Lock{}
	}

	if lock.Difficulty > 0 {
		//
		// Lock Trap Options
		//
		question = cmdPrompt.Ask(`Will this lock have a trap?`, []string{`yes`, `no`}, util.BoolYN(len(lock.TrapBuffIds) > 0))
		if !question.Done {
			return lock, true
		}

		if question.Response == `yes` {

			selectedBuffList := []int{}
			if cb, ok := cmdPrompt.Recall(`trapBuffs`); ok {
				selectedBuffList = cb.([]int)
			}

			if len(selectedBuffList) == 0 {
				selectedBuffList = append(selectedBuffList, lock.TrapBuffIds...)
			}

			// Keep track of the state
			cmdPrompt.Store(`trapBuffs`, selectedBuffList)

			selectedBuffLookup := map[int]bool{}
			for _, bId := range selectedBuffList {
				selectedBuffLookup[bId] = true
			}

			buffOptions := []templates.NameDescription{}

			for _, buffId := range buffs.GetAllBuffIds() {
				if b := buffs.GetBuffSpec(buffId); b != nil {

					if b.Name == `empty` {
						continue
					}

					marked := false
					if _, ok := selectedBuffLookup[buffId]; ok {
						marked = true
					}

					buffOptions = append(buffOptions, templates.NameDescription{Id: buffId, Marked: marked, Name: b.Name})
				}
			}

			sort.SliceStable(buffOptions, func(i, j int) bool {
				return buffOptions[i].Name < buffOptions[j].Name
			})

			question := cmdPrompt.Ask(`Select a buff to add to the trap, or nothing to continue:`, []string{}, `0`)
			if !question.Done {
				tplTxt, _ := templates.Process("tables/numbered-list-doubled", buffOptions, user.UserId)
				user.SendTextLegacy(tplTxt)
				return lock, true
			}

			buffSelected := question.Response

			if buffSelected != `0` {

				buffSelectedInt := 0

				if restNum, err := strconv.Atoi(buffSelected); err == nil {
					if restNum > 0 && restNum <= len(buffOptions) {
						buffSelectedInt = buffOptions[restNum-1].Id.(int)
					}
				}

				if buffSelectedInt == 0 {
					for _, b := range buffOptions {
						if strings.EqualFold(b.Name, buffSelected) {
							buffSelectedInt = b.Id.(int)
							break
						}
					}
				}

				if buffSelectedInt == 0 {

					user.SendTextLegacy("Invalid selection.")
					question.RejectResponse()

					tplTxt, _ := templates.Process("tables/numbered-list-doubled", buffOptions, user.UserId)
					user.SendTextLegacy(tplTxt)
					return lock, true
				}

				if _, ok := selectedBuffLookup[buffSelectedInt]; ok {

					delete(selectedBuffLookup, buffSelectedInt)
					for idx, buffId := range selectedBuffList {
						if buffId == buffSelectedInt {
							selectedBuffList = append(selectedBuffList[0:idx], selectedBuffList[idx+1:]...)
							break
						}
					}

				} else {

					selectedBuffList = append(selectedBuffList, buffSelectedInt)
					selectedBuffLookup[buffSelectedInt] = true

				}

				cmdPrompt.Store(`trapBuffs`, selectedBuffList)

				question.RejectResponse()

				for idx, data := range buffOptions {
					_, data.Marked = selectedBuffLookup[data.Id.(int)]
					buffOptions[idx] = data
				}

				tplTxt, _ := templates.Process("tables/numbered-list-doubled", buffOptions, user.UserId)
				user.SendTextLegacy(tplTxt)
				return lock, true

			}

		}

		if cb, ok := cmdPrompt.Recall(`trapBuffs`); ok {
			lock.TrapBuffIds = cb.([]int)
		}

		if lock.RelockInterval == `` {
			lock.RelockInterval = gamelock.DefaultRelockTime
		}

		question = cmdPrompt.Ask(`How long until it automatically relocks?`, []string{lock.RelockInterval}, lock.RelockInterval)
		if !question.Done {
			return lock, true
		}

		lock.RelockInterval = question.Response

		// If the default time is chosen, can just leave it blank.
		if lock.RelockInterval == gamelock.DefaultRelockTime {
			lock.RelockInterval = ``
		}

	}

	return lock, false
}

// editContainerRecipes handles the recipe questionnaire for containers.
// Returns the updated recipes and whether the caller should early-return.
func editContainerRecipes(cmdPrompt *prompt.Prompt, user *users.UserRecord, recipes map[int][]int) (map[int][]int, bool) {

	question := cmdPrompt.Ask(`Will this container have recipes?`, []string{`yes`, `no`}, util.BoolYN(len(recipes) > 0))
	if !question.Done {
		return recipes, true
	}

	if question.Response == `yes` {

		currentRecipes := map[int][]int{}
		if cr, ok := cmdPrompt.Recall(`recipes`); ok {
			currentRecipes = cr.(map[int][]int)
		}

		if len(currentRecipes) == 0 {
			for k, v := range recipes {
				currentRecipes[k] = append([]int{}, v...)
			}
		}

		recipeNow := 0
		if rNow, ok := cmdPrompt.Recall(`recipeNow`); ok {
			recipeNow = rNow.(int)
		}

		if recipeNow != 0 && items.GetItemSpec(recipeNow) == nil {
			user.SendTextLegacy(`<ansi fg="red">Invalid selection.</ansi>`)
			question.RejectResponse()
			return recipes, true
		}

		// Keep track of the state
		cmdPrompt.Store(`recipes`, currentRecipes)
		cmdPrompt.Store(`recipeNow`, recipeNow)

		// Select recipe to modify
		if _, ok := currentRecipes[recipeNow]; !ok {
			recipeOptions := []templates.NameDescription{}
			for productItemId, recipeItemList := range currentRecipes {

				itm := items.New(productItemId)
				productName := fmt.Sprintf(`%d (%s)`, productItemId, itm.DisplayName())

				allRequiredItems := []string{}
				for _, iId := range recipeItemList {
					itm := items.New(iId)
					allRequiredItems = append(allRequiredItems, fmt.Sprintf(`%d (%s)`, iId, itm.DisplayName()))
				}

				recipeOptions = append(recipeOptions,
					templates.NameDescription{
						Id:          productItemId,
						Marked:      recipeNow == productItemId,
						Name:        productName,
						Description: strings.Join(allRequiredItems, `, `),
					})

			}

			recipeOptions = append(recipeOptions,
				templates.NameDescription{
					Id:          0,
					Marked:      false,
					Name:        `new`,
					Description: `create a new recipe`,
				})

			recipeOptions = append(recipeOptions,
				templates.NameDescription{
					Id:          -1,
					Marked:      false,
					Name:        `skip`,
					Description: `skip this step`,
				})

			question := cmdPrompt.Ask(`Modify which (or new)?`, []string{`skip`}, `skip`)
			if !question.Done {
				tplTxt, _ := templates.Process("tables/numbered-list", recipeOptions, user.UserId)
				user.SendTextLegacy(tplTxt)
				return recipes, true
			}

			recipeSelected := question.Response
			if restNum, err := strconv.Atoi(recipeSelected); err == nil {
				if restNum > 0 && restNum <= len(recipeOptions) {
					recipeNow = recipeOptions[restNum-1].Id.(int)
				}
			}

			if recipeNow == 0 {
				for _, b := range recipeOptions {
					if strings.EqualFold(b.Name, recipeSelected) {
						recipeNow = b.Id.(int)
						break
					}
				}
			}

			if question.Response == `new` {

				question := cmdPrompt.Ask(`What itemId will be created?`, []string{})
				if !question.Done {
					return recipes, true
				}

				itemIdInt, _ := strconv.Atoi(question.Response)
				if items.GetItemSpec(itemIdInt) == nil {

					user.SendTextLegacy("Invalid itemId.")
					question.RejectResponse()

					return recipes, true
				}

				if _, ok := currentRecipes[itemIdInt]; !ok {
					currentRecipes[itemIdInt] = []int{}
				}

				recipeNow = itemIdInt

				// Keep track of the state
				cmdPrompt.Store(`recipes`, currentRecipes)
				cmdPrompt.Store(`recipeNow`, recipeNow)
			}
		}

		// If they're editing a recipe, lets add ingredients
		if recipeNow != -1 {

			neededItems := map[int]int{}
			for _, inputItemId := range currentRecipes[recipeNow] {
				neededItems[inputItemId] = neededItems[inputItemId] + 1
			}

			question = cmdPrompt.Ask(`Enter an itemId to add to the recipe, or nothing to continue:`, []string{``}, `skip`)
			if !question.Done {
				// They have a recipe to modify, ask for item id's
				user.SendTextLegacy(``)
				user.SendTextLegacy(`<ansi fg="cyan">Positive numbers add items, negative numbers remove items.</ansi>`)

				room_Edit_Containers_SendRecipes(user, recipeNow, neededItems)

				return recipes, true
			}

			if question.Response != `skip` {

				removeItem := false
				if question.Response[0] == '-' {
					removeItem = true
					question.Response = question.Response[1:]
				}

				recipeAdjustment := items.FindItem(question.Response)

				if itemSpec := items.GetItemSpec(recipeAdjustment); itemSpec == nil {
					user.SendTextLegacy(`<ansi fg="red">Invalid ItemId provided.</ansi>`)

					room_Edit_Containers_SendRecipes(user, recipeNow, neededItems)

					question.RejectResponse()
					return recipes, true
				}

				if removeItem {

					for idx, itemId := range currentRecipes[recipeNow] {

						if itemId == recipeAdjustment {
							currentRecipes[recipeNow] = append(currentRecipes[recipeNow][0:idx], currentRecipes[recipeNow][idx+1:]...)

							neededItems[recipeAdjustment] -= 1

							if neededItems[recipeAdjustment] == 0 {
								delete(neededItems, recipeAdjustment)
							}

							break
						}

					}

				} else {
					currentRecipes[recipeNow] = append(currentRecipes[recipeNow], recipeAdjustment)
					neededItems[recipeAdjustment] += 1
				}

				// Keep track of the state
				cmdPrompt.Store(`recipes`, currentRecipes)
				cmdPrompt.Store(`recipeNow`, recipeNow)

				room_Edit_Containers_SendRecipes(user, recipeNow, neededItems)

				question.RejectResponse()
				return recipes, true

			}

		}

		if allRecipes, ok := cmdPrompt.Recall(`recipes`); ok {
			recipes = allRecipes.(map[int][]int)

			for i, itms := range recipes {
				if len(itms) == 0 {
					delete(recipes, i)
				}
			}
		}

	} else {
		clear(recipes)
	}

	return recipes, false
}

func room_Edit_Mutators(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	allRoomMutators := []string{}
	for _, roomMut := range room.Mutators {
		allRoomMutators = append(allRoomMutators, roomMut.MutatorId)
	}

	cmdPrompt, _ := user.StartPrompt(`room edit mutators`, rest)

	selectedMutatorList := []string{}
	if muts, ok := cmdPrompt.Recall(`mutators`); ok {
		selectedMutatorList = muts.([]string)
	} else {
		if len(selectedMutatorList) == 0 {
			selectedMutatorList = append(selectedMutatorList, allRoomMutators...)
		}
	}

	// Keep track of the state
	cmdPrompt.Store(`mutators`, selectedMutatorList)

	selectedMutatorLookup := map[string]bool{}
	for _, mutId := range selectedMutatorList {
		selectedMutatorLookup[mutId] = true
	}

	mutatorOptions := []templates.NameDescription{}

	for _, mutId := range mutators.GetAllMutatorIds() {
		marked := false
		if _, ok := selectedMutatorLookup[mutId]; ok {
			marked = true
		}

		mutatorOptions = append(mutatorOptions, templates.NameDescription{Id: mutId, Marked: marked, Name: mutId})

	}

	sort.SliceStable(mutatorOptions, func(i, j int) bool {
		return mutatorOptions[i].Name < mutatorOptions[j].Name
	})

	question := cmdPrompt.Ask(`Select a mutator to add to the room, or nothing to continue:`, []string{}, `0`)
	if !question.Done {
		tplTxt, _ := templates.Process("tables/numbered-list-doubled", mutatorOptions, user.UserId)
		user.SendTextLegacy(tplTxt)
		return true, nil
	}

	if question.Response != `0` {

		mutatorSelected := ``

		if restNum, err := strconv.Atoi(question.Response); err == nil {
			if restNum > 0 && restNum <= len(mutatorOptions) {
				mutatorSelected = mutatorOptions[restNum-1].Id.(string)
			}
		}

		if mutatorSelected == `` {
			for _, b := range mutatorOptions {
				if strings.EqualFold(b.Name, question.Response) {
					mutatorSelected = b.Id.(string)
					break
				}
			}
		}

		if mutatorSelected == `` {

			user.SendTextLegacy("Invalid selection.")
			question.RejectResponse()

			tplTxt, _ := templates.Process("tables/numbered-list-doubled", mutatorOptions, user.UserId)
			user.SendTextLegacy(tplTxt)
			return true, nil
		}

		if _, ok := selectedMutatorLookup[mutatorSelected]; ok {

			delete(selectedMutatorLookup, mutatorSelected)
			for idx, mutId := range selectedMutatorList {
				if mutId == mutatorSelected {
					selectedMutatorList = append(selectedMutatorList[0:idx], selectedMutatorList[idx+1:]...)
					break
				}
			}

		} else {

			selectedMutatorList = append(selectedMutatorList, mutatorSelected)
			selectedMutatorLookup[mutatorSelected] = true

		}

		cmdPrompt.Store(`mutators`, selectedMutatorList)

		question.RejectResponse()

		for idx, data := range mutatorOptions {
			_, data.Marked = selectedMutatorLookup[data.Id.(string)]
			mutatorOptions[idx] = data
		}

		tplTxt, _ := templates.Process("tables/numbered-list-doubled", mutatorOptions, user.UserId)
		user.SendTextLegacy(tplTxt)
		return true, nil

	}

	//
	// Done editing. Save results
	//
	room.Mutators = mutators.MutatorList{}
	for _, mutId := range selectedMutatorList {
		room.Mutators = append(room.Mutators, mutators.Mutator{MutatorId: mutId})
	}
	rooms.SaveRoomTemplate(*room)

	user.SendTextLegacy(``)
	user.SendTextLegacy(`Changes saved.`)
	user.SendTextLegacy(``)

	user.ClearPrompt()

	return true, nil
}
