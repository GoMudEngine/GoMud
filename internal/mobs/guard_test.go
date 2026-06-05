package mobs

import "testing"

func TestIsGuardMob(t *testing.T) {
	if !IsGuardMob([]string{"humanoid", "guard"}) {
		t.Fatal("expected true when groups contain 'guard'")
	}
	if IsGuardMob([]string{"humanoid", "thornwall_guards"}) {
		t.Fatal("faction id 'thornwall_guards' is NOT the 'guard' marker")
	}
	if IsGuardMob([]string{"humanoid"}) {
		t.Fatal("expected false without 'guard' marker")
	}
	if IsGuardMob(nil) {
		t.Fatal("expected false for nil groups")
	}
}
