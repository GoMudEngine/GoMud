# Input Handlers System Context

## Overview

The `internal/inputhandlers` package provides comprehensive input processing and handling for the GoMud game engine. It manages user authentication, login flows, system commands, terminal protocol handling, and input validation through a sophisticated prompt-based system with multi-step workflows.

## Key Components

### Core Files
- **login.go**: User authentication and login finalization logic
- **login_prompt_handler.go**: Multi-step prompt system for interactive user input
- **systemcommands.go**: System-level command processing (quit, reload, shutdown)
- **signals.go**: Signal handling and terminal control
- **term_ansi.go**: ANSI escape sequence processing
- **term_iac.go**: Telnet IAC (Interpret As Command) protocol handling
- **cleanser.go**: Input sanitization and cleaning
- **echo.go**: Terminal echo control
- **inputhistory.go**: Command history management

### Key Structures

#### PromptHandlerState
```go
type PromptHandlerState struct {
    Steps            []*PromptStep
    CurrentStepIndex int
    Results          map[string]string
    OnComplete       CompletionFunc
    maskTemplate     string
}
```
Manages multi-step interactive prompts for login, character creation, and other workflows.

#### PromptStep
```go
type PromptStep struct {
    Key          string
    PromptTemplate string
    ValidationFunc ValidationFunc
    ConditionFunc  ConditionFunc
    DataFunc       DataFunc
    Masked         bool
}
```
Defines individual steps in interactive prompt sequences.

#### SystemCommandHelp
```go
type SystemCommandHelp struct {
    Description  string
    Details      string
    ExampleInput string
}
```
Documentation structure for system commands.

### Function Types
- **CompletionFunc**: `func(results map[string]string, sharedState map[string]any, clientInput *connections.ClientInput) bool`
- **ValidationFunc**: `func(input string, results map[string]string) (string, error)`
- **ConditionFunc**: `func(results map[string]string) bool`
- **DataFunc**: `func(results map[string]string) map[string]any`

## Core Functions

### Authentication System
- **FinalizeLoginOrCreate(results map[string]string, sharedState map[string]any, clientInput *connections.ClientInput) bool**: Completes login process
  - Handles both existing user login and new user creation
  - Manages duplicate login detection and user kicking
  - Integrates with user management system for authentication
  - Supports password validation and account creation

### System Commands
- **SystemCommandInputHandler(clientInput *connections.ClientInput, sharedState map[string]any) bool**: Processes system commands
  - Handles commands prefixed with "/" (e.g., /quit, /reload, /shutdown)
  - Provides administrative functionality during gameplay
  - Integrates with event system for server management
  - Supports graceful shutdown with countdown timers

- **trySystemCommand(cmd string, connectionId connections.ConnectionId) bool**: Executes system commands
  - Parses and validates system command syntax
  - Executes quit, reload, and shutdown operations
  - Provides feedback and confirmation for administrative actions

### Prompt System
- **Multi-Step Workflows**: Sophisticated prompt system for interactive input
  - Conditional step execution based on previous responses
  - Input validation with custom validation functions
  - Masked input for password fields
  - Dynamic data generation for prompts
  - Completion callbacks for workflow finalization

### Terminal Protocol Handling
- **ANSI Processing**: Handles ANSI escape sequences for terminal control
- **IAC Processing**: Telnet protocol IAC command handling
- **Echo Control**: Terminal echo management for password input
- **Signal Handling**: Terminal signal processing and control

## Input Processing Features

### Authentication Flow
- **User Identification**: Username validation and existence checking
- **Password Authentication**: Secure password verification
- **Account Creation**: New user account creation workflow
- **Duplicate Login Handling**: Detection and management of duplicate logins
- **Session Management**: Connection association with user accounts

### System Administration
- **Administrative Commands**: System-level commands for server management
- **Graceful Shutdown**: Controlled server shutdown with user notification
- **Hot Reloading**: Dynamic reloading of game data without restart
- **Connection Management**: User disconnection and session cleanup

### Input Validation
- **Sanitization**: Input cleaning and normalization
- **Validation Functions**: Custom validation for different input types
- **Error Handling**: Comprehensive error reporting and recovery
- **Security**: Protection against malicious input and injection attacks

### Terminal Compatibility
- **Protocol Support**: Multiple terminal protocol support (telnet, raw TCP)
- **ANSI Compatibility**: Full ANSI escape sequence support
- **Cross-Platform**: Compatible with various terminal emulators
- **Legacy Support**: Support for older terminal types and protocols

## Dependencies

### Internal Dependencies
- `internal/configs`: For accessing configuration settings
- `internal/connections`: For connection management and communication
- `internal/events`: For system event processing
- `internal/language`: For internationalization support
- `internal/mudlog`: For logging input processing operations
- `internal/templates`: For prompt template processing
- `internal/term`: For terminal control and protocol handling
- `internal/users`: For user management and authentication

### External Dependencies
- Standard library: `errors`, `fmt`, `net/mail`, `strconv`, `strings`, `syscall`, `time`

## Usage Patterns

### Login Flow Implementation
```go
// Set up multi-step login prompt
steps := []*PromptStep{
    {
        Key: "username",
        PromptTemplate: "login_username",
        ValidationFunc: validateUsername,
    },
    {
        Key: "password",
        PromptTemplate: "login_password",
        ValidationFunc: validatePassword,
        Masked: true,
    },
}

// Initialize prompt handler
state := &PromptHandlerState{
    Steps: steps,
    Results: make(map[string]string),
    OnComplete: FinalizeLoginOrCreate,
}
```

### System Command Processing
```go
// Handle system commands in input processing
if strings.HasPrefix(input, "/") {
    if trySystemCommand(input, connectionId) {
        // System command processed successfully
        return false // Stop further processing
    }
}
```

### Input Validation
```go
// Custom validation function
func validateEmail(input string, results map[string]string) (string, error) {
    input = strings.TrimSpace(input)
    if _, err := mail.ParseAddress(input); err != nil {
        return "", errors.New("invalid email format")
    }
    return input, nil
}
```

## Integration Points

### Connection Management
- **Protocol Handling**: Integration with telnet and WebSocket protocols
- **Session State**: Maintains connection-specific input state
- **Buffer Management**: Efficient input buffer handling and processing
- **Connection Lifecycle**: Input handling throughout connection lifecycle

### User Management
- **Authentication**: Seamless integration with user authentication system
- **Account Creation**: New user registration and account setup
- **Session Association**: Linking connections with user accounts
- **Permission Validation**: User permission checking for system commands

### Game Engine
- **Command Processing**: Integration with game command processing
- **Event System**: System command integration with event processing
- **Template System**: Dynamic prompt generation using templates
- **Internationalization**: Multi-language support for prompts and messages

### Administrative Tools
- **Server Management**: Administrative commands for server control
- **Hot Reloading**: Dynamic configuration and data reloading
- **Monitoring**: Input processing monitoring and logging
- **Debugging**: Debug tools for input processing analysis

## Performance Considerations

### Input Processing Efficiency
- **Buffer Management**: Efficient input buffer handling and reuse
- **Validation Caching**: Caching of validation results where appropriate
- **Protocol Optimization**: Optimized protocol handling for performance
- **Memory Management**: Efficient memory usage in input processing

### Concurrent Processing
- **Thread Safety**: Safe concurrent access to input processing state
- **Connection Isolation**: Isolated processing per connection
- **Resource Sharing**: Efficient sharing of common resources
- **Scalability**: Architecture supports high concurrent connection loads

## Error Handling and Recovery

### Input Error Management
- **Graceful Degradation**: Graceful handling of malformed input
- **Error Recovery**: Automatic recovery from input processing errors
- **User Feedback**: Clear error messages and recovery instructions
- **Logging**: Comprehensive error logging for debugging and analysis

### Connection Error Handling
- **Disconnect Handling**: Graceful handling of unexpected disconnections
- **Protocol Errors**: Recovery from protocol-level errors
- **Timeout Management**: Handling of connection timeouts and delays
- **Resource Cleanup**: Proper cleanup of resources on connection errors

## Testing and Validation

### Unit Testing
- **Input Validation**: Comprehensive testing of input validation functions
- **Protocol Handling**: Testing of terminal protocol processing
- **Authentication Flow**: Testing of complete authentication workflows
- **Error Scenarios**: Testing of error conditions and edge cases

### Integration Testing
- **End-to-End**: Complete input processing workflow testing
- **Protocol Compatibility**: Testing with various terminal emulators
- **Load Testing**: Performance testing under high connection loads
- **Security Testing**: Security vulnerability testing and validation

## Files

| File | Purpose |
|------|---------|
| `login.go` | The login/create flow, including `FinalizeLoginOrCreate` |
| `login_prompt_handler.go` | Login prompt state machine |
| `cleanser.go` | Input sanitising |
| `echo.go` | Echo control (password entry) |
| `inputhistory.go` | Per-connection command history |
| `term_iac.go` | Telnet IAC negotiation |
| `term_ansi.go` | ANSI sequence handling |
| `mssp.go` | MSSP crawler responses |
| `signals.go` | Connection signal handling |
| `systemcommands.go` | Out-of-band system commands |

**Account and IP ban rejection happens in `FinalizeLoginOrCreate`**
(`login.go`), reading `internal/moderation`. Any new path that creates a
session must perform the same check — the moderation package only stores and
answers, it does not enforce.
