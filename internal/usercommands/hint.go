package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Hint(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	allProgress := user.Character.GetQuestProgress()

	if len(allProgress) == 0 {
		user.SendText(`You don't have any active quests.`)
		return true, nil
	}

	rest = strings.TrimSpace(rest)

	// Resolve target quest ID from argument or last-progressed fallback.
	targetQuestId := 0

	if rest == "" {
		// Default: most recently progressed quest.
		targetQuestId = user.Character.LastQuestId
		if targetQuestId == 0 {
			// Fall back to any quest in progress.
			for qid := range allProgress {
				targetQuestId = qid
				break
			}
		}
		if _, ok := allProgress[targetQuestId]; !ok {
			user.SendText(`You don't have any active quests.`)
			return true, nil
		}
	} else if numId, err := strconv.Atoi(rest); err == nil {
		// Numeric ID provided.
		if _, ok := allProgress[numId]; !ok {
			user.SendText(`You don't have an active quest by that name.`)
			return true, nil
		}
		targetQuestId = numId
	} else {
		// Partial name match — case-insensitive.
		engine := questengine.GetEngine()
		searchLower := strings.ToLower(rest)
		for qid := range allProgress {
			qDef := engine.GetQuest(qid)
			if qDef == nil {
				continue
			}
			if strings.Contains(strings.ToLower(qDef.Name), searchLower) {
				targetQuestId = qid
				break
			}
		}
		if targetQuestId == 0 {
			user.SendText(`You don't have an active quest by that name.`)
			return true, nil
		}
	}

	engine := questengine.GetEngine()
	qDef := engine.GetQuest(targetQuestId)
	if qDef == nil {
		user.SendText(`You don't have any active quests.`)
		return true, nil
	}

	currentStep := allProgress[targetQuestId]

	// Completed quests have step "end".
	if currentStep == "end" {
		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow">Quest Hint</ansi> <ansi fg="yellow-bold">(%s)</ansi>: <ansi fg="white-bold">This quest is complete!</ansi>`,
			qDef.Name,
		))
		return true, nil
	}

	// Find the hint for the NEXT step after current progress.
	// If currentStep is "" (just started), show the first step's hint.
	hintText := ""
	if currentStep == "" {
		if len(qDef.Steps) > 0 {
			hintText = qDef.Steps[0].Hint
		}
	} else {
		found := false
		for i, step := range qDef.Steps {
			if step.Id == currentStep {
				// Take the next step's hint if it exists.
				if i+1 < len(qDef.Steps) {
					hintText = qDef.Steps[i+1].Hint
				}
				found = true
				break
			}
		}
		if !found {
			// Current step not located; show first step hint as fallback.
			if len(qDef.Steps) > 0 {
				hintText = qDef.Steps[0].Hint
			}
		}
	}

	if hintText == "" {
		hintText = "No hint available for this step."
	}

	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">Quest Hint</ansi> <ansi fg="yellow-bold">(%s)</ansi>: <ansi fg="white-bold">%s</ansi>`,
		qDef.Name,
		hintText,
	))

	return true, nil
}
