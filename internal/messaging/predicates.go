package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// RoomVisibility is the minimal interface CanSeeClearly / CanSeeShapes
// need from a room. *rooms.Room satisfies this implicitly. Decoupled
// so messaging/ does not import rooms/ — rooms/ imports messaging/,
// and an interface here keeps the dependency arrow one-way.
type RoomVisibility interface {
	GetVisibility() int
}

// CanSeeClearly returns true if the observer can read normal-text
// visual broadcasts in this room. Composes Perception state, room
// lighting, and the NightVision buff flag.
//
// Blinded observers (any source) return false unconditionally.
// A nil observer defaults to true (defensive — pre-init characters
// during boot must not be silently dropped).
func CanSeeClearly(observer *characters.Character, room RoomVisibility) bool {
	if observer == nil {
		return true
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return false
	}
	if room == nil || roomIsLit(room) {
		return true
	}
	return observer.HasFlagFromAnySource(buffs.NightVision)
}

// CanSeeShapes returns true if the observer can detect SOMETHING is
// happening — either full clarity (subsumes CanSeeClearly) OR
// infrared in the dark. Blindness gates this too — broken eyes don't
// see infrared.
//
// A nil observer defaults to true (matches CanSeeClearly's defensive
// behavior).
func CanSeeShapes(observer *characters.Character, room RoomVisibility) bool {
	if CanSeeClearly(observer, room) {
		return true
	}
	if observer == nil {
		return true
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return false
	}
	return observer.HasFlagFromAnySource(buffs.InfraredVision)
}

// roomIsLit returns true if the room is bright enough to read
// (visibility >= 1). Helper so callers don't need to know the
// threshold value.
func roomIsLit(room RoomVisibility) bool {
	if room == nil {
		return true
	}
	// Reflection-free nil-interface guard: a typed-nil *rooms.Room
	// would panic on GetVisibility; callers must pass nil interface,
	// not a typed-nil. The room/Room.SendText path always has a real
	// receiver, so this is safe in practice.
	return room.GetVisibility() >= 1
}
