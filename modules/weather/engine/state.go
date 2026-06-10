package engine

import (
	"encoding/json"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// StateIdentifier is the plugin-storage key for the persisted simulation state
// (plugin.WriteBytes / ReadBytes).
const StateIdentifier = "simstate"

// StateVersion is bumped whenever the persisted layout changes; a mismatched
// version is discarded (the module re-seeds) rather than migrated.
const StateVersion = 1

// persistedState wraps sim.State in a versioned envelope so the sim package
// stays free of persistence concerns.
type persistedState struct {
	Version int       `json:"version"`
	State   sim.State `json:"state"`
}

// EncodeState serializes simulation state for plugin storage.
func EncodeState(s sim.State) ([]byte, error) {
	return json.MarshalIndent(persistedState{Version: StateVersion, State: s}, "", "  ")
}

// DecodeState parses persisted state bytes, reporting ok=false for absent,
// unparseable, or version-mismatched data (caller starts fresh).
func DecodeState(b []byte) (sim.State, bool) {
	if len(b) == 0 {
		return sim.State{}, false
	}
	var ps persistedState
	if err := json.Unmarshal(b, &ps); err != nil {
		return sim.State{}, false
	}
	if ps.Version != StateVersion {
		return sim.State{}, false
	}
	return ps.State, true
}
