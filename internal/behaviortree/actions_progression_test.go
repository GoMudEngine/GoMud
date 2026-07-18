package behaviortree

import "testing"

// grant_progression forces one real skill-progression event for the triggering
// player (used by the newcomer tutorial to show the banner once). These pin the
// guardrails; the happy path (real user, skill actually increments + banner) is
// covered by the live tutorial walk-through, matching actions_move_player_test.go.

func TestGrantProgression_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["grant_progression"]; !ok {
		t.Fatal("grant_progression not registered in actionRegistry")
	}
}

func TestGrantProgression_NoTriggeringPlayer_Failure(t *testing.T) {
	ctx := &EvalContext{Event: EventContext{EventType: "room_command", UserId: 0}}
	if res := actGrantProgression(map[string]any{"skill": "spellcasting"}, ctx); res != Failure {
		t.Errorf("expected Failure with no triggering player, got %v", res)
	}
}

func TestGrantProgression_MissingSkillParam_Failure(t *testing.T) {
	// UserId 42 is not registered; but the missing-skill guard should fire first.
	ctx := &EvalContext{Event: EventContext{EventType: "room_command", UserId: 42}}
	if res := actGrantProgression(map[string]any{}, ctx); res != Failure {
		t.Errorf("expected Failure with no skill param, got %v", res)
	}
}

func TestGrantProgression_UnknownUser_Failure(t *testing.T) {
	ctx := &EvalContext{Event: EventContext{EventType: "room_command", UserId: 42}}
	if res := actGrantProgression(map[string]any{"skill": "spellcasting"}, ctx); res != Failure {
		t.Errorf("expected Failure for unknown user, got %v", res)
	}
}
