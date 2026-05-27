package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestFactionKillCounter_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["MobDeath"] {
		if reg.name == ruleNameFactionKillCounter {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("faction_kill_counter not registered for MobDeath")
	}
}

func TestFactionKillCounter_PlayerKiller_NoOp(t *testing.T) {
	// KillerMobInstanceId == 0 means player killed it (or unknown).
	// Rule must not panic and must silently return without writing counters.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()
	evt := events.MobDeath{
		InstanceId:          99,
		KillerMobInstanceId: 0, // player kill — no mob-killer attribution
	}
	factionKillCounter(evt)
}

func TestFactionKillCounter_WrongEventType_NoOp(t *testing.T) {
	// Non-MobDeath event must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic on wrong event type: %v", r)
		}
	}()
	factionKillCounter(fakeEvent{typeName: "SomethingElse"})
}

func TestFactionKillCounter_MobKillerButNilInstances_NoOp(t *testing.T) {
	// Killer is a mob (KillerMobInstanceId != 0) but neither victim nor
	// killer are loaded in the instance map. Rule must return cleanly
	// without panicking.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic on missing instances: %v", r)
		}
	}()
	evt := events.MobDeath{
		InstanceId:          99998,
		KillerMobInstanceId: 99999,
	}
	factionKillCounter(evt)
}

// Full counter-increment integration test (loaded mob + faction data →
// faction_kills_inflicted:<fid> counter increments on killer) requires
// the live instance map and faction registry to be populated. Deferred
// to Task 15 smoke per spec §6.3.
