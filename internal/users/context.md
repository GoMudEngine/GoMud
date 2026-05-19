# GoMud Users System Context

## Overview

The GoMud users system provides comprehensive user account management with support for authentication, character association, connection tracking, item storage, messaging, and configuration management. It features a sophisticated indexing system for fast user lookups, zombie connection handling, role-based permissions, and persistent user data storage with YAML serialization.

## Architecture

The users system is built around several key components:

### Core Components

**User Management:**
- Active user tracking with connection mapping
- Role-based permission system (guest, user, admin)
- Zombie connection handling for graceful disconnections
- Thread-safe user operations with proper cleanup

**User Index System:**
- High-performance binary index for user lookups
- Fixed-width record format for fast seeking
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

### 2. **High-Performance Index System**
- **Binary Index**: Fast username lookups with O(log n) performance
- **Fixed Records**: 89-byte fixed-width records for efficient seeking
- **Automatic Maintenance**: Index rebuilding and corruption recovery
- **Version Management**: Versioned index format with migration support

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
    UserId         int                   // Unique user identifier
    Role           string                // Permission role (guest/user/admin)
    Username       string                // Login username
    Password       string                // Hashed password
    Joined         time.Time             // Account creation date
    Macros         map[string]string     // User-defined command macros
    Aliases        map[string]string     // Command aliases and shortcuts
    Character      *characters.Character // Associated character data
    ItemStorage    Storage               // Personal item storage
    ConfigOptions  map[string]any        // User configuration preferences
    Inbox          Inbox                 // Message inbox with attachments
    Muted          bool                  // Communication restrictions
    Deafened       bool                  // Communication filtering
    ScreenReader   bool                  // Accessibility mode
    EmailAddress   string                // Contact email (optional)
    TipsComplete   map[string]bool       // Tutorial completion tracking
    
    // Runtime fields (not persisted)
    EventLog       UserLog               // Session event logging
    LastMusic      string                // Audio state tracking
    connectionId   uint64                // Current connection ID
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

### Active Users Management
```go
type ActiveUsers struct {
    Users             map[int]*UserRecord                 // userId -> UserRecord
    Usernames         map[string]int                      // username -> userId
    Connections       map[connections.ConnectionId]int    // connectionId -> userId
    UserConnections   map[int]connections.ConnectionId    // userId -> connectionId
    ZombieConnections map[connections.ConnectionId]uint64 // connectionId -> zombie turn
}
```

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
                       // 1 byte - Line terminator
}
```

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
    
    // Write header
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
    
    return f.Write(headerBytes)
}

// Fast username lookup
func (idx *UserIndex) GetByUsername(username string) (int, error) {
    if !idx.Exists() {
        return 0, ErrIndexMissing
    }
    
    if len(username) > 79 {
        return 0, ErrSearchNameTooLong
    }
    
    f, err := os.Open(idx.Filename)
    if err != nil {
        return 0, err
    }
    defer f.Close()
    
    // Skip header
    f.Seek(int64(idx.metaData.MetaDataSize), 0)
    
    // Binary search through fixed-width records
    for i := uint64(0); i < idx.metaData.RecordCount; i++ {
        var record IndexUserRecord
        if err := binary.Read(f, binary.LittleEndian, &record); err != nil {
            return 0, err
        }
        
        // Read terminator
        var terminator byte
        binary.Read(f, binary.LittleEndian, &terminator)
        
        recordUsername := strings.TrimRight(string(record.Username[:]), "\x00")
        if strings.EqualFold(recordUsername, username) {
            return int(record.UserID), nil
        }
    }
    
    return 0, ErrNotFound
}
```

## User Management Operations

### User Creation and Authentication
```go
// Create new user record
func NewUserRecord(userId int, connectionId uint64) *UserRecord {
    c := configs.GetGamePlayConfig()
    
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

// Password validation with multiple formats
func (u *UserRecord) PasswordMatches(input string) bool {
    // Direct match (legacy)
    if input == u.Password {
        return true
    }
    
    // Hashed match (current)
    if u.Password == util.Hash(input) {
        return true
    }
    
    return false
}
```

### Connection Management
```go
// Get connection ID for user
func GetConnectionId(userId int) connections.ConnectionId {
    if user, ok := userManager.Users[userId]; ok {
        return user.connectionId
    }
    return 0
}

// Get multiple connection IDs
func GetConnectionIds(userIds []int) []connections.ConnectionId {
    connectionIds := make([]connections.ConnectionId, 0, len(userIds))
    for _, userId := range userIds {
        if user, ok := userManager.Users[userId]; ok {
            connectionIds = append(connectionIds, user.connectionId)
        }
    }
    return connectionIds
}

// Get all active users
func GetAllActiveUsers() []*UserRecord {
    ret := []*UserRecord{}
    for _, user := range userManager.Users {
        ret = append(ret, user)
    }
    return ret
}
```

### Zombie Connection Handling
```go
// Mark user as zombie (disconnected but not cleaned up)
func RemoveZombieUser(userId int) {
    if u := userManager.Users[userId]; u != nil {
        u.Character.SetAdjective("zombie", false)
    }
    if connId, ok := userManager.UserConnections[userId]; ok {
        delete(userManager.ZombieConnections, connId)
    }
}

// Check if connection is zombie
func IsZombieConnection(connectionId connections.ConnectionId) bool {
    _, ok := userManager.ZombieConnections[connectionId]
    return ok
}

// Get expired zombie connections for cleanup
func GetExpiredZombies(expirationTurn uint64) []int {
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
```go
type Storage struct {
    Items []items.Item // Personal item storage
}

// Find item by name with fuzzy matching
func (s *Storage) FindItem(itemName string) (items.Item, bool) {
    if itemName == "" {
        return items.Item{}, false
    }
    
    closeMatchItem, matchItem := items.FindMatchIn(itemName, s.Items...)
    
    if matchItem.ItemId != 0 {
        return matchItem, true
    }
    
    if closeMatchItem.ItemId != 0 {
        return closeMatchItem, true
    }
    
    return items.Item{}, false
}

// Add item to storage
func (s *Storage) AddItem(i items.Item) bool {
    if i.ItemId < 1 {
        return false
    }
    s.Items = append(s.Items, i)
    return true
}

// Remove specific item instance
func (s *Storage) RemoveItem(i items.Item) bool {
    for j := len(s.Items) - 1; j >= 0; j-- {
        if s.Items[j].Equals(i) {
            s.Items = append(s.Items[:j], s.Items[j+1:]...)
            return true
        }
    }
    return false
}
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
```

## User Data Management

### Temporary Data Storage
```go
// Set temporary session data
func (u *UserRecord) SetTempData(key string, value any) {
    if u.tempDataStore == nil {
        u.tempDataStore = make(map[string]any)
    }
    
    if value == nil {
        delete(u.tempDataStore, key)
        return
    }
    
    u.tempDataStore[key] = value
}

// Get temporary session data
func (u *UserRecord) GetTempData(key string) any {
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

// Configuration options stored in ConfigOptions map
// Examples:
// - "auto_attack": bool
// - "brief_mode": bool
// - "color_enabled": bool
// - "sound_enabled": bool
```

## Online User Information

### Online Status Tracking
```go
type OnlineInfo struct {
    Username      string // Login username
    CharacterName string // Character display name
    Level         int    // Character level (deprecated in DOGMud — always 0)
    Alignment     string // Character alignment
    Profession    string // Character profession
    OnlineTime    int64  // Seconds online
    OnlineTimeStr string // Formatted time string
    IsAFK         bool   // Away from keyboard status
    Role          string // User role (guest/user/admin)
}
```

## Integration Patterns

### Character System Integration
```go
// Users have associated characters
- user.Character                    // Full character data
- user.Character.Name              // Character name
- user.Character.Level             // Deprecated (always 0 in DOGMud)
- user.Character.RoomId            // Current location
```

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
- user.activePrompt               // Current prompt state
- user.inputBlocked               // Input processing control
- prompt.Ask()                    // Interactive prompts
```

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
// Authenticate user login
username := "player1"
password := "secretpass"

// Look up user by username
userId, err := userIndex.GetByUsername(username)
if err != nil {
    return errors.New("user not found")
}

// Load user record
user, err := LoadUser(userId)
if err != nil {
    return err
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
userId := getNextUserId()

user := users.NewUserRecord(userId, connectionId)
user.Username = "newplayer"
user.Password = util.Hash("password123")
user.Character.Name = "NewPlayer"

// Save user
if err := user.Save(); err != nil {
    return err
}

// Add to active users
userManager.Users[userId] = user
userManager.Usernames[user.Username] = userId
userManager.Connections[connectionId] = userId
userManager.UserConnections[userId] = connectionId
```

### Item Storage Operations
```go
// Store item in user's personal storage
sword := items.New(101) // Create sword item
if user.ItemStorage.AddItem(sword) {
    user.SendText("Item stored successfully.")
}

// Retrieve item from storage
storedItem, found := user.ItemStorage.FindItem("sword")
if found {
    user.Character.GiveItem(storedItem)
    user.ItemStorage.RemoveItem(storedItem)
    user.SendText("Item retrieved from storage.")
}
```

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
recipient.SendText("You have a new message!")

// Check unread messages
unreadCount := recipient.Inbox.CountUnread()
if unreadCount > 0 {
    recipient.SendText(fmt.Sprintf("You have %d unread messages.", unreadCount))
}
```

### Zombie Connection Cleanup
```go
// Handle disconnected users
currentTurn := util.GetTurnCount()
expirationTurn := currentTurn - 100 // 100 turns ago

expiredZombies := users.GetExpiredZombies(expirationTurn)
for _, userId := range expiredZombies {
    user := users.GetByUserId(userId)
    if user != nil {
        // Save user data
        user.Save()
        
        // Remove from active users
        users.RemoveUser(userId)
        
        fmt.Printf("Cleaned up zombie user: %s\n", user.Username)
    }
}
```

## Presence Machine Integration (chunk 5)

The Presence machine lives on `Character`, not `UserRecord`, but the
player-side connection lifecycle is wired primarily inside the users
package:

- **`users.go` — TCP-close observer:** `LogOutUserByConnectionId` calls
  `u.Character.Presence.TransitionTo(presence.Disconnected, ...)` with
  `TriggerTCPClosed` when a connection is dropped. This fires the
  scheduler-cancel observer automatically.
- **`users.go` — Login-complete:** `HandleJoin` (or its callee
  `loginCharacterTo`) fires `Presence.TransitionTo(presence.Active, ...)`
  with `TriggerEnteredRoom` when the character is placed in its starting
  room, advancing from `Connecting` to `Active`.
- **`userrecord.go` — Input wake:** `SetLastInputRound` calls
  `Presence.TransitionTo(presence.Active, ...)` with
  `TriggerInputReceived` on every non-Disconnected state transition. This
  replaces the old manual-AFK clear shim that read `ManualAFK` directly.

### OnlineInfo.IsAFK (compat shim)

`OnlineInfo.IsAFK` continues to exist on the struct used by `online.go`
for the player list. In chunk 5 it is populated by:

```go
IsAFK: u.Character.Presence.State() == presence.AFK,
```

replacing the old ad-hoc computation
`ManualAFK || (roundNow - lastInputRound >= afkRounds)`. The `online`
command display and any `(afk)` marker in room descriptions still work
identically — the Presence machine is the only thing that changed.

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
- `gopkg.in/yaml.v2` - YAML serialization for user data persistence
- `internal/state/presence` - Presence state machine (chunk 5 — lives on
  Character but connection lifecycle wired here)

This comprehensive users system provides robust user account management
with authentication, connection tracking, data persistence, messaging,
and seamless integration with all game systems while maintaining high
performance through efficient indexing and caching.