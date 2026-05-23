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
// scan-test-specific helpers
// ---------------------------------------------------------------------------

// scanFakeActor is a minimal Actor with a configurable room. It records
// SendText messages for assertion. It satisfies the full Actor interface.
type scanFakeActor struct {
	char      *characters.Character
	room      *rooms.Room
	name      string
	isPlayer  bool
	userId    int
	mobInstId int
	sent      []string
}

func newScanFakeActor(name string, room *rooms.Room, isPlayer bool, userId int) *scanFakeActor {
	c := &characters.Character{
		Name:  name,
		Buffs: buffs.New(),
	}
	c.Stats.Perception.ValueAdj = 100
	return &scanFakeActor{
		char:     c,
		room:     room,
		name:     name,
		isPlayer: isPlayer,
		userId:   userId,
	}
}

func newScanMobActor(name string, room *rooms.Room, mobInstId int) *scanFakeActor {
	c := &characters.Character{
		Name:  name,
		Buffs: buffs.New(),
	}
	return &scanFakeActor{
		char:      c,
		room:      room,
		name:      name,
		isPlayer:  false,
		mobInstId: mobInstId,
	}
}

func (a *scanFakeActor) GetCharacter() *characters.Character    { return a.char }
func (a *scanFakeActor) GetRoom() *rooms.Room                   { return a.room }
func (a *scanFakeActor) GetName() string                        { return a.name }
func (a *scanFakeActor) IsPlayer() bool                         { return a.isPlayer }
func (a *scanFakeActor) GetUserId() int                         { return a.userId }
func (a *scanFakeActor) GetMobInstanceId() int                  { return a.mobInstId }
func (a *scanFakeActor) AddBuff(_ int, _ string)                {}
func (a *scanFakeActor) OnSkillUse(_ string) bool               { return false }
func (a *scanFakeActor) OnStatUse(_ string) bool                { return false }
func (a *scanFakeActor) OnCriticalSuccess(_ string)             {}
func (a *scanFakeActor) OnCriticalFailure(_ string)             {}
func (a *scanFakeActor) SendRoomCommunication(_ string, _ bool) {}
func (a *scanFakeActor) SendText(_ messaging.Category, msg string) {
	a.sent = append(a.sent, msg)
}

// newScanTestRoom builds a minimal Room with no exits.
func newScanTestRoom(roomId int) *rooms.Room {
	return &rooms.Room{RoomId: roomId}
}

// newScanTestMob builds a minimal Mob with an InstanceId.
func newScanTestMob(instId int, name string, roomId int) *mobs.Mob {
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

// TestScan_EmptyAdjacentRooms confirms Scan returns an empty (non-nil)
// Sightings slice when the actor's room has no exits.
func TestScan_EmptyAdjacentRooms(t *testing.T) {
	room := newScanTestRoom(9001)
	actor := newScanFakeActor("TestScout", room, false, 0)

	result := Scan(actor, ScanOptions{HostileOnly: false})

	if result.Sightings == nil {
		t.Error("Sightings should be an empty slice, not nil")
	}
	if len(result.Sightings) != 0 {
		t.Errorf("expected 0 sightings for room with no exits, got %d",
			len(result.Sightings))
	}
}

// TestScan_MobActorSilent confirms MobActor path: action returns structured
// data without panicking, and Sightings is non-nil.
func TestScan_MobActorSilent(t *testing.T) {
	room := newScanTestRoom(9002)
	mob := newScanTestMob(8801, "TestScoutMob", 9002)
	actor := newScanMobActor("TestScoutMob", room, mob.InstanceId)

	result := Scan(actor, ScanOptions{HostileOnly: true})

	if result.Sightings == nil {
		t.Error("Sightings should be empty slice, not nil")
	}
}

// TestScan_PlayerActorEmitsText confirms that a player actor (IsPlayer==true)
// receives at least the header "You scan the surrounding area..." line.
func TestScan_PlayerActorEmitsText(t *testing.T) {
	room := newScanTestRoom(9003)
	actor := newScanFakeActor("Scout", room, true, 1)

	Scan(actor, ScanOptions{})

	if len(actor.sent) == 0 {
		t.Fatal("player actor should receive SendText messages from Scan")
	}
	found := false
	for _, line := range actor.sent {
		if line == "You scan the surrounding area..." {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected header line in sent messages, got: %v", actor.sent)
	}
}

// TestScan_NilRoomReturnsEmpty confirms Scan handles a nil room gracefully.
func TestScan_NilRoomReturnsEmpty(t *testing.T) {
	actor := newScanFakeActor("Ghost", nil, false, 0)

	result := Scan(actor, ScanOptions{})

	if result.Sightings == nil {
		t.Error("Sightings should be empty slice, not nil, even for nil room")
	}
	if len(result.Sightings) != 0 {
		t.Errorf("expected 0 sightings for nil room, got %d", len(result.Sightings))
	}
}
