package users

import (
	"os"
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"

	"gopkg.in/yaml.v2"
)

type userEntry struct {
	UserId       int                      `json:"user_id"`
	ConnectionId connections.ConnectionId `json:"connection_id"`
	IsZombie     bool                     `json:"is_zombie"`
	ZombieTurn   uint64                   `json:"zombie_turn,omitempty"`
}

type usersState struct {
	Entries []userEntry `json:"entries"`
}

type usersCopyoverContributor struct{}

func (u *usersCopyoverContributor) CopyoverName() string {
	return "users"
}

func (u *usersCopyoverContributor) CopyoverSave(enc *copyover.Encoder) error {
	state := usersState{}

	for userId, user := range userManager.Users {
		connId := userManager.UserConnections[userId]
		isZombie := false
		zombieTurn := uint64(0)
		if turn, ok := userManager.ZombieConnections[connId]; ok {
			isZombie = true
			zombieTurn = turn
		}
		state.Entries = append(state.Entries, userEntry{
			UserId:       userId,
			ConnectionId: connId,
			IsZombie:     isZombie,
			ZombieTurn:   zombieTurn,
		})
		_ = user
	}

	return enc.WriteSection(u.CopyoverName(), state)
}

func (u *usersCopyoverContributor) CopyoverRestore(dec *copyover.Decoder) error {
	var state usersState
	if err := dec.ReadSection(u.CopyoverName(), &state); err != nil {
		return err
	}

	for _, entry := range state.Entries {
		user, err := loadUserById(entry.UserId)
		if err != nil {
			mudlog.Error("copyover: restore user", "userId", entry.UserId, "error", err)
			continue
		}

		user.connectionId = entry.ConnectionId
		user.connectionTime = time.Now()
		user.SetLastInputRound(util.GetRoundCount())

		userManager.Users[user.UserId] = user
		userManager.Usernames[user.Username] = user.UserId
		userManager.Connections[user.connectionId] = user.UserId
		userManager.UserConnections[user.UserId] = user.connectionId

		if entry.IsZombie {
			userManager.ZombieConnections[user.connectionId] = entry.ZombieTurn
		}
	}

	return nil
}

// loadUserById loads a user record directly from disk by userId.
func loadUserById(userId int) (*UserRecord, error) {
	userFilePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, strconv.Itoa(userId)+`.yaml`)

	data, err := os.ReadFile(userFilePath)
	if err != nil {
		return nil, err
	}

	user := &UserRecord{}
	if err := yaml.Unmarshal(data, user); err != nil {
		return nil, err
	}

	user.Character.Validate()

	return user, nil
}

// CopyoverContributor returns the users contributor for registration.
func CopyoverContributor() copyover.Contributor {
	return &usersCopyoverContributor{}
}
