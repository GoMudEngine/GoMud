package behaviortree

import "testing"

// player_in_room_missing_quest gates ambient idle branches (e.g. the
// newbie-hub greeter's repeated invitation) on an un-quested player
// actually being present. These pin the guard rails; the
// any-player-present logic is covered by live smoke.

func TestPlayerInRoomMissingQuest_Registered(t *testing.T) {
	if _, ok := conditionRegistry["player_in_room_missing_quest"]; !ok {
		t.Fatal("player_in_room_missing_quest not registered in conditionRegistry")
	}
}

func TestPlayerInRoomMissingQuest_NoQuestParam_Failure(t *testing.T) {
	ctx := &EvalContext{RoomId: 1}
	if res := condPlayerInRoomMissingQuest(map[string]any{}, ctx); res != Failure {
		t.Errorf("expected Failure with no quest param, got %v", res)
	}
}

func TestPlayerInRoomMissingQuest_NoRoom_Failure(t *testing.T) {
	ctx := &EvalContext{RoomId: -999999}
	params := map[string]any{"quest": "30-end"}
	if res := condPlayerInRoomMissingQuest(params, ctx); res != Failure {
		t.Errorf("expected Failure for unloadable room, got %v", res)
	}
}
