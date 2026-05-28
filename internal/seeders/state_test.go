package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestApplyCooldown_FirstCall_TrueAndWrites(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99001)}
	mob.Character.Name = "cooldown_first"

	ok := applyCooldown(mob, "test-rule", "key-1", 100)
	if !ok {
		t.Errorf("first call should return true (no active cooldown)")
	}

	// Verify the MiscData key was written.
	if got := mob.Character.GetMiscData("seed_cooldown:test-rule:key-1"); got == nil {
		t.Errorf("cooldown marker not written to MiscData")
	}
}

func TestApplyCooldown_WithinWindow_FalseNoWrite(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99002)}
	mob.Character.Name = "cooldown_within"

	// First call: write cooldown.
	applyCooldown(mob, "test-rule", "key-1", 100)
	first := mob.Character.GetMiscData("seed_cooldown:test-rule:key-1")

	// Second call immediately: should be blocked.
	ok := applyCooldown(mob, "test-rule", "key-1", 100)
	if ok {
		t.Errorf("second call within window should return false")
	}

	// Marker unchanged.
	second := mob.Character.GetMiscData("seed_cooldown:test-rule:key-1")
	if first != second {
		t.Errorf("cooldown marker was rewritten during active window: %v -> %v", first, second)
	}
}

func TestApplyCooldown_DifferentKey_NotBlocked(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99003)}
	mob.Character.Name = "cooldown_diff_key"

	applyCooldown(mob, "test-rule", "key-A", 100)
	// Different key on same rule: should NOT be blocked.
	ok := applyCooldown(mob, "test-rule", "key-B", 100)
	if !ok {
		t.Errorf("different cooldown key should not be blocked by key-A")
	}
}

func TestApplyCooldown_DifferentRule_NotBlocked(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99004)}
	mob.Character.Name = "cooldown_diff_rule"

	applyCooldown(mob, "rule-A", "shared-key", 100)
	ok := applyCooldown(mob, "rule-B", "shared-key", 100)
	if !ok {
		t.Errorf("different rule should not be blocked by rule-A on same key")
	}
}

func TestApplyCooldown_NilMob_NoOp(t *testing.T) {
	if applyCooldown(nil, "test", "key", 100) {
		t.Errorf("nil mob should return false (no work to do)")
	}
}

func TestBumpMiscInt_FromAbsent(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99005)}
	mob.Character.Name = "bump_absent"
	bumpMiscInt(mob, "counter:x", 3)
	if got := readMiscInt(mob, "counter:x", 0); got != 3 {
		t.Errorf("got %d, want 3", got)
	}
}

func TestBumpMiscInt_FromExisting(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99006)}
	mob.Character.Name = "bump_existing"
	mob.Character.SetMiscData("counter:y", 10)
	bumpMiscInt(mob, "counter:y", 5)
	if got := readMiscInt(mob, "counter:y", 0); got != 15 {
		t.Errorf("got %d, want 15", got)
	}
}

// Note: seedRevengeGoalIfAbsent tests defer to rule 4's
// friend_killed_to_revenge tests, where the helper is exercised with
// real goals.Add flow + dedup verification.
