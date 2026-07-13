package mutations

import "testing"

// TestGraphPool_ProofSliceGating proves the Wave 1 proof-slice tags actually
// gate acquisition by cluster affinity (contains() is defined in affinity_test.go).
func TestGraphPool_ProofSliceGating(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"ether-gland":   {MutationId: "ether-gland", Name: "Ether Gland", Rarity: 4, Clusters: []string{"ethereal"}, Pole: "belief"},
		"titan-growth":  {MutationId: "titan-growth", Name: "Titan Growth", Rarity: 7, Clusters: []string{"colossus"}, Pole: "body"},
		"rapid-healing": {MutationId: "rapid-healing", Name: "Rapid Healing", Rarity: 3}, // universal
	})
	defer cleanup()

	// No affinity → only the universal enabler is eligible.
	cold := GetGraphPool(map[string]int{}, map[string]float64{}, nil)
	if !contains(cold, "rapid-healing") {
		t.Fatal("universal rapid-healing must always be eligible")
	}
	if contains(cold, "ether-gland") || contains(cold, "titan-growth") {
		t.Fatal("clustered mutations must be gated out with zero affinity")
	}

	// A caster (ethereal affinity ≥ ether-gland's depth threshold) unlocks
	// ether-gland but not titan-growth (a colossus keystone, zero colossus affinity).
	caster := GetGraphPool(map[string]int{}, map[string]float64{"ethereal": depthThreshold(4) + 1}, nil)
	if !contains(caster, "ether-gland") {
		t.Fatal("ethereal affinity should unlock ether-gland")
	}
	if contains(caster, "titan-growth") {
		t.Fatal("ethereal affinity must NOT unlock the colossus keystone")
	}
}
