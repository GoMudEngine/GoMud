package migration

import "testing"

func TestSeedForCluster(t *testing.T) {
	if got := SeedForCluster("ravener"); got["rending-claws"] != 1 || got["raptor-legs"] != 1 || len(got) != 2 {
		t.Errorf("ravener seed = %v", got)
	}
	if got := SeedForCluster("chrysifier"); got["provident-hands"] != 1 || got["walking-chrysalis"] != 1 {
		t.Errorf("chrysifier seed = %v", got)
	}
	if got := SeedForCluster("generalist"); len(got) != 1 || got["keen-senses"] != 1 {
		t.Errorf("generalist seed = %v", got)
	}
	if got := SeedForCluster("admin"); len(got) < 9 { // all-access, one per cluster+
		t.Errorf("admin seed too small: %d", len(got))
	}
	if got := SeedForCluster("nonsense"); got != nil {
		t.Errorf("unknown cluster should be nil, got %v", got)
	}
}
