# GoMud Network Layer - Comprehensive Analysis

## Overview

The GoMud network layer provides a sophisticated, multi-protocol networking system that supports both traditional MUD clients via telnet and modern web-based clients via WebSocket. The architecture is designed for high concurrency, robust protocol handling, and extensive customization through a plugin system.

## Network Architecture

### Core Components

**Main Entry Point (`main.go`)**
- Primary network initialization and server startup
- Manages multiple concurrent network listeners
- Handles graceful shutdown and connection cleanup
- Coordinates between telnet and WebSocket protocols

**Connection Management (`internal/connections`)**
- Thread-safe connection tracking with unique IDs
- Dual protocol support (telnet and WebSocket)
- Connection state management (Login, LoggedIn, Zombie)
- Heartbeat monitoring for WebSocket connections
- Input processing pipeline with chainable handlers

**Web Server (`internal/web`)**
- HTTP/HTTPS server with WebSocket upgrade capability
- Administrative interface with authentication
- Template-based HTML rendering
- Plugin integration for custom web functionality

**Terminal Protocol (`internal/term`)**
- Comprehensive telnet protocol implementation
- ANSI escape sequence processing
- MUD Sound Protocol (MSP) support
- Cross-platform terminal compatibility

**Input Processing (`internal/inputhandlers`)**
- Multi-step authentication workflows
- System command processing
- Terminal protocol handling (IAC, ANSI)
- Input validation and sanitization

## Protocol Support

### Telnet Protocol
- **Default Ports**: 33333, 44444 (configurable)
- **Local Admin Port**: 9999 (localhost only, no connection limits)
- **Max Connections**: 100 (configurable)
- **Protocol Features**:
  - Full telnet option negotiation (WILL/WONT/DO/DONT)
  - Echo control for password input
  - Window size negotiation (NAWS)
  - Character encoding negotiation
  - Go-ahead suppression
  - Binary mode support

### WebSocket Protocol
- **Endpoint**: `/ws` on HTTP/HTTPS server
- **Upgrade**: Automatic HTTP to WebSocket upgrade
- **Heartbeat**: Ping/pong monitoring (60-second intervals)
- **Features**:
  - Real-time bidirectional communication
  - Cross-origin request support for development
  - Automatic connection health monitoring
  - Text masking for password input

### HTTP/HTTPS Server
- **HTTP Port**: 80 (configurable, 0 to disable)
- **HTTPS Port**: 0 (disabled by default, requires certificates)
- **HTTPS Redirect**: Optional automatic HTTP to HTTPS redirection
- **Features**:
  - Static file serving
  - Template-based dynamic content
  - Administrative interface with authentication
  - Plugin web integration

## Connection Management

### Connection Lifecycle

**Connection Establishment**:
1. Accept incoming connection (telnet or WebSocket)
2. Generate unique connection ID (atomic counter)
3. Initialize connection details structure
4. Set up input handler chain
5. Begin protocol negotiation (telnet) or heartbeat (WebSocket)

**Connection States**:
- **Login**: Initial state before authentication
- **LoggedIn**: Authenticated and active
- **Zombie**: Disconnected but not yet cleaned up (configurable timeout)

**Connection Tracking**:
- Thread-safe connection registry with RWMutex
- Unique connection IDs for identification
- Connection statistics (total connects/disconnects)
- Active connection count monitoring

### Input Processing Pipeline

**Handler Chain Architecture**:
- Chainable input processors with configurable order
- Each handler can abort or continue processing
- Shared state map for handler communication
- Handler-specific error handling and recovery

**Standard Handler Chain** (Telnet):
1. **TelnetIACHandler**: Telnet protocol command processing
2. **AnsiHandler**: ANSI escape sequence processing
3. **CleanserInputHandler**: Input sanitization
4. **LoginPromptHandler**: Multi-step authentication (initial)
5. **EchoInputHandler**: Terminal echo management (post-login)
6. **HistoryInputHandler**: Command history tracking
7. **SystemCommandInputHandler**: System commands (admin only)
8. **SignalHandler**: Terminal signal processing

**WebSocket Processing**:
- Simplified handler chain (no telnet/ANSI processing)
- Direct message processing
- Text masking for password fields
- Real-time input handling

## Authentication and Security

### Multi-Step Authentication
- **Username Validation**: Existence checking and format validation
- **Password Authentication**: Secure password verification
- **Account Creation**: New user registration workflow
- **Duplicate Login Handling**: Detection and management of concurrent sessions

### Security Features
- **Input Sanitization**: Protection against injection attacks
- **Rate Limiting**: Protection against input flooding
- **Authentication Caching**: 30-minute session caching for admin interface
- **Role-Based Access**: Admin/user role verification
- **Connection Limits**: Configurable maximum connections per protocol

### System Commands
- **Administrative Commands**: `/quit`, `/reload`, `/shutdown`
- **Permission Checking**: Admin role verification required
- **Graceful Operations**: Countdown timers for shutdown operations
- **Audit Logging**: Comprehensive logging of administrative actions

## Network Configuration

### Port Configuration
```yaml
Network:
  MaxTelnetConnections: 100
  TelnetPort: [33333, 44444]    # Multiple ports supported
  LocalPort: 9999               # Localhost admin access
  HttpPort: 80                  # Web server port
  HttpsPort: 0                  # HTTPS port (0 = disabled)
  HttpsRedirect: false          # Auto-redirect HTTP to HTTPS
  ZombieSeconds: 60             # Zombie connection timeout
  AfkSeconds: 1800              # AFK timeout
  TimeoutMods: false            # Timeout moderators/admins
```

### File Paths
```yaml
FilePaths:
  WebDomain: "localhost"
  WebCDNLocation: ""            # Optional CDN for static files
  PublicHtml: "_datafiles/html/public"
  AdminHtml: "_datafiles/html/admin"
  HttpsCertFile: ""             # TLS certificate
  HttpsKeyFile: ""              # TLS private key
```

## Advanced Features

### Heartbeat System (WebSocket)
- **Ping Interval**: 54 seconds (90% of pong timeout)
- **Pong Timeout**: 60 seconds
- **Write Timeout**: 10 seconds for control messages
- **Automatic Cleanup**: Connection removal on heartbeat failure
- **Thread Safety**: Goroutine-safe ping/pong handling

### Client Settings Management
- **Screen Dimensions**: Width/height tracking (default 80x24)
- **Protocol Capabilities**: MSP support detection
- **Display Preferences**: Color and formatting support
- **Terminal Type**: Client terminal identification

### Command History
- **History Size**: 10 commands maximum
- **Navigation**: Up/down arrow key support
- **Position Tracking**: Current history position management
- **Session Persistence**: History maintained per connection

### Input Buffer Management
- **Buffer Size**: 1024 bytes read buffer
- **Real-time Processing**: Character-by-character input handling
- **Special Keys**: Enter, Backspace, Tab detection
- **Clipboard Support**: Paste operation handling

## Plugin Integration

### Network Plugin Hooks
- **Connection Events**: `OnNetConnect` for new connections
- **IAC Command Processing**: Custom telnet command handling
- **Web Interface Extensions**: Custom admin pages and navigation
- **Command Registration**: Add custom user and system commands

### Plugin Capabilities
- **Web Pages**: Custom HTML pages with template processing
- **Navigation Links**: Add menu items to web interface
- **Static Assets**: Serve CSS, JS, images from plugin filesystem
- **Template Data**: Inject custom data into web templates

## Performance Characteristics

### Concurrency Model
- **Goroutine per Connection**: Each connection handled in separate goroutine
- **Thread-Safe Operations**: All connection operations use mutex protection
- **Non-Blocking I/O**: Asynchronous network operations
- **Resource Pooling**: Efficient buffer and resource management

### Scalability Features
- **Connection Limits**: Configurable per-protocol connection limits
- **Resource Management**: Automatic cleanup of failed connections
- **Memory Efficiency**: Minimal per-connection memory overhead
- **CPU Utilization**: Configurable CPU core usage

### Monitoring and Statistics
- **Connection Tracking**: Total connections and disconnections
- **Active Connections**: Real-time active connection count
- **Error Logging**: Comprehensive error tracking and reporting
- **Performance Metrics**: Connection timing and throughput monitoring

## Error Handling and Recovery

### Connection Error Management
- **Graceful Degradation**: Automatic handling of connection failures
- **Zombie State**: Temporary preservation of disconnected users
- **Automatic Cleanup**: Failed connection removal and resource cleanup
- **Error Logging**: Detailed error reporting with context

### Protocol Error Handling
- **Malformed Input**: Safe handling of invalid protocol data
- **Timeout Management**: Connection and operation timeouts
- **Buffer Overflow Protection**: Safe buffer handling
- **Recovery Mechanisms**: Automatic recovery from protocol errors

## Integration with Game Systems

### User Management Integration
- **Authentication**: Seamless integration with user database
- **Session Management**: Connection association with user accounts
- **Character Loading**: Automatic character data loading on login
- **Duplicate Detection**: Prevention of multiple concurrent logins

### Event System Integration
- **Connection Events**: Login/logout event generation
- **Input Events**: User command event processing
- **Broadcast Events**: Server-wide message distribution
- **Custom Events**: Plugin-generated network events

### World Manager Integration
- **Input Processing**: User command forwarding to game engine
- **Output Handling**: Game output formatting and transmission
- **State Synchronization**: Connection state with game state
- **Zombie Management**: Temporary character preservation

## Security Considerations

### Network Security
- **Input Validation**: Comprehensive input sanitization
- **Protocol Security**: Safe telnet and WebSocket handling
- **Connection Limits**: DoS protection through connection limiting
- **Authentication**: Secure multi-step authentication process

### Administrative Security
- **Role Verification**: Admin command access control
- **Audit Logging**: Complete administrative action logging
- **Session Management**: Secure admin session handling
- **Command Validation**: Strict system command syntax validation

## Deployment and Operations

### Server Startup Process
1. **Configuration Loading**: Network settings validation
2. **Port Binding**: Telnet and HTTP/HTTPS server startup
3. **Worker Initialization**: Input and main worker goroutines
4. **Plugin Loading**: Network plugin initialization
5. **Signal Handling**: Graceful shutdown signal registration

### Graceful Shutdown
1. **Connection Notification**: Broadcast shutdown message
2. **New Connection Blocking**: Prevent new connections
3. **Connection Cleanup**: Close all active connections
4. **Resource Cleanup**: Release network resources
5. **Worker Synchronization**: Wait for worker completion

### Monitoring and Maintenance
- **Connection Statistics**: Real-time connection monitoring
- **Error Tracking**: Network error logging and analysis
- **Performance Monitoring**: Connection timing and throughput
- **Resource Usage**: Memory and CPU utilization tracking

## User Input Processing Flow

### Input Processing Architecture

GoMud implements a sophisticated multi-stage input processing system that handles user commands through a series of channels, workers, and event processing stages. This design provides excellent separation of concerns, scalability, and extensibility.

### Input Flow Stages

#### Stage 1: Network Input Reception
**Location**: Connection handlers in `main.go`

**Telnet Flow**:
1. Raw bytes received from telnet connection
2. Input handler chain processes the data:
   - `TelnetIACHandler`: Processes telnet protocol commands
   - `AnsiHandler`: Handles ANSI escape sequences
   - `CleanserInputHandler`: Sanitizes input
   - `LoginPromptHandler`: Manages authentication (pre-login)
   - `EchoInputHandler`: Handles terminal echo (post-login)
   - `HistoryInputHandler`: Manages command history
   - `SystemCommandInputHandler`: Processes system commands (admin only)
   - `SignalHandler`: Handles terminal signals

**WebSocket Flow**:
1. WebSocket message received
2. Simplified handler chain (no telnet/ANSI processing)
3. Direct conversion to input event

#### Stage 2: World Input Channel
**Structure**: `WorldInput` struct
```go
type WorldInput struct {
    FromId    int     // User ID
    InputText string  // Raw command text
    ReadyTurn uint64  // Turn number when ready to process
}
```

**Channel**: `worldInput chan WorldInput`
- **Blocking Channel**: Synchronous communication between network and game layers
- **Thread Safety**: Single writer (network), single reader (InputWorker)
- **Backpressure**: Network connections block if game processing falls behind

#### Stage 3: Input Worker Processing
**Worker**: `InputWorker` goroutine in `world.go`

**Responsibilities**:
1. Receives `WorldInput` from network layer
2. Converts to `events.Input` event
3. Adds to event queue with current turn number
4. Provides isolation between network and game logic

**Code Flow**:
```go
case wi := <-w.worldInput:
    events.AddToQueue(events.Input{
        UserId:    wi.FromId,
        InputText: wi.InputText,
        ReadyTurn: util.GetTurnCount(),
    })
```

#### Stage 4: Event System Processing
**Event Type**: `events.Input`

**Event Structure**:
```go
type Input struct {
    UserId        int
    MobInstanceId int
    InputText     string
    ReadyTurn     uint64
    WaitTurns     int
    Flags         EventFlag
}
```

**Processing Logic**:
1. **Turn-Based Queuing**: Commands wait until their `ReadyTurn` is reached
2. **Input Blocking**: Commands can block further input with `CmdBlockInput` flag
3. **One Command Per Turn**: Only one command per user per turn processed
4. **Requeuing**: Commands not ready are requeued for later processing

#### Stage 5: Command Execution
**Function**: `processInput()` in `world.go`

**Processing Steps**:
1. **User Validation**: Verify user exists and is active
2. **Prompt Handling**: Process any active command prompts
3. **Macro Expansion**: Expand user-defined macros into multiple commands
4. **Command Parsing**: Split input into command and arguments
5. **Command Execution**: Try to execute via `usercommands.TryCommand()`
6. **Error Handling**: Track unrecognized commands and provide feedback

### Channel Architecture

#### Primary Channels

**World Input Channel**:
- **Type**: `chan WorldInput`
- **Purpose**: User command input from network layer
- **Flow**: Network → InputWorker → Event System

**Enter World Channel**:
- **Type**: `chan [2]int` (userId, roomId)
- **Purpose**: User login completion
- **Flow**: Authentication → MainWorker → Game World

**Leave World Channel**:
- **Type**: `chan int` (userId)
- **Purpose**: User logout/disconnect
- **Flow**: Network/System → MainWorker → Game World

**Logout Connection Channel**:
- **Type**: `chan connections.ConnectionId`
- **Purpose**: Connection-specific logout
- **Flow**: Network → MainWorker → User Management

**Zombie Flag Channel**:
- **Type**: `chan [2]int` (userId, flag)
- **Purpose**: Zombie state management
- **Flow**: Network/System → MainWorker → User Management

#### Worker Goroutines

**InputWorker**:
- **Purpose**: Convert network input to game events
- **Channels Read**: `worldInput`, `shutdown`
- **Thread Safety**: Single goroutine, no shared state
- **Lifecycle**: Started at server startup, stopped on shutdown

**MainWorker**:
- **Purpose**: Game state management and event processing
- **Channels Read**: Multiple management channels, timers
- **Responsibilities**:
  - Event loop processing
  - Turn/round timing
  - User lifecycle management
  - Room maintenance
  - Statistics updates

### Input Processing Features

#### Turn-Based Processing
- **Turn Counter**: Global turn counter incremented every `TurnMs` milliseconds
- **Command Queuing**: Commands wait for their designated turn
- **Fairness**: One command per user per turn prevents command flooding
- **Timing Control**: Commands can specify future turn execution

#### Input Blocking and Flow Control
- **Block Input Flag**: Commands can block further user input
- **Unblock Input Flag**: Commands can unblock previously blocked input
- **Ignore Map**: Tracks users with blocked input by turn number
- **Zombie Handling**: Special processing for disconnected users

#### Macro System
- **Macro Expansion**: Two-character macros expand to command sequences
- **Sequential Execution**: Macro commands executed in order with turn delays
- **Delimiter Support**: Semicolon-separated command sequences
- **Turn Scheduling**: Each macro command gets its own turn

#### Command Processing
- **Alias Resolution**: Commands resolved through keyword alias system
- **Shortcut Support**: Gossip shortcuts (`, .`) expanded automatically
- **Command Parsing**: Intelligent command/argument separation
- **Error Tracking**: Bad commands tracked for analytics

### Performance Characteristics

#### Scalability Features
- **Asynchronous Processing**: Network and game processing decoupled
- **Backpressure Handling**: Network blocks if game processing falls behind
- **Event Batching**: Multiple events processed per event loop iteration
- **Efficient Queuing**: Minimal memory allocation for event processing

#### Concurrency Safety
- **Channel Communication**: Thread-safe communication between layers
- **Single Writers**: Each channel has single writer goroutine
- **Game State Locking**: MUD state locked during processing
- **Atomic Operations**: Turn counters use atomic operations

#### Resource Management
- **Event Requeuing**: Efficient requeuing of delayed events
- **Memory Cleanup**: Event trackers cleared each turn
- **Connection Cleanup**: Failed connections automatically removed
- **Graceful Shutdown**: All workers synchronized on shutdown

### Integration Points

#### Network Layer Integration
- **Protocol Abstraction**: Same input flow for telnet and WebSocket
- **Handler Flexibility**: Configurable input handler chains
- **Error Propagation**: Network errors propagated to game layer
- **Connection State**: Input processing aware of connection state

#### Event System Integration
- **Event Priority**: Input events processed with appropriate priority
- **Event Flags**: Rich flag system for command behavior control
- **Event Requeuing**: Seamless requeuing of delayed commands
- **Event Tracking**: Per-user event tracking prevents command flooding

#### Game System Integration
- **User Management**: Direct integration with user authentication
- **Command System**: Seamless integration with command processing
- **Prompt System**: Interactive prompt handling
- **Statistics**: Input processing statistics and monitoring

This sophisticated input processing architecture provides excellent performance, scalability, and maintainability while ensuring fair command processing and robust error handling across all network protocols.

## Future Enhancements

### Protocol Extensions
- **Modern Protocols**: Enhanced WebSocket features
- **Compression**: Network traffic compression
- **Encryption**: Enhanced security protocols
- **Binary Protocols**: Efficient binary communication

### Performance Improvements
- **Connection Pooling**: Advanced connection management
- **Load Balancing**: Multi-server connection distribution
- **Caching**: Network operation caching
- **Optimization**: Protocol-specific optimizations

This comprehensive network layer provides a robust foundation for MUD server networking with excellent scalability, security, and extensibility through its plugin architecture. The dual-protocol support ensures compatibility with both traditional MUD clients and modern web-based interfaces while maintaining high performance and reliability.