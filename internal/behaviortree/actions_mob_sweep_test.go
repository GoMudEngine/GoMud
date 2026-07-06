package behaviortree

// Tests for the sweep_companions btree action (Task A5). The action itself
// only fans out to the companionSweep callback (wired at startup by main.go
// to hooks.PushCompanionsToRoom, avoiding a behaviortree -> hooks import
// cycle) — so these tests stub the callback directly rather than pulling in
// the hooks package's companion/user machinery. Gear-safety and the actual
// room-move mechanics are covered by
// internal/hooks/companion_follow_test.go's PushCompanionsToRoom tests.

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// TestActSweepCompanions_RegisteredInActionRegistry verifies the action is
// present in the registry.
func TestActSweepCompanions_RegisteredInActionRegistry(t *testing.T) {
	if LookupAction("sweep_companions") == nil {
		t.Fatal("sweep_companions must be registered in actionRegistry")
	}
}

// TestActSweepCompanions_NoCallbackWired_Failure verifies the action fails
// gracefully (does not panic) when SetCompanionSweep was never called.
func TestActSweepCompanions_NoCallbackWired_Failure(t *testing.T) {
	fn := LookupAction("sweep_companions")

	orig := companionSweep
	defer func() { companionSweep = orig }()
	companionSweep = nil

	ctx := &EvalContext{RoomId: 1}
	if result := fn(map[string]any{"dest_room": 2}, ctx); result != Failure {
		t.Errorf("expected Failure when no callback wired, got %v", result)
	}
}

// TestActSweepCompanions_InvokesCallbackForEveryPlayerInRoom verifies the
// action loops room.GetPlayers() and forwards dest_room to the callback for
// each player — the contract the Hull Sweeper boss add depends on.
func TestActSweepCompanions_InvokesCallbackForEveryPlayerInRoom(t *testing.T) {
	fn := LookupAction("sweep_companions")

	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanUser1 := seedTestUser(t, 1, "alice", "Alice", 1)
	defer cleanUser1()
	cleanUser2 := seedTestUser(t, 2, "bob", "Bob", 1)
	defer cleanUser2()

	room := rooms.LoadRoom(1)
	room.AddPlayer(1)
	room.AddPlayer(2)

	var calledUserIds []int
	var calledDestRoomIds []int
	orig := companionSweep
	defer func() { companionSweep = orig }()
	companionSweep = func(userId, destRoomId int) {
		calledUserIds = append(calledUserIds, userId)
		calledDestRoomIds = append(calledDestRoomIds, destRoomId)
	}

	ctx := &EvalContext{RoomId: 1}
	if result := fn(map[string]any{"dest_room": 99}, ctx); result != Success {
		t.Fatalf("expected Success, got %v", result)
	}

	if len(calledUserIds) != 2 {
		t.Fatalf("expected companionSweep invoked for 2 players, got %d calls: %v",
			len(calledUserIds), calledUserIds)
	}
	seen := map[int]bool{}
	for _, id := range calledUserIds {
		seen[id] = true
	}
	if !seen[1] || !seen[2] {
		t.Errorf("expected callback invoked for user 1 and user 2, got %v", calledUserIds)
	}
	for _, destRoomId := range calledDestRoomIds {
		if destRoomId != 99 {
			t.Errorf("expected dest_room=99 forwarded to every callback invocation, got %d", destRoomId)
		}
	}
}

// TestActSweepCompanions_MissingDestRoomParam_Failure verifies dest_room=0
// (unset) is rejected before touching the room or the callback.
func TestActSweepCompanions_MissingDestRoomParam_Failure(t *testing.T) {
	fn := LookupAction("sweep_companions")

	orig := companionSweep
	defer func() { companionSweep = orig }()
	companionSweep = func(userId, destRoomId int) {
		t.Fatal("callback must not be invoked when dest_room is missing")
	}

	ctx := &EvalContext{RoomId: 1}
	if result := fn(map[string]any{}, ctx); result != Failure {
		t.Errorf("expected Failure for missing dest_room, got %v", result)
	}
}

// TestActSweepCompanions_MissingRoom_Failure verifies a RoomId that doesn't
// resolve to a loaded room fails gracefully.
func TestActSweepCompanions_MissingRoom_Failure(t *testing.T) {
	fn := LookupAction("sweep_companions")

	orig := companionSweep
	defer func() { companionSweep = orig }()
	companionSweep = func(userId, destRoomId int) {
		t.Fatal("callback must not be invoked when the acting mob's room can't be loaded")
	}

	ctx := &EvalContext{RoomId: 999999} // never seeded
	if result := fn(map[string]any{"dest_room": 2}, ctx); result != Failure {
		t.Errorf("expected Failure for missing room, got %v", result)
	}
}
