package playtestprofiles

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/users"
)

// PersistOfflineUser assigns a user id, flags AI, seeds character userId,
// saves to disk, and updates the user index. It must not call CreateUser.
func PersistOfflineUser(u *users.UserRecord) error {
	if u == nil || u.Character == nil {
		return fmt.Errorf("playtestprofiles: persist requires user+character")
	}
	if u.UserId == 0 {
		u.UserId = users.GetUniqueUserId()
	}
	u.IsAI = true
	if u.Joined.IsZero() {
		u.Joined = time.Now()
	}
	u.Character.SetUserId(u.UserId)
	if u.Character.Name == "" {
		u.Character.Name = u.Username
	}

	if err := users.SaveUser(u); err != nil {
		return fmt.Errorf("playtestprofiles: save user: %w", err)
	}

	idx := users.NewUserIndex()
	if !idx.Exists() {
		if err := idx.Create(); err != nil {
			return fmt.Errorf("playtestprofiles: index create: %w", err)
		}
	}
	if err := idx.AddUser(u.UserId, u.Username); err != nil {
		return fmt.Errorf("playtestprofiles: index add: %w", err)
	}
	return nil
}
