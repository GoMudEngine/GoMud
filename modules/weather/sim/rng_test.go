package sim

import "testing"

func TestRNGDeterministic(t *testing.T) {
	a := NewRNG(42)
	b := NewRNG(42)
	for i := 0; i < 100; i++ {
		if a.Uint64() != b.Uint64() {
			t.Fatalf("same seed must produce same sequence (i=%d)", i)
		}
	}
}

func TestRNGStateRoundTrips(t *testing.T) {
	a := NewRNG(7)
	for i := 0; i < 10; i++ {
		a.Uint64()
	}
	// Reconstruct from the cursor and confirm the continuation matches.
	b := NewRNG(a.State())
	if a.Uint64() != b.Uint64() {
		t.Fatal("RNG reconstructed from State() must continue the same sequence")
	}
}

func TestRNGFloatAndIntnRanges(t *testing.T) {
	r := NewRNG(1)
	for i := 0; i < 1000; i++ {
		f := r.Float64()
		if f < 0 || f >= 1 {
			t.Fatalf("Float64 out of [0,1): %v", f)
		}
		n := r.Intn(5)
		if n < 0 || n >= 5 {
			t.Fatalf("Intn(5) out of range: %d", n)
		}
	}
	if r.Intn(0) != 0 {
		t.Fatal("Intn(0) should return 0, not panic")
	}
}
