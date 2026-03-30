package users

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"

	//
	"gopkg.in/yaml.v2"
)

var (
	userManager *ActiveUsers = newUserManager()
)

type ActiveUsers struct {
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

func RemoveZombieUser(userId int) {

	if u := userManager.Users[userId]; u != nil {
		u.Character.SetAdjective(`zombie`, false)
	}
	if connId, ok := userManager.UserConnections[userId]; ok {
		delete(userManager.ZombieConnections, connId)
	}
}

func IsZombieConnection(connectionId connections.ConnectionId) bool {
	_, ok := userManager.ZombieConnections[connectionId]
	return ok
}

func RemoveZombieConnection(connectionId connections.ConnectionId) {
	delete(userManager.ZombieConnections, connectionId)
}

// Returns a slice of userId's
// These userId's are zombies that have reached expiration
func GetExpiredZombies(expirationTurn uint64) []int {

	expiredUsers := make([]int, 0)

	for connectionId, zombieTurn := range userManager.ZombieConnections {
		if zombieTurn < expirationTurn {
			expiredUsers = append(expiredUsers, userManager.Connections[connectionId])
		}
	}

	return expiredUsers
}

func GetConnectionId(userId int) connections.ConnectionId {
	if user, ok := userManager.Users[userId]; ok {
		return user.connectionId
	}
	return 0
}

func GetConnectionIds(userIds []int) []connections.ConnectionId {

	connectionIds := make([]connections.ConnectionId, 0, len(userIds))
	for _, userId := range userIds {
		if user, ok := userManager.Users[userId]; ok {
			connectionIds = append(connectionIds, user.connectionId)
		}
	}

	return connectionIds
}

func GetAllActiveUsers() []*UserRecord {
	ret := []*UserRecord{}

	for _, userPtr := range userManager.Users {
		if !userPtr.isZombie {
			ret = append(ret, userPtr)
		}
	}

	return ret
}

func GetOnlineUserIds() []int {

	onlineList := make([]int, 0, len(userManager.Users))
	for _, user := range userManager.Users {
		onlineList = append(onlineList, user.UserId)
	}
	return onlineList
}

func GetByCharacterName(name string) *UserRecord {

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

	if user, ok := userManager.Users[userId]; ok {
		return user
	}

	return nil
}

func GetByConnectionId(connectionId connections.ConnectionId) *UserRecord {

	if userId, ok := userManager.Connections[connectionId]; ok {
		return userManager.Users[userId]
	}

	return nil
}

// First time creating a user.
func LoginUser(user *UserRecord, connectionId connections.ConnectionId) (*UserRecord, string, error) {

	mudlog.Info("LoginUser()", "username", user.Username, "connectionId", connectionId)

	user.Character.SetAdjective(`zombie`, false)

	// If they're already logged in
	if userId, ok := userManager.Usernames[user.Username]; ok {

		// Do they have a connection tracked?
		if otherConnId, ok := userManager.UserConnections[userId]; ok {

			// Is it a zombie connection? If so, lets make this new connection the owner
			if IsZombieConnection(otherConnId) {

				mudlog.Info("LoginUser()", "Zombie", true)

				if zombieUser, ok := userManager.Users[user.UserId]; ok {
					user = zombieUser
				}

				RemoveZombieConnection(otherConnId)

				user.connectionId = connectionId

				userManager.Users[user.UserId] = user
				userManager.Usernames[user.Username] = user.UserId
				userManager.Connections[user.connectionId] = user.UserId
				userManager.UserConnections[user.UserId] = user.connectionId

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

		// Otherwise, someone else is logged in, can't double-login!
		return nil, "That user is already logged in.", errors.New("user is already logged in")
	}

	mudlog.Info("LoginUser()", "Zombie", false)

	// Set their input round to current to track idle time fresh
	user.SetLastInputRound(util.GetRoundCount())

	user.connectionId = connectionId

	userManager.Users[user.UserId] = user
	userManager.Usernames[user.Username] = user.UserId
	userManager.Connections[user.connectionId] = user.UserId
	userManager.UserConnections[user.UserId] = user.connectionId

	mudlog.Info("LOGIN", "userId", user.UserId)

	user.EventLog.Add(`conn`, `Connected`)

	for _, mobInstId := range user.Character.GetCharmIds() {
		if !mobs.MobInstanceExists(mobInstId) {
			user.Character.TrackCharmed(mobInstId, false)
		}
	}

	return user, "", nil
}

func SetZombieUser(userId int) {

	if u, ok := userManager.Users[userId]; ok {

		u.Character.RemoveBuff(0)
		u.Character.SetAdjective(`zombie`, true)

		// Prevent guide mob dupes
		for _, miid := range u.Character.CharmedMobs {
			if m := mobs.GetInstance(miid); m != nil {
				if m.MobId == 38 {
					m.Character.Charmed.RoundsRemaining = 0
				}
			}
		}

		if _, ok := userManager.ZombieConnections[u.connectionId]; ok {
			return
		}

		userManager.ZombieConnections[u.connectionId] = util.GetTurnCount()
	}

}

func SaveAllUsers(isAutoSave ...bool) {

	for _, u := range userManager.Users {
		if err := SaveUser(*u, isAutoSave...); err != nil {
			mudlog.Error("SaveAllUsers()", "error", err.Error())
		}
	}

}

func LogOutUserByConnectionId(connectionId connections.ConnectionId) error {

	u := GetByConnectionId(connectionId)

	if _, ok := userManager.Connections[connectionId]; ok {

		// Make sure the user data is saved to a file.
		if u != nil {
			u.Character.Validate()
			SaveUser(*u)
		}

		delete(userManager.Users, u.UserId)
		delete(userManager.Usernames, u.Username)
		delete(userManager.Connections, u.connectionId)
		delete(userManager.UserConnections, u.UserId)

		return nil
	}
	return errors.New("user not found for connection")
}

// First time creating a user.
func CreateUser(u *UserRecord) error {

	if err := ValidateName(u.Username); err != nil {
		return errors.New("that username is not allowed: " + err.Error())
	}

	u.UserId = GetUniqueUserId()
	u.Role = RoleUser

	idx := NewUserIndex()
	idx.AddUser(u.UserId, u.Username)

	if err := SaveUser(*u); err != nil {
		return err
	}

	userManager.Users[u.UserId] = u
	userManager.Usernames[u.Username] = u.UserId
	userManager.Connections[u.connectionId] = u.UserId
	userManager.UserConnections[u.UserId] = u.connectionId

	return nil
}

func LoadUser(username string, skipValidation ...bool) (*UserRecord, error) {

	idx := NewUserIndex()
	userId, found := idx.FindByUsername(username)

	if !found {
		return nil, errors.New("user doesn't exist")
	}

	userFilePath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, strconv.Itoa(int(userId))+`.yaml`)

	userFileTxt, err := os.ReadFile(userFilePath)
	if err != nil {
		return nil, err
	}

	loadedUser := &UserRecord{}
	if err := yaml.Unmarshal([]byte(userFileTxt), loadedUser); err != nil {
		mudlog.Error("LoadUser", "error", err.Error())
	}

	if len(skipValidation) == 0 || !skipValidation[0] {
		if err := loadedUser.Character.Validate(true); err == nil {
			SaveUser(*loadedUser)
		}
	}

	// One-time migrations
	loadedUser.Character.MigratePairedSpells()
	loadedUser.Character.MigrateNeckToBack()
	loadedUser.Character.MigrateQuestSpells()
	loadedUser.Character.MigrateAlchemyPotions()
	loadedUser.Character.MigrateAlchemyRecipes()

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
			if _, ok := userManager.Usernames[uRecord.Username]; ok {
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

	validation := configs.GetValidationConfig()

	if len(name) < int(validation.NameSizeMin) || len(name) > int(validation.NameSizeMax) {
		return fmt.Errorf("name must be between %d and %d characters long", validation.NameSizeMin, validation.NameSizeMax)
	}

	if validation.NameRejectRegex != `` {
		if !regexp.MustCompile(validation.NameRejectRegex.String()).MatchString(name) {
			return errors.New(validation.NameRejectReason.String())
		}
	}

	if bannedPattern, ok := configs.GetConfig().IsBannedName(name); ok {
		return errors.New(`that username matched the prohibited name pattern: "` + bannedPattern + `"`)
	}

	for _, mobName := range mobs.GetAllMobNames() {
		if strings.EqualFold(mobName, name) {
			return errors.New("that username is in use")
		}
	}

	if Exists(name) {
		return errors.New("that username is in use")
	}

	return nil
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

func SaveUser(u UserRecord, isAutoSave ...bool) error {

	fileWritten := false
	tmpSaved := false
	tmpCopied := false
	completed := false

	defer func() {
		mudlog.Info("SaveUser()", "username", u.Username, "wrote-file", fileWritten, "tmp-file", tmpSaved, "tmp-copied", tmpCopied, "completed", completed)
	}()

	data, err := yaml.Marshal(&u)
	if err != nil {
		return err
	}

	carefulSave := configs.GetFilePathsConfig().CarefulSaveFiles

	path := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, strconv.Itoa(u.UserId)+`.yaml`)

	saveFilePath := path
	if carefulSave { // careful save first saves a {filename}.new file
		saveFilePath += `.new`
	}

	err = os.WriteFile(saveFilePath, data, 0777)
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
