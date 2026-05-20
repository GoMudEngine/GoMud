package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// UserActor adapts a *users.UserRecord so it satisfies the Actor interface.
type UserActor struct {
	User *users.UserRecord
	Room *rooms.Room
}

var _ Actor = (*UserActor)(nil)

// NewUserActor wraps a UserRecord in an Actor for polymorphic combat and
// target-resolution code paths. The returned Actor has Room == nil; callers
// that need GetRoom() to return non-nil should use NewUserActorInRoom or
// set the Room field directly on the resulting concrete *UserActor.
func NewUserActor(u *users.UserRecord) Actor {
	return &UserActor{User: u}
}

// NewUserActorInRoom is NewUserActor with a pre-populated room reference.
// Use this at sites where downstream code calls GetRoom() / SendRoomText()
// on the returned Actor.
func NewUserActorInRoom(u *users.UserRecord, room *rooms.Room) Actor {
	return &UserActor{User: u, Room: room}
}

func (a *UserActor) GetCharacter() *characters.Character {
	return a.User.Character
}

func (a *UserActor) GetRoom() *rooms.Room {
	return a.Room
}

func (a *UserActor) SendText(cat messaging.Category, msg string) {
	a.User.SendText(cat, msg)
}

func (a *UserActor) SendTextLegacy(msg string) {
	a.User.SendTextLegacy(msg)
}

func (a *UserActor) SendRoomText(msg string, excludeSelf bool) {
	if excludeSelf {
		a.Room.SendTextLegacy(msg, a.User.UserId)
	} else {
		a.Room.SendTextLegacy(msg)
	}
}

func (a *UserActor) SendRoomCommunication(msg string, excludeSelf bool) {
	if excludeSelf {
		a.Room.SendTextCommunication(msg, a.User.UserId)
	} else {
		a.Room.SendTextCommunication(msg)
	}
}

func (a *UserActor) GetName() string {
	return a.User.Character.Name
}

func (a *UserActor) IsPlayer() bool {
	return true
}

func (a *UserActor) GetUserId() int {
	return a.User.UserId
}

func (a *UserActor) GetMobInstanceId() int {
	return 0
}

func (a *UserActor) AddBuff(buffId int, source string) {
	a.User.AddBuff(buffId, source)
}

func (a *UserActor) OnSkillUse(skillName string) bool {
	return a.User.Character.OnSkillUse(skillName, a.User.UserId)
}

func (a *UserActor) OnStatUse(statName string) bool {
	return a.User.Character.OnStatUse(statName, a.User.UserId)
}

func (a *UserActor) OnCriticalSuccess(skillName string) {
	a.User.Character.OnCriticalSuccess(skillName, a.User.UserId)
}

func (a *UserActor) OnCriticalFailure(skillName string) {
	a.User.Character.OnCriticalFailure(skillName, a.User.UserId)
}
