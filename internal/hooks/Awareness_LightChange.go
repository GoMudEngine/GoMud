package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Awareness_LightChange listens for events that may change the effective
// light state of a room (someone entering or leaving with the only light
// source; an actor's own emission state changing via equipment, spell, or
// mutation). For each affected hidden actor, re-rolls detection against all
// observers using the new light-conditional modifiers; on detection failure,
// transitions Awareness Hidden → Revealing.
//
// Triggers:
//   - events.RoomChange  — actor entered or left a room (may change room
//     light balance when the actor emits light)
//   - events.EquipmentChange — equipment slot changed (may toggle EmitsLight
//     if the equipped/unequipped item has the lightsource buff flag)
//
// FUTURE expansion (not in chunk 1): light-spell cast/cancel, glow-mutation
// gain/lose. Those events don't currently have hooks; add in followups.
//
// For chunk 1, the handlers wire event subscriptions and call a re-roll
// helper stub. The full re-roll body (iterate observers, opposed roll,
// TransitionToRevealing on detection) is a known followup — existing
// room-entry detection in internal/usercommands/go.go continues to handle
// the primary case for now.
func init() {
	events.RegisterListener(events.RoomChange{}, onRoomChangeForAwareness)
	events.RegisterListener(events.EquipmentChange{}, onEquipmentChangeForAwareness)
}

// onRoomChangeForAwareness handles the case where an actor enters or leaves
// a room, potentially altering the room's effective light level (e.g., the
// only lantern-bearer leaving a dark dungeon room).
//
// FUTURE: if the actor entering emits light AND the room was previously dark
// (or vice versa), re-roll stealth for all hidden actors in the room. The
// detection-roll port is deferred to a chunk-1 followup; existing go.go
// room-entry detection handles the basic movement case.
func onRoomChangeForAwareness(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok {
		return events.Continue
	}

	// Resolve the actor's character to check if they emit light.
	var c *characters.Character
	if evt.UserId > 0 {
		if u := users.GetByUserId(evt.UserId); u != nil {
			c = u.Character
		}
	}
	if c == nil && evt.MobInstanceId > 0 {
		if m := mobs.GetInstance(evt.MobInstanceId); m != nil {
			c = &m.Character
		}
	}

	// Only act when the mover emits light — a non-emitting actor's movement
	// doesn't change room visibility.
	if c == nil || !c.HasFlagFromAnySource(buffs.EmitsLight) {
		return events.Continue
	}

	// FUTURE: compare GetVisibility() before and after; if changed, call
	// rerollHiddenActorsInRoom(toRoom) and rerollHiddenActorsInRoom(fromRoom).
	// For now, just validate the rooms are reachable (defensive nil-check).
	_ = rooms.LoadRoom(evt.ToRoomId)
	_ = rooms.LoadRoom(evt.FromRoomId)

	return events.Continue
}

// onEquipmentChangeForAwareness handles the case where an actor equips or
// removes an item that affects their EmitsLight state. If the actor is
// currently Hidden and their light emission changed, detection needs to
// be re-rolled against all observers in the same room.
//
// FUTURE: diff EmitsLight before/after the equipment change, then call
// rerollHiddenActorVsRoom(actor, room) if the state changed.
func onEquipmentChangeForAwareness(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.EquipmentChange)
	if !ok {
		return events.Continue
	}

	// Resolve the actor's character.
	var c *characters.Character
	if evt.UserId > 0 {
		if u := users.GetByUserId(evt.UserId); u != nil {
			c = u.Character
		}
	}
	if c == nil && evt.MobInstanceId > 0 {
		if m := mobs.GetInstance(evt.MobInstanceId); m != nil {
			c = &m.Character
		}
	}

	// Only re-roll if the actor is currently hidden — visible actors
	// don't need a detection check.
	if c == nil || c.Awareness.State() != awareness.Hidden {
		return events.Continue
	}

	// FUTURE: check whether the equipment change toggled EmitsLight (compare
	// HasFlagFromAnySource(buffs.EmitsLight) before and after via the
	// ItemsWorn / ItemsRemoved slices on the event). If it changed, call
	// rerollHiddenActorVsRoom to re-roll against all observers.
	//
	// Stub — deferred to chunk-1 followup. The TriggerLightChange constant
	// is referenced here to ensure it compiles and the intent is clear.
	_ = state.TransitionReason{Trigger: awareness.TriggerLightChange}

	return events.Continue
}
