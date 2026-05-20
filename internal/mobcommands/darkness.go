package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// sendRoomText is a darkness-aware drop-in replacement for room.SendTextVisual.
// In lit rooms it behaves identically. In dark rooms only players with
// nightvision receive the message; others see nothing.
//
// cat tags the broadcast for color + sight-gating in the messaging
// pipeline. Pick the most specific Category for the prose (e.g.
// CategoryMobEmote for emotes, CategoryHitMelee for attack narration).
func sendRoomText(room *rooms.Room, cat messaging.Category, msg string, excludeUserIds ...int) {
	if room.GetVisibility() >= 1 {
		room.SendTextVisual(cat, msg, excludeUserIds...)
		return
	}
	for _, uid := range room.GetPlayers() {
		if isExcludedId(uid, excludeUserIds) {
			continue
		}
		u := users.GetByUserId(uid)
		if u != nil && u.Character.HasFlagFromAnySource(buffs.NightVision) {
			u.SendText(cat, msg)
		}
	}
}

// sendAudioRoomText handles audio messages (say/shout) in dark rooms.
// Players with nightvision see the full message with mob name.
// Players without nightvision see the anonymous version.
// In lit rooms, everyone sees the full message.
//
// cat selects the audio color/category (typically CategorySpeech,
// CategoryShout, CategoryNPCDialogue, etc.).
func sendAudioRoomText(room *rooms.Room, mob *mobs.Mob, cat messaging.Category, anonMsg string, fullMsg string) {
	if room.GetVisibility() >= 1 {
		room.SendText(cat, fullMsg)
		return
	}
	for _, uid := range room.GetPlayers() {
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}
		if u.Character.HasFlagFromAnySource(buffs.NightVision) {
			u.SendText(cat, fullMsg)
		} else {
			u.SendText(cat, anonMsg)
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
