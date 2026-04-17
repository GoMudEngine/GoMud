package rooms

import (
	"testing"
)

func TestZoneActivity_IncrementDecrement(t *testing.T) {
	resetZonePlayerCount()

	incrementZonePlayerCount("alpha")
	if !ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be active after one increment")
	}
	if ZoneHasPlayers("beta") {
		t.Fatal("beta should be inactive")
	}

	incrementZonePlayerCount("alpha")
	decrementZonePlayerCount("alpha")
	if !ZoneHasPlayers("alpha") {
		t.Fatal("alpha should still be active (1 remaining)")
	}

	decrementZonePlayerCount("alpha")
	if ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be inactive after matched decrement")
	}
}

func TestZoneActivity_DecrementClampsAtZero(t *testing.T) {
	resetZonePlayerCount()
	decrementZonePlayerCount("alpha") // underflow attempt
	if ZoneHasPlayers("alpha") {
		t.Fatal("underflowed zone must not be 'active'")
	}
	// The key should either be absent or clamped to zero — either is fine.
}

func TestZoneActivity_SnapshotActiveZones(t *testing.T) {
	resetZonePlayerCount()
	incrementZonePlayerCount("alpha")
	incrementZonePlayerCount("beta")
	incrementZonePlayerCount("beta")
	// gamma never incremented
	decrementZonePlayerCount("alpha")
	// alpha now zero

	snap := SnapshotActiveZones()
	if snap["alpha"] {
		t.Fatal("alpha should not be in snapshot")
	}
	if !snap["beta"] {
		t.Fatal("beta should be in snapshot")
	}
	if snap["gamma"] {
		t.Fatal("gamma should not be in snapshot")
	}
}

func TestZoneActivity_EmptyZoneStringWorks(t *testing.T) {
	resetZonePlayerCount()
	incrementZonePlayerCount("")
	if !ZoneHasPlayers("") {
		t.Fatal(`empty zone string is a valid key`)
	}
}

func TestZoneActivity_VerifyDetectsDrift(t *testing.T) {
	resetZonePlayerCount()
	// Set up a room with 2 players.
	r := &Room{RoomId: 1001, Zone: "alpha", players: []int{10, 20}}
	originalManager := roomManager
	defer func() { roomManager = originalManager }()
	roomManager = &RoomManager{
		rooms: map[int]*Room{1001: r},
	}

	// Incrementally maintained counter is wrong (one, not two).
	incrementZonePlayerCount("alpha")

	drift := VerifyZonePlayerCount()
	if drift["alpha"] != 1 {
		t.Fatalf("expected drift of +1 for alpha (ground truth 2, counter 1), got %v", drift)
	}
}

func TestAddPlayer_IncrementsZone(t *testing.T) {
	resetZonePlayerCount()

	r := &Room{RoomId: 2001, Zone: "alpha"}
	r.AddPlayer(10)

	if !ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be active after AddPlayer")
	}
}

func TestAddPlayer_DedupeDoesNotDoubleCount(t *testing.T) {
	resetZonePlayerCount()

	r := &Room{RoomId: 2002, Zone: "alpha"}

	originalManager := roomManager
	defer func() { roomManager = originalManager }()
	roomManager = &RoomManager{rooms: map[int]*Room{2002: r}}

	r.AddPlayer(10)
	r.AddPlayer(10) // duplicate — should no-op

	drift := VerifyZonePlayerCount()
	if len(drift) != 0 {
		t.Fatalf("counter should match ground truth, drift=%v", drift)
	}
}

func TestRemovePlayer_DecrementsZone(t *testing.T) {
	resetZonePlayerCount()

	r := &Room{RoomId: 2003, Zone: "alpha"}
	r.AddPlayer(10)
	if !ZoneHasPlayers("alpha") {
		t.Fatal("precondition: alpha should be active")
	}

	r.RemovePlayer(10)
	if ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be inactive after removal")
	}
}

func TestRemovePlayer_NotMember_NoDecrement(t *testing.T) {
	resetZonePlayerCount()
	incrementZonePlayerCount("alpha") // simulate prior player in zone

	r := &Room{RoomId: 2004, Zone: "alpha"} // this room has no players
	r.RemovePlayer(999)                     // userId 999 is not a member

	if !ZoneHasPlayers("alpha") {
		t.Fatal("alpha should still be active — we didn't actually remove anyone")
	}
}

func TestAddPlayer_EmptyZoneStringDoesNotCrash(t *testing.T) {
	resetZonePlayerCount()
	r := &Room{RoomId: 2005, Zone: ""}
	r.AddPlayer(10) // must not panic; empty zone is a valid key
	if !ZoneHasPlayers("") {
		t.Fatal(`empty zone string should key into the map`)
	}
}

func TestMoveToRoom_CrossZone_MovesCount(t *testing.T) {
	resetZonePlayerCount()
	originalManager := roomManager
	defer func() { roomManager = originalManager }()

	alphaRoom := &Room{RoomId: 3001, Zone: "alpha"}
	betaRoom := &Room{RoomId: 3002, Zone: "beta"}
	roomManager = &RoomManager{
		rooms:          map[int]*Room{3001: alphaRoom, 3002: betaRoom},
		roomsWithUsers: map[int]int{},
	}

	alphaRoom.AddPlayer(42)
	if !ZoneHasPlayers("alpha") || ZoneHasPlayers("beta") {
		t.Fatal("precondition: alpha active, beta inactive")
	}

	// Simulate a move by directly removing + adding (what MoveToRoom does).
	alphaRoom.RemovePlayer(42)
	betaRoom.AddPlayer(42)

	if ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be inactive after move-out")
	}
	if !ZoneHasPlayers("beta") {
		t.Fatal("beta should be active after move-in")
	}
}

func TestMoveToRoom_SameZone_CountUnchanged(t *testing.T) {
	resetZonePlayerCount()
	originalManager := roomManager
	defer func() { roomManager = originalManager }()

	r1 := &Room{RoomId: 3003, Zone: "alpha"}
	r2 := &Room{RoomId: 3004, Zone: "alpha"}
	roomManager = &RoomManager{
		rooms: map[int]*Room{3003: r1, 3004: r2},
	}

	r1.AddPlayer(42)
	r1.RemovePlayer(42)
	r2.AddPlayer(42)

	// alpha should have exactly 1 player.
	if !ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be active")
	}

	drift := VerifyZonePlayerCount()
	if len(drift) != 0 {
		t.Fatalf("expected no drift, got %v", drift)
	}
}

func TestRebuildZonePlayerCount_MatchesGroundTruth(t *testing.T) {
	resetZonePlayerCount()
	originalManager := roomManager
	defer func() { roomManager = originalManager }()

	roomManager = &RoomManager{
		rooms: map[int]*Room{
			4001: {RoomId: 4001, Zone: "alpha", players: []int{1, 2}},
			4002: {RoomId: 4002, Zone: "beta", players: []int{3}},
			4003: {RoomId: 4003, Zone: "alpha"}, // empty
		},
	}

	// Counter intentionally wrong to start.
	incrementZonePlayerCount("ghost")

	RebuildZonePlayerCount()

	drift := VerifyZonePlayerCount()
	if len(drift) != 0 {
		t.Fatalf("expected no drift after rebuild, got %v", drift)
	}
	if !ZoneHasPlayers("alpha") || !ZoneHasPlayers("beta") {
		t.Fatal("both populated zones should be active after rebuild")
	}
	if ZoneHasPlayers("ghost") {
		t.Fatal("stale 'ghost' entry should have been cleared by rebuild")
	}
}
