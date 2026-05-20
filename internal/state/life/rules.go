package life

import "github.com/GoMudEngine/GoMud/internal/state"

// TransitionToDead is the primary death entry point. All death
// paths (damage-application, suicide command, admin kill) route
// through here.
//
// DeadData populates the cascade context — Killer and DamageMap
// drive downstream observers (kill credit, faction rep, etc.).
func (m *Machine) TransitionToDead(d DeadData, r state.TransitionReason) error {
	d.Reason = r
	m.dead = &d
	if err := m.inner.TransitionTo(Dead, r); err != nil {
		m.dead = nil
		return err
	}
	return nil
}

// TransitionToRespawning advances Dead → Respawning. Caller
// (typically Life_Cascades.go on the Dead-entry cascade) supplies
// the DestRoomId (graveyard or home).
func (m *Machine) TransitionToRespawning(d RespawningData, r state.TransitionReason) error {
	d.Reason = r
	prevDead := m.dead
	m.respawning = &d
	m.dead = nil
	if err := m.inner.TransitionTo(Respawning, r); err != nil {
		m.respawning = nil
		m.dead = prevDead
		return err
	}
	return nil
}

// TransitionToAlive advances Respawning → Alive (or Dead → Alive
// for admin restoration edge cases). Clears all per-state data.
func (m *Machine) TransitionToAlive(r state.TransitionReason) error {
	if err := m.inner.TransitionTo(Alive, r); err != nil {
		return err
	}
	m.dead = nil
	m.respawning = nil
	return nil
}

// ForceAlive transitions to Alive from any state. Used by admin
// restoration commands. Idempotent if already Alive.
func (m *Machine) ForceAlive(r state.TransitionReason) {
	if m.State() == Alive {
		return
	}
	_ = m.inner.TransitionTo(Alive, r)
	m.dead = nil
	m.respawning = nil
}
