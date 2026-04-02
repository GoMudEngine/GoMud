package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// UserActor adapts a *users.UserRecord so it satisfies the Actor interface.
type UserActor struct {
	User *users.UserRecord
	Room *rooms.Room
}

var _ Actor = (*UserActor)(nil)

func (a *UserActor) GetCharacter() *characters.Character {
	return a.User.Character
}

func (a *UserActor) GetRoom() *rooms.Room {
	return a.Room
}

func (a *UserActor) SendText(msg string) {
	a.User.SendText(msg)
}

func (a *UserActor) SendRoomText(msg string, excludeSelf bool) {
	if excludeSelf {
		a.Room.SendText(msg, a.User.UserId)
	} else {
		a.Room.SendText(msg)
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
