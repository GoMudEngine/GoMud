package usercommands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Start(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Character.RoomId != -1 {
		return false, errors.New(`only allowed in the void`)
	}

	// Get if already exists, otherwise create new
	cmdPrompt, isNew := user.StartPrompt(`start`, rest)

	if isNew {
		user.SendText(``)
		user.SendText(fmt.Sprintf(`You'll need to answer some questions.%s`, term.CRLFStr))
	}

	if user.Character.SpeciesId == 0 {
		// All players are human in Delusions of Grandeur
		if humanSpecies, ok := species.FindSpecies("human"); ok {
			user.Character.SpeciesId = humanSpecies.Id()
			user.Character.Validate()

			user.SendText(``)
			user.SendText(fmt.Sprintf(`  <ansi fg="magenta">*** Your form takes shape on the world of Gaius ***</ansi>%s`, term.CRLFStr))
		}
	}

	if strings.EqualFold(user.Character.Name, user.Username) || user.Character.Name == user.TempName() || len(user.Character.Name) == 0 || strings.ToLower(user.Character.Name) == `nameless` {

		question := cmdPrompt.Ask(`What will your character be known as (name)?`, []string{})
		if !question.Done {
			return true, nil
		}

		// Signup sets Character.Name = Username as a placeholder; this prompt
		// is how the player replaces it. Prevent them from just re-entering
		// their account Username (which would leave the placeholder unchanged).
		if strings.EqualFold(question.Response, user.Username) {
			user.SendText(`Your username cannot match your character name!`)
			question.RejectResponse()
			return true, nil
		}

		for _, c := range characters.LoadAlts(user.UserId) {
			if strings.EqualFold(question.Response, c.Name) {
				user.SendText(`Your already have a character named that!`)
				question.RejectResponse()
				return true, nil
			}
		}

		if err := users.ValidateActorName(question.Response, users.ValidateActorOpts{}); err != nil {
			user.SendText(`That name won't work: ` + err.Error())
			question.RejectResponse()
			return true, nil
		}

		usernameSelected := question.Response

		question = cmdPrompt.Ask(`Choose the name <ansi fg="username">`+usernameSelected+`</ansi>?`, []string{`yes`, `no`}, `no`)
		if !question.Done {
			return true, nil
		}

		if question.Response == `no` {
			user.ClearPrompt()
			return Start(rest, user, room, flags)
		}

		if err := user.SetCharacterName(usernameSelected); err != nil {
			user.SendText(err.Error())
			question.RejectResponse()
			return true, nil
		}

		user.SendText(fmt.Sprintf(`You will be known as <ansi fg="yellow-bold">%s</ansi>!%s`, user.Character.Name, term.CRLFStr))
	}

	user.Character.ExtraLives = int(configs.GetGamePlayConfig().LivesStart)

	user.EventLog.Add(`char`, fmt.Sprintf(`Created a new character: <ansi fg="username">%s</ansi>`, user.Character.Name))

	events.AddToQueue(events.CharacterCreated{UserId: user.UserId, CharacterName: user.Character.Name})

	duration := time.Now().Sub(user.Joined)
	if duration.Hours() > 1 {

		question := cmdPrompt.Ask(`Skip tutorial?`, []string{`yes`, `no`}, `yes`)
		if !question.Done {
			return true, nil
		}

		if question.Response != `no` {

			user.ClearPrompt()

			user.SendText(fmt.Sprintf(`<ansi fg="magenta">Suddenly, a vortex appears before you, drawing you in before you have any chance to react!</ansi>%s`, term.CRLFStr))

			if destRoom := rooms.LoadRoom(rooms.StartRoomIdAlias); destRoom != nil {

				rooms.MoveToRoom(user.UserId, destRoom.RoomId)

				// Tell the new room they have arrived

				destRoom.SendText(
					fmt.Sprintf(configs.GetTextFormatsConfig().EnterRoomMessageWrapper.String(),
						fmt.Sprintf(`<ansi fg="username">%s</ansi> enters from <ansi fg="exit">somewhere</ansi>.`, user.Character.Name),
					),
					user.UserId,
				)

				Look(``, user, destRoom, events.CmdSecretly) // Do a secret look.

				room.PlaySound(`room-exit`, `movement`, user.UserId)
				destRoom.PlaySound(`room-enter`, `movement`, user.UserId)

				return true, nil
			}

		}

	}

	user.ClearPrompt()

	tutorialRoomIds := []int{}
	startRoom := 0
	for i, roomIdStr := range configs.GetSpecialRoomsConfig().TutorialRooms {
		roomId, _ := strconv.ParseInt(roomIdStr, 10, 64)
		tutorialRoomIds = append(tutorialRoomIds, int(roomId))

		if i == 0 {
			startRoom = int(roomId)
		}
	}

	createdRoomIds, err := rooms.CreateEphemeralRoomIds(tutorialRoomIds...)
	if err != nil {
		user.SendText(`The Tutorial zone is fully occupied right now. Please try again in a few minutes`)
		return true, nil
	}

	ephemeralStartRoomId := createdRoomIds[startRoom]

	user.SendText(fmt.Sprintf(`<ansi fg="magenta">Suddenly, a vortex appears before you, drawing you in before you have any chance to react!</ansi>%s`, term.CRLFStr))

	rooms.MoveToRoom(user.UserId, ephemeralStartRoomId)

	if lookRoom := rooms.LoadRoom(ephemeralStartRoomId); lookRoom != nil {
		Look(``, user, lookRoom, events.CmdSecretly)
	}

	return true, nil
}
