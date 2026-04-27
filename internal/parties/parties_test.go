package parties

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// testActor extends fakeActor (which satisfies actorIdentity) with the three
// extra methods that partyActor requires. Returning nil/empty is fine for
// these unit tests — the API functions don't dereference those values.
type testActor struct {
	fakeActor
}

var _ partyActor = (*testActor)(nil)

func (t *testActor) GetCharacter() *characters.Character { return nil }
func (t *testActor) GetRoom() *rooms.Room                { return nil }
func (t *testActor) GetName() string                     { return "" }

// newTestActor constructs a testActor with the given userId and mobInstanceId.
// Pass userId=0 for an NPC actor; pass mobInstanceId=0 for a player actor.
func newTestActor(userId, mobInstanceId int) *testActor {
	return &testActor{fakeActor{userId: userId, mobInstanceId: mobInstanceId}}
}

func TestNewByActor_CreatesParty(t *testing.T) {
	leader := newTestActor(0, 100) // mob id 100
	p := NewByActor(leader)
	if p == nil {
		t.Fatal("NewByActor returned nil")
	}
	if p.Leader != leader {
		t.Error("Leader not set to provided actor")
	}
	if len(p.Members) != 1 || p.Members[0] != leader {
		t.Error("Leader not added to Members")
	}
}

func TestNewByActor_ReturnsNilIfActorAlreadyInParty(t *testing.T) {
	leader := newTestActor(0, 200)
	first := NewByActor(leader)
	if first == nil {
		t.Fatal("first NewByActor unexpectedly nil")
	}
	second := NewByActor(leader)
	if second != nil {
		t.Error("second NewByActor for same actor should return nil")
	}
}

func TestGetByActor_ReturnsParty(t *testing.T) {
	leader := newTestActor(0, 300)
	p := NewByActor(leader)
	got := GetByActor(leader)
	if got != p {
		t.Errorf("GetByActor returned wrong party")
	}
}

func TestGetByActor_NilForUnknown(t *testing.T) {
	a := newTestActor(0, 9999)
	if GetByActor(a) != nil {
		t.Error("GetByActor should return nil for actor not in any party")
	}
}
