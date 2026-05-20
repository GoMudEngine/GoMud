package state

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type testState int

const (
	stateIdle testState = iota
	stateRunning
	stateDone
)

func makeTestMachine() *Machine[testState] {
	return NewMachine(stateIdle, TransitionTable[testState]{
		stateIdle:    {stateRunning},
		stateRunning: {stateDone, stateIdle},
	})
}

func TestMachine_InitialState(t *testing.T) {
	m := makeTestMachine()
	require.Equal(t, stateIdle, m.State())
}

func TestMachine_ValidTransition(t *testing.T) {
	m := makeTestMachine()
	err := m.TransitionTo(stateRunning, TransitionReason{Trigger: "test"})
	require.NoError(t, err)
	require.Equal(t, stateRunning, m.State())
}

func TestMachine_InvalidTransition(t *testing.T) {
	m := makeTestMachine()
	// stateIdle → stateDone is not in the table.
	err := m.TransitionTo(stateDone, TransitionReason{Trigger: "test"})
	require.ErrorIs(t, err, ErrInvalidTransition)
	require.Equal(t, stateIdle, m.State(), "state unchanged on invalid transition")
}

func TestMachine_VetoBlocks(t *testing.T) {
	m := makeTestMachine()
	m.BeforeTransition("blocker", func(from, to testState, _ TransitionReason) error {
		return errors.New("no")
	})
	err := m.TransitionTo(stateRunning, TransitionReason{Trigger: "test"})
	require.ErrorIs(t, err, ErrVetoed)
	require.Equal(t, stateIdle, m.State(), "state unchanged when vetoed")
}

func TestMachine_CascadeFires(t *testing.T) {
	m := makeTestMachine()
	var fired bool
	m.AfterTransition("cascade", func(from, to testState, _ TransitionReason) {
		require.Equal(t, stateIdle, from)
		require.Equal(t, stateRunning, to)
		fired = true
	})
	require.NoError(t, m.TransitionTo(stateRunning, TransitionReason{}))
	require.True(t, fired, "cascade should fire after successful transition")
}

func TestMachine_CascadeRunsAllInOrder(t *testing.T) {
	m := makeTestMachine()
	var order []string
	m.AfterTransition("first", func(_, _ testState, _ TransitionReason) {
		order = append(order, "first")
	})
	m.AfterTransition("second", func(_, _ testState, _ TransitionReason) {
		order = append(order, "second")
	})
	require.NoError(t, m.TransitionTo(stateRunning, TransitionReason{}))
	require.Equal(t, []string{"first", "second"}, order)
}

func TestMachine_ObserverFiresAfterCascade(t *testing.T) {
	m := makeTestMachine()
	var order []string
	m.AfterTransition("cascade", func(_, _ testState, _ TransitionReason) {
		order = append(order, "cascade")
	})
	m.Subscribe("observer", func(_, _ testState, _ TransitionReason) {
		order = append(order, "observer")
	})
	require.NoError(t, m.TransitionTo(stateRunning, TransitionReason{}))
	require.Equal(t, []string{"cascade", "observer"}, order)
}

func TestMachine_CanTransitionDoesNotMutate(t *testing.T) {
	m := makeTestMachine()
	require.NoError(t, m.CanTransition(stateRunning, TransitionReason{}))
	require.Equal(t, stateIdle, m.State())
}
