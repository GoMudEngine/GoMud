package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/relationships"
)

func TestFriendKilledToRevenge_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["MobDeath"] {
		if reg.name == ruleNameFriendKilledToRevenge {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("friend_killed_to_revenge not registered for MobDeath")
	}
}

func TestFriendKilledToRevenge_NoKiller_NoOp(t *testing.T) {
	// KillerMobInstanceId == 0 means player killed it — rule must
	// return cleanly without panicking.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on player-kill event: %v", r)
		}
	}()
	evt := events.MobDeath{
		MobId:               999,
		InstanceId:          999,
		KillerMobInstanceId: 0,
	}
	friendKilledToRevenge(evt)
}

func TestFriendKilledToRevenge_WrongEventType_NoOp(t *testing.T) {
	// A non-MobDeath event must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on wrong event type: %v", r)
		}
	}()
	friendKilledToRevenge(fakeEvent{typeName: "SomethingElse"})
}

func TestFriendKilledToRevenge_NoRelations_NoOp(t *testing.T) {
	// Mob template 99998 has no relationships — rule must return
	// cleanly with a mob-killer event.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on mob with no relations: %v", r)
		}
	}()
	evt := events.MobDeath{
		MobId:               99998,
		InstanceId:          99998,
		KillerMobInstanceId: 99999,
	}
	friendKilledToRevenge(evt)
}

func TestIsFriendlyRelationType(t *testing.T) {
	// Friendly types: the rule should return true.
	for _, friendly := range []relationships.Type{
		relationships.TypeFriend,
		relationships.TypeFamily,
		relationships.TypeLover,
	} {
		if !isFriendlyRelationType(friendly) {
			t.Errorf("type %q should be friendly", friendly)
		}
	}
	// Non-friendly types: the rule should return false.
	for _, unfriendly := range []relationships.Type{
		relationships.TypeRival,
		relationships.TypeEmployer,
		relationships.TypeEmployee,
		relationships.Type(""),
		relationships.Type("stranger"),
	} {
		if isFriendlyRelationType(unfriendly) {
			t.Errorf("type %q should NOT be friendly", unfriendly)
		}
	}
}

// End-to-end integration (relationship edge → MobDeath → revenge goal
// on friend) deferred to Task 15 smoke: requires live instance map +
// relationship graph populated.
