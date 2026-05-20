package messaging

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state/perception"
)

// CanSeeClearly returns true if the observer can read normal-text
// visual broadcasts in this room. Composes Perception state, room
// lighting, and the NightVision buff flag.
//
// Blinded observers (any source) return false unconditionally.
// A nil observer defaults to true (defensive — pre-init characters
// during boot must not be silently dropped).
func CanSeeClearly(observer *characters.Character, room *rooms.Room) bool {
	if observer == nil {
		return true
	}
	if observer.Perception != nil && observer.Perception.State() == perception.Blinded {
		return false
	}
	if room == nil || room.GetVisibility() >= 1 {
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
func CanSeeShapes(observer *characters.Character, room *rooms.Room) bool {
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
