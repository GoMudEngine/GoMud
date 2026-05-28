package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TestOnTheft_NilVictim_NoPanic verifies that a nil victimMob is handled
// gracefully without panicking.
func TestOnTheft_NilVictim_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on nil victim: %v", r)
		}
	}()
	OnTheft(5, nil, items.Item{})
}

// TestOnTheft_ZeroThiefId_NoPanic verifies that a zero thiefUserId is
// handled gracefully without panicking.
func TestOnTheft_ZeroThiefId_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on zero thief id: %v", r)
		}
	}()
	mob := &mobs.Mob{}
	mob.MobId = mobs.MobId(99000)
	mob.Character.Name = "theft_test"
	OnTheft(0, mob, items.Item{})
}

// Full seed integration test (victim + room + witness mobs all loaded)
// requires live instance map + room state; deferred to Task 15 smoke.
