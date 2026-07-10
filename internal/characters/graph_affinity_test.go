package characters

import "testing"

func TestAddClusterAffinity_Accumulates(t *testing.T) {
	c := &Character{}
	c.AddClusterAffinity("ravener", 1.5)
	c.AddClusterAffinity("ravener", 0.5)
	if got := c.ClusterAffinity["ravener"]; got != 2.0 {
		t.Fatalf("ravener affinity = %v, want 2.0", got)
	}
}

func TestOnSkillUseScaled_FeedsClusterAffinity(t *testing.T) {
	c := &Character{Name: "T"}
	c.OnSkillUseScaled("spellcasting", 0, 1.0)
	if c.ClusterAffinity["ethereal"] <= 0 {
		t.Fatalf("expected ethereal affinity > 0, got %v", c.ClusterAffinity["ethereal"])
	}
	c.OnSkillUseScaled("cooking", 0, 1.0) // no mapping -> no new clusters
	if _, ok := c.ClusterAffinity["colossus"]; ok {
		t.Fatal("cooking should not add cluster affinity")
	}
}
