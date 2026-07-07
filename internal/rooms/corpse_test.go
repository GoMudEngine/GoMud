package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

// Corpse-loot redesign (2026-07-07): the corpse carries its own loot
// container; HasLoot reports whether anything remains to take.
func TestCorpse_HasLoot(t *testing.T) {
	var c Corpse

	if c.HasLoot() {
		t.Fatalf("empty corpse should not report loot")
	}

	c.Loot.Gold = 5
	if !c.HasLoot() {
		t.Fatalf("corpse with gold should report loot")
	}

	c.Loot.Gold = 0
	c.Loot.AddItem(items.Item{ItemId: 42})
	if !c.HasLoot() {
		t.Fatalf("corpse with an item should report loot")
	}
}

// Corpse-loot redesign (2026-07-07): LootAllowed gates who may loot a
// corpse. Free-for-all once now >= RoundOwnedUntil or when no owner is set;
// otherwise only listed owners may loot before the timeout.
func TestCorpse_LootAllowed(t *testing.T) {
	const until uint64 = 100

	// Owner in set, still owned -> allowed.
	owned := &Corpse{OwnerUserIds: []int{1, 2}, RoundOwnedUntil: until}
	assert.True(t, owned.LootAllowed(1, 50), "owner should loot while owned")

	// Non-owner, still owned -> denied.
	assert.False(t, owned.LootAllowed(3, 50), "non-owner should be denied while owned")

	// Non-owner, past the timeout -> allowed (free-for-all).
	assert.True(t, owned.LootAllowed(3, 100), "free-for-all at the timeout")
	assert.True(t, owned.LootAllowed(3, 150), "free-for-all past the timeout")

	// Empty owner set -> allowed regardless of round.
	free := &Corpse{OwnerUserIds: nil, RoundOwnedUntil: until}
	assert.True(t, free.LootAllowed(3, 0), "ownerless corpse is free-for-all")
	assert.True(t, free.LootAllowed(3, 50), "ownerless corpse is free-for-all")
}

// Test that if the corpse is already prunable, calling Update does nothing.
func TestCorpseUpdate_PrunableAlreadyTrue(t *testing.T) {
	corpse := &Corpse{
		UserId:       1,
		MobId:        2,
		Character:    characters.Character{},
		RoundCreated: 100,
		Prunable:     true,
	}

	// Even if we call Update with a large roundNow, prunable should remain true.
	corpse.Update(9999, "1 week")
	assert.True(t, corpse.Prunable, "Corpse should remain prunable when already set to true")
}

// Test that if decayRate is empty, it defaults to "1 week". We assume "1 week"
// is longer than the provided roundNow in this example, so it should remain false.
func TestCorpseUpdate_EmptyDecayRateNotEnoughTime(t *testing.T) {
	// Suppose RoundCreated=0 and roundNow=5000,
	// we assume "1 week" (whatever that translates to in rounds) is > 5000.
	corpse := &Corpse{
		UserId:       1,
		MobId:        2,
		Character:    characters.Character{},
		RoundCreated: 5000,
		Prunable:     false,
	}

	corpse.Update(5000, "")
	assert.False(t, corpse.Prunable, "Corpse should not be prunable yet if not enough time has elapsed for default decay rate")
}

// Test that if decayRate is empty, it defaults to "1 week". We assume "1 week"
// is shorter than the provided roundNow, so it becomes prunable.
func TestCorpseUpdate_EmptyDecayRateSufficientTime(t *testing.T) {
	// Suppose RoundCreated=0 and roundNow=20000,
	// we assume "1 week" < 20000 (whatever that is in your game logic).
	corpse := &Corpse{
		UserId:       1,
		MobId:        2,
		Character:    characters.Character{},
		RoundCreated: 0,
		Prunable:     false,
	}

	corpse.Update(20000, "")
	assert.True(t, corpse.Prunable, "Corpse should become prunable if enough time has elapsed for default decay rate")
}

// Test a custom decay rate, ensuring that when roundNow < decay threshold,
// the corpse is not prunable, and once roundNow >= threshold, it becomes prunable.
func TestCorpseUpdate_CustomDecayRate(t *testing.T) {
	// For demonstration, let's assume "2 weeks" is a certain number of rounds,
	// but the exact number depends on your game logic.
	// We'll test a transition point: just before and just after the threshold.
	corpse := &Corpse{
		UserId:       1,
		MobId:        2,
		Character:    characters.Character{},
		RoundCreated: 1000, // The starting round
		Prunable:     false,
	}

	// Just before decay threshold
	// e.g., if "2 weeks" from round 1000 is 21000, pass in 20999
	corpse.Update(1005, "2 weeks")
	assert.False(t, corpse.Prunable, "Corpse should not be prunable just before custom decay threshold")

	// At or after decay threshold
	// e.g., pass in 21000 or greater
	corpse.Update(21000, "2 weeks")
	assert.True(t, corpse.Prunable, "Corpse should become prunable at or after custom decay threshold")
}

// Example test that checks repeated updates before and after crossing the threshold.
func TestCorpseUpdate_MultipleCalls(t *testing.T) {
	// For demonstration, let's assume "1 week" is N rounds in your logic.
	corpse := &Corpse{
		RoundCreated: 100,
		Prunable:     false,
	}

	// First update: not enough rounds have passed
	corpse.Update(105, "1 week")
	assert.False(t, corpse.Prunable, "Corpse should still not be prunable if not enough rounds elapsed")

	// Second update: enough rounds have passed
	corpse.Update(20000, "1 week")
	assert.True(t, corpse.Prunable, "Corpse should become prunable after sufficient rounds elapsed")
}

// Stage 3.4: DisplayName helper tests.
func TestCorpse_DisplayName_CorpseNameOverride(t *testing.T) {
	c := Corpse{
		MobId:      374,
		Character:  characters.Character{Name: "a sturdy oak-and-iron supply wagon"},
		CorpseName: "splintered wagon wreckage",
	}
	if got := c.DisplayName(); got != "splintered wagon wreckage" {
		t.Errorf("DisplayName() = %q, want \"splintered wagon wreckage\"", got)
	}
}

func TestCorpse_DisplayName_DefaultsToCharacterNameCorpse(t *testing.T) {
	c := Corpse{
		MobId:     375,
		Character: characters.Character{Name: "Hob"},
	}
	if got := c.DisplayName(); got != "Hob corpse" {
		t.Errorf("DisplayName() = %q, want \"Hob corpse\"", got)
	}
}

// If you want to specifically test or mock the gametime package, you can do so
// in a separate test or by creating a mock gametime function. This example
// simply relies on the assumption that gametime.GetDate and its AddPeriod logic
// are correct. If those are crucial, consider additional testing around them.
func TestCorpseUpdate_GametimeIntegration(t *testing.T) {
	// If you have a need to test how gametime transforms RoundCreated + decayRate,
	// you might do something like:

	gd := gametime.GetDate(0)
	decayRound := gd.AddPeriod("1 week")

	// Ensure that the decayRound is what you expect...
	// But that might be tested better in the gametime package itself.
	assert.NotZero(t, decayRound, "Decay round should not be zero")
}
