package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Char.Action.Result — direct reply to an inbound Char.Action.Try.
type GMCPAction_Result struct {
	Id     int    `json:"id"`
	Status string `json:"status"` // "fired" | "deferred" | "rejected"
	Reason string `json:"reason,omitempty"`
}

// Char.Action.Interrupted — a player's queued cast was interrupted; client re-arms.
type GMCPAction_Interrupted struct {
	Spell string `json:"spell"`
}

// sendActionResult pushes a Char.Action.Result reply to the user. Signature is
// relied on by the Char.Action.Try inbound handler (next task) — keep stable.
func sendActionResult(userId, id int, status, reason string) {
	u := users.GetByUserId(userId)
	if u == nil || !isGMCPEnabled(u.ConnectionId()) {
		return
	}
	events.AddToQueue(GMCPOut{
		UserId:  userId,
		Module:  `Char.Action.Result`,
		Payload: GMCPAction_Result{Id: id, Status: status, Reason: reason},
	})
}

// castInterruptedHandler forwards events.CastInterrupted to the casting user as
// a Char.Action.Interrupted GMCP push so the web client can re-arm a queued cast.
func (g GMCPAutomationModule) castInterruptedHandler(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.CastInterrupted)
	if !ok {
		mudlog.Error("Event", "Expected Type", "CastInterrupted", "Actual Type", e.Type())
		return events.Cancel
	}
	u := users.GetByUserId(evt.UserId)
	if u == nil || !isGMCPEnabled(u.ConnectionId()) {
		return events.Continue
	}
	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  `Char.Action.Interrupted`,
		Payload: GMCPAction_Interrupted{Spell: evt.SpellId},
	})
	return events.Continue
}
