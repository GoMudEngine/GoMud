package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// SkillUseQuestNotify publishes skill_use events to the quest engine.
func SkillUseQuestNotify(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.SkillUsed)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "SkillUsed", "Actual Type", e.Type())
		return events.Cancel
	}

	// Only care about users for this stuff
	if evt.UserId == 0 {
		return events.Continue
	}

	// Fetch the user and character info
	u := users.GetByUserId(evt.UserId)
	if u == nil {
		return events.Continue
	}

	// Create a game bridge with the character's current room
	bridge := questengine.NewGameBridge(u, u.Character.RoomId)

	// Notify the quest engine
	questengine.GetEngine().Notify("skill_use", questengine.EventDetails{
		UserId: evt.UserId,
		RoomId: u.Character.RoomId,
		Skill:  string(evt.Skill),
	}, bridge, bridge)

	return events.Continue
}
