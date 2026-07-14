package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// seedSmartAmbient seeds an ambient bandolier + two distinct-buff potions.
func seedSmartAmbient() func() {
	restoreI := items.SeedItemsForTest(map[int]*items.ItemSpec{
		999954: {ItemId: 999954, Name: "Vitalis Bandolier", Type: items.Belt,
			IsBandolier: true, BandolierCapacity: 4, AmbientPotions: true},
		999955: {ItemId: 999955, Name: "vigor potion", Type: items.Potion, BuffIds: []int{54}},
		999956: {ItemId: 999956, Name: "renewal potion", Type: items.Potion, BuffIds: []int{55}},
	})
	restoreB := buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		54: {BuffId: 54, Name: "Vigor", Description: "vigor", RoundInterval: 5, TriggerCount: 100},
		55: {BuffId: 55, Name: "Renewal", Description: "renewal", RoundInterval: 5, TriggerCount: 100},
	})
	return func() { restoreI(); restoreB() }
}

// TestAmbientSmartReset_KeepsExistingWhenAddingSecond is the core "smarter reset"
// guarantee: attuning a newly-added potion must NOT drop the buff you already
// had. This is the fix for the prod complaint that crafting more potions made
// the whole bandolier feel broken.
func TestAmbientSmartReset_KeepsExistingWhenAddingSecond(t *testing.T) {
	defer seedSmartAmbient()()

	u := users.NewTestUser(720, "smart", "SmartReset", 7720)
	c := u.Character
	c.Equipment.Belt = items.New(999954)
	c.PotionItems = append(c.PotionItems, items.New(999955)) // buff 54

	// Attune the first potion.
	tickAmbientPotions(u, 100) // fp change -> attuning, nothing applied
	if c.Buffs.HasBuff(54) {
		t.Fatal("buff 54 should still be attuning on the first tick")
	}
	c.SetMiscData("pinnacle_bandolier_attune_round", uint64(100))
	tickAmbientPotions(u, 150) // attuned -> 54 applied
	if !c.Buffs.HasBuff(54) {
		t.Fatal("buff 54 should be attuned and active")
	}

	// Add a SECOND, distinct potion. The already-attuned buff 54 must NOT drop.
	c.PotionItems = append(c.PotionItems, items.New(999956)) // buff 55
	tickAmbientPotions(u, 151)                               // fp change: 55 new, 54 kept
	if !c.Buffs.HasBuff(54) {
		t.Fatal("adding a second potion must NOT revoke the already-attuned buff 54")
	}
	if c.Buffs.HasBuff(55) {
		t.Fatal("the newly-added potion (55) should still be attuning, not applied yet")
	}

	// Finish attuning the second potion; both are now active.
	c.SetMiscData("pinnacle_bandolier_attune_round", uint64(151))
	tickAmbientPotions(u, 200)
	if !c.Buffs.HasBuff(54) || !c.Buffs.HasBuff(55) {
		t.Fatalf("both buffs should be active after 55 attunes (54=%v 55=%v)",
			c.Buffs.HasBuff(54), c.Buffs.HasBuff(55))
	}
}

// TestAmbientSmartReset_RemoveRevokesOnlyThatBuff proves removing one potion
// revokes only its buff, leaving the others active (no full re-attune).
func TestAmbientSmartReset_RemoveRevokesOnlyThatBuff(t *testing.T) {
	defer seedSmartAmbient()()

	u := users.NewTestUser(722, "remove", "Remover", 7722)
	c := u.Character
	c.Equipment.Belt = items.New(999954)
	c.PotionItems = append(c.PotionItems, items.New(999955)) // 54
	c.PotionItems = append(c.PotionItems, items.New(999956)) // 55

	// Attune both.
	tickAmbientPotions(u, 100)
	c.SetMiscData("pinnacle_bandolier_attune_round", uint64(100))
	tickAmbientPotions(u, 150)
	if !c.Buffs.HasBuff(54) || !c.Buffs.HasBuff(55) {
		t.Fatalf("both should attune (54=%v 55=%v)", c.Buffs.HasBuff(54), c.Buffs.HasBuff(55))
	}

	// Remove the buff-54 potion (keep only the buff-55 one).
	c.PotionItems = c.PotionItems[1:]
	tickAmbientPotions(u, 151) // fp change: 54 gone
	c.Buffs.Prune()            // RemoveBuff marks expired; the per-turn prune evicts it
	if c.Buffs.HasBuff(54) {
		t.Fatal("the removed potion's buff 54 should be revoked")
	}
	if !c.Buffs.HasBuff(55) {
		t.Fatal("the remaining potion's buff 55 must stay active (no full re-attune)")
	}
}

// TestAmbientSmartReset_Messages proves the attuning/attuned feedback fires once
// each and steady state is silent.
func TestAmbientSmartReset_Messages(t *testing.T) {
	defer seedSmartAmbient()()

	u := users.NewTestUser(721, "msgs", "Messenger", 7721)
	c := u.Character
	c.Equipment.Belt = items.New(999954)
	c.PotionItems = append(c.PotionItems, items.New(999955))

	_ = events.DrainQueuedMessagesForTest(721)
	tickAmbientPotions(u, 100) // begins attuning
	if !containsLine(events.DrainQueuedMessagesForTest(721), "attune") {
		t.Fatal("slotting a new potion should announce that it is attuning")
	}

	c.SetMiscData("pinnacle_bandolier_attune_round", uint64(100))
	tickAmbientPotions(u, 150) // completes
	if !containsLine(events.DrainQueuedMessagesForTest(721), "resonance") {
		t.Fatal("completing attunement should announce that the virtues suffuse you")
	}

	tickAmbientPotions(u, 151) // steady state
	if m := events.DrainQueuedMessagesForTest(721); len(m) != 0 {
		t.Fatalf("steady state must be silent, got %v", m)
	}
}
