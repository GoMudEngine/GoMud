# Bad Input Tracker Context

## Overview

The `internal/badinputtracker` package provides a thread-safe system for tracking and analyzing invalid or unrecognized user commands in the GoMud game engine. This helps administrators identify common user input patterns that might indicate missing commands or user confusion.

## Key Components

### Core Files
- **badinputtracker.go**: Main bad input tracking functionality
- **badinputtracker_test.go**: Comprehensive unit tests for all tracking functions

### Key Functions

#### Command Tracking
- **TrackBadCommand(cmd string, rest string)**: Records an invalid command attempt
  - Thread-safe using mutex locking
  - Tracks both the command and its arguments separately
  - Increments counter for repeated identical bad commands
  - Creates nested map structure: `badCommands[cmd][rest]`

#### Data Retrieval
- **GetBadCommands() map[string]int**: Returns flattened view of all tracked bad commands
  - Thread-safe read operation
  - Combines command and arguments into single string key format: `"cmd rest"`
  - Returns copy of data with occurrence counts
  - Safe for concurrent access

#### Data Management
- **Clear()**: Resets all tracked bad command data
  - Thread-safe operation using mutex
  - Completely reinitializes the badCommands map
  - Used for periodic cleanup or testing

### Global State
- **badCommands**: `map[string]map[string]int` - Nested map tracking command failures
  - First level key: command name
  - Second level key: command arguments/rest
  - Value: occurrence count
- **lock**: `sync.Mutex` - Ensures thread-safe access to badCommands map

## Data Structure Design

### Nested Map Structure
```go
badCommands = map[string]map[string]int{
    "unknowncmd": {
        "arg1 arg2": 3,
        "different args": 1,
    },
    "anotherbad": {
        "some args": 2,
    },
}
```

### Flattened Output Format
```go
map[string]int{
    "unknowncmd arg1 arg2": 3,
    "unknowncmd different args": 1,
    "anotherbad some args": 2,
}
```

## Thread Safety

### Concurrency Design
- **Mutex Protection**: All operations protected by sync.Mutex
- **Read/Write Safety**: Prevents data races between tracking and retrieval
- **Lock Scope**: Minimal lock duration to reduce contention
- **Safe Iteration**: GetBadCommands creates a copy to avoid holding locks during iteration

## Usage Patterns

### Tracking Bad Commands
```go
// Track a bad command attempt
badinputtracker.TrackBadCommand("unknowncmd", "some arguments")

// Track repeated attempts (increments counter)
badinputtracker.TrackBadCommand("unknowncmd", "some arguments")
```

### Retrieving Analytics
```go
// Get all bad commands with counts
badCommands := badinputtracker.GetBadCommands()
for cmdWithArgs, count := range badCommands {
    log.Printf("Bad command '%s' attempted %d times", cmdWithArgs, count)
}
```

### Periodic Cleanup
```go
// Clear tracking data (e.g., daily reset)
badinputtracker.Clear()
```

## Integration Points

### Command Processing
- Called when user input doesn't match any valid command
- Integrated into main command parsing pipeline
- Helps identify gaps in command coverage

### Administrative Tools
- Data accessible via admin commands for analysis
- Used in web admin interface for usage statistics
- Helps guide development of new commands

### Logging and Monitoring
- Provides data for system monitoring
- Helps identify user experience issues
- Supports usage pattern analysis

## Testing Coverage

### Comprehensive Test Suite
- **Single Command Tracking**: Basic functionality verification
- **Multiple Command Handling**: Different commands and arguments
- **Counter Increment**: Repeated command tracking
- **Complex Scenarios**: Mixed command patterns
- **Clear Functionality**: Data reset verification
- **Thread Safety**: Concurrent access patterns

### Test Scenarios
- Empty state handling
- Single vs multiple command tracking
- Increment behavior for identical commands
- Mixed command and argument combinations
- Clear operation verification

## Performance Considerations

### Memory Usage
- **Bounded Growth**: Limited by unique command/argument combinations
- **Efficient Storage**: Nested map structure minimizes memory overhead
- **Cleanup Strategy**: Manual clearing prevents unbounded growth

### Lookup Performance
- **O(1) Access**: Map-based storage for fast lookups
- **Minimal Locking**: Short critical sections for good concurrency
- **Copy Strategy**: GetBadCommands creates copy to minimize lock time

## Error Handling

### Graceful Degradation
- **No Panics**: All operations handle edge cases gracefully
- **Empty State**: Safe handling of empty or uninitialized state
- **Concurrent Safety**: No data corruption under concurrent access

## Administrative Use Cases

### Usage Analysis
- Identify commonly mistyped commands
- Discover missing command aliases
- Analyze user behavior patterns
- Guide user interface improvements

### System Monitoring
- Track system usage patterns
- Identify potential bot or automated activity
- Monitor user experience quality
- Support capacity planning
