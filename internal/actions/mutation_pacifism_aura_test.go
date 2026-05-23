package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// ---------------------------------------------------------------------------
// Test stub — minimal player Actor for victimEngagedWithActor tests
// ---------------------------------------------------------------------------

// testPlayerActor satisfies the Actor interface with a fixed user ID. Only
// IsPlayer() and GetUserId() are meaningful; all other methods are no-ops or
// return zero values.
type testPlayerActor struct {
	userId int
	room   *rooms.Room
}

func (a *testPlayerActor) GetCharacter() *characters.Character { return nil }
func (a *testPlayerActor) GetRoom() *rooms.Room                { return a.room }
func (a *testPlayerActor) SendText(_ messaging.Category, _ string) {}
func (a *testPlayerActor) SendRoomCommunication(_ string, _ bool)  {}
func (a *testPlayerActor) GetName() string                         { return "TestPlayer" }
func (a *testPlayerActor) IsPlayer() bool                          { return true }
func (a *testPlayerActor) GetUserId() int                          { return a.userId }
func (a *testPlayerActor) GetMobInstanceId() int                   { return 0 }
func (a *testPlayerActor) AddBuff(_ int, _ string)                 {}
func (a *testPlayerActor) OnSkillUse(_ string) bool                { return false }
func (a *testPlayerActor) OnStatUse(_ string) bool                 { return false }
func (a *testPlayerActor) OnCriticalSuccess(_ string)              {}
func (a *testPlayerActor) OnCriticalFailure(_ string)              {}

// ---------------------------------------------------------------------------
// TriggerPacifismAura tests
// ---------------------------------------------------------------------------

// TestTriggerPacifismAura_NoMutation verifies that an actor without the
// pacifism-aura mutation is blocked with BlockReason="no-mutation".
func TestTriggerPacifismAura_NoMutation(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerPacifismAura(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false without mutation")
	}
	if res.BlockReason != "no-mutation" {
		t.Errorf("expected BlockReason=no-mutation, got %s", res.BlockReason)
	}
}

// TestTriggerPacifismAura_LowStamina verifies the stamina gate blocks the
// mutation when stamina is insufficient.
func TestTriggerPacifismAura_LowStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"pacifism-aura": 1}
	mob.Character.Stamina = 5 // Below cost of 12
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerPacifismAura(actor, MutationOpts{})
	if res.Triggered {
		t.Fatal("expected Triggered=false with low stamina")
	}
	if res.BlockReason != "low-stamina" {
		t.Errorf("expected BlockReason=low-stamina, got %s", res.BlockReason)
	}
}

// TestTriggerPacifismAura_DeductsStamina verifies that stamina is deducted on
// success and the actor receives ConditionRecoveryPenalty (the self pacifism
// penalty). No combat requirement — pacifism-aura can fire out of combat.
func TestTriggerPacifismAura_DeductsStamina(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"pacifism-aura": 1}
	mob.Character.Stamina = 100
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerPacifismAura(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	if mob.Character.Stamina != 88 {
		t.Errorf("expected stamina deducted to 88 (100-12), got %d", mob.Character.Stamina)
	}
}

// TestTriggerPacifismAura_SelfPenalty verifies that on success the actor
// receives ConditionRecoveryPenalty (the pacifism self-penalty that prevents
// attacking for 3 rounds) and Aggro is cleared.
func TestTriggerPacifismAura_SelfPenalty(t *testing.T) {
	mob := newTestMobBare(t)
	mob.Character.Mutations = map[string]int{"pacifism-aura": 1}
	mob.Character.Stamina = 100
	mob.Character.SetAggro(0, 9901, characters.DefaultAttack)
	room := newTestRoomBare(t)
	actor := NewMobActorInRoom(mob, room)

	res := TriggerPacifismAura(actor, MutationOpts{})
	if !res.Triggered {
		t.Fatalf("expected Triggered=true, got BlockReason=%s", res.BlockReason)
	}
	if !mob.Character.HasCondition(characters.ConditionRecoveryPenalty) {
		t.Error("expected actor to have ConditionRecoveryPenalty after firing pacifism-aura")
	}
	if mob.Character.IsInCombat() {
		t.Error("expected actor Aggro to be cleared (EndAggro) after firing pacifism-aura")
	}
}

// TestVictimEngagedWithActor_PlayerActor verifies that a player actor matches
// victims engaged by UserId, and does not match unrelated victims (UserId 0).
func TestVictimEngagedWithActor_PlayerActor(t *testing.T) {
	// testPlayerActor is the local stub defined at the top of this file.
	actor := &testPlayerActor{userId: 42}

	engaged := newTestMobBare(t)
	engaged.Character.SetAggro(42, 0, characters.DefaultAttack)

	unrelated := newTestMobBare(t)
	unrelated.Character.SetAggro(0, 99, characters.DefaultAttack)

	if !victimEngagedWithActor(&engaged.Character, actor) {
		t.Error("expected victim engaged with player actor UserId=42 to match")
	}
	if victimEngagedWithActor(&unrelated.Character, actor) {
		t.Error("expected victim engaged with mob (UserId=0) NOT to match player actor")
	}
}

// TestVictimEngagedWithActor_MobActor verifies that a mob actor matches only
// victims engaged by matching MobInstanceId, and does not accidentally match
// victims whose EngagedTarget has UserId == 0 (mob-vs-mob fights with a
// different target or no target at all).
func TestVictimEngagedWithActor_MobActor(t *testing.T) {
	caster := newTestMobBare(t)
	caster.InstanceId = 55
	actor := NewMobActorInRoom(caster, newTestRoomBare(t))

	// Victim engaged with caster (MobInstanceId=55).
	engaged := newTestMobBare(t)
	engaged.Character.SetAggro(0, 55, characters.DefaultAttack)

	// Victim engaged with a different mob (MobInstanceId=99 != 55).
	otherMob := newTestMobBare(t)
	otherMob.Character.SetAggro(0, 99, characters.DefaultAttack)

	// Victim not engaged with anyone.
	idle := newTestMobBare(t)

	if !victimEngagedWithActor(&engaged.Character, actor) {
		t.Error("expected victim engaged with caster mob (MobInstanceId=55) to match")
	}
	if victimEngagedWithActor(&otherMob.Character, actor) {
		t.Error("expected victim engaged with different mob (MobInstanceId=99) NOT to match")
	}
	if victimEngagedWithActor(&idle.Character, actor) {
		t.Error("expected idle victim (no engaged target) NOT to match")
	}
}
