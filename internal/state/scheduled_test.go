package state

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScheduler_TransitionFiresOnRoundTick(t *testing.T) {
	m := makeTestMachine()
	sched := NewRoundScheduler()

	// Schedule transition to Running after 2 rounds.
	m.ScheduleTransition(stateRunning, sched.RoundsFromNow(2),
		TransitionReason{Trigger: "scheduled_test"})

	require.Equal(t, stateIdle, m.State(), "should not fire before round elapses")

	sched.Tick()
	require.Equal(t, stateIdle, m.State(), "still 1 round remaining")

	sched.Tick()
	require.Equal(t, stateRunning, m.State(), "should fire at round 2")
}

func TestScheduler_TransitionFiresImmediatelyAtZero(t *testing.T) {
	m := makeTestMachine()
	sched := NewRoundScheduler()
	m.ScheduleTransition(stateRunning, sched.RoundsFromNow(0),
		TransitionReason{Trigger: "immediate"})
	sched.Tick()
	require.Equal(t, stateRunning, m.State())
}

func TestScheduler_CancelScheduled(t *testing.T) {
	m := makeTestMachine()
	sched := NewRoundScheduler()
	m.ScheduleTransition(stateRunning, sched.RoundsFromNow(2),
		TransitionReason{Trigger: "to_cancel"})
	m.CancelScheduled()
	sched.Tick()
	sched.Tick()
	require.Equal(t, stateIdle, m.State(), "canceled transition should not fire")
}

var errVetoTest = errors.New("test veto")

func TestScheduler_VetoedScheduledTransitionLeavesStateAlone(t *testing.T) {
	m := makeTestMachine()
	sched := NewRoundScheduler()
	m.BeforeTransition("blocker", func(from, to testState, r TransitionReason) error {
		if r.Trigger == "scheduled_vetoed" {
			return errVetoTest
		}
		return nil
	})
	m.ScheduleTransition(stateRunning, sched.RoundsFromNow(1),
		TransitionReason{Trigger: "scheduled_vetoed"})
	sched.Tick()
	require.Equal(t, stateIdle, m.State(), "veto blocks scheduled transition")
}
