package parser

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

// seedParserTest seeds a minimal world (items, one mob, a room with a player)
// and returns a Scope plus a cleanup func. Room 1 contains: the player Aliceia
// (user 1) and a Skeleton mob (instance 100). The caller adds whatever else it
// needs to the returned room's Items/Nouns/Containers/Corpses.
func seedParserTest(t *testing.T) (Scope, func()) {
	t.Helper()

	cleanKw := keywords.SeedKeywordsForTest()

	cleanItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		10001: {ItemId: 10001, Name: "Iron Sword", NameSimple: "Sword", Type: items.Weapon, Value: 100},
		40100: {ItemId: 40100, Name: "lake iron nodule", NameSimple: "nodule", Type: items.Object, Value: 5},
		30001: {ItemId: 30001, Name: "Healing Potion", Type: items.Potion, Uses: 3, Value: 25},
	})

	cleanMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{1: {MobId: 1, Zone: "TestZone", Character: characters.Character{Name: "Skeleton"}}},
		map[int]*mobs.Mob{100: {MobId: 1, InstanceId: 100, HomeRoomId: 1,
			Character: characters.Character{Name: "Skeleton", RoomId: 1, Buffs: buffs.New(), Cooldowns: map[string]int{}}}},
	)

	u := users.NewTestUser(1, "alice", "Aliceia", 1001)
	u.Character.RoomId = 1
	cleanUsers := users.SeedUsersForTest(map[int]*users.UserRecord{1: u})

	room := &rooms.Room{
		RoomId: 1, Zone: "TestZone", Title: "Test Room", Biome: "default",
		Exits: map[string]exit.RoomExit{"north": {RoomId: 2}},
	}
	cleanRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{1: room},
		map[string]*rooms.ZoneConfig{"TestZone": {Name: "TestZone", RoomId: 1, RoomIds: map[int]struct{}{1: {}}}},
	)
	room.AddPlayer(1)
	room.AddMob(100)

	cleanBiomes := rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"default": {BiomeId: "default", Name: "Default", Symbol: ".", LitArea: true, MovementCost: 1.0},
	})

	scope := Scope{User: u, Room: room}
	cleanup := func() {
		cleanBiomes()
		cleanRooms()
		cleanUsers()
		cleanMobs()
		cleanItems()
		cleanKw()
	}
	return scope, cleanup
}

// exactAdapter returns a Match only when the candidate equals want.
func exactAdapter(want string, kind Kind) adapter {
	return func(_ Scope, candidate string) (Match, bool) {
		if candidate == want {
			return Match{Kind: kind, Name: candidate}, true
		}
		return Match{}, false
	}
}

func TestResolveWith_LongestSpanWins(t *testing.T) {
	// Both a 2-word and a 1-word adapter can match; the 2-word span must win.
	adapters := []adapter{
		exactAdapter("hare paths", KindNoun),
		exactAdapter("hare", KindNoun),
	}
	m, ok := resolveWith([]string{"hare", "paths"}, Scope{}, adapters)
	require.True(t, ok)
	assert.Equal(t, "hare paths", m.Name)
}

func TestResolveWith_FallsBackToShorterSpan(t *testing.T) {
	adapters := []adapter{exactAdapter("hare", KindNoun)}
	m, ok := resolveWith([]string{"hare", "zzz"}, Scope{}, adapters)
	require.True(t, ok)
	assert.Equal(t, "hare", m.Name) // "hare zzz" misses, "hare" hits
}

func TestResolveWith_KindPriorityBreaksTies(t *testing.T) {
	// Two adapters match the SAME span; the first in order wins.
	adapters := []adapter{
		exactAdapter("bank clerk", KindMob),
		exactAdapter("bank clerk", KindNoun),
	}
	m, ok := resolveWith([]string{"bank", "clerk"}, Scope{}, adapters)
	require.True(t, ok)
	assert.Equal(t, KindMob, m.Kind)
}

func TestResolveWith_NoMatch(t *testing.T) {
	adapters := []adapter{exactAdapter("nope", KindNoun)}
	_, ok := resolveWith([]string{"totally", "absent"}, Scope{}, adapters)
	assert.False(t, ok)
}
