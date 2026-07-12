package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// ---------------------------------------------------------------------------
// try_mutation_active_at_target tests
// ---------------------------------------------------------------------------

// mutTestRoomId is a synthetic room seeded per-test; the behaviortree test
// environment has no on-disk room data, so rooms.LoadRoom(anything) returns
// nil without seeding.
const mutTestRoomId = 9290

func seedMutationTestRoom(t *testing.T) {
	t.Helper()
	room := &rooms.Room{
		RoomId: mutTestRoomId,
		Zone:   "testzone",
		Exits:  map[string]exit.RoomExit{},
	}
	t.Cleanup(rooms.SeedRoomsForTest(map[int]*rooms.Room{mutTestRoomId: room}, nil))
}

func TestTryMutationActiveAtTarget_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["try_mutation_active_at_target"]; !ok {
		t.Fatal("try_mutation_active_at_target not registered in actionRegistry")
	}
}

func TestTryMutationActiveAtTarget_NilMob(t *testing.T) {
	ctx := &EvalContext{
		InstanceId: 9299,
		RoomId:     mutTestRoomId,
		MobState:   NewBehaviorState(),
	}
	res := actTryMutationActiveAtTarget(map[string]any{"key": "blinding-spit"}, ctx)
	if res != Failure {
		t.Errorf("nil mob: expected Failure, got %v", res)
	}
}

// TestTryMutationActiveAtTarget_NoTargetNoStaminaWaste is the core contract:
// with no resolvable engaged target the action fails BEFORE dispatch, so the
// trigger preamble never deducts stamina or arms the special-move cooldown.
// (Dispatching without a target would burn both — the reason single-target
// mutations were excluded from try_mutation_active in the first place.)
func TestTryMutationActiveAtTarget_NoTargetNoStaminaWaste(t *testing.T) {
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9200, 99999, mutTestRoomId)
	mob.Character.Mutations = map[string]int{"blinding-spit": 1}

	ctx := &EvalContext{
		InstanceId: 9200,
		RoomId:     mutTestRoomId,
		MobState:   NewBehaviorState(),
	}
	res := actTryMutationActiveAtTarget(map[string]any{"key": "blinding-spit"}, ctx)
	if res != Failure {
		t.Errorf("no engaged target: expected Failure, got %v", res)
	}
	if mob.Character.Stamina != 100 {
		t.Errorf("no-target early exit must not consume stamina; got %d, want 100",
			mob.Character.Stamina)
	}
}

func TestTryMutationActiveAtTarget_UnknownKeySkipped(t *testing.T) {
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9201, 99999, mutTestRoomId)
	victim := buildMutationMob(t, 9202, 99998, mutTestRoomId)
	_ = victim
	mob.Character.SetAggro(0, 9202, characters.DefaultAttack)

	ctx := &EvalContext{
		InstanceId: 9201,
		RoomId:     mutTestRoomId,
		MobState:   NewBehaviorState(),
	}
	res := actTryMutationActiveAtTarget(map[string]any{"key": "not-a-real-mutation"}, ctx)
	if res != Failure {
		t.Errorf("unknown key: expected Failure, got %v", res)
	}
}

// TestTryMutationActiveAtTarget_NoKeysIgnoresSelfAoEMutations: default
// enumeration only considers single-target mutations; owning only self/AoE
// ones yields Failure even with a valid target.
func TestTryMutationActiveAtTarget_NoKeysIgnoresSelfAoEMutations(t *testing.T) {
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9207, 99999, mutTestRoomId)
	mob.Character.Mutations = map[string]int{"healing-gel": 1}
	victim := buildMutationMob(t, 9208, 99998, mutTestRoomId)
	_ = victim
	mob.Character.SetAggro(0, 9208, characters.DefaultAttack)

	ctx := &EvalContext{
		InstanceId: 9207,
		RoomId:     mutTestRoomId,
		MobState:   NewBehaviorState(),
	}
	res := actTryMutationActiveAtTarget(map[string]any{}, ctx)
	if res != Failure {
		t.Errorf("only self/AoE mutations owned: expected Failure, got %v", res)
	}
	if mob.Character.Stamina != 100 {
		t.Errorf("nothing should fire; stamina must stay 100, got %d", mob.Character.Stamina)
	}
}
