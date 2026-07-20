package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegression_MeleeSpecialsRefuseCharmedTargets locks the fix for an
// inconsistency surfaced while extracting actions.AcquireMeleeTarget.
//
// `attack` has always refused charmed mobs ("%s is someone's companion!") and
// `taunt` refused them too, but bash, drain, gore, grapple, kick, maul, pounce,
// rake, throttle and trip did not. A player could therefore not attack another
// player's companion, but could freely bash, kick, grapple or trip it — the
// companion protection was bypassable by using any special move instead of the
// plain attack verb.
//
// The gate now lives in the shared preamble, so every melee special enforces it.
// The check is deliberately player-side only: mobs SHOULD be able to attack
// player companions, and mobcommands does not route through this helper.
func TestRegression_MeleeSpecialsRefuseCharmedTargets(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	// A companion belonging to some other player.
	const companionInstanceId = 900
	newCompanion := func() {
		companion := &mobs.Mob{
			MobId:      1,
			InstanceId: companionInstanceId,
			HomeRoomId: room.RoomId,
			Character: characters.Character{
				Name:      "Loyal Hound",
				RoomId:    room.RoomId,
				Health:    100,
				Buffs:     buffs.New(),
				Cooldowns: map[string]int{},
			},
		}
		companion.Character.HealthMax.Value = 100
		companion.Character.Charm(999, 100, ``) // charmed by another player
		mobs.SetInstanceForTest(companionInstanceId, companion)
		room.AddMob(companionInstanceId)

		require.True(t, companion.Character.IsCharmed(),
			"precondition: the target must actually be charmed")
	}
	dropCompanion := func() {
		room.RemoveMob(companionInstanceId)
		mobs.SetInstanceForTest(companionInstanceId, nil)
	}

	type cmd struct {
		name string
		call func() (bool, error)
	}

	cases := []cmd{
		{"bash", func() (bool, error) { return Bash("hound", user, room, 0) }},
		{"drain", func() (bool, error) { return Drain("hound", user, room, 0) }},
		{"gore", func() (bool, error) { return Gore("hound", user, room, 0) }},
		{"grapple", func() (bool, error) { return Grapple("hound", user, room, 0) }},
		{"kick", func() (bool, error) { return Kick("hound", user, room, 0) }},
		{"maul", func() (bool, error) { return Maul("hound", user, room, 0) }},
		{"pounce", func() (bool, error) { return Pounce("hound", user, room, 0) }},
		{"rake", func() (bool, error) { return Rake("hound", user, room, 0) }},
		{"taunt", func() (bool, error) { return Taunt("hound", user, room, 0) }},
		{"throttle", func() (bool, error) { return Throttle("hound", user, room, 0) }},
		{"trip", func() (bool, error) { return Trip("hound", user, room, 0) }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			newCompanion()
			defer dropCompanion()

			user.Character.Aggro = nil

			handled, err := c.call()
			assert.True(t, handled)
			assert.NoError(t, err)

			assert.Nil(t, user.Character.Aggro,
				"%s must not engage a charmed mob — a player's companion is not a valid target", c.name)
		})
	}
}
