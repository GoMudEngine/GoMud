package users

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
	"github.com/GoMudEngine/GoMud/internal/util"

	//
	"gopkg.in/yaml.v2"
)

var (
	userManager *ActiveUsers = newUserManager()
)

type ActiveUsers struct {
	mu                sync.RWMutex                        // guards all five maps below
	Users             map[int]*UserRecord                 // userId to UserRecord
	Usernames         map[string]int                      // username to userId
	Connections       map[connections.ConnectionId]int    // connectionId to userId
	UserConnections   map[int]connections.ConnectionId    // userId to connectionId
	ZombieConnections map[connections.ConnectionId]uint64 // connectionId to turn they became a zombie
}

func newUserManager() *ActiveUsers {
	return &ActiveUsers{
		Users:             make(map[int]*UserRecord),
		Usernames:         make(map[string]int),
		Connections:       make(map[connections.ConnectionId]int),
		UserConnections:   make(map[int]connections.ConnectionId),
		ZombieConnections: make(map[connections.ConnectionId]uint64),
	}
}

// ── unguarded helpers (called only while mu is already held) ──────────────

func isZombieConnectionLocked(connectionId connections.ConnectionId) bool {
	_, ok := userManager.ZombieConnections[connectionId]
	return ok
}

func removeZombieConnectionLocked(connectionId connections.ConnectionId) {
	delete(userManager.ZombieConnections, connectionId)
}

// ── exported API ──────────────────────────────────────────────────────────

func RemoveZombieUser(userId int) {
	userManager.mu.Lock()
	u := userManager.Users[userId]
	if connId, ok := userManager.UserConnections[userId]; ok {
		delete(userManager.ZombieConnections, connId)
	}
	userManager.mu.Unlock()

	if u != nil {
		u.Character.SetAdjective(`zombie`, false)
	}
}

func IsZombieConnection(connectionId connections.ConnectionId) bool {
	userManager.mu.RLock()
	defer userManager.mu.RUnlock()

	_, ok := userManager.ZombieConnections[connectionId]
	return ok
}

func RemoveZombieConnection(connectionId connections.ConnectionId) {
	userManager.mu.Lock()
	defer userManager.mu.Unlock()

	delete(userManager.ZombieConnections, connectionId)
}

// Returns a slice of userId's
// These userId's are zombies that have reached expiration
func GetExpiredZombies(expirationTurn uint64) []int {
	userManager.mu.RLock()
	defer userManager.mu.RUnlock()

	expiredUsers := make([]int, 0)

	for connectionId, zombieTurn := range userManager.ZombieConnections {
		if zombieTurn < expirationTurn {
			expiredUsers = append(expiredUsers, userManager.Connections[connectionId])
		}
	}

	return expiredUsers
}

func GetConnectionId(userId int) connections.ConnectionId {
	userManager.mu.RLock()
	defer userManager.mu.RUnlock()

	if user, ok := userManager.Users[userId]; ok {
		return user.connectionId
	}
	return 0
}

func GetConnectionIds(userIds []int) []connections.ConnectionId {
	userManager.mu.RLock()
	defer userManager.mu.RUnlock()

	connectionIds := make([]connections.ConnectionId, 0, len(userIds))
	for _, userId := range userIds {
		if user, ok := userManager.Users[userId]; ok {
			connectionIds = append(connectionIds, user.connectionId)
		}
	}

	return connectionIds
}

func GetAllActiveUsers() []*UserRecord {
	userManager.mu.RLock()
	defer userManager.mu.RUnlock()

	ret := make([]*UserRecord, 0, len(userManager.Users))
	for _, userPtr := range userManager.Users {
		if !userPtr.isZombie {
			ret = append(ret, userPtr)
		}
	}

	return ret
}

func GetOnlineUserIds() []int {
	userManager.mu.RLock()
	defer userManager.mu.RUnlock()

	onlineList := make([]int, 0, len(userManager.Users))
	for _, user := range userManager.Users {
		onlineList = append(onlineList, user.UserId)
	}
	return onlineList
}

func GetByCharacterName(name string) *UserRecord {
	userManager.mu.RLock()
	defer userManager.mu.RUnlock()

	var closeMatch *UserRecord = nil

	name = strings.ToLower(name)
	for _, user := range userManager.Users {
		testName := strings.ToLower(user.Character.Name)
		if testName == name {
			return user
		}
		if strings.HasPrefix(testName, name) {
			closeMatch = user
		}
	}

	return closeMatch
}

func GetByUserId(userId int) *UserRecord {
	userManager.mu.RLock()
	defer userManager.mu.RUnlock()

	if user, ok := userManager.Users[userId]; ok {
		return user
	}

	return nil
}

// GetByCharacterNameOrLoad finds a user by character name, falling
// back to a disk load when no online user matches. Used by admin
// state-inspection commands (faction/crime/opinion show) that should
// work for offline players. The disk fallback passes
// skipValidation=true to avoid mutating disk state on a passive read.
//
// Disk lookup is by username, not character name; for accounts where
// the two differ, only the online-map match will succeed. DOGMud's
// single-character-per-account model keeps them aligned in practice.
func GetByCharacterNameOrLoad(name string) *UserRecord {
	if u := GetByCharacterName(name); u != nil {
		return u
	}
	if u, err := LoadUser(name, true); err == nil && u != nil {
		return u
	}
	return nil
}

func GetByConnectionId(connectionId connections.ConnectionId) *UserRecord {
	userManager.mu.RLock()
	defer userManager.mu.RUnlock()

	if userId, ok := userManager.Connections[connectionId]; ok {
		return userManager.Users[userId]
	}

	return nil
}

// First time creating a user.
func LoginUser(user *UserRecord, connectionId connections.ConnectionId) (*UserRecord, string, error) {

	mudlog.Info("LoginUser()", "username", user.Username, "connectionId", connectionId)

	user.Character.SetAdjective(`zombie`, false)

	userManager.mu.Lock()

	// If they're already logged in
	if userId, ok := userManager.Usernames[user.Username]; ok {

		// Do they have a connection tracked?
		if otherConnId, ok := userManager.UserConnections[userId]; ok {

			// Is it a zombie connection? If so, lets make this new connection the owner
			if isZombieConnectionLocked(otherConnId) {

				mudlog.Info("LoginUser()", "Zombie", true)

				if zombieUser, ok := userManager.Users[user.UserId]; ok {
					user = zombieUser
				}

				removeZombieConnectionLocked(otherConnId)

				user.connectionId = connectionId

				userManager.Users[user.UserId] = user
				userManager.Usernames[user.Username] = user.UserId
				userManager.Connections[user.connectionId] = user.UserId
				userManager.UserConnections[user.UserId] = user.connectionId

				userManager.mu.Unlock()

				// Apply persisted charset preference to connection (I/O — outside lock)
				if user.AsciiMode {
					cs := connections.GetClientSettings(connectionId)
					cs.AsciiMode = true
					connections.OverwriteClientSettings(connectionId, cs)
				}

				for _, mobInstId := range user.Character.GetCharmIds() {
					if !mobs.MobInstanceExists(mobInstId) {
						user.Character.TrackCharmed(mobInstId, false)
					}
				}

				// Set their input round to current to track idle time fresh
				user.SetLastInputRound(util.GetRoundCount())

				user.EventLog.Add(`conn`, `Reconnected`)

				return user, "Reconnecting...", nil
			}

		}

		userManager.mu.Unlock()
		// Otherwise, someone else is logged in, can't double-login!
		return nil, "That user is already logged in.", errors.New("user is already logged in")
	}

	mudlog.Info("LoginUser()", "Zombie", false)

	user.connectionId = connectionId

	userManager.Users[user.UserId] = user
	userManager.Usernames[user.Username] = user.UserId
	userManager.Connections[user.connectionId] = user.UserId
	userManager.UserConnections[user.UserId] = user.connectionId

	userManager.mu.Unlock()

	// Apply persisted charset preference to connection (I/O — outside lock)
	if user.AsciiMode {
		cs := connections.GetClientSettings(connectionId)
		cs.AsciiMode = true
		connections.OverwriteClientSettings(connectionId, cs)
	}

	mudlog.Info("LOGIN", "userId", user.UserId)

	// Defensive: re-seed the Character's userId back-reference.
	// LoadUser already does this for the fresh-login path, but a
	// zombie-reconnect path above (line 233-237) swaps in the
	// already-loaded zombieUser, whose Character may predate the
	// LoadUser fix. Setting it here too means every active session
	// has Character.userId == UserRecord.UserId regardless of how
	// the user got into the session.
	user.Character.SetUserId(user.UserId)

	// Set their input round to current to track idle time fresh
	user.SetLastInputRound(util.GetRoundCount())

	user.EventLog.Add(`conn`, `Connected`)

	for _, mobInstId := range user.Character.GetCharmIds() {
		if !mobs.MobInstanceExists(mobInstId) {
			user.Character.TrackCharmed(mobInstId, false)
		}
	}

	return user, "", nil
}

func SetZombieUser(userId int) {
	userManager.mu.Lock()

	u, ok := userManager.Users[userId]
	if !ok {
		userManager.mu.Unlock()
		return
	}

	if _, alreadyZombie := userManager.ZombieConnections[u.connectionId]; alreadyZombie {
		userManager.mu.Unlock()
		return
	}

	connId := u.connectionId
	userManager.ZombieConnections[connId] = util.GetTurnCount()
	userManager.mu.Unlock()

	// Cross-package calls outside lock
	u.Character.SetAdjective(`zombie`, true)
	u.Character.RemoveBuff(0)

	// Prevent guide mob dupes
	for _, miid := range u.Character.CharmedMobs {
		if m := mobs.GetInstance(miid); m != nil {
			if m.MobId == 38 {
				m.Character.Charmed.RoundsRemaining = 0
			}
		}
	}
}

func SaveAllUsers(isAutoSave ...bool) {
	userManager.mu.RLock()
	snapshot := make([]*UserRecord, 0, len(userManager.Users))
	for _, u := range userManager.Users {
		snapshot = append(snapshot, u)
	}
	userManager.mu.RUnlock()

	for _, u := range snapshot {
		if err := SaveUser(u, isAutoSave...); err != nil {
			mudlog.Error("SaveAllUsers()", "error", err.Error())
		}
	}
}

func LogOutUserByConnectionId(connectionId connections.ConnectionId) error {
	userManager.mu.Lock()

	userId, ok := userManager.Connections[connectionId]
	if !ok {
		userManager.mu.Unlock()
		return errors.New("user not found for connection")
	}

	u := userManager.Users[userId]

	// Capture deletion keys while we hold the lock. We'll do I/O next
	// (outside the lock), then delete by these locals — not by re-reading
	// u after a concurrent login might have replaced the map entries.
	var (
		delUserId       int
		delUsername     string
		delConnectionId connections.ConnectionId
	)
	if u != nil {
		delUserId = u.UserId
		delUsername = u.Username
		delConnectionId = u.connectionId
	}

	userManager.mu.Unlock()

	// Validate + save outside lock (I/O and cross-package calls)
	if u != nil {
		u.Character.Validate()
		SaveUser(u)

		// Chunk 5 (Presence): fire Disconnected BEFORE the user is removed
		// from the active maps so the T8 scheduler-cancel observer can still
		// find the character via its ActorRef.
		if u.Character != nil && u.Character.Presence != nil {
			_ = u.Character.Presence.TransitionTo(presence.Disconnected,
				state.TransitionReason{Trigger: presence.TriggerTCPClosed})
		}
	}

	userManager.mu.Lock()
	defer userManager.mu.Unlock()

	if u != nil {
		delete(userManager.Users, delUserId)
		delete(userManager.Usernames, delUsername)
		delete(userManager.Connections, delConnectionId)
		delete(userManager.UserConnections, delUserId)
	}

	return nil
}

// First time creating a user.
func CreateUser(u *UserRecord) error {

	if err := ValidateName(u.Username); err != nil {
		return errors.New("that username is not allowed: " + err.Error())
	}

	u.UserId = GetUniqueUserId()
	u.Role = RoleUser

	// Seed the Character's userId back-reference at creation time. Login
	// (LoginUser) and LoadUser both do this defensively, but a brand-new
	// character created via `new` plays its ENTIRE first session on this
	// in-memory record without ever going through those paths — so without
	// this, Character.GetUserId() stays 0 until the first re-login. A zero
	// userId makes Die() take the mob branch (player dies but never respawns
	// -> stuck "downed" at Health<=0) AND makes the grapple FSM build
	// ActorRef{UserId:0} -> ErrPartnerRequired. See the soft-lock fixed here.
	if u.Character != nil {
		u.Character.SetUserId(u.UserId)
	}

	idx := NewUserIndex()
	idx.AddUser(u.UserId, u.Username)

	if err := SaveUser(u); err != nil {
		return err
	}

	userManager.mu.Lock()
	defer userManager.mu.Unlock()

	userManager.Users[u.UserId] = u
	userManager.Usernames[u.Username] = u.UserId
	userManager.Connections[u.connectionId] = u.UserId
	userManager.UserConnections[u.UserId] = u.connectionId

	return nil
}

// loadUserFromPath reads and parses a single user save file, validates it
// (unless skipValidation is true), and re-saves it if validation passes.
//
// It does NOT fall through on a yaml.Unmarshal error — a partly-parsed
// record must never be "repaired" with fresh defaults and written back over
// the player's real save. shops/persistence.go and guilds/persistence.go
// already log-and-skip on a bad parse; this matches that behaviour. See the
// long comment on the historical bug in LoadUser's docs for context.
func loadUserFromPath(userFilePath string, skipValidation bool) (*UserRecord, error) {

	userFileTxt, err := os.ReadFile(userFilePath)
	if err != nil {
		return nil, err
	}

	loadedUser := &UserRecord{}
	if err := yaml.Unmarshal([]byte(userFileTxt), loadedUser); err != nil {
		mudlog.Error("LoadUser", "path", userFilePath, "error", err.Error())
		return nil, fmt.Errorf("could not parse user file %s: %w", userFilePath, err)
	}

	if loadedUser.Character == nil {
		return nil, fmt.Errorf("user file %s parsed with no character data", userFilePath)
	}

	// Retroactive ANSI hygiene (2026-08-03): mail stored before write-side
	// escaping shipped can still hold raw <ansi> payloads. Escaping is
	// idempotent, so scrubbing on every load is safe; the validation re-save
	// below persists the clean form.
	for i := range loadedUser.Inbox {
		loadedUser.Inbox[i].Message = util.EscapeAnsiTags(loadedUser.Inbox[i].Message)
	}

	if !skipValidation {
		if err := loadedUser.Character.Validate(true); err != nil {
			return nil, fmt.Errorf("user file %s failed validation: %w", userFilePath, err)
		}
		SaveUser(loadedUser)
	}

	return loadedUser, nil
}

func LoadUser(username string, skipValidation ...bool) (*UserRecord, error) {

	idx := NewUserIndex()
	userId, found := idx.FindByUsername(username)

	if !found {
		return nil, errors.New("user doesn't exist")
	}

	userFilePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, strconv.Itoa(int(userId))+`.yaml`)

	loadedUser, err := loadUserFromPath(userFilePath, len(skipValidation) > 0 && skipValidation[0])
	if err != nil {
		return nil, err
	}

	// Seed the Character's userId back-reference. UserRecord.UserId is
	// the authoritative source; Character.userId is its private mirror
	// used by Character.GetUserId() (called by combat / FSM partner-ref
	// builders, prompt rendering, btree primitives, etc.). Without this,
	// the private field stays zero post-load — fine for legacy code that
	// reads .Aggro.UserId directly, but the chunk-4b grapple FSM builds
	// ActorRef{UserId: controller.GetUserId(), ...} and rejects the
	// pair transition with ErrPartnerRequired when the resulting ref is
	// zero. The only other site calling SetUserId was the alt-character
	// switch path, so brand-new logins were the only path that broke.
	loadedUser.Character.SetUserId(loadedUser.UserId)

	// One-time migrations
	loadedUser.Character.MigratePairedSpells()
	loadedUser.Character.MigrateNeckToBack()
	loadedUser.Character.MigrateQuestSpells()
	loadedUser.Character.MigrateAlchemyPotions()
	loadedUser.Character.MigrateAlchemyRecipes()
	loadedUser.Character.MigrateDescriptionWrapping()
	loadedUser.Character.MigrateQuestFlags()
	loadedUser.Character.MigrateLegacyPotions()
	loadedUser.Character.MigrateEnchantments()
	loadedUser.Character.MigrateChrysalisAidRemoved()
	loadedUser.Character.MigrateRecipeDisciplineShuffle()
	loadedUser.Character.MigrateNewbieAwakening()
	loadedUser.ItemStorage.MigrateStorageSlots()

	if loadedUser.Joined.IsZero() {
		loadedUser.Joined = time.Now()
	}

	// Set their connection time to now
	loadedUser.connectionTime = time.Now()

	return loadedUser, nil
}

// Loads all user recvords and runs against a function.
// Stops searching if false is returned.
func SearchOfflineUsers(searchFunc func(u *UserRecord) bool) {

	basePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`)

	filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if len(path) > 10 && path[len(path)-10:] == `.alts.yaml` {
			return nil
		}

		var uRecord UserRecord

		fpathLower := path[len(path)-5:] // Only need to compare the last 5 characters
		if fpathLower == `.yaml` {

			bytes, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			err = yaml.Unmarshal(bytes, &uRecord)
			if err != nil {
				return err
			}

			// If this is an online user, skip it
			userManager.mu.RLock()
			_, isOnline := userManager.Usernames[uRecord.Username]
			userManager.mu.RUnlock()
			if isOnline {
				return nil
			}

			if res := searchFunc(&uRecord); !res {
				return errors.New(`done searching`)
			}
		}
		return nil
	})

}

func ValidateName(name string) error {
	return ValidateActorName(name, ValidateActorOpts{})
}

func ValidatePassword(pw string) error {

	validation := configs.GetValidationConfig()

	if len(pw) < int(validation.PasswordSizeMin) || len(pw) > int(validation.PasswordSizeMax) {
		return fmt.Errorf("password must be between %d and %d characters long", validation.PasswordSizeMin, validation.PasswordSizeMax)
	}

	return nil
}

// searches for a character name and returns the user that owns it
// Slow and possibly memory intensive - use strategically
func CharacterNameSearch(nameToFind string) (foundUserId int, foundUserName string) {

	foundUserId = 0
	foundUserName = ``

	SearchOfflineUsers(func(u *UserRecord) bool {

		if strings.EqualFold(u.Character.Name, nameToFind) {
			foundUserId = u.UserId
			foundUserName = u.Username
			return false
		}

		// Not found? Search alts...

		for _, char := range characters.LoadAlts(u.UserId) {
			if strings.EqualFold(char.Name, nameToFind) {
				foundUserId = u.UserId
				foundUserName = u.Username
				return false
			}
		}

		return true
	})

	return foundUserId, foundUserName
}

// CompanionNameExists checks whether any player (online or offline) has a
// companion whose Nickname matches the given string (case-insensitive exact
// match). Used to prevent new characters from taking a name already in use
// by a companion. Names are freed when companions die or are dismissed.
func CompanionNameExists(name string) bool {
	// Check online users first (fast path).
	for _, u := range GetAllActiveUsers() {
		for _, comp := range u.Character.Companions {
			if strings.EqualFold(comp.Nickname, name) {
				return true
			}
		}
	}

	// Check offline users.
	found := false
	SearchOfflineUsers(func(u *UserRecord) bool {
		for _, comp := range u.Character.Companions {
			if strings.EqualFold(comp.Nickname, name) {
				found = true
				return false // stop searching
			}
		}
		return true
	})

	return found
}

func SaveUser(u *UserRecord, isAutoSave ...bool) error {

	fileWritten := false
	tmpSaved := false
	tmpCopied := false
	completed := false

	defer func() {
		mudlog.Info("SaveUser()", "username", u.Username, "wrote-file", fileWritten, "tmp-file", tmpSaved, "tmp-copied", tmpCopied, "completed", completed)
	}()

	data, err := yaml.Marshal(u)
	if err != nil {
		return err
	}

	carefulSave := configs.GetFilePathsConfig().CarefulSaveFiles

	path := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, strconv.Itoa(u.UserId)+`.yaml`)

	saveFilePath := path
	if carefulSave { // careful save first saves a {filename}.new file
		saveFilePath += `.new`
	}

	err = os.WriteFile(saveFilePath, data, 0600)
	if err != nil {
		return err
	}
	fileWritten = true
	if carefulSave {
		tmpSaved = true
	}

	if carefulSave {
		//
		// Once the file is written, rename it to remove the .new suffix and overwrite the old file
		//
		if err := os.Rename(saveFilePath, path); err != nil {
			return err
		}
		tmpCopied = true
	}

	completed = true

	return nil
}

func GetUniqueUserId() int {

	// if highestUserId is zero, loop through users and get real highest.

	highestUserId := 0

	idx := NewUserIndex()
	if idx.Exists() {

		highestUserId = idx.GetHighestUserId()

	} else {

		// Check all user id's of offline users
		SearchOfflineUsers(func(u *UserRecord) bool {

			if u.UserId > highestUserId {
				highestUserId = u.UserId
			}

			return true
		})

		// Check all user id's of online users
		for _, u := range GetAllActiveUsers() {
			if u.UserId > highestUserId {
				highestUserId = u.UserId
			}
		}

	}

	// Increment the highestUserId before returning a new one
	highestUserId += 1

	return highestUserId
}

func Exists(name string) bool {

	for _, u := range GetAllActiveUsers() {
		if strings.ToLower(u.Username) == strings.ToLower(name) {
			return true
		}
	}

	idx := NewUserIndex()
	_, found := idx.FindByUsername(name)

	return found
}

func FindUserId(username string) int {
	idx := NewUserIndex()
	userid, _ := idx.FindByUsername(username)
	return int(userid)
}

// RemoveUserAndDisconnect tears down a user account: uncharms mobs, logs the
// user out (which saves and removes them from the in-memory indexes), deletes
// the on-disk save file, removes the username from the persistent UserIndex,
// and closes the network connection. ENOENT on the save file is non-fatal —
// the file simply may not exist in test environments.
func RemoveUserAndDisconnect(userId int) error {
	u := GetByUserId(userId)
	if u == nil {
		return errors.New("user not found")
	}

	// Uncharm any mobs before logging out so they don't dangle.
	if u.Character != nil {
		for _, mobInstanceId := range u.Character.GetCharmIds() {
			if m := mobs.GetInstance(mobInstanceId); m != nil {
				m.Character.RemoveCharm()
			}
		}
	}

	// Capture values before logout clears them from the manager.
	username := u.Username
	connId := u.connectionId
	userFilePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, strconv.Itoa(u.UserId)+`.yaml`)

	// LogOutUserByConnectionId saves + removes the user from all in-memory maps.
	// If the user has no real connection (e.g. test seed with connectionId==0),
	// the lookup by connection will fail. In that case we fall back to manual
	// map cleanup.
	if connId != 0 {
		if err := LogOutUserByConnectionId(connId); err != nil {
			mudlog.Error("RemoveUserAndDisconnect", "logoutErr", err.Error())
		}
	} else {
		// No connection — manually clean up the in-memory maps.
		userManager.mu.Lock()
		delete(userManager.Users, userId)
		delete(userManager.Usernames, strings.ToLower(username))
		userManager.mu.Unlock()
	}

	// Delete the on-disk save file. ENOENT is non-fatal.
	if err := os.Remove(userFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete user file: %w", err)
	}

	// Remove the username from the persistent UserIndex so cold lookups
	// (Exists, FindUserId, LoadUser) no longer find this account.
	idx := NewUserIndex()
	if idx.Exists() {
		if err := idx.RemoveByUsername(username); err != nil && !errors.Is(err, ErrNotFound) {
			mudlog.Error("RemoveUserAndDisconnect", "idxRemoveErr", err.Error())
		}
	}

	// Close the network connection if one exists.
	if connId != 0 {
		connections.Remove(connId) // error is non-fatal (connection may already be closed)
	}

	return nil
}

// RenameUser atomically updates Username + Character.Name + the Usernames
// index under the manager lock. The caller is responsible for calling
// SaveUser (which writes to the existing UserId-keyed file — no disk rename
// needed) and for setting LastRenameAt before saving.
func RenameUser(u *UserRecord, newName string) error {
	userManager.mu.Lock()
	defer userManager.mu.Unlock()

	oldName := u.Username
	if _, exists := userManager.Usernames[strings.ToLower(newName)]; exists {
		return errors.New("name was just claimed")
	}
	delete(userManager.Usernames, strings.ToLower(oldName))
	userManager.Usernames[strings.ToLower(newName)] = u.UserId
	u.Username = newName
	if u.Character != nil {
		u.Character.Name = newName
	}
	return nil
}
