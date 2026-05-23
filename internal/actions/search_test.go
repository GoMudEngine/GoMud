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
// search-test-specific helpers
// ---------------------------------------------------------------------------

// searchFakeActor is a minimal Actor with a configurable room. It records
// SendText messages for assertion. It satisfies the full Actor interface.
type searchFakeActor struct {
	char      *characters.Character
	room      *rooms.Room
	name      string
	isPlayer  bool
	userId    int
	mobInstId int
	sent      []string
}

func newSearchFakeActor(name string, room *rooms.Room, isPlayer bool, userId int) *searchFakeActor {
	c := &characters.Character{
		Name:  name,
		Buffs: buffs.New(),
	}
	c.Stats.Perception.ValueAdj = 100
	return &searchFakeActor{
		char:     c,
		room:     room,
		name:     name,
		isPlayer: isPlayer,
		userId:   userId,
	}
}

func newSearchMobActor(name string, room *rooms.Room, mobInstId int) *searchFakeActor {
	c := &characters.Character{
		Name:  name,
		Buffs: buffs.New(),
	}
	return &searchFakeActor{
		char:      c,
		room:      room,
		name:      name,
		isPlayer:  false,
		mobInstId: mobInstId,
	}
}

func (a *searchFakeActor) GetCharacter() *characters.Character    { return a.char }
func (a *searchFakeActor) GetRoom() *rooms.Room                   { return a.room }
func (a *searchFakeActor) GetName() string                        { return a.name }
func (a *searchFakeActor) IsPlayer() bool                         { return a.isPlayer }
func (a *searchFakeActor) GetUserId() int                         { return a.userId }
func (a *searchFakeActor) GetMobInstanceId() int                  { return a.mobInstId }
func (a *searchFakeActor) AddBuff(_ int, _ string)                {}
func (a *searchFakeActor) OnSkillUse(_ string) bool               { return false }
func (a *searchFakeActor) OnStatUse(_ string) bool                { return false }
func (a *searchFakeActor) OnCriticalSuccess(_ string)             {}
func (a *searchFakeActor) OnCriticalFailure(_ string)             {}
func (a *searchFakeActor) SendRoomCommunication(_ string, _ bool) {}
func (a *searchFakeActor) SendText(_ messaging.Category, msg string) {
	a.sent = append(a.sent, msg)
}

// newSearchTestRoom builds a minimal Room with no exits.
func newSearchTestRoom(roomId int) *rooms.Room {
	return &rooms.Room{RoomId: roomId}
}

// newSearchTestMob builds a minimal Mob with an InstanceId.
func newSearchTestMob(instId int, name string, roomId int) *mobs.Mob {
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

// TestSearch_EmptyRoomNothingFound confirms an empty room produces
// zero hits across all tiers.
func TestSearch_EmptyRoomNothingFound(t *testing.T) {
	room := newSearchTestRoom(9201)
	actor := newSearchFakeActor("SearchTester", room, false, 0)

	result := Search(actor, SearchOptions{})

	if len(result.HiddenExitsFound) != 0 ||
		len(result.HiddenContainersFound) != 0 ||
		len(result.StashedItemsFound) != 0 ||
		len(result.HiddenPlayersFound) != 0 ||
		len(result.HiddenMobsFound) != 0 ||
		len(result.HiddenNounsFound) != 0 {
		t.Error("expected all-empty result for empty room")
	}
}

// TestSearch_CooldownGate confirms two consecutive calls within the
// cooldown return OnCooldown=true on the second.
func TestSearch_CooldownGate(t *testing.T) {
	room := newSearchTestRoom(9202)
	actor := newSearchFakeActor("SearchTester2", room, false, 0)

	_ = Search(actor, SearchOptions{})
	second := Search(actor, SearchOptions{})

	if !second.OnCooldown {
		t.Error("second call within 2-round window should return OnCooldown=true")
	}
}

// TestSearch_MobActorSilent confirms mob actor invocation works
// without panic.
func TestSearch_MobActorSilent(t *testing.T) {
	room := newSearchTestRoom(9203)
	mob := newSearchTestMob(8801, "SearchMob", 9203)
	actor := newSearchMobActor("SearchMob", room, mob.InstanceId)

	_ = Search(actor, SearchOptions{})
}
