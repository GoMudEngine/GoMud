# File Loader System Context

## Overview

The `internal/fileloader` package provides a comprehensive file loading and saving system for the GoMud game engine. It handles YAML-based configuration files with validation, supports both single files and batch operations, and includes concurrent processing capabilities for performance optimization.

## Key Components

### Core Files
- **fileloader.go**: Complete file loading and saving functionality with concurrent processing

### Key Interfaces

#### LoadableSimple
```go
type LoadableSimple interface {
    Validate() error  // General validation (or none)
    Filepath() string // Relative file path to some base directory
}
```
Basic interface for loadable data structures requiring validation and file path specification.

#### Loadable[K comparable]
```go
type Loadable[K comparable] interface {
    Id() K // Must be a unique identifier for the data
    LoadableSimple
}
```
Extended interface for loadable data with unique identifiers, enabling map-based collections.

#### ReadableGroupFS
```go
type ReadableGroupFS interface {
    fs.ReadFileFS
    AllFileSubSystems(yield func(fs.ReadFileFS) bool)
}
```
File system interface supporting both standard file reading and iteration over multiple file systems.

### Key Enumerations

#### SaveOption
```go
type SaveOption uint8

const (
    SaveCareful SaveOption = iota // Save backup and rename vs. overwriting
)
```
Options for controlling file save behavior, supporting safe atomic writes.

## Core Functions

### Single File Operations
- **LoadFlatFile[T LoadableSimple](path string) (T, error)**: Loads single YAML file
  - Validates file existence and type (.yaml extension required)
  - Unmarshals YAML content into provided type
  - Validates file path consistency with Filepath() method
  - Performs data validation through Validate() method
  - Returns comprehensive error information for failures

- **SaveFlatFile[T LoadableSimple](basePath string, dataUnit T, saveOptions ...SaveOption) error**: Saves single file
  - Creates directory structure as needed
  - Supports careful save option for atomic writes
  - Marshals data to YAML format
  - Handles file permissions and error recovery

### Batch Operations
- **LoadAllFlatFilesSimple[T LoadableSimple](basePath string, filePattern ...string) ([]T, error)**: Loads multiple files into slice
  - Recursively walks directory tree
  - Supports optional file pattern matching
  - Loads only .yaml files
  - Returns slice of loaded data structures

- **LoadAllFlatFiles[K comparable, T Loadable[K]](basePath string, filePattern ...string) (map[K]T, error)**: Loads files into map
  - Similar to simple version but creates map using Id() method
  - Validates unique IDs across all loaded files
  - Returns map for efficient lookups by ID
  - Prevents duplicate ID conflicts

- **SaveAllFlatFiles[K comparable, T Loadable[K]](basePath string, data map[K]T, saveOptions ...SaveOption) (int, error)**: Concurrent batch saving
  - Uses worker goroutines for parallel processing
  - Worker count matches GOMAXPROCS for optimal CPU utilization
  - Supports careful save option for all files
  - Returns count of successfully saved files
  - Uses atomic operations for thread-safe counting

## File Processing Features

### YAML Support
- **Exclusive Format**: Only supports .yaml file extension
- **Standard Library**: Uses gopkg.in/yaml.v2 for marshaling/unmarshaling
- **Validation**: Comprehensive YAML parsing error handling
- **Structure Preservation**: Maintains YAML formatting and structure

### Path Validation
- **Consistency Checking**: Validates Filepath() method matches actual file location
- **Cross-Platform**: Uses filepath.FromSlash for platform-independent paths
- **Directory Creation**: Automatically creates necessary directory structure
- **Relative Paths**: Supports relative path specifications

### Error Handling
- **Comprehensive Errors**: Detailed error messages with file paths and context
- **Wrapped Errors**: Uses github.com/pkg/errors for error context
- **Validation Errors**: Includes data validation errors in error chain
- **Type Information**: Includes type information in error messages

## Concurrent Processing

### Worker Pool Architecture
- **Dynamic Workers**: Worker count matches available CPU cores
- **Channel Communication**: Uses buffered channels for work distribution
- **Wait Groups**: Proper synchronization using sync.WaitGroup
- **Atomic Counters**: Thread-safe counting using sync/atomic

### Performance Optimization
- **Parallel Processing**: Concurrent file operations for batch saves
- **Memory Efficiency**: Streaming processing without loading all data at once
- **CPU Utilization**: Optimal worker count based on system capabilities
- **Error Isolation**: Individual file failures don't affect other operations

## Safe Save Operations

### Careful Save Mode
- **Atomic Writes**: Uses .new suffix for temporary files
- **Rename Operation**: Atomic rename to final filename
- **Power Loss Protection**: Prevents corruption during system failures
- **Rollback Capability**: Original file preserved until successful write

### File Safety
- **Directory Creation**: Ensures target directories exist
- **Permission Handling**: Appropriate file permissions (0777 for data files)
- **Error Recovery**: Graceful handling of filesystem errors
- **Consistency**: Maintains file system consistency during operations

## Dependencies

### External Dependencies
- `github.com/pkg/errors`: Enhanced error handling and wrapping
- `gopkg.in/yaml.v2`: YAML marshaling and unmarshaling
- Standard library: `fmt`, `io`, `io/fs`, `os`, `path/filepath`, `runtime`, `strings`, `sync`, `sync/atomic`

## Usage Patterns

### Single File Loading
```go
type ConfigData struct {
    Name string `yaml:"name"`
    Value int   `yaml:"value"`
}

func (c ConfigData) Validate() error { return nil }
func (c ConfigData) Filepath() string { return "config.yaml" }

config, err := fileloader.LoadFlatFile[ConfigData]("/path/to/config.yaml")
```

### Batch Loading with IDs
```go
type ItemData struct {
    ItemId int    `yaml:"id"`
    Name   string `yaml:"name"`
}

func (i ItemData) Id() int { return i.ItemId }
func (i ItemData) Validate() error { return nil }
func (i ItemData) Filepath() string { return fmt.Sprintf("items/%d.yaml", i.ItemId) }

items, err := fileloader.LoadAllFlatFiles[int, ItemData]("/path/to/items/")
```

### Safe Batch Saving
```go
count, err := fileloader.SaveAllFlatFiles("/path/to/items/", itemsMap, fileloader.SaveCareful)
```

## Integration Points

### Configuration System
- **Game Data Loading**: Loads rooms, items, mobs, spells, and other game content
- **Settings Management**: Configuration file loading and saving
- **Hot Reloading**: Support for runtime configuration updates

### Content Management
- **World Data**: Loading and saving of world content files
- **Player Data**: Character and user data persistence
- **Dynamic Content**: Runtime-generated content saving

### Plugin System
- **Module Loading**: Plugin configuration and data file loading
- **Overlay Support**: Multiple file system support for modular content
- **Content Validation**: Ensures plugin data integrity

## Performance Considerations

### Memory Management
- **Streaming Processing**: Processes files without loading entire datasets
- **Efficient Allocation**: Pre-allocated slices with reasonable capacity
- **Garbage Collection**: Minimal allocation during batch operations

### I/O Optimization
- **Concurrent Operations**: Parallel file I/O for batch operations
- **Efficient Marshaling**: Direct YAML processing without intermediate formats
- **Directory Walking**: Efficient filesystem traversal

## Error Recovery

### Validation Framework
- **Data Validation**: Comprehensive validation through Validate() interface
- **Path Validation**: Ensures file paths match expected locations
- **Type Safety**: Generic type system prevents runtime type errors

### Failure Handling
- **Partial Success**: Batch operations continue despite individual failures
- **Detailed Reporting**: Comprehensive error information for debugging
- **Rollback Safety**: Careful save mode prevents data corruption
