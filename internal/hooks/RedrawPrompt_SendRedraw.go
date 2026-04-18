package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Checks whether their level is too high for a guide
func RedrawPrompt_SendRedraw(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.RedrawPrompt)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "RedrawPrompt", "Actual Type", e.Type())
		return events.Cancel
	}

	if user := users.GetByUserId(evt.UserId); user != nil {

		newCmdPrompt := user.GetCommandPrompt()

		if evt.OnlyIfChanged {

			oldCmdPrompt := user.GetTempData(`cmdprompt`)

			// If the prompt hasn't changed, skip redrawing
			if s, ok := oldCmdPrompt.(string); ok && s == newCmdPrompt {
				return events.Continue
			}

			// save the new prompt for next time we want to check
			user.SetTempData(`cmdprompt`, newCmdPrompt)

		}

		pTxt := templates.AnsiParse(newCmdPrompt)
		connections.SendTo([]byte(pTxt), user.ConnectionId())

	}

	return events.Continue
}
