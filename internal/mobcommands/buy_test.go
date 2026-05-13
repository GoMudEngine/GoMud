package mobcommands

import "testing"

func TestBuyRegistered(t *testing.T) {
	all := GetAllMobCommands()
	found := false
	for _, cmd := range all {
		if cmd == "buy" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'buy' to be registered in mobCommands")
	}
}

func TestBuyEmptyRest_NoOp(t *testing.T) {
	// Mob buy with empty rest is a silent no-op. Direct call (not
	// through TryCommand) since TryCommand requires a live mob
	// instance which the harness doesn't set up trivially.
	handled, err := Buy("", nil, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handled {
		t.Errorf("expected handled=true for empty rest no-op")
	}
}
