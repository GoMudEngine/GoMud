package users

import (
	"os"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestMain(m *testing.M) {
	// Initialize logger to avoid nil pointer panics in functions that log
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

// seedRegistry replaces the global userManager singleton with test data.
// Call the returned cleanup function (or use defer) to restore the original.
func seedRegistry() func() {
	origManager := userManager

	mgr := newUserManager()

	// User 1: normal active user
	u1 := &UserRecord{
		UserId:   1,
		Username: "alice",
		Role:     RoleUser,
		Character: &characters.Character{
			Name: "Aliceia",
		},
		connectionId: 1001,
	}

	// User 2: another active user
	u2 := &UserRecord{
		UserId:   2,
		Username: "bob",
		Role:     RoleUser,
		Character: &characters.Character{
			Name: "Bobrick",
		},
		connectionId: 1002,
	}

	// User 3: zombie user
	u3 := &UserRecord{
		UserId:   3,
		Username: "charlie",
		Role:     RoleUser,
		Character: &characters.Character{
			Name: "Charlton",
		},
		connectionId: 1003,
		isZombie:     true,
	}

	// Populate all maps
	mgr.Users[1] = u1
	mgr.Users[2] = u2
	mgr.Users[3] = u3

	mgr.Usernames["alice"] = 1
	mgr.Usernames["bob"] = 2
	mgr.Usernames["charlie"] = 3

	mgr.Connections[1001] = 1
	mgr.Connections[1002] = 2
	mgr.Connections[1003] = 3

	mgr.UserConnections[1] = 1001
	mgr.UserConnections[2] = 1002
	mgr.UserConnections[3] = 1003

	mgr.ZombieConnections[1003] = 100 // zombie since turn 100

	userManager = mgr

	return func() {
		userManager = origManager
	}
}

// ─── GetByUserId ────────────────────────────────────────────────────────────

func TestGetByUserId(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("found", func(t *testing.T) {
		u := GetByUserId(1)
		assert.NotNil(t, u)
		assert.Equal(t, "alice", u.Username)
	})

	t.Run("not found", func(t *testing.T) {
		u := GetByUserId(999)
		assert.Nil(t, u)
	})
}

// ─── GetByCharacterName ────────────────────────────────────────────────────

func TestGetByCharacterName(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("exact match", func(t *testing.T) {
		u := GetByCharacterName("Aliceia")
		assert.NotNil(t, u)
		assert.Equal(t, 1, u.UserId)
	})

	t.Run("case insensitive", func(t *testing.T) {
		u := GetByCharacterName("aliceia")
		assert.NotNil(t, u)
		assert.Equal(t, 1, u.UserId)
	})

	t.Run("prefix match", func(t *testing.T) {
		u := GetByCharacterName("Bob")
		assert.NotNil(t, u)
		assert.Equal(t, 2, u.UserId)
	})

	t.Run("no match", func(t *testing.T) {
		u := GetByCharacterName("Zzzzz")
		assert.Nil(t, u)
	})
}

// ─── GetByConnectionId ─────────────────────────────────────────────────────

func TestGetByConnectionId(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("found", func(t *testing.T) {
		u := GetByConnectionId(1001)
		assert.NotNil(t, u)
		assert.Equal(t, "alice", u.Username)
	})

	t.Run("not found", func(t *testing.T) {
		u := GetByConnectionId(9999)
		assert.Nil(t, u)
	})
}

// ─── GetAllActiveUsers ─────────────────────────────────────────────────────

func TestGetAllActiveUsers(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	active := GetAllActiveUsers()
	// Should return 2 (alice and bob), not charlie (zombie)
	assert.Len(t, active, 2)

	names := []string{}
	for _, u := range active {
		names = append(names, u.Username)
	}
	assert.Contains(t, names, "alice")
	assert.Contains(t, names, "bob")
	assert.NotContains(t, names, "charlie")
}

// ─── GetOnlineUserIds ──────────────────────────────────────────────────────

func TestGetOnlineUserIds(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	ids := GetOnlineUserIds()
	assert.Len(t, ids, 3) // all users including zombies
	assert.Contains(t, ids, 1)
	assert.Contains(t, ids, 2)
	assert.Contains(t, ids, 3)
}

// ─── GetConnectionId / GetConnectionIds ────────────────────────────────────

func TestGetConnectionId(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("found", func(t *testing.T) {
		connId := GetConnectionId(1)
		assert.Equal(t, connections.ConnectionId(1001), connId)
	})

	t.Run("not found returns 0", func(t *testing.T) {
		connId := GetConnectionId(999)
		assert.Equal(t, connections.ConnectionId(0), connId)
	})
}

func TestGetConnectionIds(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("multiple users", func(t *testing.T) {
		ids := GetConnectionIds([]int{1, 2})
		assert.Len(t, ids, 2)
		assert.Contains(t, ids, connections.ConnectionId(1001))
		assert.Contains(t, ids, connections.ConnectionId(1002))
	})

	t.Run("includes unknown user", func(t *testing.T) {
		ids := GetConnectionIds([]int{1, 999})
		assert.Len(t, ids, 1) // only user 1 found
	})

	t.Run("empty input", func(t *testing.T) {
		ids := GetConnectionIds([]int{})
		assert.Len(t, ids, 0)
	})
}

// ─── Zombie Management ────────────────────────────────────────────────────

func TestIsZombieConnection(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("zombie connection", func(t *testing.T) {
		assert.True(t, IsZombieConnection(1003))
	})

	t.Run("non-zombie connection", func(t *testing.T) {
		assert.False(t, IsZombieConnection(1001))
	})

	t.Run("unknown connection", func(t *testing.T) {
		assert.False(t, IsZombieConnection(9999))
	})
}

func TestRemoveZombieConnection(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	assert.True(t, IsZombieConnection(1003))
	RemoveZombieConnection(1003)
	assert.False(t, IsZombieConnection(1003))

	// Removing nonexistent is safe
	RemoveZombieConnection(9999)
}

func TestGetExpiredZombies(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("no expired before turn", func(t *testing.T) {
		expired := GetExpiredZombies(50) // zombie at turn 100 > 50
		assert.Len(t, expired, 0)
	})

	t.Run("expired after turn", func(t *testing.T) {
		expired := GetExpiredZombies(200) // zombie at turn 100 < 200
		assert.Len(t, expired, 1)
		assert.Contains(t, expired, 3) // charlie
	})

	t.Run("exact turn not expired", func(t *testing.T) {
		expired := GetExpiredZombies(100) // zombie at turn 100 == 100, not < 100
		assert.Len(t, expired, 0)
	})
}

func TestSetZombieUser(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// User 1 is not a zombie
	assert.False(t, IsZombieConnection(1001))

	SetZombieUser(1)

	// Now user 1's connection should be in zombie connections
	assert.True(t, IsZombieConnection(1001))
}

func TestRemoveZombieUser(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Charlie is zombie
	assert.True(t, IsZombieConnection(1003))

	RemoveZombieUser(3)

	assert.False(t, IsZombieConnection(1003))
}

// ─── LogOutUserByConnectionId ──────────────────────────────────────────────

func TestLogOutUserByConnectionId(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("successful logout", func(t *testing.T) {
		// Verify alice exists first
		assert.NotNil(t, GetByConnectionId(1001))

		err := LogOutUserByConnectionId(1001)
		assert.NoError(t, err)

		// Should be removed from all maps
		assert.Nil(t, GetByUserId(1))
		assert.Nil(t, GetByConnectionId(1001))
	})

	t.Run("unknown connection returns error", func(t *testing.T) {
		err := LogOutUserByConnectionId(9999)
		assert.Error(t, err)
	})
}

// ─── LoginUser ─────────────────────────────────────────────────────────────

func TestLoginUser(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	t.Run("fresh login", func(t *testing.T) {
		newUser := &UserRecord{
			UserId:   10,
			Username: "dave",
			Role:     RoleUser,
			Character: &characters.Character{
				Name: "Daveth",
			},
		}

		result, msg, err := LoginUser(newUser, 2000)
		assert.NoError(t, err)
		assert.Equal(t, "", msg)
		assert.NotNil(t, result)
		assert.Equal(t, "dave", result.Username)

		// Should be in all maps now
		assert.NotNil(t, GetByUserId(10))
		assert.NotNil(t, GetByConnectionId(2000))
	})

	t.Run("double login returns error", func(t *testing.T) {
		// alice is already logged in
		dupeUser := &UserRecord{
			UserId:   1,
			Username: "alice",
			Role:     RoleUser,
			Character: &characters.Character{
				Name: "Aliceia",
			},
		}

		result, msg, err := LoginUser(dupeUser, 3000)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, msg, "already logged in")
	})

	t.Run("zombie reconnect", func(t *testing.T) {
		// Charlie is a zombie
		reconnectUser := &UserRecord{
			UserId:   3,
			Username: "charlie",
			Role:     RoleUser,
			Character: &characters.Character{
				Name: "Charlton",
			},
		}

		result, msg, err := LoginUser(reconnectUser, 4000)
		assert.NoError(t, err)
		assert.Contains(t, msg, "Reconnecting")
		assert.NotNil(t, result)
		assert.Equal(t, "charlie", result.Username)

		// Old zombie connection should be cleaned up
		assert.False(t, IsZombieConnection(1003))

		// New connection should be active
		assert.NotNil(t, GetByConnectionId(4000))
	})
}

// ─── UserRecord Methods ────────────────────────────────────────────────────

func TestUserRecord_TempData(t *testing.T) {
	u := &UserRecord{}

	t.Run("get from nil store returns nil", func(t *testing.T) {
		assert.Nil(t, u.GetTempData("key"))
	})

	t.Run("set and get", func(t *testing.T) {
		u.SetTempData("key", "value")
		assert.Equal(t, "value", u.GetTempData("key"))
	})

	t.Run("missing key returns nil", func(t *testing.T) {
		assert.Nil(t, u.GetTempData("missing"))
	})

	t.Run("delete by nil", func(t *testing.T) {
		u.SetTempData("key", nil)
		assert.Nil(t, u.GetTempData("key"))
	})

	t.Run("multiple keys", func(t *testing.T) {
		u.SetTempData("a", 1)
		u.SetTempData("b", "two")
		assert.Equal(t, 1, u.GetTempData("a"))
		assert.Equal(t, "two", u.GetTempData("b"))
	})
}

func TestUserRecord_ConfigOption(t *testing.T) {
	u := &UserRecord{}

	t.Run("get from nil store returns nil", func(t *testing.T) {
		assert.Nil(t, u.GetConfigOption("key"))
	})

	t.Run("set and get", func(t *testing.T) {
		u.SetConfigOption("color", "red")
		assert.Equal(t, "red", u.GetConfigOption("color"))
	})

	t.Run("delete by nil", func(t *testing.T) {
		u.SetConfigOption("color", nil)
		assert.Nil(t, u.GetConfigOption("color"))
	})
}

func TestUserRecord_ShorthandId(t *testing.T) {
	u := &UserRecord{UserId: 42}
	assert.Equal(t, "@42", u.ShorthandId())
}

func TestUserRecord_LastInputRound(t *testing.T) {
	u := &UserRecord{}
	u.SetLastInputRound(100)
	assert.Equal(t, uint64(100), u.GetLastInputRound())
}

func TestUserRecord_HasShop(t *testing.T) {
	t.Run("no shop", func(t *testing.T) {
		u := &UserRecord{Character: &characters.Character{}}
		assert.False(t, u.HasShop())
	})

	t.Run("has shop", func(t *testing.T) {
		u := &UserRecord{Character: &characters.Character{
			Shop: characters.Shop{{ItemId: 1, Price: 10, Quantity: 5}},
		}}
		assert.True(t, u.HasShop())
	})
}

func TestUserRecord_InputBlocking(t *testing.T) {
	u := &UserRecord{}
	assert.False(t, u.InputBlocked())

	u.BlockInput()
	assert.True(t, u.InputBlocked())

	u.UnblockInput()
	assert.False(t, u.InputBlocked())
}

func TestUserRecord_UnsentText(t *testing.T) {
	u := &UserRecord{}

	u.SetUnsentText("hello", "world")
	unsent, suggest := u.GetUnsentText()
	assert.Equal(t, "hello", unsent)
	assert.Equal(t, "world", suggest)

	u.SetUnsentText("", "")
	unsent, suggest = u.GetUnsentText()
	assert.Equal(t, "", unsent)
	assert.Equal(t, "", suggest)
}

func TestUserRecord_ConnectionId(t *testing.T) {
	u := &UserRecord{connectionId: 12345}
	assert.Equal(t, uint64(12345), u.ConnectionId())
}

func TestUserRecord_GetConnectTime(t *testing.T) {
	u := &UserRecord{}
	// Default zero time
	assert.True(t, u.GetConnectTime().IsZero())
}

func TestUserRecord_RoundTick(t *testing.T) {
	u := &UserRecord{}
	// Should not panic (empty implementation)
	u.RoundTick()
}

func TestUserRecord_ReplaceCharacter(t *testing.T) {
	u := &UserRecord{Character: &characters.Character{Name: "Old"}}
	newChar := &characters.Character{Name: "New"}
	u.ReplaceCharacter(newChar)
	assert.Equal(t, "New", u.Character.Name)
}

func TestUserRecord_DidTip(t *testing.T) {
	u := &UserRecord{}

	t.Run("nil tips returns false", func(t *testing.T) {
		assert.False(t, u.DidTip("welcome"))
	})

	t.Run("set tip complete", func(t *testing.T) {
		u.DidTip("welcome", true)
		assert.True(t, u.DidTip("welcome"))
	})

	t.Run("unset tip", func(t *testing.T) {
		u.DidTip("welcome", false)
		assert.False(t, u.DidTip("welcome"))
	})
}

func TestUserRecord_HasRolePermission(t *testing.T) {
	t.Run("admin has all permissions", func(t *testing.T) {
		u := &UserRecord{
			Role:      RoleAdmin,
			Character: &characters.Character{},
		}
		assert.True(t, u.HasRolePermission("anything"))
	})

	t.Run("user has no custom permissions", func(t *testing.T) {
		u := &UserRecord{
			Role:      RoleUser,
			Character: &characters.Character{},
		}
		assert.False(t, u.HasRolePermission("room.info"))
	})
}

func TestUserRecord_TempName(t *testing.T) {
	u := &UserRecord{Username: "testuser"}
	name := u.TempName()
	assert.Contains(t, name, "nameless-")
	// Should be deterministic
	assert.Equal(t, name, u.TempName())
}

func TestUserRecord_PasswordMatches(t *testing.T) {
	// Set bcrypt hash directly (bypasses config-dependent SetPassword validation)
	bcryptHash, _ := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	u := &UserRecord{Password: string(bcryptHash)}

	t.Run("bcrypt match", func(t *testing.T) {
		assert.True(t, u.PasswordMatches("testpass123"))
	})

	t.Run("wrong password", func(t *testing.T) {
		assert.False(t, u.PasswordMatches("wrong"))
	})

	t.Run("plaintext no longer matches", func(t *testing.T) {
		u2 := &UserRecord{Password: "plaintext123"}
		assert.False(t, u2.PasswordMatches("plaintext123"))
	})

	t.Run("sha256 migration", func(t *testing.T) {
		// Simulate old SHA256-hashed password
		u3 := &UserRecord{Password: util.Hash("oldpass")}
		assert.True(t, u3.PasswordMatches("oldpass"))
		// After migration, password should be bcrypt
		assert.True(t, strings.HasPrefix(u3.Password, "$2a$"))
	})
}

func TestUserRecord_AddCommandAlias(t *testing.T) {
	u := &UserRecord{}

	t.Run("add alias", func(t *testing.T) {
		added, deleted := u.AddCommandAlias("n", "north")
		assert.Equal(t, "n", added)
		assert.Equal(t, "", deleted)
	})

	t.Run("alias keyword rejected", func(t *testing.T) {
		added, deleted := u.AddCommandAlias("alias", "something")
		assert.Equal(t, "", added)
		assert.Equal(t, "", deleted)
	})

	t.Run("delete alias with empty output", func(t *testing.T) {
		added, deleted := u.AddCommandAlias("n", "")
		assert.Equal(t, "", added)
		assert.Equal(t, "n", deleted)
	})

	t.Run("long input truncated", func(t *testing.T) {
		longInput := "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz"
		added, _ := u.AddCommandAlias(longInput, "short")
		assert.Len(t, added, 64)
	})
}

func TestUserRecord_TryCommandAlias(t *testing.T) {
	t.Run("nil aliases returns input", func(t *testing.T) {
		u := &UserRecord{}
		assert.Equal(t, "north", u.TryCommandAlias("north"))
	})

	t.Run("alias found", func(t *testing.T) {
		u := &UserRecord{Aliases: map[string]string{"n": "north"}}
		assert.Equal(t, "north", u.TryCommandAlias("n"))
	})

	t.Run("no match returns input", func(t *testing.T) {
		u := &UserRecord{Aliases: map[string]string{"n": "north"}}
		assert.Equal(t, "south", u.TryCommandAlias("south"))
	})
}

// ─── Storage ───────────────────────────────────────────────────────────────

func TestStorage_GetItems(t *testing.T) {
	s := &Storage{Items: []items.Item{{ItemId: 1}, {ItemId: 2}}}
	got := s.GetItems()
	assert.Len(t, got, 2)

	// Verify it returns a copy
	got[0].ItemId = 999
	assert.Equal(t, 1, s.Items[0].ItemId)
}

func TestStorage_AddItem(t *testing.T) {
	s := &Storage{}

	t.Run("add valid item", func(t *testing.T) {
		ok := s.AddItem(items.Item{ItemId: 1})
		assert.True(t, ok)
		assert.Len(t, s.Items, 1)
	})

	t.Run("reject zero item", func(t *testing.T) {
		ok := s.AddItem(items.Item{ItemId: 0})
		assert.False(t, ok)
	})

	t.Run("reject negative item", func(t *testing.T) {
		ok := s.AddItem(items.Item{ItemId: -1})
		assert.False(t, ok)
	})
}

func TestStorage_RemoveItem(t *testing.T) {
	uuid1 := [16]byte{1}
	s := &Storage{Items: []items.Item{{ItemId: 1, UUID: uuid1}}}

	t.Run("remove existing", func(t *testing.T) {
		ok := s.RemoveItem(items.Item{ItemId: 1, UUID: uuid1})
		assert.True(t, ok)
		assert.Len(t, s.Items, 0)
	})

	t.Run("remove nonexistent", func(t *testing.T) {
		ok := s.RemoveItem(items.Item{ItemId: 999})
		assert.False(t, ok)
	})
}

func TestStorage_FindItem(t *testing.T) {
	s := &Storage{}

	t.Run("empty name returns not found", func(t *testing.T) {
		_, found := s.FindItem("")
		assert.False(t, found)
	})

	t.Run("no items returns not found", func(t *testing.T) {
		_, found := s.FindItem("sword")
		assert.False(t, found)
	})
}

// ─── Inbox ─────────────────────────────────────────────────────────────────

func TestInbox_Add(t *testing.T) {
	inbox := Inbox{}

	inbox.Add(Message{FromName: "Alice", Message: "Hello"})
	assert.Len(t, inbox, 1)
	assert.Equal(t, "Alice", inbox[0].FromName)

	// Add another — should prepend
	inbox.Add(Message{FromName: "Bob", Message: "Hi"})
	assert.Len(t, inbox, 2)
	assert.Equal(t, "Bob", inbox[0].FromName)
	assert.Equal(t, "Alice", inbox[1].FromName)
}

func TestInbox_CountReadUnread(t *testing.T) {
	inbox := Inbox{
		{FromName: "A", Read: true},
		{FromName: "B", Read: false},
		{FromName: "C", Read: true},
	}

	assert.Equal(t, 2, inbox.CountRead())
	assert.Equal(t, 1, inbox.CountUnread())
}

func TestInbox_Empty(t *testing.T) {
	inbox := Inbox{
		{FromName: "A"},
		{FromName: "B"},
	}
	inbox.Empty()
	assert.Len(t, inbox, 0)
}

// ─── UserLog ───────────────────────────────────────────────────────────────

func TestUserLog_Add(t *testing.T) {
	log := UserLog{}
	log.Add("combat", "Hit the goblin")
	assert.Len(t, log, 1)
	assert.Equal(t, "combat", log[0].Category)
	assert.Equal(t, "Hit the goblin", log[0].What)
}

func TestUserLog_Items(t *testing.T) {
	log := UserLog{}
	log.Add("a", "first")
	log.Add("b", "second")

	var collected []string
	log.Items(func(e UserLogEntry) bool {
		collected = append(collected, e.What)
		return true
	})
	assert.Len(t, collected, 2)
}

func TestUserLog_ItemsEarlyExit(t *testing.T) {
	log := UserLog{}
	log.Add("a", "first")
	log.Add("b", "second")
	log.Add("c", "third")

	var collected []string
	log.Items(func(e UserLogEntry) bool {
		collected = append(collected, e.What)
		return len(collected) < 2 // stop after 2
	})
	assert.Len(t, collected, 2)
}

func TestUserLog_Growth(t *testing.T) {
	log := UserLog{}
	// Add enough entries to trigger capacity growth
	for i := 0; i < LogMinAllocation+5; i++ {
		log.Add("test", "entry")
	}
	assert.Equal(t, LogMinAllocation+5, len(log))
}

// ─── GetMemoryUsage ────────────────────────────────────────────────────────

func TestGetMemoryUsage(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mem := GetMemoryUsage()
	assert.Contains(t, mem, "Users")
	assert.Contains(t, mem, "Usernames")
	assert.Contains(t, mem, "Connections")
	assert.Contains(t, mem, "UserConnections")
	assert.Equal(t, 3, mem["Users"].Count)
}

// ─── Prompt ────────────────────────────────────────────────────────────────

func TestUserRecord_Prompt(t *testing.T) {
	u := &UserRecord{}

	t.Run("no active prompt", func(t *testing.T) {
		assert.Nil(t, u.GetPrompt())
	})

	t.Run("start prompt", func(t *testing.T) {
		p, isNew := u.StartPrompt("say", "hello")
		assert.True(t, isNew)
		assert.NotNil(t, p)
	})

	t.Run("same prompt returns existing", func(t *testing.T) {
		p, isNew := u.StartPrompt("say", "hello")
		assert.False(t, isNew)
		assert.NotNil(t, p)
	})

	t.Run("different prompt creates new", func(t *testing.T) {
		p, isNew := u.StartPrompt("tell", "world")
		assert.True(t, isNew)
		assert.NotNil(t, p)
	})

	t.Run("get prompt returns active", func(t *testing.T) {
		p := u.GetPrompt()
		assert.NotNil(t, p)
	})

	t.Run("clear prompt", func(t *testing.T) {
		u.ClearPrompt()
		assert.Nil(t, u.GetPrompt())
	})
}

// ─── SendText / AddBuff / SendWebClientCommand ────────────────────────────

func TestUserRecord_SendText(t *testing.T) {
	u := &UserRecord{UserId: 1}
	// Should not panic — adds event to queue
	u.SendText("Hello, world!")
}

func TestUserRecord_AddBuff(t *testing.T) {
	u := &UserRecord{UserId: 1}
	// Should not panic
	u.AddBuff(1, "test")
}

func TestUserRecord_SendWebClientCommand(t *testing.T) {
	u := &UserRecord{connectionId: 1001}
	// Should not panic
	u.SendWebClientCommand("refresh")
}

// ─── WimpyCheck ────────────────────────────────────────────────────────────

func TestUserRecord_WimpyCheck(t *testing.T) {
	t.Run("no wimpy set", func(t *testing.T) {
		u := &UserRecord{Character: &characters.Character{}}
		// Should not panic
		u.WimpyCheck()
	})
}

// ─── UserLog overflow ─────────────────────────────────────────────────────

func TestUserLog_Overflow(t *testing.T) {
	log := UserLog{}
	// Fill to max capacity
	for i := 0; i < LogMaxAllocation+10; i++ {
		log.Add("test", "entry")
	}
	// Should be capped at LogMaxAllocation
	assert.True(t, len(log) <= LogMaxAllocation)
}

// ─── SetZombieUser duplicate ──────────────────────────────────────────────

func TestSetZombieUserDuplicate(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Charlie is already zombie
	SetZombieUser(3) // should be a no-op (already in zombie connections)
	assert.True(t, IsZombieConnection(1003))
}

// ─── SetZombieUser nonexistent user ───────────────────────────────────────

func TestSetZombieUserNonexistent(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Should not panic for unknown userId
	SetZombieUser(999)
}

// ─── PasswordMatches additional paths ─────────────────────────────────────

func TestUserRecord_PasswordMatchesHashed(t *testing.T) {
	t.Run("hash-of-hash bypass blocked", func(t *testing.T) {
		// Old vulnerability: attacker with file access could compute
		// SHA256 of stored hash to log in. This must be blocked.
		u := &UserRecord{Password: util.Hash("somepassword")}
		hashOfHash := util.Hash(u.Password)
		assert.False(t, u.PasswordMatches(hashOfHash))
	})
}

// ─── OnlineInfo struct ────────────────────────────────────────────────────

func TestOnlineInfo(t *testing.T) {
	info := OnlineInfo{
		Username:      "alice",
		CharacterName: "Aliceia",
		Title:         "scrub warrior",
		OnlineTime:    3600,
		OnlineTimeStr: "1h0m",
		IsAFK:         false,
		IsAI:          true,
		Role:          "user",
	}
	assert.Equal(t, "alice", info.Username)
	assert.Equal(t, "Aliceia", info.CharacterName)
	assert.True(t, info.IsAI)
}

// ─── Roles ────────────────────────────────────────────────────────────────

func TestRoleConstants(t *testing.T) {
	assert.Equal(t, "guest", RoleGuest)
	assert.Equal(t, "user", RoleUser)
	assert.Equal(t, "admin", RoleAdmin)
}

// ─── renderVitalBar ───────────────────────────────────────────────────────

func TestRenderVitalBar(t *testing.T) {
	t.Run("full health", func(t *testing.T) {
		bar := renderVitalBar(100, 100, 0)
		assert.Contains(t, bar, "█")
		assert.Contains(t, bar, "82") // green
	})

	t.Run("half health", func(t *testing.T) {
		bar := renderVitalBar(50, 100, 0)
		assert.Contains(t, bar, "█")
		assert.Contains(t, bar, "226") // yellow
	})

	t.Run("low health", func(t *testing.T) {
		bar := renderVitalBar(10, 100, 0)
		assert.Contains(t, bar, "█")
		assert.Contains(t, bar, "196") // red
	})

	t.Run("zero health", func(t *testing.T) {
		bar := renderVitalBar(0, 100, 0)
		assert.Contains(t, bar, "░") // empty blocks
	})

	t.Run("with reservation", func(t *testing.T) {
		bar := renderVitalBar(50, 100, 20)
		assert.Contains(t, bar, "▓") // reserved blocks
	})

	t.Run("zero max defaults to 1", func(t *testing.T) {
		bar := renderVitalBar(0, 0, 0)
		assert.NotEmpty(t, bar)
	})

	t.Run("negative current clamped to 0", func(t *testing.T) {
		bar := renderVitalBar(-10, 100, 0)
		assert.NotEmpty(t, bar)
	})

	t.Run("negative reserved clamped to 0", func(t *testing.T) {
		bar := renderVitalBar(50, 100, -5)
		assert.NotEmpty(t, bar)
	})
}

// ─── targetHealthDesc ─────────────────────────────────────────────────────

func TestTargetHealthDesc(t *testing.T) {
	tests := []struct {
		name     string
		health   int
		max      int
		wantDesc string
		wantClass string
	}{
		{"healthy", 90, 100, "healthy", "health-100"},
		{"bruised", 65, 100, "bruised", "health-60"},
		{"wounded", 45, 100, "wounded", "health-40"},
		{"badly wounded", 25, 100, "badly wounded", "health-20"},
		{"near death", 5, 100, "near death", "health-10"},
		{"dead", 0, 100, "dead", "health-0"},
		{"negative health", -5, 100, "dead", "health-0"},
		{"zero max", 0, 0, "dead", "health-0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc, class := targetHealthDesc(tt.health, tt.max)
			assert.Equal(t, tt.wantDesc, desc)
			assert.Equal(t, tt.wantClass, class)
		})
	}
}

// ─── getPromptToggle ──────────────────────────────────────────────────────

func TestGetPromptToggle(t *testing.T) {
	t.Run("default returns true", func(t *testing.T) {
		u := &UserRecord{}
		assert.True(t, u.getPromptToggle("bars"))
	})

	t.Run("explicit true", func(t *testing.T) {
		u := &UserRecord{}
		u.SetConfigOption("fprompt-tog-bars", true)
		assert.True(t, u.getPromptToggle("bars"))
	})

	t.Run("explicit false", func(t *testing.T) {
		u := &UserRecord{}
		u.SetConfigOption("fprompt-tog-bars", false)
		assert.False(t, u.getPromptToggle("bars"))
	})

	t.Run("non-bool value returns true", func(t *testing.T) {
		u := &UserRecord{}
		u.SetConfigOption("fprompt-tog-bars", "yes")
		assert.True(t, u.getPromptToggle("bars"))
	})
}

// ─── BuildFightPromptTemplate ─────────────────────────────────────────────

func TestBuildFightPromptTemplate(t *testing.T) {
	u := &UserRecord{}

	t.Run("default template includes all toggles", func(t *testing.T) {
		tmpl := u.BuildFightPromptTemplate()
		assert.Contains(t, tmpl, "{t}")
		assert.Contains(t, tmpl, "{hpbar}")
		assert.Contains(t, tmpl, "{target}")
	})

	t.Run("disable bars", func(t *testing.T) {
		u.SetConfigOption("fprompt-tog-bars", false)
		tmpl := u.BuildFightPromptTemplate()
		assert.NotContains(t, tmpl, "{hpbar}")
		u.SetConfigOption("fprompt-tog-bars", nil) // reset
	})

	t.Run("disable target", func(t *testing.T) {
		u.SetConfigOption("fprompt-tog-target", false)
		tmpl := u.BuildFightPromptTemplate()
		assert.NotContains(t, tmpl, "» {target}")
		u.SetConfigOption("fprompt-tog-target", nil)
	})
}

// ─── SaveAllUsers ─────────────────────────────────────────────────────────

func TestSaveAllUsers(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	// Should not panic even though file paths don't exist
	SaveAllUsers(true)
}
