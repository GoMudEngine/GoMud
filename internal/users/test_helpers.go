package users

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
)

// SeedUsersForTest replaces the global userManager with a fresh instance
// populated from the supplied map and returns a cleanup function.
// Intended for cross-package integration tests (hooks, commands).
func SeedUsersForTest(testUsers map[int]*UserRecord) func() {
	origManager := userManager

	mgr := newUserManager()

	for _, u := range testUsers {
		mgr.Users[u.UserId] = u
		mgr.Usernames[u.Username] = u.UserId
		connId := u.ConnectionId()
		if connId > 0 {
			mgr.Connections[connId] = u.UserId
			mgr.UserConnections[u.UserId] = connId
		}
		if u.isZombie {
			mgr.ZombieConnections[connId] = 100
		}
	}

	userManager = mgr

	return func() {
		userManager = origManager
	}
}

// NewTestUser creates a UserRecord suitable for testing. The character will
// have basic defaults (name, health, stamina, conviction pools set).
func NewTestUser(userId int, username string, charName string, connId uint64) *UserRecord {
	ch := &characters.Character{
		Name:      charName,
		RoomId:    1,
		Health:    100,
		Stamina:   100,
		Buffs:     buffs.New(),
		Cooldowns: map[string]int{},
		Awareness: awareness.NewMachine(),
	}
	ch.HealthMax.Value = 100
	ch.StaminaMax.Value = 100
	ch.ConvictionMax.Value = 50
	ch.Conviction = 50
	ch.ActionPointsMax.Value = 10
	ch.ActionPoints = 5
	ch.Stats.Strength.ValueAdj = 100
	ch.Stats.Dexterity.ValueAdj = 100
	ch.Stats.Perception.ValueAdj = 100
	ch.Stats.Vitality.ValueAdj = 100
	ch.Stats.Willpower.ValueAdj = 100
	ch.Stats.Charisma.ValueAdj = 100

	return &UserRecord{
		UserId:       userId,
		Username:     username,
		Role:         RoleUser,
		Character:    ch,
		connectionId: connId,
	}
}
