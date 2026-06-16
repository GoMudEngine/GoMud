package characters

import "testing"

func TestMigrateNewbieAwakening_Veteran(t *testing.T) {
	c := &Character{QuestProgress: map[int]string{8: "end"}} // old-world progress, no q30
	c.MigrateNewbieAwakening()
	if c.QuestProgress[30] != "end" {
		t.Errorf("veteran: want 30=end, got %q", c.QuestProgress[30])
	}
}

func TestMigrateNewbieAwakening_FreshChar(t *testing.T) {
	c := &Character{} // brand-new: empty progress
	c.MigrateNewbieAwakening()
	if _, ok := c.QuestProgress[30]; ok {
		t.Errorf("fresh char: 30 should be unset, got %q", c.QuestProgress[30])
	}
}

func TestMigrateNewbieAwakening_MidRite(t *testing.T) {
	c := &Character{QuestProgress: map[int]string{30: "start"}} // doing the rite
	c.MigrateNewbieAwakening()
	if c.QuestProgress[30] != "start" {
		t.Errorf("mid-rite: must not overwrite, got %q", c.QuestProgress[30])
	}
}

func TestMigrateNewbieAwakening_RunsOnce(t *testing.T) {
	c := &Character{QuestProgress: map[int]string{8: "end"}}
	c.MigrateNewbieAwakening()
	delete(c.QuestProgress, 30) // simulate a later quest-30 reset
	c.MigrateNewbieAwakening()  // must NOT re-grant (guard set)
	if _, ok := c.QuestProgress[30]; ok {
		t.Errorf("run-once: should not re-grant after guard set")
	}
}
