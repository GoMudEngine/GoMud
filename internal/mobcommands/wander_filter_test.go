package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Finding 11: `wander loot` / `wander players` computed a filtered exit list
// and then discarded it, calling GetRandomExit() unconditionally. The filter
// had no effect whatsoever. These pin the selection contract.

func wanderTestRoom() *rooms.Room {
	return &rooms.Room{
		RoomId: 1,
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 100},
			"south": {RoomId: 200},
			"east":  {RoomId: 300},
			"west":  {RoomId: 400},
		},
	}
}

// The regression itself: a filtered candidate must actually be chosen.
func TestPickWanderExit_UsesFilteredCandidates(t *testing.T) {
	room := wanderTestRoom()
	options := []string{"north"}

	// Run repeatedly: pre-fix this picked uniformly from all four exits, so
	// a single lucky iteration would not prove anything.
	for i := 0; i < 200; i++ {
		name, roomId := pickWanderExit(options, room)
		if name != "north" {
			t.Fatalf("iteration %d: picked %q, want the only filtered candidate %q", i, name, "north")
		}
		if roomId != 100 {
			t.Fatalf("iteration %d: picked roomId %d, want 100 for exit north", i, roomId)
		}
	}
}

func TestPickWanderExit_StaysWithinCandidateSet(t *testing.T) {
	room := wanderTestRoom()
	options := []string{"north", "south"}
	allowed := map[string]int{"north": 100, "south": 200}

	for i := 0; i < 200; i++ {
		name, roomId := pickWanderExit(options, room)
		wantRoom, ok := allowed[name]
		if !ok {
			t.Fatalf("iteration %d: picked %q, which is outside the candidate set", i, name)
		}
		if roomId != wantRoom {
			t.Fatalf("iteration %d: exit %q gave roomId %d, want %d", i, name, roomId, wantRoom)
		}
	}
}

// Empty candidates is the common case (no adjacent room had loot/players).
// The mob must still wander rather than freeze.
func TestPickWanderExit_EmptyCandidatesFallsBackToRandomExit(t *testing.T) {
	room := wanderTestRoom()

	name, roomId := pickWanderExit(nil, room)
	if name == "" {
		t.Fatal("empty candidates returned no exit; mob would stand still instead of wandering")
	}
	if _, ok := room.Exits[name]; !ok {
		t.Errorf("fallback picked %q, which is not an exit of this room", name)
	}
	if roomId == 0 {
		t.Errorf("fallback picked exit %q with roomId 0", name)
	}
}

// A candidate naming an exit the room does not have must not be trusted.
func TestPickWanderExit_StaleCandidateFallsBack(t *testing.T) {
	room := wanderTestRoom()

	name, roomId := pickWanderExit([]string{"up"}, room)
	if name == "up" {
		t.Fatal("picked a candidate that is not an exit of the room")
	}
	if _, ok := room.Exits[name]; !ok {
		t.Errorf("fallback picked %q, which is not an exit of this room", name)
	}
	if roomId == 0 {
		t.Errorf("fallback picked exit %q with roomId 0", name)
	}
}

// A room with no exits at all must not panic.
func TestPickWanderExit_NoExitsIsSafe(t *testing.T) {
	room := &rooms.Room{RoomId: 1, Exits: map[string]exit.RoomExit{}}

	if name, _ := pickWanderExit(nil, room); name != "" {
		t.Errorf("expected no exit from an exitless room, got %q", name)
	}
}
