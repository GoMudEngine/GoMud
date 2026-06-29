package usercommands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
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
		user.SendText(messaging.CategorySystem, ``)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You'll need to answer some questions.%s`, term.CRLFStr))
	}

	if user.Character.SpeciesId == 0 {
		// All players are human in Delusions of Grandeur
		if humanSpecies, ok := species.FindSpecies("human"); ok {
			user.Character.SpeciesId = humanSpecies.Id()
			user.Character.Validate()

			user.SendText(messaging.CategorySystem, ``)
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`  <ansi fg="magenta">*** Your form takes shape on the world of Gaius ***</ansi>%s`, term.CRLFStr))
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
			user.SendText(messaging.CategorySystem, `Your username cannot match your character name!`)
			question.RejectResponse()
			return true, nil
		}

		for _, c := range characters.LoadAlts(user.UserId) {
			if strings.EqualFold(question.Response, c.Name) {
				user.SendText(messaging.CategorySystem, `Your already have a character named that!`)
				question.RejectResponse()
				return true, nil
			}
		}

		if err := users.ValidateActorName(question.Response, users.ValidateActorOpts{}); err != nil {
			user.SendText(messaging.CategorySystem, `That name won't work: ` + err.Error())
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
			user.SendText(messaging.CategorySystem, err.Error())
			question.RejectResponse()
			return true, nil
		}

		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You will be known as <ansi fg="yellow-bold">%s</ansi>!%s`, user.Character.Name, term.CRLFStr))
	}

	user.EventLog.Add(`char`, fmt.Sprintf(`Created a new character: <ansi fg="username">%s</ansi>`, user.Character.Name))

	events.AddToQueue(events.CharacterCreated{UserId: user.UserId, CharacterName: user.Character.Name})

	question := cmdPrompt.Ask(
		"How much of Gaius do you already know?\n"+
			"  1) New to text MUDs       -- I'll teach you the basics first.\n"+
			"  2) New to DOGMud          -- I know MUDs; show me what's different.\n"+
			"  3) Veteran                -- skip all tutorials; drop me into the city.",
		[]string{"1", "2", "3"}, "2")
	if !question.Done {
		return true, nil
	}

	switch onboardingRoute(question.Response) {
	case routeVeteran:
		confirm := cmdPrompt.Ask("Skip everything and start in the city, already Awakened? (y/n)", []string{"y", "n"}, "n")
		if !confirm.Done {
			return true, nil
		}
		if confirm.Response != "y" {
			question.RejectResponse()
			return true, nil
		}
		user.ClearPrompt()
		autoAwaken(user)
		startVeteranInThornwall(user)
		return true, nil
	case routeNewbie:
		user.ClearPrompt()
		grantNewcomerMarker(user)
		if !startInAntechamber(user) {
			startInCoulee(user, room)
		}
		return true, nil
	default: // routeMudVet
		user.ClearPrompt()
		startInCoulee(user, room)
		return true, nil
	}
}

// ─── Onboarding route ────────────────────────────────────────────────────────

type onboardingRouteKind int

const (
	routeMudVet  onboardingRouteKind = iota // tier B (default): today's Coulee
	routeNewbie                             // tier A: tutorial antechamber
	routeVeteran                            // tier C: skip to Thornwall, Awakened
)

// onboardingRoute maps the poll answer to a route. Anything unrecognized
// (incl. the empty default) maps to the safe mud-vet path.
func onboardingRoute(answer string) onboardingRouteKind {
	switch strings.TrimSpace(answer) {
	case "1":
		return routeNewbie
	case "3":
		return routeVeteran
	default:
		return routeMudVet
	}
}

// ─── Routing helpers ─────────────────────────────────────────────────────────

// startInCoulee drops the character at the Awakening Pool (StartRoom 5200) —
// the standard new-player path (mud-vet, and newbie until the antechamber
// route lands in a later phase).
func startInCoulee(user *users.UserRecord, room *rooms.Room) {
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="magenta">Suddenly, a vortex appears before you, drawing you in before you have any chance to react!</ansi>%s`, term.CRLFStr))
	if destRoom := rooms.LoadRoom(rooms.StartRoomIdAlias); destRoom != nil {
		rooms.MoveToRoom(user.UserId, destRoom.RoomId)
		Look(``, user, destRoom, events.CmdSecretly)
	}
}

// startInAntechamber drops a new-to-MUDs player into a private, instanced copy
// of the tutorial antechamber (the TutorialRooms). Returns false if the
// instance could not be created (caller falls back to startInCoulee).
func startInAntechamber(user *users.UserRecord) bool {
	cfg := configs.GetSpecialRoomsConfig()
	ids := []int{}
	first := 0
	for i, s := range cfg.TutorialRooms {
		id, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		ids = append(ids, id)
		if i == 0 {
			first = id
		}
	}
	if len(ids) == 0 || first == 0 {
		return false
	}
	created, err := rooms.CreateEphemeralRoomIds(ids...)
	if err != nil {
		return false
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="magenta">The grey takes you, gentle as sleep...</ansi>%s`, term.CRLFStr))
	rooms.MoveToRoom(user.UserId, created[first])
	if r := rooms.LoadRoom(created[first]); r != nil {
		Look("", user, r, events.CmdSecretly)
	}
	return true
}

// startVeteranInThornwall vortexes an already-Awakened veteran to Thornwall
// (468) and prints the back-door hint.
func startVeteranInThornwall(user *users.UserRecord) {
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="magenta">The pool is behind you before you ever saw it. You step out already changed, into a city that does not know your name.</ansi>%s`, term.CRLFStr))
	if destRoom := rooms.LoadRoom(468); destRoom != nil {
		rooms.MoveToRoom(user.UserId, destRoom.RoomId)
		Look(``, user, destRoom, events.CmdSecretly)
	}
	user.SendText(messaging.CategorySystem, `New to Gaius after all? Type <ansi fg="command">tutorial</ansi> to be taken to the newcomers' pool.`)
}

// grantNewcomerMarker flags a total-newbie character with the hidden
// Newcomer's Path token, which gates the newbie-only teaching beats (Cleric
// Hadwen's mutations/progression talk; Crier Toke's player-interaction talk).
func grantNewcomerMarker(user *users.UserRecord) {
	events.AddToQueue(events.Quest{
		UserId:     user.UserId,
		QuestToken: "21-start",
	})
}

// ─── Veteran auto-awaken ─────────────────────────────────────────────────────

// autoAwaken gives a veteran-tier character the two effects of the Awakening
// Rite without making them perform it: the 30-end "Opened" token (clears the
// 5200 movement block and Warden Esk's gate) and a starting mutation.
func autoAwaken(user *users.UserRecord) {
	user.Character.GrantRandomMutation()
	events.AddToQueue(events.Quest{
		UserId:     user.UserId,
		QuestToken: "30-end",
	})
}
