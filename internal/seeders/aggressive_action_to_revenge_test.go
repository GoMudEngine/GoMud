package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestAggressiveActionToRevenge_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["PlayerAttackedMob"] {
		if reg.name == ruleNameAggressiveActionToRevenge {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("aggressive_action_to_revenge not registered for PlayerAttackedMob")
	}
}

func TestAggressiveActionToRevenge_ZeroFields_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on zero-field event: %v", r)
		}
	}()
	aggressiveActionToRevenge(events.PlayerAttackedMob{})
}

func TestAggressiveActionToRevenge_ZeroUserId_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on zero UserId: %v", r)
		}
	}()
	// MobInstanceId set but UserId zero — should return early cleanly.
	aggressiveActionToRevenge(events.PlayerAttackedMob{UserId: 0, MobInstanceId: 999})
}

func TestAggressiveActionToRevenge_ZeroMobInstanceId_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on zero MobInstanceId: %v", r)
		}
	}()
	// UserId set but MobInstanceId zero — should return early cleanly.
	aggressiveActionToRevenge(events.PlayerAttackedMob{UserId: 5, MobInstanceId: 0})
}

func TestAggressiveActionToRevenge_WrongEventType_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on wrong event type: %v", r)
		}
	}()
	// Dispatch a completely different event type — type assertion must
	// return early without panicking.
	aggressiveActionToRevenge(fakeEvent{typeName: "NotPlayerAttackedMob"})
}

// Full seed test (attacked mob gets revenge goal, witnesses get lower-
// priority revenge, auto-aggro mobs are skipped) requires live mob +
// room fixtures with working GetInstance and LoadRoom; deferred to
// Task 15 smoke per spec §6.3.
