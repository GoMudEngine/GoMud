package sim

import "encoding/json"

// NewState returns the initial simulation state for a fresh run.
func NewState(seed uint64) State {
	return State{
		RNGState: seed,
		NextID:   1,
		Weather:  map[ZoneId]WeatherType{},
	}
}

// DeriveSeed produces a stable default seed from the graph's sorted zone names
// (FNV-1a), so each world gets the same seed on every boot but two worlds
// differ. Used when the configured Seed is 0.
func DeriveSeed(g *Graph) uint64 {
	// FNV-1a 64-bit offset basis / prime.
	const prime = 1099511628211
	h := uint64(14695981039346656037)
	for _, z := range g.Zones() {
		for i := 0; i < len(z); i++ {
			h ^= uint64(z[i])
			h *= prime
		}
		h ^= 0xff // name separator so ["ab","c"] != ["a","bc"]
		h *= prime
	}
	return h
}

// ToJSON serializes the full simulation state (RNG cursor + fronts + resolved
// weather) for persistence.
func (s State) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// StateFromJSON restores a State from its serialized form.
func StateFromJSON(b []byte) (State, error) {
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return State{}, err
	}
	return s, nil
}
