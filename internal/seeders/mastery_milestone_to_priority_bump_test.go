package seeders

import "testing"

func TestMasteryMilestone_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["SkillUsed"] {
		if reg.name == ruleNameMasteryMilestone {
			found = true
			break
		}
	}
	if !found {
		// Expected: events.SkillUsed does not carry MobInstanceId or
		// NewRank fields, so the Register call in init() is intentionally
		// commented out. The rule ships as a documented stub pending an
		// event-shape extension (see the init() comment in
		// mastery_milestone_to_priority_bump.go for the exact blocker
		// and what changes would unblock it).
		t.Skip("mastery_milestone not registered — SkillUsed event deferred (no MobInstanceId or NewRank field)")
	}
}

func TestMasteryMilestoneInterval_Constants(t *testing.T) {
	// Sanity-check constant values so a future editor can't accidentally
	// set the soft cap below the milestone interval.
	if masteryMilestoneInterval <= 0 {
		t.Errorf("masteryMilestoneInterval must be positive, got %d", masteryMilestoneInterval)
	}
	if masterySoftCap <= masteryMilestoneInterval {
		t.Errorf("masterySoftCap (%d) must exceed masteryMilestoneInterval (%d)",
			masterySoftCap, masteryMilestoneInterval)
	}
}

func TestMasteryMilestoneSeed_NotMilestoneRank_ExitsEarly(t *testing.T) {
	// When newRank is hard-wired to 0 (TODO-ADAPT stub), the rule
	// exits at the "newRank <= 0" guard. Confirm no panic.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic in masteryMilestoneSeed: %v", r)
		}
	}()

	// fakeSkillEvent satisfies events.Event; the concrete type assertion
	// to events.SkillUsed will fail, so masteryMilestoneSeed returns at
	// the first type-assertion check — exercising the defensive guard.
	masteryMilestoneSeed(fakeEvent{typeName: "SkillUsed"})
}

// Behavior tests (milestone math, mob lookup, goals.Add) require a
// mob rank-up event with MobInstanceId + NewRank fields that do not
// yet exist in events.SkillUsed. Deferred to Task 15 smoke once the
// event-shape extension lands.
