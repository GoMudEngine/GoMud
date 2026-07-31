# Logging System Context

## Overview

The `internal/mudlog` package provides a comprehensive structured logging system for the GoMud game engine. It features colored console output, file rotation, configurable log levels, and integration with in-game logging systems using Go's structured logging (slog) framework.

## Key Components

### Core Files
- **mudlog.go**: Main logging setup and configuration
- **loghandler.go**: Custom log handler with color formatting and tee functionality

### Key Structures

#### LogHandler
```go
type LogHandler struct {
    slog.Handler
    l                    *log.Logger
    minimumMessageLength int
    lTee                 teeLogger
    noColorHandler       *slog.TextHandler
}
```
Custom log handler providing:
- **Colored Output**: ANSI color codes for different log levels and data types
- **Tee Functionality**: Simultaneous output to multiple destinations
- **Message Formatting**: Consistent message padding and alignment
- **Attribute Processing**: Type-specific formatting for structured data

#### teeLogger Interface
```go
type teeLogger interface {
    Println(level string, v ...any)
}
```
Interface for additional log destinations, enabling in-game log display.

### Global State
- **slogInstance**: `*slog.Logger` - Global logger instance
- **logLevel**: `*slog.LevelVar` - Thread-safe log level control

## Core Functions

### Logger Setup
- **SetupLogger(inGameLogger teeLogger, logLevel string, logPath string, colorLogs bool)**: Main logger configuration
  - Configures file or stderr output based on logPath parameter
  - Sets up log rotation using lumberjack for file logging
  - Integrates tee logger for in-game log display
  - Handles directory validation and file creation

- **SetLogLevel(lvl string)**: Dynamic log level configuration
  - Supports string-based level setting ("M" = Info, "L" = Warn, default = Debug)
  - Thread-safe level changes during runtime
  - Case-insensitive level specification

### Logging Functions
- **Debug(msg string, args ...any)**: Debug level logging
- **Info(msg string, args ...any)**: Info level logging  
- **Warn(msg string, args ...any)**: Warning level logging
- **Error(msg string, args ...any)**: Error level logging

All logging functions support structured logging with key-value pairs.

## Log Formatting Features

### Color Coding System
- **Log Levels**: Different colors for Debug (magenta), Info (green), Warn (yellow), Error (red)
- **Data Types**: Type-specific colors for strings (yellow), numbers (red), booleans (bright green), etc.
- **Special Fields**: Error fields highlighted with red background
- **Timestamps**: Gray timestamps in [HH:MM:SS] format

### Message Formatting
- **Consistent Padding**: Messages padded to maintain column alignment
- **Attribute Formatting**: Key-value pairs with proper spacing and colors
- **Line Handling**: Newlines escaped in string values for clean output
- **Length Limits**: Messages truncated to 24 characters for consistency

### Structured Data Handling
- **Type-Aware Formatting**: Different colors and formatting based on data type
- **String Escaping**: Newlines and carriage returns properly escaped
- **Error Highlighting**: Special formatting for error fields
- **Duration/Time**: Special formatting for temporal values

## File Logging Features

### Log Rotation
- **Automatic Rotation**: Files rotated at 100MB size limit
- **Backup Retention**: Keeps 10 old log files
- **Compression**: Rotated files compressed to save space
- **Path Validation**: Ensures log directory exists and path is valid

### File Safety
- **Directory Validation**: Checks log directory exists before writing
- **Path Verification**: Ensures log path is not a directory
- **Error Handling**: Comprehensive error reporting for file issues
- **Atomic Operations**: Safe file operations to prevent corruption

## Dependencies

### Internal Dependencies
- None - serves as base logging for other internal packages

### External Dependencies
- `github.com/natefinch/lumberjack`: Log rotation and management
- Standard library: `context`, `fmt`, `io`, `log`, `log/slog`, `os`, `path/filepath`, `strings`, `time`

## Usage Patterns

### Basic Logging
```go
mudlog.Info("Server started", "port", 8080, "version", "1.0.0")
mudlog.Error("Connection failed", "error", err, "address", "127.0.0.1")
mudlog.Debug("Processing request", "user", "player1", "command", "look")
```

### Logger Setup
```go
// File logging with rotation
mudlog.SetupLogger(inGameLogger, "INFO", "/var/log/gomud.log", true)

// Console logging only
mudlog.SetupLogger(nil, "DEBUG", "", true)

// Change log level at runtime
mudlog.SetLogLevel("WARN")
```

### Structured Data
```go
mudlog.Info("Player action",
    "player", playerName,
    "action", "attack",
    "target", targetName,
    "damage", damageAmount,
    "success", true,
)
```

## Integration Points

### Game Engine Integration
- **System Events**: Logs server startup, shutdown, and configuration changes
- **Player Actions**: Tracks player commands and interactions
- **Error Reporting**: Comprehensive error logging with context
- **Performance Monitoring**: Logs timing and performance metrics

### In-Game Display
- **Tee Logger**: Displays logs in-game for administrators
- **Real-Time Monitoring**: Live log viewing through game interface
- **Filtered Display**: Different log levels for different audiences

### External Systems
- **Log Aggregation**: Compatible with external log collection systems
- **Monitoring**: Structured format suitable for monitoring tools
- **Debugging**: Detailed logging for development and troubleshooting

## Performance Considerations

### Efficient Processing
- **Structured Logging**: Uses slog for efficient structured data handling
- **Color Caching**: Efficient ANSI color code generation
- **String Building**: Uses strings.Builder for efficient string concatenation
- **Conditional Processing**: Respects log levels to avoid unnecessary work

### Memory Management
- **Log Rotation**: Prevents unbounded log file growth
- **Efficient Formatting**: Minimal allocations during log formatting
- **Reusable Handlers**: Single handler instance for all logging operations

## Configuration Options

### Log Levels
- **DEBUG**: All messages including detailed debugging information
- **INFO**: General information and system events
- **WARN**: Warning messages and non-critical issues
- **ERROR**: Error conditions and failures

### Output Destinations
- **File Logging**: Rotated log files with compression
- **Console Output**: Colored output to stderr
- **Tee Logging**: Simultaneous output to multiple destinations
- **No Color Mode**: Plain text output for log aggregation systems

### Formatting Options
- **Color Control**: Enable/disable color output
- **Timestamp Format**: Configurable time display format
- **Message Padding**: Consistent message alignment
- **Attribute Formatting**: Structured data presentation

## Error Handling

### Setup Validation
- **Path Validation**: Ensures log paths are valid and accessible
- **Directory Checking**: Verifies log directories exist
- **Permission Validation**: Checks write permissions for log files
- **Configuration Errors**: Clear error messages for setup issues

### Runtime Safety
- **Panic Prevention**: Graceful handling of logging errors
- **Fallback Behavior**: Falls back to stderr if file logging fails
- **Error Recovery**: Continues operation despite logging issues
