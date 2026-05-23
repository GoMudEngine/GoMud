package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// ---------------------------------------------------------------------------
// salvage-test-specific helpers
// ---------------------------------------------------------------------------

// salvageFakeActor is a minimal Actor with a configurable room. It records
// SendText messages for assertion. It satisfies the full Actor interface.
type salvageFakeActor struct {
	char      *characters.Character
	room      *rooms.Room
	name      string
	isPlayer  bool
	userId    int
	mobInstId int
	sent      []string
}

func newSalvageFakeActor(t *testing.T, name string, room *rooms.Room, isPlayer bool, userId int) *salvageFakeActor {
	t.Helper()
	c := &characters.Character{
		Name:  name,
		Buffs: buffs.New(),
	}
	c.Stats.Perception.ValueAdj = 100
	return &salvageFakeActor{
		char:     c,
		room:     room,
		name:     name,
		isPlayer: isPlayer,
		userId:   userId,
	}
}

func newSalvageMobActor(t *testing.T, mob *mobs.Mob, room *rooms.Room) *salvageFakeActor {
	t.Helper()
	c := &characters.Character{
		Name:  mob.Character.Name,
		Buffs: buffs.New(),
	}
	return &salvageFakeActor{
		char:      c,
		room:      room,
		name:      mob.Character.Name,
		isPlayer:  false,
		mobInstId: mob.InstanceId,
	}
}

func (a *salvageFakeActor) GetCharacter() *characters.Character    { return a.char }
func (a *salvageFakeActor) GetRoom() *rooms.Room                   { return a.room }
func (a *salvageFakeActor) GetName() string                        { return a.name }
func (a *salvageFakeActor) IsPlayer() bool                         { return a.isPlayer }
func (a *salvageFakeActor) GetUserId() int                         { return a.userId }
func (a *salvageFakeActor) GetMobInstanceId() int                  { return a.mobInstId }
func (a *salvageFakeActor) AddBuff(_ int, _ string)                {}
func (a *salvageFakeActor) OnSkillUse(_ string) bool               { return false }
func (a *salvageFakeActor) OnStatUse(_ string) bool                { return false }
func (a *salvageFakeActor) OnCriticalSuccess(_ string)             {}
func (a *salvageFakeActor) OnCriticalFailure(_ string)             {}
func (a *salvageFakeActor) SendRoomCommunication(_ string, _ bool) {}
func (a *salvageFakeActor) SendText(_ messaging.Category, msg string) {
	a.sent = append(a.sent, msg)
}

// newSalvageTestRoom builds a minimal Room with the specified RoomId.
func newSalvageTestRoom(t *testing.T, roomId int) *rooms.Room {
	t.Helper()
	return &rooms.Room{RoomId: roomId}
}

// newSalvageTestMob builds a minimal Mob with an InstanceId.
func newSalvageTestMob(t *testing.T, instId int, name string, roomId int) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{
		InstanceId: instId,
	}
	m.Character.Name = name
	m.Character.Buffs = buffs.New()
	m.Character.RoomId = roomId
	return m
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestSalvage_NoTargetFails confirms an empty options struct
// (no corpse, no item) returns a reason without rolling.
func TestSalvage_NoTargetFails(t *testing.T) {
	room := newSalvageTestRoom(t, 9401)
	user := newSalvageFakeActor(t, "SalvageTester", room, true, 1)

	result := Salvage(user, SalvageOptions{})

	if result.Succeeded {
		t.Error("expected Succeeded=false with no target")
	}
	if result.Reason == "" {
		t.Error("expected Reason to be set")
	}
	if result.RollHappened {
		t.Error("expected RollHappened=false with no target")
	}
}

// TestSalvage_CorpseModeNoCorpse confirms TargetCorpse with no
// eligible corpse in the room returns Failure cleanly.
func TestSalvage_CorpseModeNoCorpse(t *testing.T) {
	room := newSalvageTestRoom(t, 9402)
	mob := newSalvageTestMob(t, 9998, "SalvageMob", room.RoomId)
	actor := newSalvageMobActor(t, mob, room)

	result := Salvage(actor, SalvageOptions{TargetCorpse: true})

	if result.Succeeded {
		t.Error("expected Succeeded=false with no corpse in room")
	}
}

// TestSalvage_ItemModeInvalidUuid confirms TargetItemUuid pointing
// at a non-existent UUID returns Failure with a reason.
func TestSalvage_ItemModeInvalidUuid(t *testing.T) {
	room := newSalvageTestRoom(t, 9403)
	user := newSalvageFakeActor(t, "SalvageTester2", room, true, 2)

	result := Salvage(user, SalvageOptions{TargetItemUuid: "nonexistent-uuid"})

	if result.Succeeded {
		t.Error("expected Succeeded=false for invalid UUID")
	}
}

// TestSalvage_MobActorSilent confirms MobActor.SendText is not called.
func TestSalvage_MobActorSilent(t *testing.T) {
	room := newSalvageTestRoom(t, 9404)
	mob := newSalvageTestMob(t, 9997, "SalvageSilentMob", room.RoomId)
	actor := newSalvageMobActor(t, mob, room)

	_ = Salvage(actor, SalvageOptions{TargetCorpse: true})

	if len(actor.sent) != 0 {
		t.Errorf("MobActor should be silent; got %d messages", len(actor.sent))
	}
}
