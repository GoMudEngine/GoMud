package sim

// RNG is a small, deterministic, serializable PRNG (splitmix64). Its entire
// state is a single uint64, so it round-trips trivially through State.
type RNG struct{ state uint64 }

// NewRNG creates a PRNG positioned at the given cursor (use a seed to start, or
// a value from State() to resume an exact sequence).
func NewRNG(seed uint64) *RNG { return &RNG{state: seed} }

// State returns the current cursor, for serialization.
func (r *RNG) State() uint64 { return r.state }

// Uint64 returns the next pseudo-random 64-bit value (splitmix64).
func (r *RNG) Uint64() uint64 {
	r.state += 0x9E3779B97F4A7C15
	z := r.state
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// Float64 returns a value in [0, 1).
func (r *RNG) Float64() float64 {
	return float64(r.Uint64()>>11) / float64(uint64(1)<<53)
}

// Intn returns a value in [0, n); returns 0 for n <= 0.
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.Uint64() % uint64(n))
}
