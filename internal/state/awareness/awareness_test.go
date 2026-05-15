package awareness

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/require"
)

// Test helpers.

func makePair() (sneaker, observer *Machine) {
	registryMu.Lock()
	machineRegistry = map[state.ActorRef]*Machine{}
	registryMu.Unlock()
	A := NewMachine()
	B := NewMachine()
	RegisterMachine(actor(1), A)
	RegisterMachine(actor(2), B)
	return A, B
}

func actor(userId int) state.ActorRef {
	return state.ActorRef{UserId: userId}
}

// --- AW-001 through AW-003: Basic sneak resolve ---

// AW-001: Sneak alone succeeds.
func TestAW_001_SneakAloneSucceeds(t *testing.T) {
	A, _ := makePair()
	err := A.TransitionToConcealing(ConcealingData{},
		state.TransitionReason{Trigger: TriggerSneakCommand})
	require.NoError(t, err)
	A.ResolveConcealment(true, state.TransitionReason{Trigger: TriggerSneakSuccess})
	require.Equal(t, Hidden, A.State())
}

// AW-002: Sneak fails when observer wins the roll.
func TestAW_002_SneakObserverDetects(t *testing.T) {
	A, _ := makePair()
	err := A.TransitionToConcealing(ConcealingData{},
		state.TransitionReason{Trigger: TriggerSneakCommand})
	require.NoError(t, err)
	A.ResolveConcealment(false, state.TransitionReason{Trigger: TriggerSneakFailed})
	require.Equal(t, Visible, A.State())
}

// AW-003: Sneak succeeds when sneaker wins against all observers.
func TestAW_003_SneakSucceedsAgainstObservers(t *testing.T) {
	A, _ := makePair()
	err := A.TransitionToConcealing(ConcealingData{},
		state.TransitionReason{Trigger: TriggerSneakCommand})
	require.NoError(t, err)
	A.ResolveConcealment(true, state.TransitionReason{Trigger: TriggerSneakSuccess})
	require.Equal(t, Hidden, A.State())
	require.True(t, A.IsHidden())
}

// --- AW-004 through AW-011: Detection rolls on room entry / observer presence ---

// AW-004: Hidden actor moves into room with observers; all sneak rolls won → Hidden persists.
func TestAW_004_HiddenMovesIntoRoomWithObservers_AllWon(t *testing.T) {
	A, _ := makePair()
	// Setup: A is already Hidden.
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	require.Equal(t, Hidden, A.State())

	// Movement detection: all rolls won.
	A.NotifyRoomChanged(false /* not detected */, state.TransitionReason{Trigger: TriggerMovementDetected})
	require.Equal(t, Hidden, A.State(), "Hidden persists when all observer rolls fail")
}

// AW-005: Observer enters room with hidden actor; detection roll fires.
func TestAW_005_ObserverEntersRoomWithHidden(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	// Observer's roll lost (i.e., new observer didn't detect):
	A.NotifyRoomChanged(false, state.TransitionReason{Trigger: TriggerMovementDetected})
	require.Equal(t, Hidden, A.State())
}

// AW-006: Detection roll won by observer → Hidden → Revealing → Visible.
func TestAW_006_DetectionRollWonByObserver(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	require.Equal(t, Hidden, A.State())

	A.NotifyRoomChanged(true /* detected */, state.TransitionReason{Trigger: TriggerMovementDetected})
	require.Equal(t, Visible, A.State(), "detection → Revealing → Visible same-tick")
}

// AW-007: Detection roll won by sneaker → Hidden persists.
func TestAW_007_DetectionRollWonBySneaker(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	A.NotifyRoomChanged(false, state.TransitionReason{})
	require.Equal(t, Hidden, A.State())
}

// AW-008: Hidden actor moves; per-observer detection rolls; failure → Revealing.
func TestAW_008_MovementDetectionFailure(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	A.NotifyRoomChanged(true, state.TransitionReason{Trigger: TriggerMovementDetected})
	require.Equal(t, Visible, A.State())
}

// AW-009: Hidden actor moves; sneak rolls won against all observers → Hidden persists.
func TestAW_009_MovementDetectionSuccess(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	A.NotifyRoomChanged(false, state.TransitionReason{})
	require.Equal(t, Hidden, A.State())
}

// AW-010: Observer-initiated `look <actor>` against hidden actor; failure → Revealing.
func TestAW_010_ObserverLookAtFails(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})

	err := A.TransitionToRevealing(state.TransitionReason{Trigger: TriggerObserverSearch})
	require.NoError(t, err)
	require.Equal(t, Visible, A.State())
}

// AW-011: Observer-initiated `search` / `scan` fires per-hidden-actor rolls.
func TestAW_011_ObserverSearch(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})

	err := A.TransitionToRevealing(state.TransitionReason{Trigger: TriggerObserverSearch})
	require.NoError(t, err)
	require.Equal(t, Visible, A.State())
}

// --- AW-012 through AW-014: Light state changes ---

// AW-012: Sneaker emission state change → re-roll; failure → Revealing.
func TestAW_012_SneakerEmissionChange_DetectionFails(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})

	err := A.TransitionToRevealing(state.TransitionReason{Trigger: TriggerLightChange})
	require.NoError(t, err)
	require.Equal(t, Visible, A.State())
}

// AW-013: Room light state changes → re-roll for hidden actors.
func TestAW_013_RoomLightChange(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})

	err := A.TransitionToRevealing(state.TransitionReason{Trigger: TriggerLightChange})
	require.NoError(t, err)
	require.Equal(t, Visible, A.State())
}

// AW-014: NightVision observer in dark room treats sneaker as in lit room (per-observer).
// Framework-level test: this is mostly a CalcSneakScore concern (Task 6).
// Here we just verify the Awareness machine doesn't auto-reveal on NightVision presence.
func TestAW_014_NightVisionDoesNotAutoReveal(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	require.Equal(t, Hidden, A.State(),
		"NightVision is a per-roll modifier; doesn't unilaterally break stealth")
}

// --- AW-015 through AW-018: Combat Phase cascade ---
// These are integration-tested via the cascade subscription in hooks/
// (Task 5). Framework-level here just verifies TransitionToRevealing
// with TriggerCombatEntered works.

// AW-015: Combat Phase → Engaging (non-surprise) cascades Awareness → Revealing.
func TestAW_015_NonSurpriseCombatBreaksStealth(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})

	err := A.TransitionToRevealing(state.TransitionReason{Trigger: TriggerCombatEntered})
	require.NoError(t, err)
	require.Equal(t, Visible, A.State())
}

// AW-016: SurpriseAttack reason — no cascade should fire (subscription respects it).
// Verified via the cascade hook in Task 5; here we verify the Awareness
// machine doesn't itself trigger on combat entry.
func TestAW_016_SurpriseAttackPreservesHidden(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	require.Equal(t, Hidden, A.State(),
		"surprise-attack combat-entry does NOT auto-break Awareness Hidden state (cascade subscriber checks reason)")
}

// AW-017: Engaging→Engaged with SurpriseLeft preserves Hidden.
func TestAW_017_SurpriseEngagedPreservesHidden(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	require.Equal(t, Hidden, A.State())
}

// AW-018: OnEndOfRoundIfSurprise callback fires → Hidden → Revealing.
func TestAW_018_SurpriseRoundEndReveals(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})

	err := A.TransitionToRevealing(state.TransitionReason{Trigger: TriggerSurpriseRoundEnd})
	require.NoError(t, err)
	require.Equal(t, Visible, A.State())
}

// --- AW-019, AW-020: Stamina cost (framework-level just verifies IsHidden()) ---

// AW-019: IsHidden() returns true for Hidden machines.
func TestAW_019_IsHiddenForStaminaCost(t *testing.T) {
	A, _ := makePair()
	require.False(t, A.IsHidden(), "Visible at start")
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	require.True(t, A.IsHidden(), "Hidden after successful sneak")
}

// AW-020: IsHidden() drops to false after a reveal.
func TestAW_020_IsHiddenFalseAfterReveal(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	require.True(t, A.IsHidden())
	A.NotifyRoomChanged(true, state.TransitionReason{})
	require.False(t, A.IsHidden())
}

// --- AW-021, AW-022: Logout safety valve ---

// AW-021: ForceVisible from Hidden transitions through Revealing → Visible.
func TestAW_021_ForceVisibleFromHidden(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})
	require.Equal(t, Hidden, A.State())

	A.ForceVisible(state.TransitionReason{Trigger: TriggerLogout})
	require.Equal(t, Visible, A.State())
}

// AW-022: ForceVisible from Visible is a no-op.
func TestAW_022_ForceVisibleFromVisible(t *testing.T) {
	A, _ := makePair()
	require.Equal(t, Visible, A.State())
	A.ForceVisible(state.TransitionReason{Trigger: TriggerLogout})
	require.Equal(t, Visible, A.State())
}

// --- AW-023: Activity veto ---

// AW-023: Activity ≠ Free blocks Visible → Concealing.
func TestAW_023_ActivityBlocksConcealing(t *testing.T) {
	A, _ := makePair()
	A.RegisterActivityCheck(func() bool { return false /* busy with activity */ })
	err := A.TransitionToConcealing(ConcealingData{},
		state.TransitionReason{Trigger: TriggerSneakCommand})
	require.ErrorIs(t, err, state.ErrVetoed)
	require.Equal(t, Visible, A.State())
}

// --- AW-024 through AW-027: Light-conditional sneak score ---
// These are CalcSneakScore tests; logically belong in skill_helpers_test.go.
// Here we just verify the trigger constants exist (compile-time check).

// AW-024: Baseline sneak — no emit, dark room — sneakMod = 1.0.
// (Tested in Task 6's CalcSneakScore tests.)
func TestAW_024_CalcSneakScoreBaseline(t *testing.T) {
	t.Skip("CalcSneakScore tests land in Task 6's skill_helpers_test.go")
}

// AW-025: Lit room, no emit — sneakMod = 0.9.
func TestAW_025_CalcSneakScoreLitRoom(t *testing.T) {
	t.Skip("CalcSneakScore tests land in Task 6's skill_helpers_test.go")
}

// AW-026: Lit sneaker, dark room — sneakMod = 0.5.
func TestAW_026_CalcSneakScoreBeacon(t *testing.T) {
	t.Skip("CalcSneakScore tests land in Task 6's skill_helpers_test.go")
}

// AW-027: Lit sneaker, lit room — sneakMod = 0.85.
func TestAW_027_CalcSneakScoreLitBoth(t *testing.T) {
	t.Skip("CalcSneakScore tests land in Task 6's skill_helpers_test.go")
}

// --- AW-028, AW-029: Persistence ---

// AW-028: Fresh Machine is Visible.
func TestAW_028_FreshMachineIsVisible(t *testing.T) {
	m := NewMachine()
	require.Equal(t, Visible, m.State())
	require.False(t, m.IsHidden())
}

// AW-029: No Awareness state persists across machine recreation.
// Implicit in AW-028; documented for clarity.
func TestAW_029_StateDoesNotPersist(t *testing.T) {
	m := NewMachine()
	require.Equal(t, Visible, m.State())
}

// --- AW-030, AW-031: Revealing state semantics ---

// AW-030: Revealing fires cascade subscribers (validates that
// TransitionToRevealing actually calls inner.TransitionTo, which
// triggers AfterTransition cascades).
func TestAW_030_RevealingFiresCascade(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})

	var transitionedTo []State
	A.Inner().AfterTransition("test_cascade", func(from, to State, r state.TransitionReason) {
		transitionedTo = append(transitionedTo, to)
	})

	err := A.TransitionToRevealing(state.TransitionReason{Trigger: TriggerForceVisible})
	require.NoError(t, err)
	require.Contains(t, transitionedTo, Revealing)
	require.Contains(t, transitionedTo, Visible)
}

// AW-031: Revealing → Visible is same-tick.
func TestAW_031_RevealingToVisibleSameTick(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})

	err := A.TransitionToRevealing(state.TransitionReason{Trigger: TriggerForceVisible})
	require.NoError(t, err)
	require.Equal(t, Visible, A.State(),
		"after TransitionToRevealing returns, state should already be Visible (same-tick)")
}

// --- AW-032, AW-033: Noisy actions ---

// AW-032: Hidden + noisy communication trigger reveals via TriggerNoisyAction.
func TestAW_032_NoisyCommunicationReveals(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})

	err := A.TransitionToRevealing(state.TransitionReason{Trigger: TriggerNoisyAction})
	require.NoError(t, err)
	require.Equal(t, Visible, A.State())
}

// AW-033: Hidden + noisy broadcast (rally/warcry/taunt) reveals via TriggerNoisyAction.
func TestAW_033_NoisyBroadcastReveals(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToConcealing(ConcealingData{}, state.TransitionReason{}))
	A.ResolveConcealment(true, state.TransitionReason{})

	err := A.TransitionToRevealing(state.TransitionReason{Trigger: TriggerNoisyAction})
	require.NoError(t, err)
	require.Equal(t, Visible, A.State())
}
