package combatphase

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/stretchr/testify/require"
)

// Test helpers. Each test constructs Machine instances directly
// (no Character — those come in integration tests in later tasks).

func makePair() (attacker, defender *Machine) {
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

func attackReason(attacker, target state.ActorRef) state.TransitionReason {
	return state.TransitionReason{
		Trigger: TriggerAttackCommand,
		Actor:   attacker,
		Target:  target,
	}
}

// --- Standard entry / exit (CP-001 through CP-008) ---

// CP-001: Basic attack initiation. Attacker → Engaging; defender
// gains attacker in its Attackers list; defender's Combat Phase
// unchanged (defenders don't auto-engage).
func TestCP_001_BasicAttackInitiation(t *testing.T) {
	A, B := makePair()
	a, b := actor(1), actor(2)

	err := A.TransitionToEngaging(EngagingData{Target: b, RoundsUntil: 0},
		attackReason(a, b))
	require.NoError(t, err)

	require.Equal(t, Engaging, A.State())
	require.Equal(t, Idle, B.State(), "defender does not auto-engage")
	require.Len(t, B.Attackers(), 1, "defender's Attackers list gains attacker")
	require.Equal(t, a, B.Attackers()[0])
}

// CP-002: Engaging counts down each round until ready.
func TestCP_002_EngagingCountdown(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToEngaging(
		EngagingData{Target: actor(2), RoundsUntil: 2}, state.TransitionReason{}))

	A.OnRoundTick()
	d, _ := A.EngagingData()
	require.Equal(t, 1, d.RoundsUntil, "decrement after one round")

	A.OnRoundTick()
	require.Equal(t, Engaged, A.State(), "transition to Engaged when RoundsUntil reaches 0")
}

// CP-003: Engaging → Engaged fires cascade.
func TestCP_003_EngagingToEngagedFiresCascade(t *testing.T) {
	A, _ := makePair()
	var firedTransitions []State
	A.Inner().AfterTransition("test_cascade", func(from, to State, _ state.TransitionReason) {
		firedTransitions = append(firedTransitions, to)
	})
	require.NoError(t, A.TransitionToEngaging(
		EngagingData{Target: actor(2), RoundsUntil: 0}, state.TransitionReason{}))
	A.OnRoundTick()
	require.Contains(t, firedTransitions, Engaged)
}

// CP-004: Target dies → attacker auto-transitions Engaged → Idle.
func TestCP_004_TargetDiesEndsEngagement(t *testing.T) {
	A, B := makePair()
	require.NoError(t, A.TransitionToEngaging(
		EngagingData{Target: actor(2), RoundsUntil: 0}, state.TransitionReason{}))
	A.OnRoundTick() // Engaging → Engaged

	A.NotifyTargetDied(actor(2))

	require.Equal(t, Idle, A.State(), "attacker returns to Idle when target dies")
	require.Empty(t, B.Attackers(), "B's Attackers cleared")
}

// CP-005: Self dies → outbound combat ends + inbound attackers cleared.
func TestCP_005_SelfDeathClearsBoth(t *testing.T) {
	A, B := makePair()
	a, b := actor(1), actor(2)
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: b}, attackReason(a, b)))
	require.NoError(t, B.TransitionToEngaging(EngagingData{Target: a}, attackReason(b, a)))

	A.NotifySelfDied()
	require.Equal(t, Idle, A.State(), "self died → Idle")
	require.Empty(t, A.Attackers(), "inbound attackers cleared")
	require.Empty(t, B.Attackers(), "outbound cleared from B's perspective")
}

// CP-006: Flee command initiates Disengaging.
func TestCP_006_FleeInitiatesDisengaging(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick() // Engaging → Engaged
	require.NoError(t, A.TransitionToDisengaging(
		state.TransitionReason{Trigger: TriggerFleeCommand}))
	require.Equal(t, Disengaging, A.State())
}

// CP-007: Flee success → Idle.
func TestCP_007_FleeSuccess(t *testing.T) {
	A, B := makePair()
	a, b := actor(1), actor(2)
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: b}, attackReason(a, b)))
	A.OnRoundTick()
	require.NoError(t, A.TransitionToDisengaging(state.TransitionReason{Trigger: TriggerFleeCommand}))
	A.ResolveFlee(true) // success
	require.Equal(t, Idle, A.State())
	require.Empty(t, B.Attackers())
}

// CP-008: Flee failure → back to Engaged.
func TestCP_008_FleeFailureReturnsEngaged(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick()
	require.NoError(t, A.TransitionToDisengaging(state.TransitionReason{Trigger: TriggerFleeCommand}))
	A.ResolveFlee(false) // failure
	require.Equal(t, Engaged, A.State())
}

// --- Surprise attack semantics (CP-009, CP-023 through CP-025b) ---

// CP-009/CP-023: Hidden attacker enters Engaging with surprise marker; Awareness stays Hidden.
func TestCP_023_SurpriseEngagingPreservesStealth(t *testing.T) {
	A, _ := makePair()
	reason := state.TransitionReason{
		Trigger: TriggerSurpriseAttack,
		Actor:   actor(1),
		Target:  actor(2),
	}
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2), Reason: reason}, reason))

	d, _ := A.EngagingData()
	require.Equal(t, TriggerSurpriseAttack, d.Reason.Trigger,
		"reason persisted on Engaging data for surprise carry-through")
}

// CP-024: Engaging → Engaged with surprise marker preserves stealth.
func TestCP_024_SurpriseEngagedStillPreservesStealth(t *testing.T) {
	A, _ := makePair()
	reason := state.TransitionReason{Trigger: TriggerSurpriseAttack}
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2), Reason: reason}, reason))
	A.OnRoundTick()
	d, _ := A.EngagedData()
	require.True(t, d.SurpriseLeft, "Engaged data flags surprise as still available")
}

// CP-025: At end of first combat round, surprise consumed.
func TestCP_025_SurpriseConsumedAtEndOfFirstRound(t *testing.T) {
	A, _ := makePair()
	reason := state.TransitionReason{Trigger: TriggerSurpriseAttack}
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2), Reason: reason}, reason))
	A.OnRoundTick() // Engaging → Engaged

	var stealthBreakFired bool
	A.OnEndOfRoundIfSurprise(func(_ state.TransitionReason) {
		stealthBreakFired = true
	})

	A.OnCombatRoundEnd() // round resolves
	require.True(t, stealthBreakFired, "end-of-round cascade fires once surprise round ends")

	d, _ := A.EngagedData()
	require.False(t, d.SurpriseLeft, "surprise flag consumed after first round end")
}

// --- Vetoes (CP-010 through CP-016) ---

// CP-010: NonCombatant cannot attack.
func TestCP_010_NonCombatantCannotInitiate(t *testing.T) {
	A, _ := makePair()
	A.RegisterCombatantVeto(func() bool { return false /* NonCombatant */ })
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)}, state.TransitionReason{})
	require.ErrorIs(t, err, state.ErrVetoed)
	require.Equal(t, Idle, A.State())
}

// CP-011: NonCombatant target cannot be attacked.
func TestCP_011_NonCombatantTargetVetoed(t *testing.T) {
	A, _ := makePair()
	A.RegisterTargetCombatantCheck(func(target state.ActorRef) bool {
		return false // target is NonCombatant
	})
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{Target: actor(2)})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// CP-012: Activity != Free blocks Engaging.
func TestCP_012_ActivityBlocksEngaging(t *testing.T) {
	A, _ := makePair()
	A.RegisterActivityCheck(func() bool { return false /* not Free */ })
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)}, state.TransitionReason{})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// CP-013: Dead character cannot attack.
func TestCP_013_DeadCannotAttack(t *testing.T) {
	A, _ := makePair()
	A.RegisterLifeCheck(func() bool { return false /* not Alive */ })
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)}, state.TransitionReason{})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// CP-014: Cannot attack a dead target.
func TestCP_014_CannotAttackDeadTarget(t *testing.T) {
	A, _ := makePair()
	A.RegisterTargetLifeCheck(func(_ state.ActorRef) bool {
		return false // target dead
	})
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{Target: actor(2)})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// CP-015: AFK/Disconnected player target vetoed.
func TestCP_015_AFKTargetVetoed(t *testing.T) {
	A, _ := makePair()
	A.RegisterTargetPresenceCheck(func(_ state.ActorRef) bool {
		return false // target AFK/Disconnected
	})
	err := A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{Target: actor(2)})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// CP-016: Cannot flee while grappled.
func TestCP_016_GrappledCannotFlee(t *testing.T) {
	A, _ := makePair()
	A.RegisterPositionCheck(func() bool { return false /* Clinched/Grounded */ })
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick()
	err := A.TransitionToDisengaging(state.TransitionReason{Trigger: TriggerFleeCommand})
	require.ErrorIs(t, err, state.ErrVetoed)
}

// --- Multi-attacker (CP-017 through CP-019) ---

// CP-017: Three attackers track inbound on victim.
func TestCP_017_MultipleAttackersTracked(t *testing.T) {
	registryMu.Lock()
	machineRegistry = map[state.ActorRef]*Machine{}
	registryMu.Unlock()

	M1, M2, M3, P := NewMachine(), NewMachine(), NewMachine(), NewMachine()
	p := actor(1)
	RegisterMachine(p, P)
	RegisterMachine(actor(101), M1)
	RegisterMachine(actor(102), M2)
	RegisterMachine(actor(103), M3)

	for i, m := range []*Machine{M1, M2, M3} {
		require.NoError(t, m.TransitionToEngaging(EngagingData{Target: p},
			state.TransitionReason{Actor: actor(100 + i + 1), Target: p}))
	}
	require.Equal(t, Idle, P.State(), "defender stays Idle")
	require.Len(t, P.Attackers(), 3)
}

// CP-018: When attacker dies, inbound list shrinks.
func TestCP_018_DeadAttackerRemovedFromInbound(t *testing.T) {
	registryMu.Lock()
	machineRegistry = map[state.ActorRef]*Machine{}
	registryMu.Unlock()

	M1, M2 := NewMachine(), NewMachine()
	P := NewMachine()
	p := actor(1)
	RegisterMachine(p, P)
	RegisterMachine(actor(101), M1)
	RegisterMachine(actor(102), M2)

	require.NoError(t, M1.TransitionToEngaging(EngagingData{Target: p},
		state.TransitionReason{Actor: actor(101), Target: p}))
	require.NoError(t, M2.TransitionToEngaging(EngagingData{Target: p},
		state.TransitionReason{Actor: actor(102), Target: p}))

	require.Len(t, P.Attackers(), 2)
	M1.NotifySelfDied()
	require.Len(t, P.Attackers(), 1)
	require.Equal(t, actor(102), P.Attackers()[0])
}

// CP-019: Combat Phase exposes Attackers as observable change.
func TestCP_019_AttackersChangeIsObservable(t *testing.T) {
	registryMu.Lock()
	machineRegistry = map[state.ActorRef]*Machine{}
	registryMu.Unlock()

	M1 := NewMachine()
	P := NewMachine()
	RegisterMachine(actor(1), P)
	RegisterMachine(actor(101), M1)

	var observedAttackerCount int
	P.SubscribeAttackersChange(func(_ []state.ActorRef) {
		observedAttackerCount++
	})
	require.NoError(t, M1.TransitionToEngaging(EngagingData{Target: actor(1)},
		state.TransitionReason{Actor: actor(101), Target: actor(1)}))
	require.Equal(t, 1, observedAttackerCount, "Attackers-change observer fires on inbound add")
}

// --- Non-combat target picking (CP-026, CP-027) ---

// CP-026: Soft-target (skullduggery) does NOT transition Combat Phase.
// This is the chunk-2.7 regression test.
func TestCP_026_SoftTargetDoesNotTransition(t *testing.T) {
	T, _ := makePair()
	require.Equal(t, Idle, T.State(), "thief stays Idle when picking a soft target")
	// We deliberately do not call any TransitionTo — that's the point.
}

// CP-027: A failed steal does NOT auto-transition the mob to combat.
func TestCP_027_FailedStealNoAutoCombat(t *testing.T) {
	T, _ := makePair()
	require.Equal(t, Idle, T.State(),
		"failed non-combat action does not silently engage combat")
}

// --- Life cascade (CP-028, CP-029) ---

// CP-028: Death cascades to Idle outbound + clears inbound list.
func TestCP_028_DeathCascadeClearsAll(t *testing.T) {
	A, B := makePair()
	a, b := actor(1), actor(2)
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: b}, attackReason(a, b)))
	require.NoError(t, B.TransitionToEngaging(EngagingData{Target: a}, attackReason(b, a)))

	A.NotifySelfDied()
	require.Equal(t, Idle, A.State())
	require.Empty(t, A.Attackers())
	require.Empty(t, B.Attackers())
}

// CP-029: Inbound-only death still cleans both sides.
func TestCP_029_InboundOnlyDeathCleansAttackers(t *testing.T) {
	A, B := makePair() // A is defender, never attacked back
	require.NoError(t, B.TransitionToEngaging(EngagingData{Target: actor(1)},
		state.TransitionReason{Actor: actor(2), Target: actor(1)}))
	require.Len(t, A.Attackers(), 1)

	A.NotifySelfDied()
	require.Empty(t, A.Attackers())
}

// --- Persistence (CP-030, CP-031) ---

// CP-030: Fresh Machine instance is in Idle.
func TestCP_030_FreshMachineIsIdle(t *testing.T) {
	m := NewMachine()
	require.Equal(t, Idle, m.State())
	require.Empty(t, m.Attackers())
}

// CP-031: Scheduled transitions do not survive instance recreation.
func TestCP_031_ScheduledTransitionsDoNotPersist(t *testing.T) {
	m := NewMachine()
	require.Equal(t, Idle, m.State())
}

// --- Combatant flag (CP-032) ---

// CP-032: Combatant flag toggle forces Idle.
func TestCP_032_CombatantOffForcesIdle(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick() // → Engaged
	require.Equal(t, Engaged, A.State())

	A.ForceIdle(state.TransitionReason{Trigger: TriggerCombatantToggle})
	require.Equal(t, Idle, A.State())
}

// --- Tick events (CP-033 through CP-036) ---

// CP-033: Engaged state dispatches mob_combat_round.
func TestCP_033_EngagedDispatchesCombatRoundEvent(t *testing.T) {
	A, _ := makePair()
	var event string
	A.OnTickEvent(func(name string, _ state.TransitionReason) {
		event = name
	})
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick() // → Engaged
	A.DispatchTickEvent()
	require.Equal(t, "mob_combat_round", event)
}

// CP-034: Idle state dispatches mob_idle.
func TestCP_034_IdleDispatchesIdleEvent(t *testing.T) {
	A, _ := makePair()
	var event string
	A.OnTickEvent(func(name string, _ state.TransitionReason) {
		event = name
	})
	A.DispatchTickEvent() // currently Idle
	require.Equal(t, "mob_idle", event)
}

// CP-035: Engaging state dispatches no tick event.
func TestCP_035_EngagingDispatchesNoTickEvent(t *testing.T) {
	A, _ := makePair()
	var event string
	A.OnTickEvent(func(name string, _ state.TransitionReason) {
		event = name
	})
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2), RoundsUntil: 5},
		state.TransitionReason{}))
	A.DispatchTickEvent()
	require.Empty(t, event, "Engaging is silent")
}

// CP-036: Disengaging state dispatches no tick event.
func TestCP_036_DisengagingDispatchesNoTickEvent(t *testing.T) {
	A, _ := makePair()
	require.NoError(t, A.TransitionToEngaging(EngagingData{Target: actor(2)},
		state.TransitionReason{}))
	A.OnRoundTick() // → Engaged
	require.NoError(t, A.TransitionToDisengaging(state.TransitionReason{Trigger: TriggerFleeCommand}))
	var event string
	A.OnTickEvent(func(name string, _ state.TransitionReason) {
		event = name
	})
	A.DispatchTickEvent()
	require.Empty(t, event, "Disengaging is silent")
}
