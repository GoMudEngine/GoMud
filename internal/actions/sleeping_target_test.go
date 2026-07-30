package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
)

// sleepBuffId is the buff the sleep action applies. Pinned here so a renumber
// fails loudly in this test instead of silently disabling every sleep guard.
const sleepBuffId = 15

// newChar builds a character the way sleep_test.go does — characters.New()
// plus an initialized Buffs, or AddBuff nil-panics.
func newChar() *characters.Character {
	c := characters.New()
	c.Name = "Marn"
	c.Buffs = buffs.New()
	return c
}

// sleeper returns a character carrying the Sleeping buff flag. Reuses
// seedSleepBuff from sleep_test.go — the buff spec registry is not loaded in
// unit tests, so AddBuff fails without it.
func sleeper(t *testing.T) *characters.Character {
	t.Helper()
	t.Cleanup(seedSleepBuff(t))
	c := newChar()
	if err := c.AddBuff(sleepBuffId, true); err != nil {
		t.Fatalf("could not apply the sleep buff (id %d): %v — has it been renumbered?", sleepBuffId, err)
	}
	if !c.HasBuffFlag(buffs.Sleeping) {
		t.Fatalf("buff %d applied but does not carry the Sleeping flag", sleepBuffId)
	}
	return c
}

func TestTargetAsleep(t *testing.T) {
	if TargetAsleep(nil) {
		t.Error("nil character must not report asleep")
	}
	awake := newChar()
	if TargetAsleep(awake) {
		t.Error("a fresh character must not report asleep")
	}
	if !TargetAsleep(sleeper(t)) {
		t.Error("a character with the Sleeping buff must report asleep")
	}
}

// The guard must be a pure predicate when there is no user to message — the
// shop/command paths call it in contexts where user may legitimately be nil.
func TestRefuseIfAsleepHandlesNilUser(t *testing.T) {
	if RefuseIfAsleep(sleeper(t), "Marn", nil) != true {
		t.Error("asleep target must still refuse when there is no user to tell")
	}
	if RefuseIfAsleep(newChar(), "Marn", nil) != false {
		t.Error("awake target must not refuse")
	}
	if RefuseIfAsleep(nil, "Marn", nil) != false {
		t.Error("nil target must not refuse")
	}
}

func TestRefuseMobIfAsleepNilMob(t *testing.T) {
	if RefuseMobIfAsleep(nil, nil) {
		t.Error("nil mob must not refuse (callers rely on this to fall through)")
	}
}
