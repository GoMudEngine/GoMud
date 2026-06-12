package behaviortree

import "testing"

// move_player is the portal/teleport action used by quest NPC btrees
// (e.g. Warden Esk's threshold portal in Pothole Coulee). These tests
// pin the guard rails; the happy path (real user, loaded room) is
// covered by live smoke tests.

func TestMovePlayer_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["move_player"]; !ok {
		t.Fatal("move_player not registered in actionRegistry")
	}
}

func TestMovePlayer_NoTriggeringPlayer_Failure(t *testing.T) {
	ctx := &EvalContext{
		Event: EventContext{EventType: "player_ask", UserId: 0},
	}
	if res := actMovePlayer(map[string]any{"room_id": 468}, ctx); res != Failure {
		t.Errorf("expected Failure with no triggering player, got %v", res)
	}
}

func TestMovePlayer_MissingRoomParam_Failure(t *testing.T) {
	ctx := &EvalContext{
		Event: EventContext{EventType: "player_ask", UserId: 42},
	}
	if res := actMovePlayer(map[string]any{}, ctx); res != Failure {
		t.Errorf("expected Failure with no room_id param, got %v", res)
	}
}

func TestMovePlayer_UnknownUser_Failure(t *testing.T) {
	// UserId 42 is not registered in the user manager, so MoveToRoom
	// must error and the action must surface that as Failure.
	ctx := &EvalContext{
		Event: EventContext{EventType: "player_ask", UserId: 42},
	}
	if res := actMovePlayer(map[string]any{"room_id": 468}, ctx); res != Failure {
		t.Errorf("expected Failure for unknown user, got %v", res)
	}
}
