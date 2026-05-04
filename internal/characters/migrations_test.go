package characters

import "testing"

func TestMigrateRecipeDisciplineShuffle_BumpsBlacksmithingForLockpicks(t *testing.T) {
	c := &Character{
		Skills:       map[string]int{"blacksmithing": 5},
		KnownRecipes: map[string]int{"master-lockpicks": 1},
	}
	c.MigrateRecipeDisciplineShuffle()
	if got := c.Skills["blacksmithing"]; got != 20 {
		t.Errorf("blacksmithing = %d, want 20", got)
	}
}

func TestMigrateRecipeDisciplineShuffle_BumpsJewelcraftingForDisarmKit(t *testing.T) {
	c := &Character{
		Skills:       map[string]int{"jewelcrafting": 0},
		KnownRecipes: map[string]int{"reinforced-disarm-kit": 1},
	}
	c.MigrateRecipeDisciplineShuffle()
	if got := c.Skills["jewelcrafting"]; got != 15 {
		t.Errorf("jewelcrafting = %d, want 15", got)
	}
}

func TestMigrateRecipeDisciplineShuffle_PreservesHigherSkill(t *testing.T) {
	c := &Character{
		Skills:       map[string]int{"blacksmithing": 35, "jewelcrafting": 25},
		KnownRecipes: map[string]int{"master-lockpicks": 1, "reinforced-disarm-kit": 1},
	}
	c.MigrateRecipeDisciplineShuffle()
	if got := c.Skills["blacksmithing"]; got != 35 {
		t.Errorf("blacksmithing = %d, want 35 (preserved)", got)
	}
	if got := c.Skills["jewelcrafting"]; got != 25 {
		t.Errorf("jewelcrafting = %d, want 25 (preserved)", got)
	}
}

func TestMigrateRecipeDisciplineShuffle_NoOpWithoutRecipes(t *testing.T) {
	c := &Character{
		Skills:       map[string]int{"blacksmithing": 5},
		KnownRecipes: map[string]int{"healing-salve": 1},
	}
	c.MigrateRecipeDisciplineShuffle()
	if got := c.Skills["blacksmithing"]; got != 5 {
		t.Errorf("blacksmithing = %d, want 5 (untouched)", got)
	}
}

func TestMigrateRecipeDisciplineShuffle_RunsOnce(t *testing.T) {
	c := &Character{
		Skills:       map[string]int{"blacksmithing": 5},
		KnownRecipes: map[string]int{"master-lockpicks": 1},
		MiscData:     map[string]any{},
	}
	c.MigrateRecipeDisciplineShuffle()
	// Manually drop the skill back to simulate post-migration state.
	c.Skills["blacksmithing"] = 1
	c.MigrateRecipeDisciplineShuffle()
	if got := c.Skills["blacksmithing"]; got != 1 {
		t.Errorf("blacksmithing = %d, want 1 (migration should have re-run-skipped)", got)
	}
}
