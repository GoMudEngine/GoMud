package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// validateFoldRecall is called during the onCast phase. Returns false to
// abort the spell.
func validateFoldRecall(actor actions.Actor) bool {
	char := actor.GetCharacter()
	if char == nil {
		return false
	}
	currentRoomId := char.RoomId

	// Check if recall is blocked in the current room (instanced zones with
	// allow_recall: false).
	if currentRoom := rooms.LoadRoom(currentRoomId); currentRoom != nil {
		if blocked, ok := currentRoom.GetTempData("allow_recall").(bool); ok && !blocked {
			actor.SendText(messaging.CategorySpellFold, "Something about this place prevents you from recalling.")
			return false
		}
	}

	// A no-go root (Psychic Anchor, or a Jailed holding-cell buff — 5.1c) pins
	// the body in place; recall can't slip free of it either.
	if char.HasBuffFlag(buffs.NoMovement) {
		actor.SendText(messaging.CategorySpellFold, "You are held fast and cannot recall away.")
		return false
	}

	anchorRoom := getMiscDataInt(char, "fold-anchor-room")
	if anchorRoom <= 0 {
		actor.SendText(messaging.CategorySpellFold, `You reach for the Veil, but there is no anchor to `+
			`pull you. Set one first with `+
			`<ansi fg="command">cast fold-anchor</ansi>.`)
		return false
	}

	if anchorRoom == currentRoomId {
		actor.SendText(messaging.CategorySpellFold, "You are already standing on your anchor.")
		return false
	}

	return true
}

// resolveFoldRecall is called during the onMagic phase.
func resolveFoldRecall(actor actions.Actor) {
	char := actor.GetCharacter()
	if char == nil {
		return
	}
	anchorRoom := getMiscDataInt(char, "fold-anchor-room")
	currentRoomId := char.RoomId

	if anchorRoom <= 0 || anchorRoom == currentRoomId {
		actor.SendText(messaging.CategorySpellFold, "The fold collapses — no valid anchor found.")
		return
	}

	// Clear combat state before teleporting.
	char.EndAggro()

	// Move the actor first; only broadcast on success so a failed teleport
	// doesn't leave the departure room thinking the actor vanished.
	if !teleportActor(actor, anchorRoom) {
		actor.SendText(messaging.CategorySpellFold, "The fold collapses — no valid anchor found.")
		return
	}

	// Departure broadcast on the room the actor LEFT (use the snapshotted
	// currentRoomId — char.RoomId has been updated by teleport).
	if oldRoom := rooms.LoadRoom(currentRoomId); oldRoom != nil {
		oldRoom.SendText(messaging.CategorySpellManifestation, fmt.Sprintf(
			`<ansi fg="username">%s</ansi> folds through the Veil and vanishes!`,
			actor.GetName()), actor.GetUserId())
	}

	actor.SendText(messaging.CategorySpellFold, "You fold through the Veil and arrive at your anchor point!")

	// Arrival broadcast on the new room.
	if newRoom := rooms.LoadRoom(anchorRoom); newRoom != nil {
		newRoom.SendText(messaging.CategorySpellManifestation, fmt.Sprintf(
			`<ansi fg="username">%s</ansi> folds through the Veil and appears!`,
			actor.GetName()), actor.GetUserId())
	}

	// Auto-look so the player sees their new room without typing it manually.
	// Mirrors the death-respawn auto-look pattern in Respawn_PlayerAutoLook.go.
	if actor.IsPlayer() {
		if u := users.GetByUserId(actor.GetUserId()); u != nil {
			u.Command("look")
		}
	}
}

// teleportActor moves the actor to the destination room. For players this
// goes through rooms.MoveToRoom (handles cross-zone bookkeeping). For mobs
// it manipulates room membership directly. Returns false if the destination
// room can't be loaded.
func teleportActor(actor actions.Actor, toRoomId int) bool {
	if actor.IsPlayer() {
		// Players: existing helper handles the cross-zone case.
		if err := rooms.MoveToRoom(actor.GetUserId(), toRoomId); err != nil {
			return false
		}
		return true
	}

	// Mobs: manual room membership update.
	char := actor.GetCharacter()
	fromRoom := rooms.LoadRoom(char.RoomId)
	toRoom := rooms.LoadRoom(toRoomId)
	if toRoom == nil {
		return false
	}
	instId := actor.GetMobInstanceId()
	if fromRoom != nil {
		fromRoom.RemoveMob(instId)
	}
	toRoom.AddMob(instId) // AddMob sets mob.Character.RoomId internally (rooms.go:827)
	return true
}

// getMiscDataInt retrieves an integer stored in MiscData, handling both int
// and float64 (the latter can occur after YAML round-trips).
// getMiscDataInt reads an integer out of MiscData.
//
// It MUST handle every numeric shape a value can arrive in, because MiscData is
// an any-typed bag written from several places and round-tripped through YAML:
//   - int      — written directly by most callers
//   - uint64   — written by anything storing util.GetRoundCount()
//   - float64  — what YAML/JSON gives back after a save/load cycle
//
// Missing uint64 here was a silent, long-lived bug: OnSleeperWoken stamps
// "schedule_wake_round" with util.GetRoundCount() (a uint64), this returned 0
// for it, and so the ScheduleWakeGraceRounds cooldown in the schedule executor
// never applied. Every wake of a scheduled NPC — shout, damage, a failed steal,
// a light entering the room — was undone on the very next tick, because
// WantsSleep saw lastWoken == 0 and put them straight back to sleep.
func getMiscDataInt(char *characters.Character, key string) int {
	val := char.GetMiscData(key)
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case uint64:
		return int(v)
	case uint:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

// getMiscDataString retrieves a string stored in MiscData; returns "" for
// unset keys or non-string values.
func getMiscDataString(char *characters.Character, key string) string {
	v := char.GetMiscData(key)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
