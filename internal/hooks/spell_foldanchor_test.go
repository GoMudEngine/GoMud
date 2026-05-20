package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
)

// fakeActor is a minimal Actor implementation for unit tests. It records
// SendText / SendRoomText calls so assertions can verify behavior without
// touching the user/mob packages.
type fakeActor struct {
	char      *characters.Character
	room      *rooms.Room
	name      string
	isPlayer  bool
	userId    int
	mobInstId int
	selfTexts []string
	roomTexts []string
}

func (f *fakeActor) GetCharacter() *characters.Character { return f.char }
func (f *fakeActor) GetRoom() *rooms.Room                { return f.room }
func (f *fakeActor) SendText(_ messaging.Category, msg string) {
	f.selfTexts = append(f.selfTexts, msg)
}
func (f *fakeActor) SendTextLegacy(msg string) { f.selfTexts = append(f.selfTexts, msg) }
func (f *fakeActor) SendRoomText(msg string, excludeSelf bool) {
	f.roomTexts = append(f.roomTexts, msg)
}
func (f *fakeActor) SendRoomCommunication(msg string, excludeSelf bool) {}
func (f *fakeActor) GetName() string                                    { return f.name }
func (f *fakeActor) IsPlayer() bool                                     { return f.isPlayer }
func (f *fakeActor) GetUserId() int                                     { return f.userId }
func (f *fakeActor) GetMobInstanceId() int                              { return f.mobInstId }
func (f *fakeActor) AddBuff(buffId int, source string)                  {}
func (f *fakeActor) OnSkillUse(skillName string) bool                   { return false }
func (f *fakeActor) OnStatUse(statName string) bool                     { return false }
func (f *fakeActor) OnCriticalSuccess(skillName string)                 {}
func (f *fakeActor) OnCriticalFailure(skillName string)                 {}

// compile-time check
var _ actions.Actor = (*fakeActor)(nil)

// Resolving fold-anchor must write the actor's current room ID into
// MiscData["fold-anchor-room"]. Works for both player and mob actors.
func TestResolveFoldAnchor_PlayerActor_SetsMiscData(t *testing.T) {
	c := characters.New()
	c.RoomId = 4036

	a := &fakeActor{
		char:     c,
		name:     "TestPlayer",
		isPlayer: true,
		userId:   42,
	}

	resolveFoldAnchor(a)

	got := c.GetMiscData("fold-anchor-room")
	assert.Equal(t, 4036, got, "MiscData should hold the actor's current room ID")
	assert.Len(t, a.selfTexts, 1, "player should receive one self message")
	assert.Len(t, a.roomTexts, 1, "room should receive one shimmer broadcast")
}

func TestResolveFoldAnchor_MobActor_SetsMiscData(t *testing.T) {
	c := characters.New()
	c.RoomId = 4036

	a := &fakeActor{
		char:      c,
		name:      "Old Edrin",
		isPlayer:  false,
		mobInstId: 99,
	}

	resolveFoldAnchor(a)

	got := c.GetMiscData("fold-anchor-room")
	assert.Equal(t, 4036, got, "MiscData should hold the actor's current room ID")
	assert.Len(t, a.roomTexts, 1, "room should still get the shimmer broadcast for mob actors")
}
