package hooks

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// File: combat_verbosity_wiring_test.go
//
// Pins the Task-5 drain-side glue: the verbosity gate applied at the
// combat message drain (drainParticipantLines / drainSpectatorLines),
// the light-mode tally recording (recordTallyFor / recordSpectatorTallies),
// and the round-end flush (flushCombatTallies). These functions are
// exercised directly with seeded fixtures rather than driving a full
// combat round so each verbosity level can be asserted deterministically
// without seeding the RNG for a specific hit/miss outcome.
//
// Test message fixtures use natural prose tokens ("batter"/"dodge"/
// "parry") so the messaging pipeline (capitalize + end-punct normalize)
// leaves a recognizable substring; assertions are case-insensitive.

// distinctive line texts for category gating assertions.
const (
	// "batter" deliberately avoids words the tally renderer emits
	// (strike/lands/blow/guard/trade) so a summary line can't false-match.
	vbHitText   = "you batter the foe relentlessly"
	vbDodgeText = "you twist away in a dodge"
	vbParryText = "you turn the blow with a parry"
	vbPlainText = "the dust settles around you"
)

// vbHitMsgs / vbDefenseMsgs build representative tagged-message batches.
func vbHitDodgeBatch() []combat.TaggedMessage {
	return []combat.TaggedMessage{
		{Category: messaging.CategoryHitMelee, Text: vbHitText},
		{Category: messaging.CategoryDodge, Text: vbDodgeText},
	}
}

func vbHitParryBatch() []combat.TaggedMessage {
	return []combat.TaggedMessage{
		{Category: messaging.CategoryHitMelee, Text: vbHitText},
		{Category: messaging.CategoryParry, Text: vbParryText},
	}
}

// vbLandingResult is an AttackResult that lands a blow — enough to
// produce a non-empty tally line.
func vbLandingResult() *combat.AttackResult {
	return &combat.AttackResult{
		Hit:                 true,
		DamageToTarget:      10,
		DefenderWasAttacked: true,
		SwingEvents:         []combat.SwingEvent{{Hit: true, Damage: 10}},
	}
}

// textsForUser returns the rendered texts of every captured direct
// message bound for userId.
func textsForUser(captured []events.Message, userId int) []string {
	out := []string{}
	for _, m := range captured {
		if m.UserId == userId {
			out = append(out, m.Text)
		}
	}
	return out
}

// countContaining counts texts that contain needle (case-insensitive).
func countContaining(texts []string, needle string) int {
	n := 0
	low := strings.ToLower(needle)
	for _, t := range texts {
		if strings.Contains(strings.ToLower(t), low) {
			n++
		}
	}
	return n
}

func TestCombatVerbosityWiring(t *testing.T) {
	// ── 1. Full participant: hit AND dodge both delivered ────────────────
	t.Run("FullParticipant_BothLinesDelivered", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		roundTallies = newCombatTallies()

		u := users.GetByUserId(1)
		require.NotNil(t, u)
		u.CombatVerbosity = "" // full (default)

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		drainParticipantLines(u, vbHitDodgeBatch(), u.GetCombatVerbosity(), false)
		events.ProcessEvents()

		texts := textsForUser(*captured, 1)
		assert.Equal(t, 1, countContaining(texts, "batter"), "full: hit line must be delivered")
		assert.Equal(t, 1, countContaining(texts, "dodge"), "full: dodge line must be delivered")
	})

	// ── 2. Medium attacker: hit delivered, dodge suppressed ──────────────
	t.Run("MediumAttacker_DodgeSuppressed", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		roundTallies = newCombatTallies()

		u := users.GetByUserId(1)
		require.NotNil(t, u)
		u.CombatVerbosity = "medium"

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		drainParticipantLines(u, vbHitDodgeBatch(), u.GetCombatVerbosity(), false)
		events.ProcessEvents()

		texts := textsForUser(*captured, 1)
		assert.Equal(t, 1, countContaining(texts, "batter"), "medium: hit line must be delivered")
		assert.Equal(t, 0, countContaining(texts, "dodge"), "medium: dodge line must be suppressed")
	})

	// ── 3. Medium defender: incoming hit floor-protected, parry dropped ──
	t.Run("MediumDefender_IncomingHitFloorProtected", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		roundTallies = newCombatTallies()

		u := users.GetByUserId(1)
		require.NotNil(t, u)
		u.CombatVerbosity = "medium"

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		// incoming=true: CategoryHit* lines describe damage TO the viewer.
		drainParticipantLines(u, vbHitParryBatch(), u.GetCombatVerbosity(), true)
		events.ProcessEvents()

		texts := textsForUser(*captured, 1)
		assert.Equal(t, 1, countContaining(texts, "batter"),
			"medium defender: incoming hit (damage to you) must always show")
		assert.Equal(t, 0, countContaining(texts, "parry"),
			"medium defender: parry line must be suppressed")
	})

	// ── 4. Light attacker: lines suppressed, one tally on flush ──────────
	t.Run("LightAttacker_LinesSuppressedTallyFlushed", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		roundTallies = newCombatTallies()

		u := users.GetByUserId(1)
		require.NotNil(t, u)
		u.CombatVerbosity = "light"

		room1 := rooms.LoadRoom(1)
		require.NotNil(t, room1)
		atk := actions.NewUserActorInRoom(u, room1)
		mob := mobs.GetInstance(100)
		require.NotNil(t, mob)
		def := actions.NewMobActorInRoom(mob, room1)

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		drainParticipantLines(u, vbHitDodgeBatch(), u.GetCombatVerbosity(), false)
		recordTallyFor(u.UserId, atk, def, vbLandingResult())
		flushCombatTallies()
		events.ProcessEvents()

		texts := textsForUser(*captured, 1)
		assert.Equal(t, 0, countContaining(texts, "batter"), "light: per-swing hit line suppressed")
		assert.Equal(t, 0, countContaining(texts, "dodge"), "light: per-swing dodge line suppressed")
		assert.Equal(t, 1, len(texts), "light: exactly one summary line should arrive after flush")
		assert.Equal(t, 1, countContaining(texts, "Skeleton"),
			"light: the summary line must mention the opponent")
	})

	// ── 5. Spectator at full → effective medium ──────────────────────────
	t.Run("SpectatorFull_EffectiveMedium", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		roundTallies = newCombatTallies()

		spectator := users.GetByUserId(2)
		require.NotNil(t, spectator)
		spectator.CombatVerbosity = "" // full → spectator tier medium

		room1 := rooms.LoadRoom(1)
		require.NotNil(t, room1)

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		// Combatant user 1 is excluded; user 2 is the spectator.
		drainSpectatorLines(room1, vbHitDodgeBatch(), []int{1})
		events.ProcessEvents()

		texts := textsForUser(*captured, 2)
		assert.Equal(t, 1, countContaining(texts, "batter"),
			"spectator(full→medium): room hit line delivered")
		assert.Equal(t, 0, countContaining(texts, "dodge"),
			"spectator(full→medium): room dodge line suppressed")
	})

	// ── 6. Spectator at medium → effective light ─────────────────────────
	t.Run("SpectatorMedium_EffectiveLight", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		roundTallies = newCombatTallies()

		spectator := users.GetByUserId(2)
		require.NotNil(t, spectator)
		spectator.CombatVerbosity = "medium" // → spectator tier light

		room1 := rooms.LoadRoom(1)
		require.NotNil(t, room1)
		u1 := users.GetByUserId(1)
		require.NotNil(t, u1)
		atk := actions.NewUserActorInRoom(u1, room1)
		mob := mobs.GetInstance(100)
		require.NotNil(t, mob)
		def := actions.NewMobActorInRoom(mob, room1)

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		drainSpectatorLines(room1, vbHitDodgeBatch(), []int{1})
		recordSpectatorTallies(room1, room1, atk, def, vbLandingResult(), []int{1})
		flushCombatTallies()
		events.ProcessEvents()

		texts := textsForUser(*captured, 2)
		assert.Equal(t, 0, countContaining(texts, "batter"),
			"spectator(medium→light): all per-swing room lines suppressed")
		assert.Equal(t, 0, countContaining(texts, "dodge"),
			"spectator(medium→light): all per-swing room lines suppressed")
		assert.Equal(t, 1, len(texts),
			"spectator(medium→light): exactly one summary line after flush")
		assert.Equal(t, 1, countContaining(texts, "Skeleton"),
			"spectator(medium→light): summary mentions the combatants")
	})

	// ── 7. Non-suppressible category passes at light ─────────────────────
	t.Run("LightParticipant_DefaultCategoryPasses", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		roundTallies = newCombatTallies()

		u := users.GetByUserId(1)
		require.NotNil(t, u)
		u.CombatVerbosity = "light"

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		drainParticipantLines(u,
			[]combat.TaggedMessage{{Category: messaging.CategoryDefault, Text: vbPlainText}},
			u.GetCombatVerbosity(), false)
		events.ProcessEvents()

		texts := textsForUser(*captured, 1)
		assert.Equal(t, 1, countContaining(texts, "dust settles"),
			"light: a category outside the suppress table must always pass")
	})

	// ── 8. Dark-room spectator: sight gate suppresses tally ───────────────
	// Fix 1: recordSpectatorTallies must not record a tally for a spectator
	// who cannot see the fight (dark room, no night-vision). The named
	// summary would reveal combatant identities they have no line-of-sight to.
	t.Run("DarkRoomSpectator_NoTally", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		roundTallies = newCombatTallies()

		spectator := users.GetByUserId(2)
		require.NotNil(t, spectator)
		spectator.CombatVerbosity = "medium" // → effective light tier

		// Make room 2 dark by setting the cave biome (DarkArea:true →
		// GetVisibility() == 0 → CanSeeClearly returns false).
		room1 := rooms.LoadRoom(1)
		room2 := rooms.LoadRoom(2)
		require.NotNil(t, room1)
		require.NotNil(t, room2)
		room2.Biome = "cave"
		room1.RemovePlayer(2)
		room2.AddPlayer(2)
		spectator.Character.RoomId = 2

		// Attacker = mob in dark room; defender = user 1 (excluded as combatant).
		u1 := users.GetByUserId(1)
		require.NotNil(t, u1)
		mob := mobs.GetInstance(100)
		require.NotNil(t, mob)
		atk := actions.NewMobActorInRoom(mob, room2)
		def := actions.NewUserActorInRoom(u1, room1)

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		recordSpectatorTallies(room2, room1, atk, def, vbLandingResult(), []int{1})
		flushCombatTallies()
		events.ProcessEvents()

		texts := textsForUser(*captured, 2)
		assert.Equal(t, 0, len(texts),
			"spectator in dark room must not receive any tally summary")
	})

	// ── 9. Blind attacker: participant tally gate blocks summary ──────────
	// Fix 2: a light-verbosity attacker whose prose was darkness-substituted
	// must not receive a named tally summary via the round-end flush either.
	// Tested end-to-end through dispatchCritAndMessaging so the srcCanSee
	// gate wiring is exercised directly.
	t.Run("BlindAttacker_NoParticipantTally", func(t *testing.T) {
		cleanup := seedAllRegistries()
		defer cleanup()
		roundTallies = newCombatTallies()

		u1 := users.GetByUserId(1)
		require.NotNil(t, u1)
		u1.CombatVerbosity = "light"

		// Move attacker into dark room 2 (cave biome).
		room1 := rooms.LoadRoom(1)
		room2 := rooms.LoadRoom(2)
		require.NotNil(t, room1)
		require.NotNil(t, room2)
		room2.Biome = "cave"
		room1.RemovePlayer(1)
		room2.AddPlayer(1)
		u1.Character.RoomId = 2

		mob := mobs.GetInstance(100)
		require.NotNil(t, mob)
		atk := actions.NewUserActorInRoom(u1, room2)
		def := actions.NewMobActorInRoom(mob, room2)

		captured, capCleanup := captureMessages(t)
		defer capCleanup()

		dispatchCritAndMessaging(atk, def, vbLandingResult())
		flushCombatTallies()
		events.ProcessEvents()

		// The tally would say "You strike Skeleton (…)" — confirm that
		// text never arrives. Generic dark-room fallback messages are
		// excluded because the attacker is in the combatant-exclude list.
		texts := textsForUser(*captured, 1)
		assert.Equal(t, 0, countContaining(texts, "Skeleton"),
			"blind attacker must not receive a tally summary naming the target")
	})
}
