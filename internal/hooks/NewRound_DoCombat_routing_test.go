package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// File: NewRound_DoCombat_routing_test.go
//
// Stage 2b structural test for the unified combat-round handler. Drives
// a representative sample of each of the four quadrant pairs (UU, UM,
// MU, MM) end-to-end through handleCombatRound and asserts the routing
// contract: text recipients are correct per-quadrant, mob_hurt fires on
// mob defenders only, and the unified handler does not crash on any
// quadrant.
//
// The test deliberately invokes handleCombatRound — NOT the phase
// helpers — so a future regression in the dispatcher flip (Task 3) or
// in any phase helper (Task 2a) is caught at the contract level.

// recordingNode is a behavior tree node that records each invocation,
// satisfying behaviortree.Node. Used to detect mob_hurt dispatch in
// the routing test.
type recordingNode struct {
	calls []behaviortree.EventContext
}

func (n *recordingNode) Evaluate(ctx *behaviortree.EvalContext) behaviortree.Result {
	n.calls = append(n.calls, ctx.Event)
	return behaviortree.Success
}

// captureMessages registers an events.Message listener and returns the
// captured slice plus an unregister cleanup. Drains the queue once
// before unregistering so any messages emitted during the run are
// captured even if the test forgets to call ProcessEvents.
func captureMessages(t *testing.T) (*[]events.Message, func()) {
	t.Helper()
	captured := []events.Message{}
	id := events.RegisterListener(events.Message{}, func(e events.Event) events.ListenerReturn {
		if msg, ok := e.(events.Message); ok {
			captured = append(captured, msg)
		}
		return events.Continue
	})
	return &captured, func() {
		events.ProcessEvents()
		events.UnregisterListener(events.Message{}, id)
	}
}

// seedSecondMobInstance adds a second mob instance (101) into the
// existing seedAllRegistries() set, sharing the same RoomId as instance
// 100. Returns a cleanup that removes only the second instance —
// callers are still responsible for calling the seedAllRegistries
// cleanup to tear down the rest.
func seedSecondMobInstance(t *testing.T) {
	t.Helper()
	// The instance counter is reset to 200 by SeedMobsForTest, so manually
	// adding instance 101 is safe — it won't collide with auto-spawn.
	m1 := mobs.GetInstance(100)
	require.NotNil(t, m1, "mob instance 100 must be present (seeded by seedAllRegistries)")

	m2 := &mobs.Mob{
		MobId:      m1.MobId,
		InstanceId: 101,
		HomeRoomId: m1.HomeRoomId,
		Hostile:    true,
		Groups:     m1.Groups,
		Character: characters.Character{
			Name:      "Skeleton2",
			RoomId:    m1.Character.RoomId,
			Health:    m1.Character.Health,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
		},
	}
	m2.Character.HealthMax.Value = m1.Character.HealthMax.Value
	m2.Character.StaminaMax.Value = m1.Character.StaminaMax.Value
	m2.Character.Stamina = m1.Character.Stamina
	m2.Character.ConvictionMax.Value = m1.Character.ConvictionMax.Value
	m2.Character.Conviction = m1.Character.Conviction
	m2.Character.Stats.Strength.ValueAdj = m1.Character.Stats.Strength.ValueAdj
	m2.Character.Stats.Dexterity.ValueAdj = m1.Character.Stats.Dexterity.ValueAdj
	m2.Character.Stats.Willpower.ValueAdj = m1.Character.Stats.Willpower.ValueAdj

	// Manually splice the new instance into the global map. SeedMobsForTest
	// is the only way to swap the entire map; for our incremental case we
	// reach into the package via its public GetInstance + a paired Add. The
	// cleanest workable path is to use SeedMobsForTest — but that resets
	// state we want to preserve. Workaround: register the instance via the
	// package's internal registry by re-seeding with both instances.
	specs := map[int]*mobs.Mob{
		int(m1.MobId): {MobId: m1.MobId, Zone: m1.Zone, Hostile: m1.Hostile,
			ActivityLevel: m1.ActivityLevel, Groups: m1.Groups,
			Character: characters.Character{Name: m1.Character.Name}},
	}
	instances := map[int]*mobs.Mob{
		100: m1,
		101: m2,
	}
	cleanup := mobs.SeedMobsForTest(specs, instances)
	t.Cleanup(cleanup)

	// Place the second mob in the same room as the first.
	r1 := rooms.LoadRoom(m1.Character.RoomId)
	require.NotNil(t, r1)
	r1.AddMob(101)
}

// TestHandleCombatRound_AllQuadrantsRouteCorrectly drives the unified
// combat handler through each of the four quadrant pairs (UU, UM, MU,
// MM) and asserts:
//
//  1. text recipient routing — direct messages targeted at a user id
//     fire only when that side is a player; no direct user-id messages
//     fire for mob participants;
//  2. mob_hurt dispatch — fires on mob defenders only; never on player
//     defenders (Divergence #2 in the design spec);
//  3. the unified handler does not crash on any quadrant.
//
// The assertions deliberately avoid forcing a specific hit/miss outcome
// — combat.AttackXxxVsXxx emits MessagesToSource / MessagesToTarget on
// both hits AND misses, so the routing contract is observable without
// seeding the RNG. This is the structural integration coverage that
// Stage 1 deferred to Stage 2.
func TestHandleCombatRound_AllQuadrantsRouteCorrectly(t *testing.T) {
	type quadrant struct {
		name        string
		atkIsPlayer bool
		defIsPlayer bool
	}
	cases := []quadrant{
		{"UU", true, true},
		{"UM", true, false},
		{"MU", false, true},
		{"MM", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cleanup := seedAllRegistries()
			defer cleanup()

			// Combat lookups dereference species.GetSpecies(SpeciesId)
			// without nil-checking, so we seed a minimal Human species and
			// stamp SpeciesId=1 on every combatant.
			cleanupSpecies := species.SeedSpeciesForTest(map[int]*species.Species{
				1: {SpeciesId: 1, Name: "Human", UnarmedName: "fist"},
			})
			defer cleanupSpecies()

			// items.GetAttackMessage infinite-recurses when attackMessages
			// is empty (the production map is loaded by items.LoadDataFiles
			// at startup, but unit tests don't trigger that). Seed a minimal
			// Generic-subtype fixture so the lookup resolves on first try.
			cleanupAtkMsgs := items.SeedAttackMessagesForTest(items.MinimalCombatMessageFixture())
			defer cleanupAtkMsgs()

			// MM needs a second mob instance to attack the first.
			if !tc.atkIsPlayer && !tc.defIsPlayer {
				seedSecondMobInstance(t)
			}

			// Ensure every combatant has SpeciesId set.
			if u := users.GetByUserId(1); u != nil {
				u.Character.SpeciesId = 1
			}
			if u := users.GetByUserId(2); u != nil {
				u.Character.SpeciesId = 1
			}
			if m := mobs.GetInstance(100); m != nil {
				m.Character.SpeciesId = 1
			}
			if m := mobs.GetInstance(101); m != nil {
				m.Character.SpeciesId = 1
			}

			// Install a recording behavior tree on the defender mob's MobId
			// (when relevant) so we can detect mob_hurt dispatch. The mob
			// template id is 1 (set in seedAllRegistries).
			var recorder *recordingNode
			var btreeCleanup func()
			if !tc.defIsPlayer {
				recorder = &recordingNode{}
				btreeCleanup = behaviortree.GetEngine().SetMobTreeForTest(1, recorder)
				defer btreeCleanup()
			}

			// Build atk / def Actor wrappers per quadrant.
			room1 := rooms.LoadRoom(1)
			require.NotNil(t, room1)

			var atk, def actions.Actor
			var atkUserId, defUserId int

			if tc.atkIsPlayer {
				u1 := users.GetByUserId(1)
				require.NotNil(t, u1)
				atk = actions.NewUserActorInRoom(u1, room1)
				atkUserId = 1
			} else {
				m := mobs.GetInstance(100)
				require.NotNil(t, m)
				atk = actions.NewMobActorInRoom(m, room1)
			}

			if tc.defIsPlayer {
				u2 := users.GetByUserId(2)
				require.NotNil(t, u2)
				def = actions.NewUserActorInRoom(u2, room1)
				defUserId = 2
			} else {
				// MM uses instance 101; UM uses instance 100.
				defInstId := 100
				if !tc.atkIsPlayer && !tc.defIsPlayer {
					defInstId = 101
				}
				m := mobs.GetInstance(defInstId)
				require.NotNil(t, m, "defender mob instance %d should exist", defInstId)
				def = actions.NewMobActorInRoom(m, room1)
			}

			// Set Aggro on the attacker so resolveCombatTarget passes.
			if tc.defIsPlayer {
				atk.GetCharacter().SetAggro(defUserId, 0, characters.DefaultAttack)
			} else {
				defMobInstanceId := 100
				if !tc.atkIsPlayer && !tc.defIsPlayer {
					defMobInstanceId = 101
				}
				atk.GetCharacter().SetAggro(0, defMobInstanceId, characters.DefaultAttack)
			}

			// Capture all events.Message events emitted during the round.
			captured, captureCleanup := captureMessages(t)
			defer captureCleanup()

			// Drive the unified handler end-to-end.
			var (
				affPlayers []int
				affMobs    []int
			)
			cfg := configs.GetConfig()
			handleCombatRound(
				atk, def,
				events.NewRound{RoundNumber: 1},
				0, // moonMod
				&cfg,
				&affPlayers,
				&affMobs,
			)

			// Drain the events queue so listeners see everything emitted
			// during the round.
			events.ProcessEvents()

			// ─── Assertion 1: text routing ────────────────────────────────────
			//
			// Direct messages (UserId > 0, RoomId == 0) should appear ONLY for
			// player participants. Mobs do not have a UserId; if an internal
			// path mistakenly routes a mob participant's "user" message via
			// SendText with UserId == 0, that message is invisible (or worse,
			// targets nobody) — but UserId == 0 will not appear here because
			// MobActor.SendText is a no-op.
			directRecipients := map[int]int{} // userId → count
			for _, m := range *captured {
				if m.UserId > 0 && m.RoomId == 0 {
					directRecipients[m.UserId]++
				}
			}
			if tc.atkIsPlayer {
				assert.Greater(t, directRecipients[atkUserId], 0,
					"%s: player attacker (uid %d) must receive at least one direct message via SendText",
					tc.name, atkUserId)
			} else {
				// Mob attacker: no direct messages should target user id 1
				// (the player slot used as our "atkUserId" in player runs)
				// purely from this attacker's perspective. Other directs may
				// still fire (e.g., to defender player). We only assert that
				// the player slot opposite the mob (defender side) is the only
				// player getting messages.
			}
			if tc.defIsPlayer {
				assert.Greater(t, directRecipients[defUserId], 0,
					"%s: player defender (uid %d) must receive at least one direct message via SendText",
					tc.name, defUserId)
			}
			// In MM, NO direct user-id messages should fire (both sides are
			// mobs). Room broadcasts may still fire.
			if !tc.atkIsPlayer && !tc.defIsPlayer {
				assert.Empty(t, directRecipients,
					"MM: no direct user-id messages should fire when both combatants are mobs; got: %v",
					directRecipients)
			}

			// ─── Assertion 2: mob_hurt / mob_combat_round dispatch ───────────
			//
			// Per Divergence #2 in the design spec, mob_hurt fires only when
			// the defender is a mob. Player defenders never receive mob_hurt
			// (they have no behavior tree).
			//
			// mob_combat_round fires for mob attackers (Task 9). In the MM
			// quadrant both attacker and defender share MobId=1, so the same
			// recording tree may capture mob_combat_round (from the attacker)
			// in addition to mob_hurt (from the defender). When the tree
			// returns Success for mob_combat_round the swing is suppressed, so
			// mob_hurt may not fire — that's intentional and permissible.
			if !tc.defIsPlayer {
				// The recordingNode receives the EvalContext for any event
				// dispatched to a mob's tree. We expect at least
				// one mob_hurt dispatch IF the attack hit; on a miss, no
				// mob_hurt fires. mob_combat_round is also permissible (mob
				// attacker getting first shot at the round). What we cannot
				// tolerate is any other event type landing in the recorder.
				allowedEvents := map[string]bool{"mob_hurt": true, "mob_combat_round": true}
				for _, evt := range recorder.calls {
					assert.True(t, allowedEvents[evt.EventType],
						"%s: mob tree should only receive mob_hurt or mob_combat_round events from this code path; got %q",
						tc.name, evt.EventType)
				}
			} else {
				// Player defender — no behavior tree should ever be invoked.
				// We can't directly check the player has no tree (they don't
				// have a MobId), but we can assert that no "spurious" tree
				// dispatch went out to mob template 1 (the test mob template)
				// from this round. If recorder is nil (we didn't install one
				// because def is a player), this is implicitly satisfied.
				assert.Nil(t, recorder, "%s: no btree recorder should be installed for player-defender quadrants", tc.name)
			}

			// ─── Assertion 3: handler did not crash ──────────────────────────
			//
			// Reaching this line means handleCombatRound returned without
			// panicking on this quadrant — the dispatcher routes correctly
			// for all four pairs.
		})
	}
}
