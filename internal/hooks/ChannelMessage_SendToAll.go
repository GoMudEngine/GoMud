package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/channels"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ChannelMessage_SendToAll delivers a global chat-channel line as terminal text
// to every online user who has that channel on (the sender always sees their own
// echo). The web/GMCP copy is handled separately by gmcp.Comm.
func ChannelMessage_SendToAll(e events.Event) events.ListenerReturn {
	msg, typeOk := e.(events.ChannelMessage)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "ChannelMessage", "Actual Type", e.Type())
		return events.Continue
	}

	ch, ok := channels.Get(msg.Channel)
	if !ok {
		return events.Continue
	}

	for _, u := range users.GetAllActiveUsers() {
		if !channels.ShouldReceive(u.UserId == msg.SourceUserId, u.Deafened, u.GetConfigOption(ch.ConfigKey)) {
			continue
		}
		u.SendText(messaging.CategoryBroadcast, msg.Text)
		events.AddToQueue(events.RedrawPrompt{UserId: u.UserId}, 100)
	}

	return events.Continue
}
