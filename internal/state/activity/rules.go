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

// AdvanceCastingFolds increments folds accumulated and conviction
// spent on the current cast. Returns the updated data and true if
// the cast just completed (FoldsAccumulated >= FoldsNeeded).
//
// Caller is responsible for transitioning to Free + resolving the
// spell when complete = true.
func (m *Machine) AdvanceCastingFolds(folds int, convictionCost int) (CastingData, bool) {
	if m.State() != Casting || m.casting == nil {
		return CastingData{}, false
	}
	m.casting.FoldsAccumulated += folds
	m.casting.ConvictionSpent += convictionCost
	complete := m.casting.FoldsAccumulated >= m.casting.FoldsNeeded
	return *m.casting, complete
}

// AdvanceCraftingRound increments rounds complete on the current
// craft. Returns the updated data and true if the craft just
// completed.
func (m *Machine) AdvanceCraftingRound() (CraftingData, bool) {
	if m.State() != Crafting || m.crafting == nil {
		return CraftingData{}, false
	}
	m.crafting.RoundsComplete++
	complete := m.crafting.RoundsComplete >= m.crafting.RoundsTotal
	return *m.crafting, complete
}

// AdvanceSalvagingRound is the Salvaging equivalent of AdvanceCraftingRound.
func (m *Machine) AdvanceSalvagingRound() (SalvagingData, bool) {
	if m.State() != Salvaging || m.salvaging == nil {
		return SalvagingData{}, false
	}
	m.salvaging.RoundsComplete++
	complete := m.salvaging.RoundsComplete >= m.salvaging.RoundsTotal
	return *m.salvaging, complete
}
