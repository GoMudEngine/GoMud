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

func TestAddActor_AddsToMembers(t *testing.T) {
	leader := newTestActor(0, 400)
	p := NewByActor(leader)
	member := newTestActor(0, 401)
	if !p.AddActor(member) {
		t.Error("AddActor returned false for new member")
	}
	if len(p.Members) != 2 {
		t.Errorf("Members len = %d, want 2", len(p.Members))
	}
	if GetByActor(member) != p {
		t.Error("member's GetByActor lookup did not return party")
	}
}

func TestAddActor_RejectsActorAlreadyInOtherParty(t *testing.T) {
	leaderA := newTestActor(0, 410)
	leaderB := newTestActor(0, 411)
	a := NewByActor(leaderA)
	NewByActor(leaderB)
	other := newTestActor(0, 412)
	a.AddActor(other)
	// Try adding `other` to the second party — should fail.
	b := GetByActor(leaderB)
	if b.AddActor(other) {
		t.Error("AddActor should reject an actor already in another party")
	}
}

func TestRemoveActor_RemovesFromMembers(t *testing.T) {
	leader := newTestActor(0, 420)
	member := newTestActor(0, 421)
	p := NewByActor(leader)
	p.AddActor(member)
	if !p.RemoveActor(member) {
		t.Error("RemoveActor returned false for present member")
	}
	if len(p.Members) != 1 {
		t.Errorf("Members len after remove = %d, want 1", len(p.Members))
	}
	if GetByActor(member) != nil {
		t.Error("member should no longer be tracked in any party")
	}
}

func TestGetByMobInstanceId_FindsParty(t *testing.T) {
	leader := newTestActor(0, 500)
	p := NewByActor(leader)
	if got := GetByMobInstanceId(500); got != p {
		t.Error("GetByMobInstanceId did not find the party")
	}
}

func TestGetByMobInstanceId_NilForUnknown(t *testing.T) {
	if GetByMobInstanceId(99999) != nil {
		t.Error("expected nil for unknown mob instance")
	}
}

func TestListAllParties_DeduplicatesByPointer(t *testing.T) {
	leader := newTestActor(0, 600)
	m1 := newTestActor(0, 601)
	m2 := newTestActor(0, 602)
	p := NewByActor(leader)
	p.AddActor(m1)
	p.AddActor(m2)
	all := ListAllParties()
	count := 0
	for _, party := range all {
		if party == p {
			count++
		}
	}
	if count != 1 {
		t.Errorf("party should appear once in ListAllParties, got %d", count)
	}
}

func TestDissolve_RemovesAllMembersFromRegistry(t *testing.T) {
	leader := newTestActor(0, 430)
	m1 := newTestActor(0, 431)
	m2 := newTestActor(0, 432)
	p := NewByActor(leader)
	p.AddActor(m1)
	p.AddActor(m2)
	p.Dissolve("test")
	if GetByActor(leader) != nil || GetByActor(m1) != nil || GetByActor(m2) != nil {
		t.Error("Dissolve did not remove all members from registry")
	}
}
