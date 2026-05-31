package planners

import "testing"

func TestHuntBountyTarget_Registered(t *testing.T) {
	if LookupPlanner("hunt_bounty_target") == nil {
		t.Fatalf("hunt_bounty_target planner not registered")
	}
}

func TestHuntDecision(t *testing.T) {
	// jailed target → hold (empty command)
	if cmd, _ := huntDecision(true, 10, 10, 7); cmd != "" {
		t.Fatalf("jailed target must hold (empty command), got %q", cmd)
	}
	// same room → attack
	if cmd, _ := huntDecision(false, 10, 10, 7); cmd != "attack @7" {
		t.Fatalf("same room should attack, got %q", cmd)
	}
	// different room → pathto target room
	if cmd, _ := huntDecision(false, 10, 25, 7); cmd != "pathto 25" {
		t.Fatalf("pursue should pathto target room, got %q", cmd)
	}
}
