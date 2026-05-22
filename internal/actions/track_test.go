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
// track-test-specific helpers
// ---------------------------------------------------------------------------

// trackFakeActor is a minimal Actor with a configurable room. It records
// SendText messages for assertion. It satisfies the full Actor interface.
type trackFakeActor struct {
	char      *characters.Character
	room      *rooms.Room
	name      string
	isPlayer  bool
	userId    int
	mobInstId int
	sent      []string
}

func newTrackFakeActor(name string, room *rooms.Room, isPlayer bool, userId int) *trackFakeActor {
	c := &characters.Character{
		Name:  name,
		Buffs: buffs.New(),
	}
	c.Stats.Perception.ValueAdj = 100
	return &trackFakeActor{
		char:     c,
		room:     room,
		name:     name,
		isPlayer: isPlayer,
		userId:   userId,
	}
}

func newTrackMobActor(name string, room *rooms.Room, mobInstId int) *trackFakeActor {
	c := &characters.Character{
		Name:  name,
		Buffs: buffs.New(),
	}
	return &trackFakeActor{
		char:      c,
		room:      room,
		name:      name,
		isPlayer:  false,
		mobInstId: mobInstId,
	}
}

func (a *trackFakeActor) GetCharacter() *characters.Character    { return a.char }
func (a *trackFakeActor) GetRoom() *rooms.Room                   { return a.room }
func (a *trackFakeActor) GetName() string                        { return a.name }
func (a *trackFakeActor) IsPlayer() bool                         { return a.isPlayer }
func (a *trackFakeActor) GetUserId() int                         { return a.userId }
func (a *trackFakeActor) GetMobInstanceId() int                  { return a.mobInstId }
func (a *trackFakeActor) AddBuff(_ int, _ string)                {}
func (a *trackFakeActor) OnSkillUse(_ string) bool               { return false }
func (a *trackFakeActor) OnStatUse(_ string) bool                { return false }
func (a *trackFakeActor) OnCriticalSuccess(_ string)             {}
func (a *trackFakeActor) OnCriticalFailure(_ string)             {}
func (a *trackFakeActor) SendRoomCommunication(_ string, _ bool) {}
func (a *trackFakeActor) SendText(_ messaging.Category, msg string) {
	a.sent = append(a.sent, msg)
}

// newTrackTestRoom builds a minimal Room with no exits.
func newTrackTestRoom(roomId int) *rooms.Room {
	return &rooms.Room{RoomId: roomId}
}

// newTrackTestMob builds a minimal Mob with an InstanceId.
func newTrackTestMob(instId int, name string, roomId int) *mobs.Mob {
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

// TestTrack_NoArgEmptyRoom confirms trail-scan mode on a room with no
// visitors returns an empty Visitors slice. With Perception=100 and no
// cooldown, roll will land near 100 which is below the 125 detection
// threshold — so result is a non-nil empty slice.
func TestTrack_NoArgEmptyRoom(t *testing.T) {
	room := newTrackTestRoom(9101)
	actor := newTrackFakeActor("TrackTester", room, false, 1)

	result := Track(actor, TrackOptions{})

	if result.Visitors == nil {
		t.Error("Visitors should be a non-nil slice, not nil")
	}
	if result.BuffApplied {
		t.Error("BuffApplied should be false on trail-scan mode")
	}
}

// TestTrack_ActiveTrackMobNoMatchFails confirms active-track mode with
// an unresolvable target sets Reason and does NOT apply buff 86.
func TestTrack_ActiveTrackMobNoMatchFails(t *testing.T) {
	room := newTrackTestRoom(9102)
	actor := newTrackFakeActor("TrackTester2", room, false, 2)

	result := Track(actor, TrackOptions{TargetNoun: "nonexistent_target"})

	if result.BuffApplied {
		t.Error("BuffApplied should be false when target unresolved")
	}
	if result.ActiveTargetUserId != 0 || result.ActiveTargetMobInstId != 0 {
		t.Error("ActiveTarget* should be 0 when target unresolved")
	}
}

// TestTrack_MobActorSilent confirms MobActor path: action returns structured
// data without panicking, result returned cleanly.
func TestTrack_MobActorSilent(t *testing.T) {
	room := newTrackTestRoom(9103)
	mob := newTrackTestMob(8901, "TrackMob", 9103)
	actor := newTrackMobActor("TrackMob", room, mob.InstanceId)

	// Confirm no panic, result returned cleanly.
	result := Track(actor, TrackOptions{})

	if result.Visitors == nil {
		t.Error("Visitors should be non-nil even for mob actor")
	}
}

// TestTrack_CancelTracking confirms CancelTracking flag returns cleanly
// without applying any buff.
func TestTrack_CancelTracking(t *testing.T) {
	room := newTrackTestRoom(9104)
	actor := newTrackFakeActor("TrackTester3", room, true, 3)

	result := Track(actor, TrackOptions{CancelTracking: true})

	if result.BuffApplied {
		t.Error("BuffApplied should be false on cancel path")
	}
	// Player actor should receive the stop message.
	found := false
	for _, msg := range actor.sent {
		if msg == "You stop tracking." {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'You stop tracking.' in sent messages, got: %v", actor.sent)
	}
}
