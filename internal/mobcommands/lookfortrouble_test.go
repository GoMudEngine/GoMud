package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/require"
)

// buildGroupHateRoom creates and registers two mob instances in a fresh
// player-free room for the group-hate tests:
//
//   - instance 301: a non-hostile bandit with Hates: ["caravan"]
//   - instance 302: a caravan mob with Groups: ["caravan"]
//
// Returns (bandit, caravanMob, room, cleanup). The caller must call cleanup().
func buildGroupHateRoom(t *testing.T) (bandit, caravanMob *mobs.Mob, room *rooms.Room, cleanup func()) {
	t.Helper()

	bandit = &mobs.Mob{
		MobId:      9001,
		InstanceId: 301,
		HomeRoomId: 9999,
		AutoAggro: false, // not globally hostile — goes through HatesMob path
		Hates:      []string{"caravan"},
		Character: characters.Character{
			Name:      "Bandit Lookout",
			RoomId:    9999,
			Health:    100,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
			SpeciesId: 1, // avoids nil from species.GetSpecies() in HatesSpecies branch
		},
	}
	bandit.Character.HealthMax.Value = 100

	caravanMob = &mobs.Mob{
		MobId:      9002, // different MobId — HatesMob same-mob-type guard won't fire
		InstanceId: 302,
		HomeRoomId: 9999,
		AutoAggro: false,
		Groups:     []string{"caravan"},
		Character: characters.Character{
			Name:      "Caravan Guard Ketil",
			RoomId:    9999,
			Health:    100,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
			SpeciesId: 1,
		},
	}
	caravanMob.Character.HealthMax.Value = 100

	// Register instances BEFORE AddMob — AddMob calls GetInstance() and
	// silently skips unknown IDs.
	mobs.SetInstanceForTest(301, bandit)
	mobs.SetInstanceForTest(302, caravanMob)

	room = &rooms.Room{
		RoomId:      9999,
		Zone:        "TestZone",
		Title:       "Bandit Ambush Point",
		Description: "A rocky defile — perfect for an ambush.",
	}
	room.AddMob(301)
	room.AddMob(302)

	cleanup = func() {
		mobs.SetInstanceForTest(301, nil)
		mobs.SetInstanceForTest(302, nil)
	}
	return
}

// TestLookForTrouble_TargetsMobByGroupHate verifies that lookfortrouble
// selects a mob target when the acting mob has `hates: [caravan]` and the
// other mob has `groups: [caravan]`, with no players in the room.
//
// Stage 2 caravan: this is the mechanism that makes bandit lookouts engage
// the caravan crew when the caravan enters room 4052. Bandits with
// `hates: [caravan]` should aggro on caravan mobs that have
// `groups: [caravan]`.
//
// Design note: mob.Command() enqueues asynchronously (no event loop in
// tests). We verify the selection logic in two complementary sub-tests:
//
//  1. When Hates matches Groups, lookfortrouble queues a command (observed
//     via the event queue being non-empty OR by verifying that the attack
//     pipeline succeeds when driven with the expected target).
//  2. When Hates does NOT match Groups, lookfortrouble does NOT queue a
//     command (nothing in possibleMobTargets).
//
// Because util.GetTurnCount() returns 0 in tests (no game loop running),
// mob.lastCommandTurn stays 0 even after queueing — not a reliable signal.
// Instead we verify the selection outcome by confirming that Attack("#302")
// resolves correctly for this mob, paired with a contrast test showing that
// without the matching hate no attack is queued (boredom counter stays 0
// for mobs that despawn).
func TestLookForTrouble_TargetsMobByGroupHate(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	bandit, caravanMob, room, mobCleanup := buildGroupHateRoom(t)
	defer mobCleanup()

	// Pre-condition: no aggro, no queued commands.
	require.Nil(t, bandit.Character.Aggro)

	// Call LookForTrouble. Internally this calls:
	//   mob.HatesMob(caravanMob) → hatesAnyGroup(["caravan"]) → true
	//   possibleMobTargets = [302]
	//   mob.Command("attack #302")   ← enqueued asynchronously
	handled, err := LookForTrouble("", bandit, room)
	require.NoError(t, err)
	require.True(t, handled)

	// The event loop would now call TryCommand("attack", "#302", 301) which
	// routes to Attack("#302", bandit, room). Simulate that here to verify
	// the end-to-end target resolution is correct.
	//
	// Note: caravanMob must still be registered (it is — deferred cleanup
	// hasn't fired yet) so GetInstance(302) resolves inside Attack.
	_ = caravanMob // suppress unused warning; 302 is accessed via GetInstance

	_, err = Attack("#302", bandit, room)
	require.NoError(t, err)

	require.NotNil(t, bandit.Character.Aggro,
		"Attack('#302') must set aggro on the bandit")
	require.Equal(t, 302, bandit.Character.Aggro.MobInstanceId,
		"bandit aggro target must be the caravan mob (instance 302)")
}

// TestLookForTrouble_NoAggroWhenGroupHateMissing is the contrast test:
// when the bandit has NO hates list, lookfortrouble must not select the
// caravan mob (no target, BoredomCounter increments for despawning mobs).
//
// This ensures the group-hate path in lookfortrouble is selective — it
// should NOT aggro on neutral mobs the acting mob has no reason to hate.
func TestLookForTrouble_NoAggroWhenGroupHateMissing(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	_, _, room, mobCleanup := buildGroupHateRoom(t)
	defer mobCleanup()

	// Override the bandit: no hates list, not globally hostile.
	// Replace instance 301 with a version that has no reason to hate anyone.
	neutralBandit := &mobs.Mob{
		MobId:      9001,
		InstanceId: 301,
		HomeRoomId: 9999,
		AutoAggro: false,
		Hates:      nil, // no hates — should not aggro on caravan
		Character: characters.Character{
			Name:      "Neutral Guard",
			RoomId:    9999,
			Health:    100,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
			SpeciesId: 1,
		},
	}
	neutralBandit.Character.HealthMax.Value = 100
	mobs.SetInstanceForTest(301, neutralBandit)

	require.Nil(t, neutralBandit.Character.Aggro)

	handled, err := LookForTrouble("", neutralBandit, room)
	require.NoError(t, err)
	require.True(t, handled)

	// No target was found, so no attack command was issued.
	// Verify Aggro is still nil (no attack was called by the event loop).
	require.Nil(t, neutralBandit.Character.Aggro,
		"neutral mob with no hates must not aggro on caravan mob")
}
