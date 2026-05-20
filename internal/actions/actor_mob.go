package actions

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
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

// NewMobActor wraps a Mob in an Actor for polymorphic combat and
// target-resolution code paths. The returned Actor has Room == nil; callers
// that need GetRoom() to return non-nil should use NewMobActorInRoom or set
// the Room field directly on the resulting concrete *MobActor.
func NewMobActor(m *mobs.Mob) Actor {
	return &MobActor{Mob: m}
}

// NewMobActorInRoom is NewMobActor with a pre-populated room reference.
// Use this at sites where downstream code calls GetRoom() / SendRoomText()
// on the returned Actor.
func NewMobActorInRoom(m *mobs.Mob, room *rooms.Room) Actor {
	return &MobActor{Mob: m, Room: room}
}

func (a *MobActor) GetCharacter() *characters.Character {
	return &a.Mob.Character
}

func (a *MobActor) GetRoom() *rooms.Room {
	return a.Room
}

// SendText is a no-op for mobs — they have no player connection.
func (a *MobActor) SendText(cat messaging.Category, msg string) {}

// SendTextLegacy is a no-op for mobs — they have no player connection.
func (a *MobActor) SendTextLegacy(msg string) {}

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

func (a *MobActor) OnSkillUse(skillName string) bool {
	return a.Mob.Character.OnSkillUse(skillName, 0)
}

func (a *MobActor) OnStatUse(statName string) bool {
	return a.Mob.Character.OnStatUse(statName, 0)
}

func (a *MobActor) OnCriticalSuccess(skillName string) {
	a.Mob.Character.OnCriticalSuccess(skillName, 0)
}

func (a *MobActor) OnCriticalFailure(skillName string) {
	a.Mob.Character.OnCriticalFailure(skillName, 0)
}

// sendRoomTextDarknessAware is a darkness-aware room broadcast. In lit rooms
// it delegates to room.SendTextLegacy() directly. In dark rooms only players who
// have the NightVision flag receive the message.
func sendRoomTextDarknessAware(room *rooms.Room, msg string, excludeUserIds ...int) {
	if room.GetVisibility() >= 1 {
		room.SendTextLegacy(msg, excludeUserIds...)
		return
	}
	for _, uid := range room.GetPlayers() {
		if slices.Contains(excludeUserIds, uid) {
			continue
		}
		u := users.GetByUserId(uid)
		if u != nil && u.Character.HasFlagFromAnySource(buffs.NightVision) {
			u.SendTextLegacy(msg)
		}
	}
}
