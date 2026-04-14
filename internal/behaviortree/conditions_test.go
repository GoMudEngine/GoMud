package behaviortree

import (
	"testing"
)

func TestCondKeywordMatch_Hit(t *testing.T) {
	fn := LookupCondition("keyword_match")
	if fn == nil {
		t.Fatal("keyword_match not registered")
	}

	ctx := &EvalContext{
		Event: EventContext{Text: "hello world"},
	}
	params := map[string]any{
		"keywords": []any{"world", "foo"},
	}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success, got %v", result)
	}
}

func TestCondKeywordMatch_Miss(t *testing.T) {
	fn := LookupCondition("keyword_match")

	ctx := &EvalContext{
		Event: EventContext{Text: "hello world"},
	}
	params := map[string]any{
		"keywords": []any{"foo", "bar"},
	}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure, got %v", result)
	}
}

func TestCondKeywordMatch_CaseInsensitive(t *testing.T) {
	fn := LookupCondition("keyword_match")

	ctx := &EvalContext{
		Event: EventContext{Text: "Hello WORLD"},
	}
	params := map[string]any{
		"keywords": []any{"world"},
	}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success, got %v", result)
	}
}

func TestCondStateEquals_Match(t *testing.T) {
	fn := LookupCondition("state_equals")
	if fn == nil {
		t.Fatal("state_equals not registered")
	}

	state := NewBehaviorState()
	state.Set("mood", "angry")

	ctx := &EvalContext{
		MobState: state,
	}
	params := map[string]any{
		"key":   "mood",
		"value": "angry",
	}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success, got %v", result)
	}
}

func TestCondStateEquals_Miss(t *testing.T) {
	fn := LookupCondition("state_equals")

	state := NewBehaviorState()
	state.Set("mood", "calm")

	ctx := &EvalContext{
		MobState: state,
	}
	params := map[string]any{
		"key":   "mood",
		"value": "angry",
	}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure, got %v", result)
	}
}

func TestCondStateEquals_NilState(t *testing.T) {
	fn := LookupCondition("state_equals")

	ctx := &EvalContext{}
	params := map[string]any{
		"key":   "mood",
		"value": "angry",
	}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure for nil state, got %v", result)
	}
}

func TestCondRandomChance_AlwaysSucceeds(t *testing.T) {
	fn := LookupCondition("random_chance")
	if fn == nil {
		t.Fatal("random_chance not registered")
	}

	ctx := &EvalContext{}
	params := map[string]any{
		"percent": 100,
	}
	for i := 0; i < 50; i++ {
		if result := fn(params, ctx); result != Success {
			t.Fatalf("100%% chance failed on iteration %d", i)
		}
	}
}

func TestCondRandomChance_AlwaysFails(t *testing.T) {
	fn := LookupCondition("random_chance")

	ctx := &EvalContext{}
	params := map[string]any{
		"percent": 0,
	}
	for i := 0; i < 50; i++ {
		if result := fn(params, ctx); result != Failure {
			t.Fatalf("0%% chance succeeded on iteration %d", i)
		}
	}
}

func TestCondRandomChance_Float64Param(t *testing.T) {
	fn := LookupCondition("random_chance")

	ctx := &EvalContext{}
	// YAML parses numbers as float64
	params := map[string]any{
		"percent": float64(100),
	}
	if result := fn(params, ctx); result != Success {
		t.Error("expected Success with float64(100)")
	}
}

func TestCondItemMatches_Hit(t *testing.T) {
	fn := LookupCondition("item_matches")
	if fn == nil {
		t.Fatal("item_matches not registered")
	}

	ctx := &EvalContext{
		Event: EventContext{ItemId: 42},
	}
	params := map[string]any{
		"item_id": 42,
	}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success, got %v", result)
	}
}

func TestCondItemMatches_Miss(t *testing.T) {
	fn := LookupCondition("item_matches")

	ctx := &EvalContext{
		Event: EventContext{ItemId: 42},
	}
	params := map[string]any{
		"item_id": 99,
	}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure, got %v", result)
	}
}

func TestCondItemMatches_Float64Param(t *testing.T) {
	fn := LookupCondition("item_matches")

	ctx := &EvalContext{
		Event: EventContext{ItemId: 42},
	}
	params := map[string]any{
		"item_id": float64(42),
	}
	if result := fn(params, ctx); result != Success {
		t.Error("expected Success with float64 item_id")
	}
}

func TestCondLookup_AllRegistered(t *testing.T) {
	names := []string{
		"keyword_match", "player_has_quest", "player_missing_quest",
		"player_has_item", "player_has_gold", "player_has_flag",
		"mob_in_combat", "mob_health_below", "mob_at_home",
		"time_of_day", "round_mod", "random_chance",
		"state_equals", "players_in_room", "item_matches",
	}
	for _, name := range names {
		if LookupCondition(name) == nil {
			t.Errorf("condition %q not registered", name)
		}
	}
}

func TestCondLookup_Unknown(t *testing.T) {
	if LookupCondition("nonexistent") != nil {
		t.Error("expected nil for unknown condition")
	}
}
