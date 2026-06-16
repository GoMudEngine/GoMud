package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestQuestCooldown_ActiveThenExpires(t *testing.T) {
	defer util.ResetRoundCountForTest()
	util.SetRoundCountForTest(1000)

	c := &Character{}
	// No cooldown set yet -> not active.
	if c.QuestCooldownActive(53) {
		t.Fatalf("no cooldown set: want inactive, got active")
	}

	// Set a 200-round cooldown at round 1000 -> available again at 1200.
	c.SetQuestCooldown(53, 200)
	if !c.QuestCooldownActive(53) {
		t.Fatalf("just-set cooldown at round 1000: want active, got inactive")
	}

	// Still inside the window.
	util.SetRoundCountForTest(1199)
	if !c.QuestCooldownActive(53) {
		t.Fatalf("round 1199 (< 1200): want active, got inactive")
	}

	// At/after the threshold -> expired.
	util.SetRoundCountForTest(1200)
	if c.QuestCooldownActive(53) {
		t.Fatalf("round 1200 (>= 1200): want inactive, got active")
	}
}

func TestQuestCooldown_CoercesReloadedInt(t *testing.T) {
	// MiscData round-trips through YAML as an int; QuestCooldownActive must
	// coerce a plain int value, not only the uint64 it was written as.
	defer util.ResetRoundCountForTest()
	util.SetRoundCountForTest(500)

	c := &Character{}
	c.SetMiscData("questcd-53", int(1000)) // simulate a reloaded value
	if !c.QuestCooldownActive(53) {
		t.Fatalf("int-typed cooldown 1000 at round 500: want active, got inactive")
	}
	util.SetRoundCountForTest(1000)
	if c.QuestCooldownActive(53) {
		t.Fatalf("int-typed cooldown 1000 at round 1000: want inactive, got active")
	}
}
