package awareness

// vetoChain holds the registered veto functions. Each returns
// true if the transition is OK, false to block it.
//
// activitySelf is the chunk-1 pre-wire for the future Activity
// machine (chunk 3 will repoint it to the real Activity query).
// For now it consults CastingState/CraftingState via the
// registered callback.
type vetoChain struct {
	activitySelf func() bool
}

// RegisterActivityCheck registers the Activity-state veto.
// Pre-wire for chunk 3; today the callback consults the existing
// CastingState/CraftingState pointers on Character.
func (m *Machine) RegisterActivityCheck(check func() bool) {
	m.vetoes.activitySelf = check
}
