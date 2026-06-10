package skills

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/stats"
)

func TestGetTitle_IsTitleCased(t *testing.T) {
	var s stats.Statistics
	s.Strength.Value = 200
	s.Dexterity.Value = 100
	s.Perception.Value = 100
	s.Vitality.Value = 100
	s.Willpower.Value = 100
	s.Charisma.Value = 100

	got := GetTitle(map[string]int{}, map[string]int{}, s)
	if got != "Scrub Warrior" {
		t.Errorf("GetTitle = %q, want %q", got, "Scrub Warrior")
	}
}

// TestRangedCombat_Registered verifies the revived ranged-combat skill is fully
// wired into the registry: listed by GetAllSkillNames, has a progression
// multiplier, and is governed by Perception.
func TestRangedCombat_Registered(t *testing.T) {
	if !SkillExists(string(RangedCombat)) {
		t.Errorf("ranged-combat must be registered in allSkillNames")
	}

	found := false
	for _, sk := range GetAllSkillNames() {
		if sk == RangedCombat {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ranged-combat must appear in GetAllSkillNames()")
	}

	if _, ok := SkillProgressionMultipliers[RangedCombat]; !ok {
		t.Errorf("ranged-combat must have a progression multiplier entry")
	}

	if stat := GetSkillPrimaryStat(string(RangedCombat)); stat != "perception" {
		t.Errorf("ranged-combat primary stat = %q, want perception", stat)
	}
}
