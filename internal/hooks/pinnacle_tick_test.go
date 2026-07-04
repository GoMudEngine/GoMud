package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/itemvoices"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// setPinnacleEnabled flips the PinnacleItemsEnabled master toggle in-memory for
// the test process (AddOverlayOverrides, the same file-free mechanism
// enableItemProcs uses). The gate-off test relies on being able to set it back
// to false explicitly, since overlay overrides persist across tests.
func setPinnacleEnabled(t *testing.T, on bool) {
	t.Helper()
	if err := configs.AddOverlayOverrides(map[string]any{
		"GamePlay.PinnacleItemsEnabled": on,
	}); err != nil {
		t.Fatalf("failed to set PinnacleItemsEnabled=%v: %v", on, err)
	}
}

// ─── Hunger ───────────────────────────────────────────────────────────────────

// TestPinnacleHunger_DrainAndClock covers the anchor-on-first-tick behavior,
// the no-drain-before-threshold guard, and the escalating drain after.
func TestPinnacleHunger_DrainAndClock(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999950: {ItemId: 999950, Name: "blackrazor", Type: items.Weapon, Hands: 1,
			HungerRounds: 10, HungerDrainPct: 0.1},
	})()

	u := users.NewTestUser(701, "hungry", "Hungerer", 7701)
	c := u.Character
	c.HealthMax.Value = 100
	c.Health = 100
	c.Equipment.Weapon = items.New(999950)

	// First tick with no anchor: starts the clock, no drain.
	tickHunger(c, u, 50)
	if c.Health != 100 {
		t.Fatalf("first tick should not drain, health=%d", c.Health)
	}
	if a, ok := readMiscRound(c.GetMiscData("pinnacle_hunger_anchor")); !ok || a != 50 {
		t.Fatalf("first tick should set anchor to 50, got %v (ok=%v)", a, ok)
	}

	// elapsed 5 <= HungerRounds 10 → still no drain.
	tickHunger(c, u, 55)
	if c.Health != 100 {
		t.Fatalf("before threshold should not drain, health=%d", c.Health)
	}

	// elapsed 15, overdue 5, escalation 1 + 5/10 = 1.5 → drain 100*0.1*1.5 = 15.
	tickHunger(c, u, 65)
	if c.Health != 85 {
		t.Fatalf("expected drain to 85 (escalation 1.5), got %d", c.Health)
	}
}

// TestPinnacleHunger_NeverLethalAndKillReset proves hunger clamps at 1 HP and
// that a recent kill re-bases the anchor (feeding the blade).
func TestPinnacleHunger_NeverLethalAndKillReset(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999950: {ItemId: 999950, Name: "blackrazor", Type: items.Weapon, Hands: 1,
			HungerRounds: 10, HungerDrainPct: 0.1},
	})()

	u := users.NewTestUser(702, "hungry2", "Hungerer2", 7702)
	c := u.Character
	c.HealthMax.Value = 100
	c.Health = 5
	c.Equipment.Weapon = items.New(999950)

	// Massively overdue: escalation caps at 3x → nominal drain 30, but clamps so
	// Health floors at 1 (never lethal).
	c.SetMiscData("pinnacle_hunger_anchor", uint64(0))
	tickHunger(c, u, 1000)
	if c.Health != 1 {
		t.Fatalf("hunger must never kill; expected health 1, got %d", c.Health)
	}

	// A kill more recent than the anchor resets the hunger clock.
	c.Health = 100
	c.SetMiscData("pinnacle_hunger_anchor", uint64(50))
	c.SetMiscData("pinnacle_last_kill_round", uint64(60))
	tickHunger(c, u, 65) // anchor advances to 60, elapsed 5 <= 10 → no drain
	if c.Health != 100 {
		t.Fatalf("recent kill should reset the hunger clock, health=%d", c.Health)
	}
	if a, _ := readMiscRound(c.GetMiscData("pinnacle_hunger_anchor")); a != 60 {
		t.Fatalf("kill should advance anchor to 60, got %d", a)
	}
}

// TestPinnacleHunger_FeedingLineCooldown proves the drain repeats every overdue
// round but the feeding LINE is paced by pinnacle_hunger_msg_next_round
// (SentientChatterCooldownRounds): two consecutive overdue ticks both drain,
// only the first messages.
func TestPinnacleHunger_FeedingLineCooldown(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999950: {ItemId: 999950, Name: "blackrazor", Type: items.Weapon, Hands: 1,
			HungerRounds: 10, HungerDrainPct: 0.1},
	})()

	u := users.NewTestUser(708, "hungry3", "Hungerer3", 7708)
	c := u.Character
	c.HealthMax.Value = 1000
	c.Health = 1000
	c.Equipment.Weapon = items.New(999950)
	c.SetMiscData("pinnacle_hunger_anchor", uint64(0))

	// First overdue tick: drains AND messages (and arms the message cooldown).
	_ = events.DrainQueuedMessagesForTest(708)
	tickHunger(c, u, 20)
	healthAfterFirst := c.Health
	if healthAfterFirst >= 1000 {
		t.Fatal("first overdue tick should drain")
	}
	if msgs := events.DrainQueuedMessagesForTest(708); len(msgs) == 0 {
		t.Fatal("first overdue tick should emit the feeding line")
	}
	if _, ok := readMiscRound(c.GetMiscData("pinnacle_hunger_msg_next_round")); !ok {
		t.Fatal("feeding line should arm pinnacle_hunger_msg_next_round")
	}

	// Second consecutive overdue tick (well inside the 20-round chatter
	// cooldown): drains again, but stays silent.
	tickHunger(c, u, 21)
	if c.Health >= healthAfterFirst {
		t.Fatal("second overdue tick should drain again")
	}
	if msgs := events.DrainQueuedMessagesForTest(708); len(msgs) != 0 {
		t.Fatalf("second overdue tick within the message cooldown must be silent, got %v", msgs)
	}
}

// ─── Aging freeze ───────────────────────────────────────────────────────────

func TestPinnacleAgingFreeze(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999951: {ItemId: 999951, Name: "vitalis bandolier", Type: items.Belt,
			IsBandolier: true, BandolierCapacity: 4, PreservesContents: true},
		999952: {ItemId: 999952, Name: "plain belt", Type: items.Belt},
		999953: {ItemId: 999953, Name: "test potion", Type: items.Potion},
	})()

	// With preserves_contents: CraftedRound advances 1:1 (elapsed frozen).
	c := characters.New()
	c.Equipment.Belt = items.New(999951)
	p := items.New(999953)
	p.CraftedRound = 1000
	c.PotionItems = append(c.PotionItems, p)
	tickPreserveContents(c)
	if c.PotionItems[0].CraftedRound != 1001 {
		t.Fatalf("preserves belt should advance CraftedRound to 1001, got %d", c.PotionItems[0].CraftedRound)
	}

	// Without the flag: unchanged.
	c2 := characters.New()
	c2.Equipment.Belt = items.New(999952)
	p2 := items.New(999953)
	p2.CraftedRound = 1000
	c2.PotionItems = append(c2.PotionItems, p2)
	tickPreserveContents(c2)
	if c2.PotionItems[0].CraftedRound != 1000 {
		t.Fatalf("plain belt must not freeze aging, got %d", c2.PotionItems[0].CraftedRound)
	}
}

// ─── Ambient potions ────────────────────────────────────────────────────────

func TestPinnacleAmbientPotions(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999954: {ItemId: 999954, Name: "ambient bandolier", Type: items.Belt,
			IsBandolier: true, BandolierCapacity: 4, AmbientPotions: true},
		999955: {ItemId: 999955, Name: "vigor potion", Type: items.Potion, BuffIds: []int{54}},
	})()
	defer buffs.SeedBuffsForTest(map[int]*buffs.BuffSpec{
		54: {BuffId: 54, Name: "Vigor", Description: "test vigor", RoundInterval: 5, TriggerCount: 100},
	})()

	u := users.NewTestUser(703, "bandit", "Bandolier", 7703)
	c := u.Character
	c.Equipment.Belt = items.New(999954)
	c.PotionItems = append(c.PotionItems, items.New(999955))

	// First tick: fingerprint changes ("" → belt+potion), stamps the attunement
	// cooldown and applies NOTHING yet.
	tickAmbientPotions(u, 100)
	if c.Buffs.HasBuff(54) {
		t.Fatal("buff must not apply during the initial attunement window")
	}

	// Simulate attunement having expired, then tick with unchanged contents.
	c.SetMiscData("pinnacle_bandolier_attune_round", uint64(100))
	tickAmbientPotions(u, 150)
	if !c.Buffs.HasBuff(54) {
		t.Fatal("ambient buff should apply once attuned")
	}

	// Remove the potion → fingerprint changes → its buff is revoked (marked
	// expired, then evicted by the engine's per-turn prune — the same
	// mark-expired + prune path WornBuffIds use on unequip).
	c.PotionItems = nil
	tickAmbientPotions(u, 151)
	c.Buffs.Prune()
	if c.Buffs.HasBuff(54) {
		t.Fatal("removing the potion should revoke its ambient buff")
	}

	// Re-slot the potion → re-attunement → after it expires, the buff re-applies.
	c.PotionItems = append(c.PotionItems, items.New(999955))
	tickAmbientPotions(u, 152) // fingerprint change re-stamps attunement
	c.SetMiscData("pinnacle_bandolier_attune_round", uint64(152))
	tickAmbientPotions(u, 200)
	if !c.Buffs.HasBuff(54) {
		t.Fatal("ambient buff should re-apply after re-attunement expires")
	}
}

// ─── Mutation tick ──────────────────────────────────────────────────────────

func TestPinnacleMutationTick(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999956: {ItemId: 999956, Name: "seething prism", Type: items.Neck,
			MutationTickInterval: 5, MutationTickChance: 100, MutationRarityFloor: 8},
	})()
	// Exactly one qualifying (rarity >= floor) mutation → deterministic grant.
	defer mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"third_eye": {MutationId: "third_eye", Name: "Third Eye", Rarity: 8},
		"common_wart": {MutationId: "common_wart", Name: "Common Wart", Rarity: 2},
	})()

	u := users.NewTestUser(704, "prism", "Prismatic", 7704)
	c := u.Character
	c.Equipment.Neck = items.New(999956)

	// Off-interval round (11 % 5 != 0) grants nothing.
	tickMutationItems(u, c.GetAllWornItems(), 11)
	if len(c.Mutations) != 0 {
		t.Fatalf("off-interval tick must not grant, got %d mutations", len(c.Mutations))
	}

	// Interval-aligned round with chance 100 grants the rarity-8 mutation.
	tickMutationItems(u, c.GetAllWornItems(), 10)
	if _, ok := c.Mutations["third_eye"]; !ok {
		t.Fatalf("interval-aligned tick should grant the rarity-floored mutation, got %v", c.Mutations)
	}
	if _, ok := c.Mutations["common_wart"]; ok {
		t.Fatal("below-floor mutation must never be granted")
	}
}

// ─── Voices ─────────────────────────────────────────────────────────────────

// TestPinnaclePickVoiceEvent unit-tests the pure event-selection logic (no RNG,
// no side effects).
func TestPinnaclePickVoiceEvent(t *testing.T) {
	hungerSpec := items.ItemSpec{HungerRounds: 10}
	plainSpec := items.ItemSpec{}

	// In combat → taunt (takes precedence over everything).
	c := characters.New()
	c.Aggro = &characters.Aggro{UserId: 1, Type: characters.DefaultAttack}
	if ev := pickVoiceEvent(c, hungerSpec, 100); ev != "on_taunt" {
		t.Fatalf("in combat should pick on_taunt, got %q", ev)
	}

	// Hungry weapon past 3/4 of its window (anchor 0, now 8 > 7.5) → warning.
	c2 := characters.New()
	c2.SetMiscData("pinnacle_hunger_anchor", uint64(0))
	if ev := pickVoiceEvent(c2, hungerSpec, 8); ev != "on_hunger_warning" {
		t.Fatalf("hungry weapon near feeding should pick on_hunger_warning, got %q", ev)
	}

	// Well within the hunger window → idle.
	c3 := characters.New()
	c3.SetMiscData("pinnacle_hunger_anchor", uint64(0))
	if ev := pickVoiceEvent(c3, hungerSpec, 3); ev != "on_idle" {
		t.Fatalf("early in the hunger window should pick on_idle, got %q", ev)
	}

	// Non-hunger, non-combat item → idle.
	if ev := pickVoiceEvent(characters.New(), plainSpec, 100); ev != "on_idle" {
		t.Fatalf("plain sentient item should pick on_idle, got %q", ev)
	}
}

// TestPinnacleVoiceCooldownGates proves the chatter-cooldown MiscData gates
// re-entry deterministically: when pinnacle_voice_next_round is in the future,
// tickVoices returns before any RNG and emits nothing.
func TestPinnacleVoiceCooldownGates(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999957: {ItemId: 999957, Name: "whispering blade", Type: items.Weapon, Hands: 1,
			VoiceId: "test-voice"},
	})()
	defer itemvoices.SeedVoicesForTest(map[string]*itemvoices.VoiceSpec{
		"test-voice": {VoiceId: "test-voice", Lines: map[string][]string{
			"on_idle": {"I thirst."},
		}},
	})()

	u := users.NewTestUser(705, "whisper", "Whisperer", 7705)
	c := u.Character
	c.Equipment.Weapon = items.New(999957)
	room := &rooms.Room{RoomId: 500, Zone: "TestZone"}

	// Cooldown active (next round far in the future).
	c.SetMiscData("pinnacle_voice_next_round", uint64(1000))
	_ = events.DrainQueuedMessagesForTest(705) // clear the queue
	tickVoices(u, room, c.GetAllWornItems(), 100)
	if msgs := events.DrainQueuedMessagesForTest(705); len(msgs) != 0 {
		t.Fatalf("active chatter cooldown must gate all output, got %v", msgs)
	}
}

// TestPinnacleEmitVoiceLine covers the shared emit helper: authored line vs
// fallback vs silent.
func TestPinnacleEmitVoiceLine(t *testing.T) {
	defer itemvoices.SeedVoicesForTest(map[string]*itemvoices.VoiceSpec{
		"blade-voice": {VoiceId: "blade-voice", Lines: map[string][]string{
			"on_hunger_feeding": {"Yesss."},
		}},
	})()

	u := users.NewTestUser(706, "voice", "Voicer", 7706)

	// Item with a voice + authored line → emits it, returns true.
	voiced := items.ItemSpec{Name: "hungry blade", VoiceId: "blade-voice"}
	_ = events.DrainQueuedMessagesForTest(706)
	if !emitVoiceLine(u, nil, voiced, "on_hunger_feeding", "fallback flavor") {
		t.Fatal("emitVoiceLine should return true when an authored line exists")
	}
	if msgs := events.DrainQueuedMessagesForTest(706); len(msgs) == 0 {
		t.Fatal("authored-line emit should queue a message")
	}

	// No authored line for the event, but a fallback → emits the fallback.
	if !emitVoiceLine(u, nil, voiced, "on_idle", "fallback flavor") {
		t.Fatal("emitVoiceLine should return true when falling back")
	}
	if msgs := events.DrainQueuedMessagesForTest(706); len(msgs) == 0 {
		t.Fatal("fallback emit should queue a message")
	}

	// No voice and empty fallback → says nothing, returns false.
	plain := items.ItemSpec{Name: "plain sword"}
	if emitVoiceLine(u, nil, plain, "on_idle", "") {
		t.Fatal("emitVoiceLine should return false with no line and no fallback")
	}
	if msgs := events.DrainQueuedMessagesForTest(706); len(msgs) != 0 {
		t.Fatalf("silent emit should queue nothing, got %v", msgs)
	}
}

// ─── Master gate ────────────────────────────────────────────────────────────

// TestPinnacleUserTick_MasterGate proves the whole tick is a no-op when the
// feature is disabled, and runs when enabled — observed via whether tickHunger
// stamps the first-tick anchor (independent of the global round counter).
func TestPinnacleUserTick_MasterGate(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		999958: {ItemId: 999958, Name: "blackrazor", Type: items.Weapon, Hands: 1,
			HungerRounds: 10, HungerDrainPct: 0.1},
	})()

	u := users.NewTestUser(707, "gate", "Gatekeeper", 7707)
	c := u.Character
	c.HealthMax.Value = 100
	c.Health = 100
	c.Equipment.Weapon = items.New(999958)
	room := &rooms.Room{RoomId: 500, Zone: "TestZone"}

	// Gate OFF → tick does nothing, so hunger never even stamps its anchor.
	setPinnacleEnabled(t, false)
	pinnacleUserTick(u, room)
	if c.GetMiscData("pinnacle_hunger_anchor") != nil {
		t.Fatal("disabled feature must not run any sub-tick (anchor should be unset)")
	}

	// Gate ON → tick runs, hunger stamps the first-tick anchor.
	setPinnacleEnabled(t, true)
	pinnacleUserTick(u, room)
	if c.GetMiscData("pinnacle_hunger_anchor") == nil {
		t.Fatal("enabled feature should run sub-ticks (hunger anchor should be set)")
	}
}
