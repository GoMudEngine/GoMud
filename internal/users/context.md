# GoMud Users System Context

## Overview

The GoMud users system provides comprehensive user account management with support for authentication, character association, connection tracking, item storage, messaging, and configuration management. It features a sophisticated indexing system for user lookups, zombie connection handling, role-based permissions, and persistent user data storage with YAML serialization.

## Architecture

The users system is built around several key components:

### Core Components

**User Management:**
- Active user tracking with connection mapping
- Role-based permission system (guest, user, admin)
- Zombie connection handling for graceful disconnections
- Thread-safe user operations with proper cleanup

**User Index System:**
- Fixed-width record index for user lookups
- Username-to-UserID mapping with collision handling
- Automatic index rebuilding and maintenance

**User Storage:**
- YAML-based persistent user data storage
- Item storage system for player belongings
- Inbox messaging system with attachments
- Configuration options and customization settings

**Connection Integration:**
- Connection ID to User ID mapping
- Real-time connection state tracking
- Input handling and prompt system integration
- Client settings and display preferences

## Key Features

### 1. **Comprehensive User Management**
- **Authentication**: Password hashing and validation
- **Role System**: Guest, user, and admin roles with permissions
- **Connection Tracking**: Real-time user connection mapping
- **Zombie Handling**: Graceful disconnection and cleanup

### 2. **Fixed-Width Index System**
- **Fixed Records**: 89-byte fixed-width records seeked by offset (`MetaDataSize + i*RecordSize`); lookups are a linear scan, not a binary search
- **Automatic Maintenance**: Index rebuilding and corruption recovery (`Rebuild()`/`rebuildFromDir` in `index_rebuild.go`)
- **Version Management**: Versioned index format via `IndexMetaData`

### 3. **Rich User Data Model**
- **Character Integration**: Full character system association
- **Item Storage**: Personal item storage separate from inventory
- **Messaging System**: Inbox with item and gold attachments
- **Customization**: Macros, aliases, and configuration options

### 4. **Advanced Features**
- **Screen Reader Support**: Accessibility features for visually impaired users
- **Audio Integration**: Music and sound effect tracking
- **Tip System**: Tutorial completion tracking
- **Temporary Data**: Session-based data storage for scripting

## User Structure

### User Record Structure
```go
type UserRecord struct {
    UserId          int                   // Unique user identifier
    Role            string                // Permission role (guest/user/admin)
    Username        string                // Login username
    Password        string                // Hashed password
    Joined          time.Time             // Account creation date
    LastRenameAt    time.Time             // Last time the rename command was used (cooldown gate)
    Macros          map[string]string     // Up to 10 user-defined command macros
    Aliases         map[string]string     // Command aliases and shortcuts
    Ticks           []UserTick            // Client-run interval timers (web automation panel)
    Triggers        []UserTrigger         // Client-run text-pattern automation (web automation panel)
    Character       *characters.Character // Associated character data
    ItemStorage     Storage               // Personal item storage (bank)
    ConfigOptions   map[string]any        // User configuration preferences
    Inbox           Inbox                 // Message inbox with attachments
    Muted           bool                  // Cannot SEND custom communications to anyone but admin/mods
    Deafened        bool                  // Cannot HEAR custom communications from anyone but admin/mods
    IsAI            bool                  // Flagged as an AI account (playtest harness)
    ScreenReader    bool                  // Accessibility mode
    AsciiMode       bool                  // Convert UTF-8 decorative chars to ASCII for legacy clients
    LineWidth       int                   // Column width for line wrapping; 0 = default 80
    CombatVerbosity string                // Combat text level: ""/full, medium (hits only), light (round tally)
    EmailAddress    string                // Contact email (optional)
    TipsComplete    map[string]bool       // Tutorial completion tracking

    // Runtime fields (not persisted, yaml:"-")
    EventLog        UserLog               // Session event logging
    LastMusic       string                // Audio state tracking
    LastWhisperFrom int                   // UserId of last person who whispered to us

    // Unexported (not marshalled)
    connectionId   uint64                // Current connection ID
    sessionMu      sync.Mutex            // Guards unsentText, suggestText, tempDataStore and
                                          // activePrompt: the only UserRecord state written from
                                          // both the per-connection goroutine and the tick loop.
                                          // tempDataStore is a map, so a concurrent read+write is
                                          // a hard runtime panic, not a benign race.
    unsentText     string                // Buffered output
    suggestText    string                // Input suggestions
    connectionTime time.Time             // Connection timestamp
    lastInputRound uint64                // Last input round number
    tempDataStore  map[string]any        // Temporary session data
    activePrompt   *prompt.Prompt        // Current prompt state
    isZombie       bool                  // Zombie connection flag
    inputBlocked   bool                  // Input processing control
}
```

`UserTick` (`Id, Name, Commands, IntervalSec, Enabled`) and `UserTrigger`
are both defined in `userrecord.go` alongside `UserRecord`; they back the
web client's automation panel and are not otherwise covered here.

### Active Users Management
```go
type ActiveUsers struct {
    mu                sync.RWMutex                        // guards all five maps below
    Users             map[int]*UserRecord                 // userId -> UserRecord
    Usernames         map[string]int                      // username -> userId
    Connections       map[connections.ConnectionId]int    // connectionId -> userId
    UserConnections   map[int]connections.ConnectionId    // userId -> connectionId
    ZombieConnections map[connections.ConnectionId]uint64 // connectionId -> zombie turn
}
```

There is a single package-level instance, `userManager`, created by
`newUserManager()`. Every exported function in `users.go` (`GetByUserId`,
`GetAllActiveUsers`, `LoginUser`, etc.) takes `userManager.mu`, an
`RLock` for read-only functions and a full `Lock` for anything that
mutates one of the five maps, before touching them. This locking is
per-call only: it makes each individual function call atomic, not a
sequence of calls. A caller that reads with one call and then writes
based on that read with a second call gets no atomicity guarantee across
the gap between the two: another goroutine can change state in between.

## User Index System

### Index Structure
```go
type IndexMetaData struct {
    MetaDataSize uint64 // Header size in bytes (100)
    IndexVersion uint64 // Index format version (1)
    RecordCount  uint64 // Number of user records
    RecordSize   uint64 // Fixed record size (89 bytes)
}

type IndexUserRecord struct {
    UserID   int64     // 8 bytes - User identifier
    Username [80]byte  // 80 bytes - Fixed-width username
}
```
On disk each record is username bytes, then the little-endian `UserID`,
then a single line-terminator byte (`IndexLineTerminatorV1`) written
separately by the read/write code. The terminator is not a struct field.

### Index Operations
```go
// Create new index from scratch
func (idx *UserIndex) Create() error {
    idx.Delete() // Remove existing index

    f, err := os.Create(idx.Filename)
    if err != nil {
        return err
    }
    defer f.Close()

    // Reset metadata / write header
    idx.metaData = IndexMetaData{
        MetaDataSize: FixedHeaderTotalLength,
        IndexVersion: IndexVersion,
        RecordCount:  0,
        RecordSize:   IndexRecordSizeV1,
    }

    headerBytes, err := idx.metaData.Format()
    if err != nil {
        return err
    }
    if _, err := f.Write(headerBytes); err != nil {
        return err
    }
    return nil
}

// Username lookup: a full sequential scan of the fixed-width records (not
// a binary search despite the earlier upstream naming). Each record is an
// exact, case-insensitive match, not a prefix match: the first
// len(username) bytes are compared byte-for-byte, then the record's next
// byte must be a zero (the padding after the stored username), so a query
// for "bob" does NOT match a stored "bobby": only equal-length,
// equal-content usernames match. Case-insensitivity comes from both sides being
// lowercased (the query here, the stored value in AddUser).
func (idx *UserIndex) FindByUsername(username string) (int64, bool) {
    if len(username) > 80 {
        return 0, false
    }

    f, err := os.Open(idx.Filename)
    if err != nil {
        return 0, false
    }
    defer f.Close()

    username = strings.ToLower(username)

    for i := uint64(0); i < idx.metaData.RecordCount; i++ {
        offset := int64(idx.metaData.MetaDataSize) + int64(i*idx.metaData.RecordSize)
        f.Seek(offset, io.SeekStart)

        var recUsername [80]byte
        io.ReadFull(f, recUsername[:])

        var userId int64
        binary.Read(f, binary.LittleEndian, &userId)

        // ... reads and validates the terminator byte, then applies the
        // exact-match check described above ...

        // returns (userId, true) on match
    }
    return 0, false
}
```
`FindByUserId(userId int64) (string, bool)` does the mirror-image scan
(compare `UserID`, return the stored username). Neither method returns an
error; both use a `(value, found bool)` shape.

## User Management Operations

### User Creation and Authentication
```go
// Create new user record
func NewUserRecord(userId int, connectionId uint64) *UserRecord {
    u := &UserRecord{
        connectionId:   connectionId,
        UserId:         userId,
        Role:           RoleUser,
        Username:       "",
        Password:       "",
        Macros:         make(map[string]string),
        Character:      characters.New(),
        ConfigOptions:  map[string]any{},
        Joined:         time.Now(),
        connectionTime: time.Now(),
        tempDataStore:  make(map[string]any),
        EventLog:       UserLog{},
    }

    return u
}

// PasswordMatches tries three formats in order:
//   1. bcrypt (current format): compared via bcrypt.CompareHashAndPassword
//   2. legacy unsalted SHA256 (util.Hash): on match, re-hashes with bcrypt
//      in place so the next login uses the secure path
//   3. plaintext fallback: only allowed when HasPlaintextPassword() is true
//      (stored value is neither bcrypt nor a 64-char hex SHA256 digest)
//
// The plaintext branch does NOT migrate the password. u.Password is left
// as plaintext on disk and PasswordMatches will keep accepting it on every
// future login until something else changes it. The "forced" part of
// "authenticate once, then change it" is enforced outside this function:
// HasPlaintextPassword() gates the command dispatcher
// (internal/usercommands/usercommands.go) and the post-login hook
// (internal/hooks/PlayerSpawn_HandleJoin.go) so every command except
// `password` is refused until the player runs `password` and the stored
// value becomes a bcrypt hash.
func (u *UserRecord) PasswordMatches(input string) bool {
    // see internal/users/userrecord.go for the real three-step logic
}
```

### Connection Management
```go
// Get connection ID for user. Takes userManager.mu.RLock like every
// other exported function in this file.
func GetConnectionId(userId int) connections.ConnectionId {
    userManager.mu.RLock()
    defer userManager.mu.RUnlock()

    if user, ok := userManager.Users[userId]; ok {
        return user.connectionId
    }
    return 0
}

// Get multiple connection IDs. Takes userManager.mu.RLock for the loop.
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

// Get all active users. Zombies are excluded: a zombie is still in
// userManager.Users (it hasn't been logged out yet) but isZombie is
// true, so it's filtered out of this list. Takes userManager.mu.RLock.
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
```

### Zombie Connection Handling
```go
// Clear a user's zombie status (they've reconnected or been cleaned up).
// The function to MARK a user as a zombie is SetZombieUser, not this one.
// Takes userManager.mu for the map access, then calls into characters
// (a different package) outside the lock.
func RemoveZombieUser(userId int) {
    userManager.mu.Lock()
    u := userManager.Users[userId]
    if connId, ok := userManager.UserConnections[userId]; ok {
        delete(userManager.ZombieConnections, connId)
    }
    userManager.mu.Unlock()

    if u != nil {
        u.Character.SetAdjective("zombie", false)
    }
}

// Check if connection is zombie. Takes userManager.mu.RLock.
func IsZombieConnection(connectionId connections.ConnectionId) bool {
    userManager.mu.RLock()
    defer userManager.mu.RUnlock()

    _, ok := userManager.ZombieConnections[connectionId]
    return ok
}

// Get expired zombie connections for cleanup. Takes userManager.mu.RLock
// for the whole scan.
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
```

## Storage Systems

### Item Storage
Storage is the per-player bank inventory. It is slot-based, not a flat
item list: stackable item types (components, food, potions, etc.)
accumulate a `Count` on one `StorageSlot` so, e.g., 50 iron ore occupies a
single slot. Non-stackable types always have `Count == 1`.
```go
type StorageSlot struct {
    Item  items.Item // one representative item for the stack
    Count int        // >= 1
}

type Storage struct {
    Slots []StorageSlot // canonical field

    // Legacy field, kept with omitempty for one release so old save
    // files still deserialize; MigrateStorageSlots folds it into Slots
    // on first load and clears it.
    Items []items.Item
}

func (s *Storage) SlotCount() int             // number of occupied slots (stacks)
func (s *Storage) GetItems() []items.Item     // one Item value per logical unit (a Count-3 slot yields 3 values)
func (s *Storage) GetSlots() []StorageSlot    // copy of the slot list; use when reasoning about stacks
func (s *Storage) FindItem(itemName string) (items.Item, bool) // fuzzy match via items.FindMatchIn
func (s *Storage) AddItem(i items.Item) bool  // increments an existing stack (via items.SameStack) or appends a new slot
func (s *Storage) RemoveItem(i items.Item) bool // decrements/drops a stackable slot, or removes a non-stackable slot by UUID (items.Item.Equals)
func (s *Storage) RemoveSlot(idx int) StorageSlot // removes and returns the whole slot at idx; panics if out of range
func (s *Storage) MigrateStorageSlots() bool  // storage_migrate.go: folds legacy Items into Slots
```

### Inbox Messaging System
```go
type Inbox []Message

type Message struct {
    FromUserId int         // Sender user ID
    FromName   string      // Sender display name
    Message    string      // Message content
    Item       *items.Item // Attached item (optional)
    Gold       int         // Attached gold amount
    Read       bool        // Read status
    DateSent   time.Time   // Timestamp
}

// Add message to inbox (newest first)
func (i *Inbox) Add(msg Message) {
    msg.DateSent = time.Now()
    
    newInbox := &Inbox{msg}
    
    if i == nil {
        (*i) = *newInbox
        return
    }
    
    // Prepend new message
    (*i) = append(*newInbox, (*i)...)
}

// Count read/unread messages
func (i *Inbox) CountRead() int {
    ct := 0
    for _, msg := range *i {
        if msg.Read {
            ct++
        }
    }
    return ct
}

func (i *Inbox) CountUnread() int {
    ct := 0
    for _, msg := range *i {
        if !msg.Read {
            ct++
        }
    }
    return ct
}

// Empty clears the inbox entirely.
func (i *Inbox) Empty() {
    (*i) = Inbox{}
}

// DateString formats msg.DateSent using the configured TextFormats.Time layout.
func (m Message) DateString() string {
    return m.DateSent.Format(string(configs.GetConfig().TextFormats.Time))
}
```

## User Data Management

### Temporary Data Storage
```go
// Set temporary session data
func (u *UserRecord) SetTempData(key string, value any) {
    u.sessionMu.Lock()
    defer u.sessionMu.Unlock()

    if u.tempDataStore == nil {
        u.tempDataStore = make(map[string]any)
    }
    
    if value == nil {
        delete(u.tempDataStore, key)
        return
    }
    
    u.tempDataStore[key] = value
}

// Get temporary session data. Takes a full Lock rather than an RLock
// because this "getter" lazily allocates tempDataStore on first use.
func (u *UserRecord) GetTempData(key string) any {
    u.sessionMu.Lock()
    defer u.sessionMu.Unlock()

    if u.tempDataStore == nil {
        u.tempDataStore = make(map[string]any)
    }
    
    if value, ok := u.tempDataStore[key]; ok {
        return value
    }
    
    return nil
}
```

### Configuration Management
```go
// Get client display settings
func (u *UserRecord) ClientSettings() connections.ClientSettings {
    return connections.GetClientSettings(u.connectionId)
}

// Configuration options stored in ConfigOptions map (any type per key).
// Real keys in use include:
// - "wimpy": int             // health% threshold for auto-flee
// - "prompt" / "fprompt": string // prompt template overrides
// - "tinymap": bool          // compact map display (defaults true if unset)
// - "shortadjectives": bool
// - "auction": bool          // opt out of auction house broadcast messages
```

## Online User Information

### Online Status Tracking
```go
type OnlineInfo struct {
    Username      string // Login username
    CharacterName string // Character display name
    Title         string // Skill/mutation-derived title (skills.GetTitle), NOT a level or class name
    OnlineTime    int64  // Seconds online
    OnlineTimeStr string // Formatted time string
    IsAFK         bool   // Away from keyboard status
    IsAI          bool   // True if the connection is an AI-flagged playtest session
    Role          string // User role (guest/user/admin)
}
```
There is no `Level`, `Alignment`, or `Profession` field. DOGMud has no
character levels or classes; `Title` is derived from the character's
mutations, skill ranks, and stats via `skills.GetTitle(...)`, called from
`(*UserRecord).GetOnlineInfo()`.

## Integration Patterns

### Character System Integration
```go
// Users have associated characters
- user.Character                    // Full character data
- user.Character.Name               // Character name
- user.Character.RoomId             // Current location
```
`Character` has no `Level` field. DOGMud has no character levels;
progression is use-based (see CLAUDE.md and `internal/characters/context.md`
if present).

### Connection System Integration
```go
// Users map to network connections
- user.connectionId                // Current connection
- connections.GetClientSettings()  // Display preferences
- connections.SendTo()             // Send messages to user
```

### Prompt System Integration
```go
// Users can have active prompts
- user.StartPrompt(command, rest) (*prompt.Prompt, bool) // creates or reuses the active prompt
- user.GetPrompt() *prompt.Prompt                        // read the active prompt
- user.ClearPrompt()                                     // clear it
- user.inputBlocked                                      // Input processing control (via BlockInput/UnblockInput)
```
`prompt.Ask()` is not a package-level function: it is a method,
`(*prompt.Prompt).Ask(question, responseOptions, defaultOption ...)`,
called on the prompt returned by `StartPrompt`/`GetPrompt`.

### Event System Integration
```go
// Users participate in game events
- events.AddToQueue()             // Queue user actions
- user.EventLog                   // Track user events
- user.lastInputRound             // Input timing
```

## Usage Examples

### User Authentication
```go
// Authenticate user login (LoadUser looks the username up in the
// UserIndex internally: there is no separate userIndex.GetByUsername step)
username := "player1"
password := "secretpass"

user, err := LoadUser(username)
if err != nil {
    return errors.New("user not found")
}

// Verify password
if !user.PasswordMatches(password) {
    return errors.New("invalid password")
}

// User authenticated successfully
fmt.Printf("Welcome back, %s!\n", user.Username)
```

### User Management
```go
// Create new user
connectionId := uint64(12345)
userId := users.GetUniqueUserId()

user := users.NewUserRecord(userId, connectionId)
user.Username = "newplayer"
user.Character.Name = "NewPlayer"
if err := user.SetPassword("password123"); err != nil {
    return err
}

// CreateUser assigns UserId/Role, saves the file, registers the user in
// the UserIndex, and adds it to the active-users maps: no manual map
// manipulation needed.
if err := users.CreateUser(user); err != nil {
    return err
}
```

### Item Storage Operations
```go
// Store item in user's bank storage (AddItem stacks stackable types
// automatically; see the slot-based Storage struct above)
sword := items.New(101) // Create sword item
if user.ItemStorage.AddItem(sword) {
    user.SendText(messaging.CategorySystem, "Item stored successfully.")
}

// Retrieve item from storage
storedItem, found := user.ItemStorage.FindItem("sword")
if found {
    user.Character.StoreItem(storedItem)
    user.ItemStorage.RemoveItem(storedItem)
    user.SendText(messaging.CategorySystem, "Item retrieved from storage.")
}
```
`Character` has no `GiveItem` method; the equivalent is `StoreItem`.
`SendText` takes a `messaging.Category`, not just a string.

### Messaging System
```go
// Send message with attachment
message := users.Message{
    FromUserId: senderUserId,
    FromName:   sender.Character.Name,
    Message:    "Here's that sword I promised you!",
    Item:       &sword,
    Gold:       100,
    Read:       false,
}

recipient.Inbox.Add(message)
recipient.SendText(messaging.CategorySystem, "You have a new message!")

// Check unread messages
unreadCount := recipient.Inbox.CountUnread()
if unreadCount > 0 {
    recipient.SendText(messaging.CategorySystem, fmt.Sprintf("You have %d unread messages.", unreadCount))
}
```

### Zombie Connection Cleanup
The real cleanup path (`internal/hooks/NewTurn_CleanupZombies.go`) does not
call `user.Save()` or a `RemoveUser` directly. It queues a `leaveworld`
system event per expired userId, which the normal disconnect flow then
handles:
```go
expTurns := uint64(et.SecondsToTurns(int(gp.ZombieSeconds)))

expZombies := users.GetExpiredZombies(evt.TurnNumber - expTurns)
for _, userId := range expZombies {
    events.AddToQueue(events.System{
        Command:     `leaveworld`,
        Data:        userId,
        Description: `Zombie Expired`,
    })
}
```
There is no `users.RemoveUser` function. Tearing down an account entirely
(deleting its save file, index entry, and disconnecting it) is
`users.RemoveUserAndDisconnect(userId int) error` (a different,
much more destructive operation than zombie cleanup).

## Presence Machine Integration (chunk 5)

The Presence machine lives on `Character`, not `UserRecord`. Only the
TCP-close side of the connection lifecycle is wired inside this package;
login-complete and input-wake are wired elsewhere:

- **`users.go`, TCP-close observer:** `LogOutUserByConnectionId` calls
  `u.Character.Presence.TransitionTo(presence.Disconnected, ...)` with
  `TriggerTCPClosed` when a connection is dropped. This fires the
  scheduler-cancel observer automatically.
- **Login-complete (NOT in this package):**
  `internal/hooks/PlayerSpawn_HandleJoin.go` fires
  `Presence.TransitionTo(presence.Active, ...)` with `TriggerEnteredRoom`
  when the character is placed in its starting room.
- **Input wake (NOT in this package):** `internal/usercommands` (both
  `afk.go` and `usercommands.go`) fires
  `Presence.TransitionTo(presence.Active, ...)` with
  `TriggerInputReceived` when a command is dispatched.
  `(*UserRecord).SetLastInputRound` in `userrecord.go` only updates the
  `lastInputRound` counter; it does not touch Presence itself (see the
  comment on that function for why the wake logic had to move to the
  per-command entry point).

### OnlineInfo.IsAFK (compat shim)

`OnlineInfo.IsAFK` continues to exist on the struct used by `online.go`
for the player list. In chunk 5 it is populated by:

```go
IsAFK: u.Character.Presence.State() == presence.AFK,
```

replacing the old ad-hoc computation
`ManualAFK || (roundNow - lastInputRound >= afkRounds)`. The `online`
command display and any `(afk)` marker in room descriptions still work
identically. The Presence machine is the only thing that changed.

The `Idle` Presence state is intentionally invisible to the UI in v1.
Only `AFK` surfaces to players via the `(afk)` tag.

## Dependencies

- `internal/characters` - Character system integration for user avatars
- `internal/connections` - Network connection management and client settings
- `internal/items` - Item system for storage and inventory management
- `internal/configs` - Configuration management for user settings
- `internal/prompt` - Interactive prompt system for user input
- `internal/util` - Utility functions for hashing, file operations, and validation
- `internal/mudlog` - Logging system for user events and debugging
- `internal/events` - Event queue (login/logout, web client commands)
- `internal/messaging` - `SendText` categories and combat-verbosity parsing
- `internal/mobs` - Charm tracking cleanup on login/zombie/removal
- `internal/quests` - Quest step migration (`DoQuestStepMigration`)
- `internal/skills` - Title derivation for `OnlineInfo.Title`
- `internal/spells`, `internal/gametime`, `internal/term` - prompt token
  rendering (`userrecord.prompt.go`)
- `internal/audio` - Music/sound tracking (`PlayMusic`/`PlaySound`)
- `internal/copyover` - Copyover contributor interface (`copyover.go`)
- `internal/state`, `internal/state/presence`, `internal/state/position` -
  Presence/position state machines (chunk 5, lives on Character but
  connection lifecycle wired here)
- `internal/buffs`, `internal/state/awareness` - only pulled in by
  `test_helpers.go` (`NewTestUser`), not by any production path
- `golang.org/x/crypto/bcrypt` - Password hashing (current format)
- `gopkg.in/yaml.v2` - YAML serialization for user data persistence

## Files

| File | Purpose |
|------|---------|
| `users.go` | Registry, connect/disconnect, lookup, save file read/write (`LoadUser`, `loadUserFromPath`, `SaveUser`, `SaveAllUsers`) |
| `userrecord.go` | The `UserRecord` type |
| `userrecord.prompt.go` | Prompt rendering and tokens |
| `storage.go` / `storage_migrate.go` | Bank inventory (`Storage`, `StorageSlot`) and its legacy `Items`-to-`Slots` shape migration |
| `index.go` / `index_rebuild.go` / `character_index.go` | Name/character indexes |
| `migration.go` | Per-user migrations |
| `validate_actor_name.go` | Name validation |
| `inbox.go` | Player mudmail |
| `onlineinfo.go` | Who-list / online presence data |
| `userlog.go` | Per-user log |
| `copyover.go` | Copyover contributor |
| `memory.go` | Memory reporting |

Saves live at `_datafiles/world/dogmud/users/<id>.yaml`. That directory has its
contents gitignored but ships a tracked `.gitkeep`. Without it a fresh clone
fails `ValidateWorldFiles` and the server dies before loading a room.
