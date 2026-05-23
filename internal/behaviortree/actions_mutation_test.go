package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// buildMutationMob creates a minimal mob instance for mutation action tests.
// Uses the same pattern as buildForagerMob in actions_forager_test.go.
func buildMutationMob(t *testing.T, instanceId int, mobId mobs.MobId, roomId int) *mobs.Mob {
	t.Helper()
	mob := &mobs.Mob{
		MobId:      mobId,
		InstanceId: instanceId,
		HomeRoomId: roomId,
	}
	mob.Character.RoomId = roomId
	mob.Character.Health = 100
	mob.Character.HealthMax.Value = 100
	mob.Character.Stamina = 100
	mob.Character.StaminaMax.Value = 100
	mob.Character.Buffs = buffs.New()
	mob.Character.Stats.Strength.ValueAdj = 100
	mob.Character.Stats.Dexterity.ValueAdj = 100
	mob.Character.Stats.Vitality.ValueAdj = 100
	mob.Character.Stats.Perception.ValueAdj = 100
	mob.Character.Stats.Willpower.ValueAdj = 100
	mob.Character.Stats.Charisma.ValueAdj = 100
	mobs.SetInstanceForTest(instanceId, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(instanceId, nil) })
	return mob
}

// TestTryMutationActive_RegisteredInActionRegistry verifies the action is
// present in the actionRegistry.
func TestTryMutationActive_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["try_mutation_active"]; !ok {
		t.Fatal("try_mutation_active not registered in actionRegistry")
	}
}

// TestTryMutationActive_NoKeyOrKeys verifies that a node with neither `key`
// nor `keys` returns Failure (and logs an error).
func TestTryMutationActive_NoKeyOrKeys(t *testing.T) {
	mob := buildMutationMob(t, 9100, 99999, 1)
	_ = mob

	ctx := &EvalContext{
		InstanceId: 9100,
		RoomId:     1,
		MobState:   NewBehaviorState(),
	}
	res := actTryMutationActive(map[string]any{}, ctx)
	if res != Failure {
		t.Errorf("expected Failure with neither key nor keys, got %v", res)
	}
}

// TestTryMutationActive_SingleKey_MobLacksMutation verifies that a mob
// without the requested mutation returns Failure (the mutationPreamble
// gate blocks with "no-mutation").
func TestTryMutationActive_SingleKey_MobLacksMutation(t *testing.T) {
	mob := buildMutationMob(t, 9101, 99999, 1)
	_ = mob

	ctx := &EvalContext{
		InstanceId: 9101,
		RoomId:     1,
		MobState:   NewBehaviorState(),
	}
	params := map[string]any{
		"key": "healing-gel",
	}
	res := actTryMutationActive(params, ctx)
	if res != Failure {
		t.Errorf("mob without mutation: expected Failure, got %v", res)
	}
}

// TestTryMutationActive_UnknownKeySkipped verifies that an unrecognised
// mutation key (not in mutationTriggers) is skipped with a warn log and the
// action ultimately returns Failure when no valid candidate fires.
func TestTryMutationActive_UnknownKeySkipped(t *testing.T) {
	mob := buildMutationMob(t, 9102, 99999, 1)
	_ = mob

	ctx := &EvalContext{
		InstanceId: 9102,
		RoomId:     1,
		MobState:   NewBehaviorState(),
	}
	params := map[string]any{
		"key": "not-a-real-mutation",
	}
	// Should not panic; should return Failure because there are no valid
	// candidates to fire.
	res := actTryMutationActive(params, ctx)
	if res != Failure {
		t.Errorf("unknown key: expected Failure, got %v", res)
	}
}

// TestTryMutationActive_NilMob verifies that a missing mob instance returns
// Failure without panicking.
func TestTryMutationActive_NilMob(t *testing.T) {
	// No mob registered for this instance ID — mobs.GetInstance returns nil.
	ctx := &EvalContext{
		InstanceId: 9199,
		RoomId:     1,
		MobState:   NewBehaviorState(),
	}
	params := map[string]any{
		"key": "healing-gel",
	}
	res := actTryMutationActive(params, ctx)
	if res != Failure {
		t.Errorf("nil mob: expected Failure, got %v", res)
	}
}

// TestCollectMutationKeys_KeyOnly verifies single-key collection.
func TestCollectMutationKeys_KeyOnly(t *testing.T) {
	params := map[string]any{"key": "healing-gel"}
	got := collectMutationKeys(params)
	if len(got) != 1 || got[0] != "healing-gel" {
		t.Errorf("key only: got %v, want [healing-gel]", got)
	}
}

// TestCollectMutationKeys_KeysOnly verifies list-key collection.
func TestCollectMutationKeys_KeysOnly(t *testing.T) {
	params := map[string]any{
		"keys": []any{"blinding-flash", "healing-gel"},
	}
	got := collectMutationKeys(params)
	if len(got) != 2 || got[0] != "blinding-flash" || got[1] != "healing-gel" {
		t.Errorf("keys only: got %v, want [blinding-flash healing-gel]", got)
	}
}

// TestCollectMutationKeys_BothKeyAndKeys verifies that `key` is prepended
// before `keys` when both are set.
func TestCollectMutationKeys_BothKeyAndKeys(t *testing.T) {
	params := map[string]any{
		"key":  "sonic-shout",
		"keys": []any{"blinding-flash", "healing-gel"},
	}
	got := collectMutationKeys(params)
	want := []string{"sonic-shout", "blinding-flash", "healing-gel"}
	if len(got) != len(want) {
		t.Fatalf("both key+keys: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("both key+keys[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestCollectMutationKeys_Empty verifies empty params yield empty slice.
func TestCollectMutationKeys_Empty(t *testing.T) {
	got := collectMutationKeys(map[string]any{})
	if len(got) != 0 {
		t.Errorf("empty params: got %v, want []", got)
	}
}

// TestMutationTriggers_AllSixPresent verifies that all six expected
// mutation keys are registered in the mutationTriggers map.
func TestMutationTriggers_AllSixPresent(t *testing.T) {
	expected := []string{
		"blinding-flash",
		"blinding-spit",
		"healing-gel",
		"pacifism-aura",
		"sonic-shout",
		"toxic-bite",
	}
	for _, key := range expected {
		if _, ok := mutationTriggers[key]; !ok {
			t.Errorf("mutationTriggers missing key %q", key)
		}
	}
}
