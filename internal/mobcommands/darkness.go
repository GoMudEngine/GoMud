package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// sendRoomText is a darkness-aware drop-in replacement for room.SendTextVisual().
// In lit rooms it behaves identically. In dark rooms only players with
// nightvision receive the message; others see nothing.
func sendRoomText(room *rooms.Room, msg string, excludeUserIds ...int) {
	if room.GetVisibility() >= 1 {
		room.SendTextVisual(msg, excludeUserIds...)
		return
	}
	for _, uid := range room.GetPlayers() {
		if isExcludedId(uid, excludeUserIds) {
			continue
		}
		u := users.GetByUserId(uid)
		if u != nil && u.Character.HasFlagFromAnySource(buffs.NightVision) {
			u.SendText(msg)
		}
	}
}

// sendAudioRoomText handles audio messages (say/shout) in dark rooms.
// Players with nightvision see the full message with mob name.
// Players without nightvision see the anonymous version.
// In lit rooms, everyone sees the full message.
func sendAudioRoomText(room *rooms.Room, mob *mobs.Mob, anonMsg string, fullMsg string) {
	if room.GetVisibility() >= 1 {
		room.SendTextVisual(fullMsg)
		return
	}
	for _, uid := range room.GetPlayers() {
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}
		if u.Character.HasFlagFromAnySource(buffs.NightVision) {
			u.SendText(fullMsg)
		} else {
			u.SendText(anonMsg)
		}
	}
}

// canSeeInDark returns true if the user has nightvision or the room is lit.
func canSeeInDark(u *users.UserRecord, room *rooms.Room) bool {
	return room.GetVisibility() >= 1 || u.Character.HasFlagFromAnySource(buffs.NightVision)
}

func isExcludedId(uid int, excludeIds []int) bool {
	for _, id := range excludeIds {
		if uid == id {
			return true
		}
	}
	return false
}
