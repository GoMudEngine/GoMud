# GoMud Connections System Context

## Overview

The GoMud connections system provides comprehensive network connection management with support for both traditional telnet and WebSocket connections. It features connection lifecycle management, input handling with history, heartbeat monitoring for WebSocket connections, client settings management, and thread-safe connection operations with graceful shutdown support.

## Architecture

The connections system is built around several key components:

### Core Components

**Connection Management:**
- Thread-safe connection tracking with unique IDs
- Dual protocol support (telnet and WebSocket)
- Connection state management (Login, LoggedIn, Zombie)
- Automatic connection cleanup and resource management

**Input Processing:**
- Chainable input handler system with state management
- Command history with navigation (up/down arrows)
- Special key handling (Enter, Backspace, Tab)
- Input buffering and clipboard support

**Client Settings:**
- Screen dimension tracking and defaults
- MSP (MUD Sound Protocol) support detection
- Telnet protocol option management
- Display preference configuration

**Heartbeat System:**
- WebSocket connection monitoring with ping/pong
- Configurable timeout and interval settings
- Automatic connection cleanup on timeout
- Thread-safe heartbeat management

## Key Features

### 1. **Dual Protocol Support**
- **Telnet Connections**: Traditional MUD protocol with full terminal control
- **WebSocket Connections**: Modern web-based connections with heartbeat monitoring
- **Unix Socket Support**: Local connections for development and administration
- **Protocol Detection**: Automatic handling based connection type

### 2. **Advanced Input Management**
- **Handler Chaining**: Multiple input processors with configurable order
- **Command History**: 10-command history with navigation
- **Special Key Support**: Enter, Backspace, Tab handling
- **Input Buffering**: Real-time input processing with buffer management

### 3. **Connection State Management**
- **Login State**: Initial connection before authentication
- **LoggedIn State**: Authenticated and active connections
- **Zombie State**: Disconnected but not yet cleaned up
- **Thread-Safe Operations**: All connection operations are mutex-protected

### 4. **Robust Heartbeat System**
- **WebSocket Monitoring**: Automatic ping/pong for connection health
- **Configurable Timeouts**: Customizable ping intervals and pong wait times
- **Graceful Degradation**: Automatic cleanup on connection failure
- **Resource Management**: Proper goroutine cleanup on disconnection

## Connection Structure

### Connection Details Structure
```go
type ConnectionDetails struct {
    connectionId      ConnectionId    // Unique connection identifier
    state             ConnectState    // Current connection state
    lastInputTime     time.Time       // Last input received timestamp
    conn              net.Conn        // Raw network connection
    wsConn            *websocket.Conn // WebSocket connection (if applicable)
    wsLock            sync.Mutex      // WebSocket write synchronization
    handlerMutex      sync.Mutex      // Input handler synchronization
    inputHandlerNames []string        // Handler names for management
    inputHandlers     []InputHandler  // Handler function chain
    inputDisabled     bool            // Input processing toggle
    clientSettings    ClientSettings  // Client configuration
    heartbeat         *heartbeatManager // WebSocket heartbeat manager
}
```

### Client Input Structure
```go
type ClientInput struct {
    ConnectionId  ConnectionId // Connection identifier
    DataIn        []byte       // Raw input data
    Buffer        []byte       // Current input buffer
    Clipboard     []byte       // Clipboard content for paste operations
    LastSubmitted []byte       // Previous command for reference
    EnterPressed  bool         // Enter key detection
    BSPressed     bool         // Backspace key detection
    TabPressed    bool         // Tab key detection
    History       InputHistory // Command history management
}
```

### Client Settings Structure
```go
type ClientSettings struct {
    Display           DisplaySettings // Screen dimensions and display options
    MSPEnabled        bool           // MUD Sound Protocol support
    SendTelnetGoAhead bool           // Telnet IAC GA after prompts
}

type DisplaySettings struct {
    ScreenWidth  uint32 // Terminal width (default: 80)
    ScreenHeight uint32 // Terminal height (default: 24)
}
```

## Connection Management

### Connection Lifecycle
```go
// Add new connection to management system
func Add(conn net.Conn, wsConn *websocket.Conn) *ConnectionDetails {
    lock.Lock()
    defer lock.Unlock()
    
    connectCounter++
    
    connDetails := NewConnectionDetails(
        connectCounter,
        conn,
        wsConn,
        nil, // Default heartbeat config
    )
    
    netConnections[connDetails.ConnectionId()] = connDetails
    
    return connDetails
}

// Remove connection and cleanup resources
func Remove(id ConnectionId) error {
    lock.Lock()
    defer lock.Unlock()
    
    if cd, ok := netConnections[id]; ok {
        cd.Close()                    // Close underlying connection
        disconnectCounter++           // Track disconnection
        delete(netConnections, id)    // Remove from tracking
        return nil
    }
    
    return errors.New("connection not found")
}

// Forcibly disconnect with reason
func Kick(id ConnectionId, reason string) error {
    lock.Lock()
    defer lock.Unlock()
    
    if cd, ok := netConnections[id]; ok {
        cd.Close()
        disconnectCounter++
        mudlog.Info("connection kicked", 
            "connectionId", id, 
            "remoteAddr", cd.RemoteAddr().String(), 
            "reason", reason)
        return nil
    }
    
    return errors.New("connection not found")
}
```

### Connection Discovery
```go
// Get connection by ID
func Get(id ConnectionId) *ConnectionDetails {
    lock.Lock()
    defer lock.Unlock()
    return netConnections[id]
}

// Check if connection is WebSocket
func IsWebsocket(id ConnectionId) bool {
    lock.Lock()
    defer lock.Unlock()
    
    if cd, ok := netConnections[id]; ok {
        return cd.IsWebSocket()
    }
    return false
}

// Get all active connection IDs
func GetAllConnectionIds() []ConnectionId {
    lock.Lock()
    defer lock.Unlock()
    
    ids := make([]ConnectionId, len(netConnections))
    for id := range netConnections {
        ids = append(ids, id)
    }
    return ids
}

// Get connection statistics
func Stats() (connections uint64, disconnections uint64) {
    lock.RLock()
    defer lock.RUnlock()
    return connectCounter, disconnectCounter
}
```

## Input Handling System

### Input Handler Chain
```go
type InputHandler func(ci *ClientInput, handlerState map[string]any) (doNextHandler bool)

// Add input handler to chain
func (cd *ConnectionDetails) AddInputHandler(name string, newInputHandler InputHandler, after ...string) {
    cd.handlerMutex.Lock()
    defer cd.handlerMutex.Unlock()
    
    // Insert after specific handler if specified
    if len(after) > 0 {
        for i, handlerName := range cd.inputHandlerNames {
            if handlerName == after[0] {
                cd.inputHandlerNames = append(cd.inputHandlerNames[:i+1], 
                    append([]string{name}, cd.inputHandlerNames[i+1:]...)...)
                cd.inputHandlers = append(cd.inputHandlers[:i+1], 
                    append([]InputHandler{newInputHandler}, cd.inputHandlers[i+1:]...)...)
                return
            }
        }
    }
    
    // Append to end of chain
    cd.inputHandlerNames = append(cd.inputHandlerNames, name)
    cd.inputHandlers = append(cd.inputHandlers, newInputHandler)
}

// Remove input handler from chain
func (cd *ConnectionDetails) RemoveInputHandler(name string) {
    cd.handlerMutex.Lock()
    defer cd.handlerMutex.Unlock()
    
    for i := len(cd.inputHandlerNames) - 1; i >= 0; i-- {
        if cd.inputHandlerNames[i] == name {
            cd.inputHandlerNames = append(cd.inputHandlerNames[:i], cd.inputHandlerNames[i+1:]...)
            cd.inputHandlers = append(cd.inputHandlers[:i], cd.inputHandlers[i+1:]...)
        }
    }
}
```

### Input Processing
```go
// Process input through handler chain
func (cd *ConnectionDetails) HandleInput(ci *ClientInput, handlerState map[string]any) (doNextHandler bool, lastHandler string, err error) {
    cd.handlerMutex.Lock()
    defer cd.handlerMutex.Unlock()
    
    cd.lastInputTime = time.Now()
    
    if len(cd.inputHandlers) < 1 {
        return false, lastHandler, errors.New("no input handlers")
    }
    
    // Execute handler chain
    for i, inputHandler := range cd.inputHandlers {
        lastHandler = cd.inputHandlerNames[i]
        if runNextHandler := inputHandler(ci, handlerState); !runNextHandler {
            return false, lastHandler, nil
        }
    }
    
    return true, lastHandler, nil
}

// Reset input state after processing
func (ci *ClientInput) Reset() {
    ci.DataIn = ci.DataIn[:0]
    ci.Buffer = ci.Buffer[:0]
    ci.EnterPressed = false
}
```

## Command History System

### History Management
```go
type InputHistory struct {
    inhistory bool     // Currently navigating history
    position  int      // Current position in history
    history   [][]byte // Command history buffer (max 10)
}

// Add command to history
func (ih *InputHistory) Add(input []byte) {
    // Maintain maximum history size
    if len(ih.history) >= MaxHistory {
        ih.history = ih.history[1:]
    }
    
    // Add new command
    ih.history = append(ih.history, make([]byte, len(input)))
    ih.position = len(ih.history) - 1
    copy(ih.history[ih.position], input)
    ih.inhistory = false
}

// Navigate to previous command
func (ih *InputHistory) Previous() {
    if !ih.inhistory {
        ih.inhistory = true
        return
    }
    if ih.position <= 0 {
        return
    }
    ih.position--
}

// Navigate to next command
func (ih *InputHistory) Next() {
    if !ih.inhistory {
        ih.inhistory = true
        return
    }
    if ih.position >= len(ih.history)-1 {
        return
    }
    ih.position++
}

// Get current history command
func (ih *InputHistory) Get() []byte {
    if len(ih.history) < 1 {
        return nil
    }
    return ih.history[ih.position]
}
```

## Communication System

### Broadcasting and Messaging
```go
// Broadcast message to all connections (with exclusions)
func Broadcast(colorizedText []byte, skipConnectionIds ...ConnectionId) []ConnectionId {
    lock.Lock()
    
    removeIds := []ConnectionId{}
    sentToIds := []ConnectionId{}
    
    for id, cd := range netConnections {
        // Skip login state connections
        if cd.state == Login {
            continue
        }
        
        // Skip excluded connections
        skip := false
        for _, skipId := range skipConnectionIds {
            if skipId == id {
                skip = true
                break
            }
        }
        if skip {
            continue
        }
        
        // Send message
        if _, err := cd.Write(colorizedText); err != nil {
            mudlog.Warn("Broadcast()", "connectionId", id, "error", err)
            removeIds = append(removeIds, id)
        } else {
            sentToIds = append(sentToIds, id)
        }
    }
    
    lock.Unlock()
    
    // Cleanup failed connections
    for _, id := range removeIds {
        Remove(id)
    }
    
    return sentToIds
}

// Send message to specific connections
func SendTo(b []byte, ids ...ConnectionId) {
    lock.Lock()
    
    removeIds := []ConnectionId{}
    
    for _, id := range ids {
        if cd, ok := netConnections[id]; ok {
            if _, err := cd.Write(b); err != nil {
                mudlog.Warn("SendTo()", "connectionId", id, "error", err)
                removeIds = append(removeIds, id)
            }
        }
    }
    
    lock.Unlock()
    
    // Cleanup failed connections
    for _, id := range removeIds {
        Remove(id)
    }
}
```

### Protocol-Specific I/O
```go
// Write data with protocol handling
func (cd *ConnectionDetails) Write(p []byte) (n int, err error) {
    // Convert line endings for all protocols
    p = []byte(strings.ReplaceAll(string(p), "\n", "\r\n"))
    
    if len(p) == 0 {
        return 0, nil
    }
    
    if cd.wsConn != nil {
        cd.wsLock.Lock()
        defer cd.wsLock.Unlock()
        
        // Prevent telnet commands to WebSocket
        if p[0] == term.TELNET_IAC {
            mudlog.Error("conn.Write", "error", "Trying to send telnet command to websocket!")
            return 0, nil
        }
        
        err := cd.wsConn.WriteMessage(websocket.TextMessage, p)
        if err != nil {
            return 0, err
        }
        return len(p), nil
    }
    
    return cd.conn.Write(p)
}

// Read data with protocol handling
func (cd *ConnectionDetails) Read(p []byte) (n int, err error) {
    if cd.wsConn != nil {
        _, message, err := cd.wsConn.ReadMessage()
        if err != nil {
            return 0, err
        }
        copy(p, message)
        return len(message), nil
    }
    
    return cd.conn.Read(p)
}
```

## Heartbeat System

### WebSocket Heartbeat Management
```go
type HeartbeatConfig struct {
    PongWait   time.Duration // Maximum time to wait for pong
    PingPeriod time.Duration // Ping interval (90% of PongWait)
    WriteWait  time.Duration // Write timeout for control messages
}

var DefaultHeartbeatConfig = HeartbeatConfig{
    PongWait:   60 * time.Second,
    PingPeriod: (60 * time.Second * 9) / 10, // 54 seconds
    WriteWait:  10 * time.Second,
}

// Start heartbeat monitoring for WebSocket
func (cd *ConnectionDetails) StartHeartbeat(config HeartbeatConfig) error {
    if cd.wsConn == nil {
        return ErrNotWebsocket
    }
    
    hm := newHeartbeatManager(cd, config)
    
    // Set up pong handler
    cd.wsConn.SetReadDeadline(time.Now().Add(hm.config.PongWait))
    cd.wsConn.SetPongHandler(func(string) error {
        mudlog.Debug("Heartbeat::Pong", "connectionId", hm.cd.connectionId)
        cd.wsConn.SetReadDeadline(time.Now().Add(hm.config.PongWait))
        return nil
    })
    
    // Start ping loop
    hm.wg.Add(1)
    go hm.runPingLoop()
    
    return nil
}

// Ping loop for connection monitoring
func (hm *heartbeatManager) runPingLoop() {
    defer hm.wg.Done()
    ticker := time.NewTicker(hm.config.PingPeriod)
    defer ticker.Stop()
    
    for {
        select {
        case <-hm.stopChan:
            return
        case <-ticker.C:
            if err := hm.writePing(); err != nil {
                mudlog.Warn("Heartbeat::Error", "connectionId", hm.cd.connectionId, "error", err)
                return
            }
        }
    }
}
```

## Client Settings Management

### Settings Configuration
```go
// Get client settings for connection
func GetClientSettings(id ConnectionId) ClientSettings {
    lock.Lock()
    defer lock.Unlock()
    
    if cd, ok := netConnections[id]; ok {
        return cd.clientSettings
    }
    
    return ClientSettings{} // Return defaults
}

// Update client settings
func OverwriteClientSettings(id ConnectionId, cs ClientSettings) {
    lock.Lock()
    defer lock.Unlock()
    
    if cd, ok := netConnections[id]; ok {
        cd.clientSettings = cs
    }
}

// Display settings with defaults
func (c DisplaySettings) GetScreenWidth() int {
    if c.ScreenWidth == 0 {
        return DefaultScreenWidth // 80
    }
    return int(c.ScreenWidth)
}

func (c DisplaySettings) GetScreenHeight() int {
    if c.ScreenHeight == 0 {
        return DefaultScreenHeight // 24
    }
    return int(c.ScreenHeight)
}
```

## Connection State Management

### State Transitions
```go
type ConnectState uint32

const (
    Login    ConnectState = iota // Initial connection state
    LoggedIn                     // Authenticated and active
    Zombie                       // Disconnected but not cleaned up
)

// Thread-safe state management
func (cd *ConnectionDetails) State() ConnectState {
    return ConnectState(atomic.LoadUint32((*uint32)(&cd.state)))
}

func (cd *ConnectionDetails) SetState(state ConnectState) {
    atomic.StoreUint32((*uint32)(&cd.state), uint32(state))
}

// Input control
func (cd *ConnectionDetails) InputDisabled(setTo ...bool) bool {
    if len(setTo) > 0 {
        cd.inputDisabled = setTo[0]
    }
    return cd.inputDisabled
}
```

### Connection Properties
```go
// Check if connection is local
func (cd *ConnectionDetails) IsLocal() bool {
    var remoteAddrStr string
    
    if cd.wsConn == nil {
        // Unix sockets are always local
        if _, ok := cd.conn.(*net.UnixConn); ok {
            return true
        }
        remoteAddrStr = cd.conn.RemoteAddr().String()
    } else {
        remoteAddrStr = cd.wsConn.RemoteAddr().String()
    }
    
    host, _, err := net.SplitHostPort(remoteAddrStr)
    if err != nil {
        return false
    }
    
    ip := net.ParseIP(host)
    if ip == nil {
        return false
    }
    
    return ip.IsLoopback()
}
```

## Shutdown and Cleanup

### Graceful Shutdown
```go
// Set shutdown signal channel
func SetShutdownChan(osSignalChan chan os.Signal) {
    lock.Lock()
    defer lock.Unlock()
    
    if shutdownChannel != nil {
        panic("Can't set shutdown channel a second time!")
    }
    shutdownChannel = osSignalChan
}

// Signal shutdown to all systems
func SignalShutdown(s os.Signal) {
    if shutdownChannel != nil {
        shutdownChannel <- s
    }
}

// Cleanup all connections
func Cleanup() {
    for _, id := range GetAllConnectionIds() {
        Remove(id)
    }
}
```

## Usage Examples

### Basic Connection Management
```go
// Accept new connection
conn, err := listener.Accept()
if err != nil {
    return err
}

// Add to connection management
connDetails := connections.Add(conn, nil) // nil for telnet
fmt.Printf("New connection: %d\n", connDetails.ConnectionId())

// Set up input handlers
connDetails.AddInputHandler("auth", authHandler)
connDetails.AddInputHandler("game", gameHandler, "auth")

// Set connection state
connDetails.SetState(connections.LoggedIn)
```

### WebSocket Connection Setup
```go
// Upgrade HTTP to WebSocket
wsConn, err := upgrader.Upgrade(w, r, nil)
if err != nil {
    return err
}

// Add WebSocket connection
connDetails := connections.Add(nil, wsConn)

// Heartbeat is automatically started for WebSocket connections
// Custom heartbeat config can be provided during NewConnectionDetails
```

### Input Handler Implementation
```go
// Example input handler
func gameInputHandler(ci *connections.ClientInput, handlerState map[string]any) bool {
    if ci.EnterPressed {
        command := string(ci.Buffer)
        
        // Add to history
        ci.History.Add(ci.Buffer)
        
        // Process game command
        processGameCommand(ci.ConnectionId, command)
        
        // Reset input
        ci.Reset()
        
        return false // Stop handler chain
    }
    
    return true // Continue to next handler
}

// Add handler to connection
connDetails.AddInputHandler("game", gameInputHandler)
```

### Broadcasting Messages
```go
// Broadcast to all logged-in users
message := []byte("Server announcement: Maintenance in 5 minutes!")
sentTo := connections.Broadcast(message)
fmt.Printf("Message sent to %d connections\n", len(sentTo))

// Send to specific users (excluding sender)
userMessage := []byte("You have a new message!")
connections.SendTo(userMessage, userId1, userId2)

// Broadcast excluding specific user
announcement := []byte("Player John has joined the game!")
connections.Broadcast(announcement, johnConnectionId)
```

## Dependencies

- `net` - Network connection handling and address management
- `sync` - Thread synchronization and atomic operations
- `github.com/gorilla/websocket` - WebSocket protocol implementation
- `internal/mudlog` - Logging system for connection events and debugging
- `internal/term` - Terminal control codes and telnet protocol handling

This comprehensive connections system provides robust network connection management with support for multiple protocols, advanced input processing, heartbeat monitoring, and thread-safe operations for reliable MUD server networking.
## Files

| File | Purpose |
|------|---------|
| `connections.go` | The connection registry and lifecycle |
| `connectiondetails.go` | Per-connection state |
| `clientsettings.go` | Negotiated client capabilities |
| `heartbeat.go` | Keepalive / zombie detection |
| `copyover.go` | Handing sockets across a hot restart |
| `fd_unix.go` / `fd_windows.go` | Platform file-descriptor handling |

The `fd_*` split is what makes copyover possible: a hot restart passes live
file descriptors to the new process so telnet sessions survive it.
