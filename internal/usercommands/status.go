package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Status(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	//possibleStatuses := []string{`strength`, `speed`, `smarts`, `vitality`, `mysticism`, `perception`}

	if rest != `` {

		if rest != `train` {
			user.SendText("status WHAT???")
			return true, nil
		}

		user.DidTip(`status train`, true)

		cmdPrompt, isNew := user.StartPrompt(`status`, rest)

		if isNew {
			tplTxt, _ := templates.Process("character/status-train", user, user.UserId)
			user.SendText(tplTxt)
		}

		question := cmdPrompt.Ask(`Increase which?`, []string{`strength`, `dexterity`, `perception`, `vitality`, `willpower`, `charisma`, `quit`}, `quit`)
		if !question.Done {
			return true, nil
		}

		if question.Response == `quit` {
			user.ClearPrompt()
			return true, nil
		}

		match, closeMatch := util.FindMatchIn(question.Response, []string{`strength`, `dexterity`, `perception`, `vitality`, `willpower`, `charisma`}...)

		question.RejectResponse() // Always reset this question, since we want to keep reusing it.

		if user.Character.StatPoints < 1 {
			user.SendText(`Oops! You have no stat points to spend!`)
			user.ClearPrompt()
			return true, nil
		}
		selection := match
		if match == `` {
			selection = closeMatch
		}

		before := 0
		after := 0
		spent := 0

		switch selection {
		case `strength`:
			before = user.Character.Stats.Strength.Value - user.Character.Stats.Strength.Mods
			user.Character.Stats.Strength.Training += 1
			spent = 1
		case `dexterity`:
			before = user.Character.Stats.Dexterity.Value - user.Character.Stats.Dexterity.Mods
			user.Character.Stats.Dexterity.Training += 1
			spent = 1
		case `perception`:
			before = user.Character.Stats.Perception.Value - user.Character.Stats.Perception.Mods
			user.Character.Stats.Perception.Training += 1
			spent = 1
		case `vitality`:
			before = user.Character.Stats.Vitality.Value - user.Character.Stats.Vitality.Mods
			user.Character.Stats.Vitality.Training += 1
			spent = 1
		case `willpower`:
			before = user.Character.Stats.Willpower.Value - user.Character.Stats.Willpower.Mods
			user.Character.Stats.Willpower.Training += 1
			spent = 1
		case `charisma`:
			before = user.Character.Stats.Charisma.Value - user.Character.Stats.Charisma.Mods
			user.Character.Stats.Charisma.Training += 1
			spent = 1
		}

		if spent > 0 {
			after = before + 1
			user.Character.StatPoints -= 1

			user.Character.Validate()

			user.SendText(fmt.Sprintf(term.CRLFStr+`<ansi fg="210">Your <ansi fg="yellow">%s</ansi> training improves from <ansi fg="201">%d</ansi> to <ansi fg="201">%d</ansi>!</ansi>`, selection, before, after))

			events.AddToQueue(events.CharacterTrained{UserId: user.UserId})
		}

		tplTxt, _ := templates.Process("character/status-train", user, user.UserId)

		if spent > 0 {
			tplTxt = strings.Replace(tplTxt, `fakeprop="`+selection+`"`, `bg="highlight"`, 1)
		}

		user.SendText(tplTxt)

		return true, nil
	}

	tplTxt, _ := templates.Process("character/status", user, user.UserId)
	user.SendText(tplTxt)

	Inventory(``, user, room, flags)

	return true, nil
}
