package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestCombatAssistToOpinion_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["PlayerAttackedMob"] {
		if reg.name == ruleNameCombatAssistToOpinion {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("combat_assist_to_opinion_boost not registered for PlayerAttackedMob")
	}
}

func TestCombatAssistToOpinion_ZeroFields_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on zero-field event: %v", r)
		}
	}()
	combatAssistToOpinion(events.PlayerAttackedMob{})
}

func TestCombatAssistToOpinion_ZeroUserId_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on zero UserId: %v", r)
		}
	}()
	combatAssistToOpinion(events.PlayerAttackedMob{UserId: 0, MobInstanceId: 999})
}

func TestCombatAssistToOpinion_ZeroMobInstanceId_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on zero MobInstanceId: %v", r)
		}
	}()
	combatAssistToOpinion(events.PlayerAttackedMob{UserId: 5, MobInstanceId: 0})
}

func TestCombatAssistToOpinion_WrongEventType_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on wrong event type: %v", r)
		}
	}()
	// Dispatch a completely different event type — type assertion must
	// return early without panicking.
	combatAssistToOpinion(fakeEvent{typeName: "NotPlayerAttackedMob"})
}

func TestCombatAssistToOpinion_BothRulesRegisteredForSameEvent(t *testing.T) {
	// Sanity check: rule 6 (aggressive_action_to_revenge) and rule 9
	// (combat_assist_to_opinion_boost) both register for PlayerAttackedMob.
	// This verifies the multi-consumer pattern of Option B architecture —
	// both rules fire independently on the same event.
	registryMu.RLock()
	defer registryMu.RUnlock()
	count := len(registry["PlayerAttackedMob"])
	if count < 2 {
		t.Errorf("expected ≥2 rules on PlayerAttackedMob, got %d "+
			"(rule 6 aggressive_action_to_revenge + rule 9 combat_assist_to_opinion_boost "+
			"should both be registered)", count)
	}
}

func TestResolveAttackerMobTarget_NilAggro_ReturnsZero(t *testing.T) {
	if got := resolveAttackerMobTarget(nil); got != 0 {
		t.Errorf("nil Aggro: got %d, want 0", got)
	}
}

func TestResolveAttackerMobTarget_MobTarget_ReturnsInstanceId(t *testing.T) {
	a := &characters.Aggro{MobInstanceId: 42, UserId: 0}
	if got := resolveAttackerMobTarget(a); got != 42 {
		t.Errorf("mob target: got %d, want 42", got)
	}
}

func TestResolveAttackerMobTarget_PlayerTarget_ReturnsZero(t *testing.T) {
	// UserId != 0 means the target is a player — combat-assist should
	// not fire (player is helping nothing in a mob-vs-mob fight).
	a := &characters.Aggro{MobInstanceId: 0, UserId: 7}
	if got := resolveAttackerMobTarget(a); got != 0 {
		t.Errorf("player target: got %d, want 0", got)
	}
}

func TestResolveAttackerMobTarget_ZeroAggro_ReturnsZero(t *testing.T) {
	// Empty aggro (no current target) — should return 0.
	a := &characters.Aggro{}
	if got := resolveAttackerMobTarget(a); got != 0 {
		t.Errorf("zero Aggro: got %d, want 0", got)
	}
}

// Full assist-flow test (peaceful beneficiary + IsPeacefulToward gate +
// cooldown round-trip + opinions.Bump) requires live mob + faction
// fixtures; deferred to Task 15 smoke per spec §6.3.
