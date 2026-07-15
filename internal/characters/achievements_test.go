package characters

import "testing"

func TestCharacterAchievements(t *testing.T) {
	c := &Character{}
	if c.HasAchievement("first-blood") {
		t.Error("fresh character has no achievements")
	}
	c.GrantAchievement("first-blood", 42)
	if !c.HasAchievement("first-blood") {
		t.Error("granted achievement should be present")
	}
	if c.Achievements["first-blood"] != 42 {
		t.Errorf("unlock round = %d, want 42", c.Achievements["first-blood"])
	}
	// Idempotent: re-grant keeps the original round.
	c.GrantAchievement("first-blood", 99)
	if c.Achievements["first-blood"] != 42 {
		t.Errorf("re-grant should not overwrite the original unlock round")
	}
}
