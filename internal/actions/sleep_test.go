package actions

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// ---------------------------------------------------------------------------
// Sleep test helpers
// ---------------------------------------------------------------------------

// sleepFakeActor is a minimal Actor for sleep tests. It exposes the full
// Actor interface with a real *characters.Character so that
// Character.AddBuff / Character.HasBuffFlag work correctly without a real
// server event queue. SendText messages are captured for assertion.
type sleepFakeActor struct {
	char     *characters.Character
	room     *rooms.Room
	isPlayer bool
	userId   int
	sent     []string
}

func (a *sleepFakeActor) GetCharacter() *characters.Character    { return a.char }
func (a *sleepFakeActor) GetRoom() *rooms.Room                   { return a.room }
func (a *sleepFakeActor) GetName() string                        { return a.char.Name }
func (a *sleepFakeActor) IsPlayer() bool                         { return a.isPlayer }
func (a *sleepFakeActor) GetUserId() int                         { return a.userId }
func (a *sleepFakeActor) GetMobInstanceId() int                  { return 0 }
func (a *sleepFakeActor) AddBuff(_ int, _ string)                {} // no-op: Sleep calls c.AddBuff directly
func (a *sleepFakeActor) OnSkillUse(_ string) bool               { return false }
func (a *sleepFakeActor) OnStatUse(_ string) bool                { return false }
func (a *sleepFakeActor) OnCriticalSuccess(_ string)             {}
func (a *sleepFakeActor) OnCriticalFailure(_ string)             {}
func (a *sleepFakeActor) SendRoomCommunication(_ string, _ bool) {}
func (a *sleepFakeActor) SendText(_ messaging.Category, msg string) {
	a.sent = append(a.sent, msg)
}

// newSleepActor builds a minimal actor for sleep tests.
// inCombat=true sets Aggro on the character to simulate combat.
func newSleepActor(t *testing.T, inCombat bool, isPlayer bool) *sleepFakeActor {
	t.Helper()
	c := characters.New()
	c.Name = "Sleeper"
	c.Buffs = buffs.New()
	if inCombat {
		// SetAggro requires a valid target; use the compat helper.
		c.SetAggro(0, 9001, characters.DefaultAttack)
	}
	room := &rooms.Room{RoomId: 9900}
	return &sleepFakeActor{
		char:     c,
		room:     room,
		isPlayer: isPlayer,
		userId:   1,
	}
}

// seedSleepBuff seeds buff spec 15 (Sleeping) so that Character.AddBuff(15,
// false) succeeds. Also re-seeds buff 9 (Hidden) to avoid breaking shadow
// tests that share the same test binary. Returns a cleanup func.
func seedSleepBuff(t *testing.T) func() {
	t.Helper()
	return buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		9: {
			BuffId:       9,
			Name:         "Hidden",
			Flags:        []buffs.Flag{buffs.Hidden},
			TriggerCount: 1000000,
		},
		15: {
			BuffId:       15,
			Name:         "Sleeping",
			Flags:        []buffs.Flag{buffs.Sleeping, buffs.CancelOnDamage},
			TriggerCount: 1000000, // effectively infinite — governed by cancel flags
		},
	})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestSleep_AppliesBuffOnSuccess verifies that Sleep returns Success and
// sets the Sleeping buff flag on the character.
func TestSleep_AppliesBuffOnSuccess(t *testing.T) {
	cleanup := seedSleepBuff(t)
	defer cleanup()

	actor := newSleepActor(t, false /* notInCombat */, true)

	res := Sleep(actor, SleepOptions{})

	if !res.Success {
		t.Fatalf("expected Success, got %+v", res)
	}
	if !actor.char.HasBuffFlag(buffs.Sleeping) {
		t.Errorf("expected Sleeping flag applied after Sleep()")
	}
}

// TestSleep_BuffUnavailable_NoRawErrorLeak verifies that when the Sleeping
// buff spec can't be applied (e.g. the buff is missing from the world data),
// the player gets a clean message rather than the raw internal AddBuff error
// that leaks the buff id ("...buffId: 15"). Regression guard for the dogmud
// world shipping without buff 15.
func TestSleep_BuffUnavailable_NoRawErrorLeak(t *testing.T) {
	// Deliberately seed an EMPTY buff registry so AddBuff(15) fails.
	cleanup := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{})
	defer cleanup()

	actor := newSleepActor(t, false /* notInCombat */, true)

	res := Sleep(actor, SleepOptions{})

	if res.Success {
		t.Fatalf("expected failure when the Sleeping buff is unavailable, got %+v", res)
	}
	if len(actor.sent) == 0 {
		t.Fatal("expected a player-facing message on sleep failure")
	}
	joined := strings.Join(actor.sent, " ")
	if strings.Contains(joined, "buffId") || strings.Contains(joined, "failed to add") {
		t.Errorf("player message leaked the raw internal error: %q", joined)
	}
}

// TestSleep_BlocksInCombat verifies that Sleep returns Failure and sends
// a user-facing message when the actor is in combat.
func TestSleep_BlocksInCombat(t *testing.T) {
	cleanup := seedSleepBuff(t)
	defer cleanup()

	actor := newSleepActor(t, true /* inCombat */, true)

	res := Sleep(actor, SleepOptions{})

	if res.Success {
		t.Errorf("expected failure when in combat, got %+v", res)
	}
	if actor.char.HasBuffFlag(buffs.Sleeping) {
		t.Errorf("expected Sleeping NOT applied during combat")
	}
	if len(actor.sent) == 0 {
		t.Errorf("expected player-facing message on combat block, got none")
	}
}

// TestSleep_MobActorSilentInCombat verifies that a mob actor in combat
// receives no message (mob.SendText is a no-op, actor.sent stays empty).
func TestSleep_MobActorSilentInCombat(t *testing.T) {
	cleanup := seedSleepBuff(t)
	defer cleanup()

	actor := newSleepActor(t, true /* inCombat */, false /* mob, not player */)

	res := Sleep(actor, SleepOptions{})

	if res.Success {
		t.Errorf("expected failure for mob in combat, got %+v", res)
	}
	if len(actor.sent) != 0 {
		t.Errorf("mob actor should be silent; got messages: %v", actor.sent)
	}
}

// TestSleep_IdempotentWhenAlreadySleeping verifies that calling Sleep on a
// character that is already sleeping returns Success without error.
func TestSleep_IdempotentWhenAlreadySleeping(t *testing.T) {
	cleanup := seedSleepBuff(t)
	defer cleanup()

	actor := newSleepActor(t, false, true)

	// First call applies the buff.
	first := Sleep(actor, SleepOptions{})
	if !first.Success {
		t.Fatalf("first Sleep() expected Success, got %+v", first)
	}

	// Second call should be a no-op success.
	second := Sleep(actor, SleepOptions{})
	if !second.Success {
		t.Errorf("idempotent re-sleep expected Success, got %+v", second)
	}
}
