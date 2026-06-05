# Template System Context

## Overview

The `internal/templates` package provides a comprehensive template processing system for the GoMud game engine. It handles text templating with ANSI color support, markdown processing, screen reader compatibility, caching, and multi-source file system integration for dynamic content generation.

## Key Components

### Core Files
- **templates.go**: Complete template processing and management system
- **templatesfunctions.go**: Template function library and utilities
- **templatesfunctions_test.go**: Comprehensive unit tests for template functions
- **layout.go**: Template layout and structure management
- **layout_test.go**: Layout system unit tests
- **name_description.go**: Name and description template utilities

### Key Structures

#### AnsiFlag
```go
type AnsiFlag uint8

const (
    AnsiTagsDefault AnsiFlag = iota // Do not parse tags
    AnsiTagsParse                   // Parse ansi tags before returning
    AnsiTagsStrip                   // Strip out all ansi tags
    AnsiTagsMono                    // Parse tags but strip color info
    AnsiTagsNone    = AnsiTagsDefault
)
```
Controls ANSI tag processing behavior for different output contexts.

#### templateConfig
```go
type templateConfig struct {
    ScreenReader bool     // Screen reader friendly templates
    AnsiFlags    AnsiFlag // ANSI processing configuration
}
```
Per-user template configuration for accessibility and display preferences.

#### cacheEntry
```go
type cacheEntry struct {
    tpl           *template.Template
    ansiPreparsed bool
    modified      time.Time
}
```
Cached template with metadata for efficient reuse and cache invalidation.

### Global State
- **templateCache**: `map[string]cacheEntry` - Compiled template cache
- **templateConfigCache**: `sync.Map` (key: int userId → value: templateConfig) — per-user configuration cache; concurrency-safe, accessed via the `getTemplateConfig(userId)` helper
- **fileSystems**: `[]fs.ReadFileFS` - Registered file systems for template loading
- **forceAnsiFlags**: Global ANSI flag override
- **ansiLock**: Read-write mutex for thread-safe ANSI processing

## Core Functions

### Template Processing
- **Process(fname string, data any, receivingUserId ...int) (string, error)**: Main template processing function
  - Loads and compiles templates with caching
  - Applies user-specific configuration (screen reader, ANSI preferences)
  - Processes markdown content with dividers
  - Handles ANSI tag parsing based on user settings
  - Supports template inheritance and layout systems

### File System Management
- **RegisterFS(f fs.ReadFileFS)**: Registers additional file systems
  - Enables plugin-based template overrides
  - Supports multiple template sources with priority ordering
  - Used for modular template systems and customization

- **Exists(name string) bool**: Checks template existence across file systems
  - Searches registered file systems first (plugins)
  - Falls back to core data files
  - Enables conditional template loading

### Configuration Management
- **SetAnsiFlag(flag AnsiFlag)**: Sets global ANSI processing override
- **ClearTemplateConfigCache(userId int)**: Clears user-specific template cache
  - Used when user preferences change
  - Forces reloading of user-specific template configuration

### Markdown Integration
- **processMarkdown(in string) string**: Processes markdown content
  - Converts markdown to ANSI-formatted text
  - Adds decorative dividers for visual separation
  - Integrates with markdown package for rich text formatting

## Template Features

### ANSI Color Support
- **Dynamic Processing**: ANSI tags processed based on user preferences
- **Screen Reader Mode**: Strips colors for accessibility
- **Mono Mode**: Preserves formatting without colors
- **Full Color**: Complete ANSI color and formatting support

### Caching System
- **Template Compilation**: Compiled templates cached for performance
- **Cache Invalidation**: File modification time tracking for automatic updates
- **User Configuration**: Per-user configuration caching
- **Thread Safety**: Concurrent access protection with mutexes

### Accessibility Features
- **Screen Reader Support**: Alternative templates for screen reader users
- **ANSI Stripping**: Removes color codes for text-only output
- **Alternative Layouts**: Screen reader optimized template variants
- **Configurable Output**: User-controllable output formatting

### Multi-Source Loading
- **Plugin Templates**: Templates from plugin file systems
- **Core Templates**: Built-in game templates
- **Priority System**: Plugin templates override core templates
- **Fallback Mechanism**: Graceful fallback to available templates

## Dependencies

### Internal Dependencies
- `internal/colorpatterns`: For color pattern application in templates
- `internal/configs`: For accessing configuration file paths
- `internal/fileloader`: For template file loading operations
- `internal/markdown`: For markdown processing and formatting
- `internal/mudlog`: For logging template operations
- `internal/users`: For user preference and configuration access
- `internal/util`: For utility functions and file path operations

### External Dependencies
- `github.com/GoMudEngine/ansitags`: For ANSI tag parsing and processing
- `github.com/mattn/go-runewidth`: For accurate text width calculations
- `gopkg.in/yaml.v2`: For YAML template configuration parsing
- Standard library: `bytes`, `fmt`, `io/fs`, `os`, `strconv`, `strings`, `sync`, `text/template`, `time`

## Usage Patterns

### Basic Template Processing
```go
// Process template with data
result, err := templates.Process("character_sheet", characterData, userId)
if err != nil {
    // Handle template processing error
}

// Process without user-specific settings
result, err := templates.Process("global_message", data)
```

### Plugin Template Registration
```go
// Register plugin file system for templates
templates.RegisterFS(pluginFileSystem)

// Check if template exists
if templates.Exists("custom_template") {
    // Use custom template
}
```

### User Configuration
```go
// Clear user template cache when preferences change
templates.ClearTemplateConfigCache(userId)

// Set global ANSI behavior
templates.SetAnsiFlag(templates.AnsiTagsStrip)
```

## Integration Points

### User Interface
- **Dynamic Content**: Generate dynamic game content from templates
- **User Customization**: Per-user template rendering preferences
- **Accessibility**: Screen reader compatible output generation
- **Rich Formatting**: ANSI color and formatting for enhanced display

### Plugin System
- **Template Overrides**: Plugins can override core templates
- **Custom Templates**: Plugins can add new template types
- **Asset Integration**: Templates can reference plugin assets
- **Modular Design**: Independent template systems per plugin

### Web Interface
- **HTML Generation**: Templates for web interface content
- **Dynamic Pages**: Server-side template rendering for web
- **Asset Integration**: Template-based asset serving
- **Responsive Design**: Template-based responsive layouts

### Game Systems
- **Character Sheets**: Dynamic character information display
- **Room Descriptions**: Rich room and environment descriptions
- **Item Information**: Detailed item descriptions and stats
- **Help System**: Formatted help and documentation content

## Performance Considerations

### Caching Strategy
- **Template Compilation**: Expensive compilation cached for reuse
- **File System Caching**: Reduce file system access through caching
- **User Configuration**: Cache user preferences to avoid repeated lookups
- **Memory Management**: Efficient memory usage for large template sets

### Concurrent Access
- **Thread Safety**: Mutex protection for concurrent template access
- **Read-Write Locks**: Optimized locking for read-heavy workloads
- **Cache Contention**: Minimize lock contention in high-traffic scenarios
- **Scalable Design**: Architecture supports high concurrent usage

### File System Optimization
- **Multi-Source Priority**: Efficient search across multiple file systems
- **Existence Checking**: Fast template existence validation
- **Lazy Loading**: Templates loaded only when needed
- **Modification Tracking**: Efficient cache invalidation based on file changes

## Security Considerations

### Template Safety
- **Input Sanitization**: Prevent template injection attacks
- **Data Validation**: Validate template data before processing
- **Path Traversal**: Prevent unauthorized file access through template names
- **Resource Limits**: Prevent resource exhaustion through template processing

### User Data Protection
- **Configuration Privacy**: Protect user template preferences
- **Data Isolation**: Ensure user data doesn't leak between templates
- **Access Control**: Validate user permissions for template access
- **Audit Logging**: Log template access for security monitoring

## Future Enhancements

### Advanced Features
- **Template Inheritance**: Hierarchical template systems with inheritance
- **Conditional Rendering**: Advanced conditional logic in templates
- **Internationalization**: Multi-language template support
- **Dynamic Loading**: Runtime template compilation and loading

### Performance Improvements
- **Precompilation**: Ahead-of-time template compilation
- **Streaming**: Streaming template output for large content
- **Compression**: Compressed template storage and transmission
- **CDN Integration**: Content delivery network integration for templates

### Enhanced Accessibility
- **Voice Output**: Integration with text-to-speech systems
- **Braille Support**: Braille-friendly template formatting
- **High Contrast**: High contrast template variants
- **Font Scaling**: Scalable font support in templates

### Developer Tools
- **Template Editor**: Visual template editing tools
- **Debug Mode**: Template debugging and profiling tools
- **Validation**: Template syntax and data validation
- **Hot Reload**: Development-time template hot reloading