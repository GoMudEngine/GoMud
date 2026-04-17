package rooms

import (
	"testing"
)

func TestZoneActivity_IncrementDecrement(t *testing.T) {
	ResetZonePlayerCount()

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
	ResetZonePlayerCount()
	decrementZonePlayerCount("alpha") // underflow attempt
	if ZoneHasPlayers("alpha") {
		t.Fatal("underflowed zone must not be 'active'")
	}
	// The key should either be absent or clamped to zero — either is fine.
}

func TestZoneActivity_SnapshotActiveZones(t *testing.T) {
	ResetZonePlayerCount()
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
	ResetZonePlayerCount()
	incrementZonePlayerCount("")
	if !ZoneHasPlayers("") {
		t.Fatal(`empty zone string is a valid key`)
	}
}

func TestZoneActivity_VerifyDetectsDrift(t *testing.T) {
	ResetZonePlayerCount()
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
