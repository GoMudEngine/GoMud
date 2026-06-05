package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestIsGuardMob(t *testing.T) {
	if !mobs.IsGuardMob([]string{"humanoid", "guard", "thornwall_guards"}) {
		t.Error("expected guard group to be detected")
	}
	if mobs.IsGuardMob([]string{"humanoid", "merchant"}) {
		t.Error("non-guard must not be detected")
	}
	if mobs.IsGuardMob(nil) {
		t.Error("nil groups must not be detected")
	}
}
