package hooks

import "testing"

func TestIsGuardMob(t *testing.T) {
	if !isGuardMob([]string{"humanoid", "guard", "thornwall_guards"}) {
		t.Error("expected guard group to be detected")
	}
	if isGuardMob([]string{"humanoid", "merchant"}) {
		t.Error("non-guard must not be detected")
	}
	if isGuardMob(nil) {
		t.Error("nil groups must not be detected")
	}
}
