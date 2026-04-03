package actions

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MobActor adapts a *mobs.Mob so it satisfies the Actor interface.
type MobActor struct {
	Mob  *mobs.Mob
	Room *rooms.Room
}

var _ Actor = (*MobActor)(nil)

func (a *MobActor) GetCharacter() *characters.Character {
	return &a.Mob.Character
}

func (a *MobActor) GetRoom() *rooms.Room {
	return a.Room
}

// SendText is a no-op for mobs — they have no player connection.
func (a *MobActor) SendText(msg string) {}

func (a *MobActor) SendRoomText(msg string, excludeSelf bool) {
	sendRoomTextDarknessAware(a.Room, msg)
}

// SendRoomCommunication is identical to SendRoomText for mobs: mobs do not
// respect client-side mute/deafen settings.
func (a *MobActor) SendRoomCommunication(msg string, excludeSelf bool) {
	sendRoomTextDarknessAware(a.Room, msg)
}

func (a *MobActor) GetName() string {
	return a.Mob.Character.Name
}

func (a *MobActor) IsPlayer() bool {
	return false
}

func (a *MobActor) GetUserId() int {
	return 0
}

func (a *MobActor) GetMobInstanceId() int {
	return a.Mob.InstanceId
}

func (a *MobActor) AddBuff(buffId int, source string) {
	a.Mob.AddBuff(buffId, source)
}

// sendRoomTextDarknessAware is a darkness-aware room broadcast. In lit rooms
// it delegates to room.SendText() directly. In dark rooms only players who
// have the NightVision flag receive the message.
func sendRoomTextDarknessAware(room *rooms.Room, msg string, excludeUserIds ...int) {
	if room.GetVisibility() >= 1 {
		room.SendText(msg, excludeUserIds...)
		return
	}
	for _, uid := range room.GetPlayers() {
		if slices.Contains(excludeUserIds, uid) {
			continue
		}
		u := users.GetByUserId(uid)
		if u != nil && u.Character.HasFlagFromAnySource(buffs.NightVision) {
			u.SendText(msg)
		}
	}
}
