package activity

import "github.com/GoMudEngine/GoMud/internal/state"

// TransitionToCasting moves Free → Casting and stores the casting
// context. Caller is responsible for any pre-checks (e.g., "is the
// character free", "does the character have enough conviction").
func (m *Machine) TransitionToCasting(d CastingData, r state.TransitionReason) error {
	d.Reason = r
	prev := m.casting
	m.casting = &d
	if err := m.inner.TransitionTo(Casting, r); err != nil {
		m.casting = prev
		return err
	}
	return nil
}

// TransitionToCrafting moves Free → Crafting and stores the
// crafting context.
func (m *Machine) TransitionToCrafting(d CraftingData, r state.TransitionReason) error {
	d.Reason = r
	prev := m.crafting
	m.crafting = &d
	if err := m.inner.TransitionTo(Crafting, r); err != nil {
		m.crafting = prev
		return err
	}
	return nil
}

// TransitionToSalvaging moves Free → Salvaging and stores the
// salvaging context.
func (m *Machine) TransitionToSalvaging(d SalvagingData, r state.TransitionReason) error {
	d.Reason = r
	prev := m.salvaging
	m.salvaging = &d
	if err := m.inner.TransitionTo(Salvaging, r); err != nil {
		m.salvaging = prev
		return err
	}
	return nil
}

// TransitionToFree returns the machine to Free, clearing all
// per-state data. All cancel / complete / interrupt paths route
// through here.
func (m *Machine) TransitionToFree(r state.TransitionReason) error {
	if err := m.inner.TransitionTo(Free, r); err != nil {
		return err
	}
	m.casting = nil
	m.crafting = nil
	m.salvaging = nil
	return nil
}

// ForceFree transitions to Free from any state. Used by admin
// commands and emergency cleanup. Idempotent if already Free.
func (m *Machine) ForceFree(r state.TransitionReason) {
	if m.State() == Free {
		return
	}
	_ = m.inner.TransitionTo(Free, r)
	m.casting = nil
	m.crafting = nil
	m.salvaging = nil
}
