package gmcp

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// GMCPCharOp carries an inbound, state-touching Char.* request from the
// per-connection goroutine to MainWorker.
//
// HandleIAC runs on the connection goroutine, which is genuinely concurrent
// with the world tick. Char.Automation.*, Char.Action.Try and Char.Quests.Focus
// all read/write shared game state (user records, the room manager's plain maps
// via rooms.LoadRoom, quest progress). Doing that work inline races MainWorker
// and can trigger Go's uncatchable "fatal error: concurrent map read and map
// write". This mirrors GMCPBuildOp: copy the payload, queue the op, and do the
// real work in handleCharOp, which runs on MainWorker under util.LockMud().
type GMCPCharOp struct {
	ConnectionId uint64
	Command      string
	Payload      []byte
}

func (e GMCPCharOp) Type() string { return `GMCPCharOp` }

// handleCharOp processes a deferred Char.* request on the MainWorker goroutine.
// The bodies below are the former inline HandleIAC cases, unchanged apart from
// resolving the connection id from the event.
func (g *GMCPModule) handleCharOp(e events.Event) events.ListenerReturn {

	evt, ok := e.(GMCPCharOp)
	if !ok {
		return events.Cancel
	}

	switch evt.Command {

	case `Char.Automation.Set`:
		// Peek the kind, then unmarshal the payload into the matching type.
		var kindOnly struct {
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal(evt.Payload, &kindOnly); err == nil {
			switch kindOnly.Kind {
			case `tick`:
				var decoded users.UserTick
				if err := json.Unmarshal(evt.Payload, &decoded); err == nil {
					if uid := userIdForConnection(evt.ConnectionId); uid > 0 {
						if u := users.GetByUserId(uid); u != nil {
							u.SetTick(decoded)
							events.AddToQueue(events.AutomationChanged{UserId: uid})
						}
					}
				}
			case `trigger`:
				var decoded users.UserTrigger
				if err := json.Unmarshal(evt.Payload, &decoded); err == nil {
					if uid := userIdForConnection(evt.ConnectionId); uid > 0 {
						if u := users.GetByUserId(uid); u != nil {
							u.SetTrigger(decoded)
							events.AddToQueue(events.AutomationChanged{UserId: uid})
						}
					}
				}
			}
		}

	case `Char.Automation.Remove`:
		var decoded struct {
			Kind string `json:"kind"`
			Id   int    `json:"id"`
		}
		if err := json.Unmarshal(evt.Payload, &decoded); err == nil {
			if uid := userIdForConnection(evt.ConnectionId); uid > 0 {
				if u := users.GetByUserId(uid); u != nil {
					switch decoded.Kind {
					case `tick`:
						u.RemoveTick(decoded.Id)
						events.AddToQueue(events.AutomationChanged{UserId: uid})
					case `trigger`:
						u.RemoveTrigger(decoded.Id)
						events.AddToQueue(events.AutomationChanged{UserId: uid})
					}
				}
			}
		}

	case `Char.Action.Try`:
		var req struct {
			Id  int    `json:"id"`
			Cmd string `json:"cmd"`
		}
		if err := json.Unmarshal(evt.Payload, &req); err != nil {
			break
		}
		uid := userIdForConnection(evt.ConnectionId)
		if uid <= 0 {
			break
		}
		u := users.GetByUserId(uid)
		if u == nil {
			break
		}
		room := rooms.LoadRoom(u.Character.RoomId)
		actor := actions.NewUserActorInRoom(u, room)
		result := actions.ActionReadiness(actor, req.Cmd)
		switch result.Status {
		case actions.ActionReady:
			u.Command(req.Cmd)
			sendActionResult(uid, req.Id, "fired", "")
		case actions.ActionDeferred:
			sendActionResult(uid, req.Id, "deferred", result.Reason)
		case actions.ActionRejected:
			u.Command(req.Cmd) // run normally so the player sees the real error message
			sendActionResult(uid, req.Id, "rejected", result.Reason)
		}

	case `Char.Quests.Focus`:
		var req struct {
			Id int `json:"id"`
		}
		if err := json.Unmarshal(evt.Payload, &req); err != nil {
			break
		}
		uid := userIdForConnection(evt.ConnectionId)
		if uid <= 0 {
			break
		}
		u := users.GetByUserId(uid)
		if u == nil {
			break
		}
		// Only allow focusing an active quest.
		if _, ok := u.Character.GetQuestProgress()[req.Id]; !ok {
			break
		}
		u.Character.LastQuestId = req.Id
		// Re-emit Char.Quests so the panel, marker, and `hint` all follow.
		events.AddToQueue(GMCPCharUpdate{
			UserId:     uid,
			Identifier: `Char.Quests`,
		})

	}

	return events.Continue
}
