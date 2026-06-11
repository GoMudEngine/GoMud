package planners

import "testing"

func TestSkillTrainingContextOf_CombatSkills(t *testing.T) {
	for _, name := range []string{"weapon-combat", "unarmed-combat", "ranged-combat", "spellcasting", "manifestation"} {
		if got := SkillTrainingContextOf(name); got != TrainingCombat {
			t.Errorf("%s: got %v, want TrainingCombat", name, got)
		}
	}
}

func TestSkillTrainingContextOf_CraftingSkills(t *testing.T) {
	for _, name := range []string{"blacksmithing", "alchemy", "cooking", "salvage"} {
		if got := SkillTrainingContextOf(name); got != TrainingCrafting {
			t.Errorf("%s: got %v, want TrainingCrafting", name, got)
		}
	}
}

func TestSkillTrainingContextOf_Foraging(t *testing.T) {
	if got := SkillTrainingContextOf("search"); got != TrainingForaging {
		t.Errorf("got %v, want TrainingForaging", got)
	}
}

func TestSkillTrainingContextOf_Unknown(t *testing.T) {
	if got := SkillTrainingContextOf("nonexistent-skill"); got != TrainingUnknown {
		t.Errorf("got %v, want TrainingUnknown", got)
	}
}
