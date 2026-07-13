package characters

import "testing"

func TestDriftFromCombat_OncePerRound(t *testing.T) {
	c := &Character{}
	c.DriftFromCombat("ironhide", 5)
	first := c.ClusterAffinity["ironhide"]
	if first <= 0 {
		t.Fatal("first combat drift should grant affinity")
	}
	c.DriftFromCombat("ironhide", 5) // same round -> no-op
	if c.ClusterAffinity["ironhide"] != first {
		t.Error("second grant in the same round should be a no-op")
	}
	c.DriftFromCombat("ironhide", 6) // next round -> grants again
	if c.ClusterAffinity["ironhide"] <= first {
		t.Error("next round should grant again")
	}
	// distinct clusters are tracked independently within a round
	c.DriftFromCombat("weaver", 6)
	if c.ClusterAffinity["weaver"] <= 0 {
		t.Error("a different cluster should still grant in the same round")
	}
}
