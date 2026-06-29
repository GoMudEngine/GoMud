package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// onboardingAllowedQuests are the only quest ids a character may have progress
// in and still use the `tutorial` back-door. Keeps an established character
// from yanking themselves back to the newcomers' pool. (30 = Awakening,
// 31 = Find Your Footing, 21 = Newcomer's Path marker.)
var onboardingAllowedQuests = map[int]struct{}{30: {}, 31: {}, 21: {}}

// Tutorial teleports a low-progress character to the newcomers' pool (the
// Awakening Pool, StartRoom 5200). Advertised in the veteran welcome as the
// mis-pick safety net. Refuses if the character has progress beyond the
// onboarding quests.
func Tutorial(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	for questId := range user.Character.QuestProgress {
		if _, ok := onboardingAllowedQuests[questId]; !ok {
			user.SendText(messaging.CategorySystem, `The newcomers' coulee is for those just arriving. You are past that now.`)
			return true, nil
		}
	}

	user.SendText(messaging.CategorySystem, `A familiar pull takes you back toward the glowing pool...`)
	if destRoom := rooms.LoadRoom(rooms.StartRoomIdAlias); destRoom != nil {
		rooms.MoveToRoom(user.UserId, destRoom.RoomId)
		Look("", user, destRoom, events.CmdSecretly)
	}
	return true, nil
}
