package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
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

func TestCondStateGreaterThan_Above(t *testing.T) {
	state := NewBehaviorState()
	state.Set("counter", 5)
	ctx := &EvalContext{MobState: state}
	fn := LookupCondition("state_greater_than")
	if fn == nil {
		t.Fatal("state_greater_than not registered")
	}
	result := fn(map[string]any{"key": "counter", "value": 3}, ctx)
	if result != Success {
		t.Error("expected Success when 5 > 3")
	}
}

func TestCondStateGreaterThan_Equal(t *testing.T) {
	state := NewBehaviorState()
	state.Set("counter", 3)
	ctx := &EvalContext{MobState: state}
	fn := LookupCondition("state_greater_than")
	result := fn(map[string]any{"key": "counter", "value": 3}, ctx)
	if result != Failure {
		t.Error("expected Failure when 3 == 3 (not greater)")
	}
}

func TestCondStateGreaterThan_Below(t *testing.T) {
	state := NewBehaviorState()
	state.Set("counter", 1)
	ctx := &EvalContext{MobState: state}
	fn := LookupCondition("state_greater_than")
	result := fn(map[string]any{"key": "counter", "value": 3}, ctx)
	if result != Failure {
		t.Error("expected Failure when 1 < 3")
	}
}

func TestCondStateGreaterThan_NilState(t *testing.T) {
	ctx := &EvalContext{}
	fn := LookupCondition("state_greater_than")
	result := fn(map[string]any{"key": "counter", "value": 0}, ctx)
	if result != Failure {
		t.Error("expected Failure for nil state")
	}
}

func TestCondStateGreaterThan_Float64Param(t *testing.T) {
	state := NewBehaviorState()
	state.Set("counter", 5)
	ctx := &EvalContext{MobState: state}
	fn := LookupCondition("state_greater_than")
	// YAML parses numbers as float64
	result := fn(map[string]any{"key": "counter", "value": float64(3)}, ctx)
	if result != Success {
		t.Error("expected Success with float64 threshold")
	}
}

func TestCondAllNewRegistered(t *testing.T) {
	for _, name := range []string{
		"mob_has_buff", "player_has_spell",
		"player_has_misc_data", "state_greater_than",
		"multiple_enemies",
	} {
		if LookupCondition(name) == nil {
			t.Errorf("condition %q not registered", name)
		}
	}
}

// Phase 4c condition tests

func TestCondCommandMatches_Hit(t *testing.T) {
	fn := LookupCondition("command_matches")
	if fn == nil {
		t.Fatal("command_matches not registered")
	}

	params := map[string]any{"commands": []any{"look", "examine"}}
	ctx := &EvalContext{Event: EventContext{Command: "look"}}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success for command 'look', got %v", result)
	}
}

func TestCondCommandMatches_Miss(t *testing.T) {
	fn := LookupCondition("command_matches")

	params := map[string]any{"commands": []any{"look", "examine"}}
	ctx := &EvalContext{Event: EventContext{Command: "east"}}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure for command 'east', got %v", result)
	}
}

func TestCondCommandMatches_MissingParam(t *testing.T) {
	fn := LookupCondition("command_matches")

	params := map[string]any{}
	ctx := &EvalContext{Event: EventContext{Command: "look"}}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure when 'commands' param absent, got %v", result)
	}
}

func TestCondCommandRestContains_Hit(t *testing.T) {
	fn := LookupCondition("command_rest_contains")
	if fn == nil {
		t.Fatal("command_rest_contains not registered")
	}

	// "chest" in mixed-case rest — verify case-insensitive match
	params := map[string]any{"keywords": []any{"chest"}}
	ctx := &EvalContext{Event: EventContext{Rest: "open the wooden CHEST"}}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success for case-insensitive rest match, got %v", result)
	}
}

func TestCondCommandRestContains_EmptyRest(t *testing.T) {
	fn := LookupCondition("command_rest_contains")

	params := map[string]any{"keywords": []any{"chest"}}
	ctx := &EvalContext{Event: EventContext{Rest: ""}}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure for empty rest, got %v", result)
	}
}

func TestCondMobInRoom_Hit(t *testing.T) {
	fn := LookupCondition("mob_in_room")
	if fn == nil {
		t.Fatal("mob_in_room not registered")
	}

	cleanRoom := seedTestRoom(t, 1, "TestZone")
	defer cleanRoom()
	cleanMob := seedTestMob(t, 5, 105, 1, "Goblin")
	defer cleanMob()

	// seedTestMob registers the instance in the mob registry but does NOT
	// wire it into the room's mob list — add it explicitly.
	room := rooms.LoadRoom(1)
	if room == nil {
		t.Fatal("seedTestRoom did not register room 1")
	}
	room.AddMob(105)

	params := map[string]any{"mob_id": 5}
	ctx := &EvalContext{RoomId: 1}
	if result := fn(params, ctx); result != Success {
		t.Errorf("expected Success when mob template 5 is in room, got %v", result)
	}
}

func TestCondMobInRoom_NoRoom(t *testing.T) {
	fn := LookupCondition("mob_in_room")

	// No room seeded — LoadRoom(99) should return nil; condition must Fail
	// without panicking.
	params := map[string]any{"mob_id": 5}
	ctx := &EvalContext{RoomId: 99}
	if result := fn(params, ctx); result != Failure {
		t.Errorf("expected Failure when room does not exist, got %v", result)
	}
}
