package crafting

import "testing"

// ─── learn_only discovery exclusion ──────────────────────────────────────────

func TestLearnOnlyExcludedFromDiscovery(t *testing.T) {
	allRecipes = map[string]*RecipeSpec{
		"normal-x": {
			RecipeId:     "normal-x",
			Name:         "Normal X",
			Skill:        "blacksmithing",
			SkillMinimum: 10,
		},
		"pinnacle-x": {
			RecipeId:     "pinnacle-x",
			Name:         "Pinnacle X",
			Skill:        "blacksmithing",
			SkillMinimum: 65,
			LearnOnly:    true,
		},
	}

	// A blacksmith at skill 70 who knows neither recipe.
	known := map[string]int{}
	skills := map[string]int{"blacksmithing": 70}

	eligible := GetEligibleRecipes(known, skills, "blacksmithing")

	sawNormal, sawPinnacle := false, false
	for _, id := range eligible {
		if id == "normal-x" {
			sawNormal = true
		}
		if id == "pinnacle-x" {
			sawPinnacle = true
		}
	}
	if !sawNormal {
		t.Error("normal recipe should be discoverable")
	}
	if sawPinnacle {
		t.Error("learn_only recipe must NOT be discoverable")
	}
}
