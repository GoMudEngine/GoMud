package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKnowledge_ForgetMultiWordMob is the Stage 2 fix target: admin commands
// mis-parse multi-word mob template names (positional strings.Fields). Before
// the fix, "forget bank clerk <player>" resolves mob="bank" (Unknown) and does
// NOT forget; after the fix the greedy split resolves "bank clerk" and forgets.
// Observable via knowledge.Get (SendText is not capturable in-test).
func TestKnowledge_ForgetMultiWordMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Seed a two-word mob template alongside the defaults.
	cleanMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{
			1: {MobId: 1, Zone: "TestZone", Character: characters.Character{Name: "Skeleton"}},
			3: {MobId: 3, Zone: "TestZone", Character: characters.Character{Name: "Bank Clerk"}},
		},
		map[int]*mobs.Mob{},
	)
	defer cleanMobs()

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)
	require.NotNil(t, target)
	subj := knowledge.PlayerSubject(target.UserId)

	// Seed a knowledge record: mob 3 ("Bank Clerk") has met the player.
	knowledge.RecordMet(3, subj, room.RoomId, knowledge.SourceWitnessed)
	require.NotNil(t, knowledge.Get(3, subj), "record should exist before forget")

	// "forget bank clerk <player>" must resolve the multi-word mob and forget.
	_, err := Knowledge("forget bank clerk "+target.Character.Name, admin, room, 0)
	require.NoError(t, err)

	assert.Nil(t, knowledge.Get(3, subj), "record should be forgotten (multi-word mob resolved)")
}
