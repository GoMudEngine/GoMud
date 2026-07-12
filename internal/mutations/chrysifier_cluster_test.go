package mutations

import "testing"

func TestSkillClusters_Chrysifier(t *testing.T) {
	crafts := []string{"blacksmithing", "alchemy", "tailoring", "cooking",
		"jewelcrafting", "enchanting", "salvage", "foraging"}
	for _, s := range crafts {
		cls := ClustersForSkill(s)
		if len(cls) != 1 || cls[0] != "chrysifier" {
			t.Fatalf("ClustersForSkill(%q) = %v, want [chrysifier]", s, cls)
		}
	}
}
