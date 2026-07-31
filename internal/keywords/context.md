# Keywords and Aliases System Context

## Overview

The `internal/keywords` package provides a comprehensive alias and keyword management system for the GoMud game engine. It handles command aliases, help system organization, direction shortcuts, and map legend customization, supporting both base configuration and overlay systems for modular content.

## Key Components

### Core Files
- **keywords.go**: Complete keyword and alias management system

### Key Structures

#### Aliases
```go
type Aliases struct {
    Help               map[string]map[string][]string `yaml:"help"`
    HelpAliases        map[string][]string            `yaml:"help-aliases"`
    CommandAliases     map[string][]string            `yaml:"command-aliases"`
    DirectionAliases   map[string]string              `yaml:"direction-aliases"`
    MapLegendOverrides map[string]map[string]string   `yaml:"legend-overrides"`
    
    // Internal processed data
    helpTopics         map[string]HelpTopic
    helpAliases        map[string]string
    commandAliases     map[string]string
    mapLegendOverrides map[string]map[rune]string
}
```
Main structure containing all alias and keyword configurations with both raw YAML data and processed lookup maps.

#### HelpTopic
```go
type HelpTopic struct {
    Command   string
    Type      string // command/skill
    Category  string
    AdminOnly bool
}
```
Represents a help topic with categorization and access control information.

### Global State
- **loadedKeywords**: `*Aliases` - Currently loaded keyword configuration
- **fileSystems**: `[]fileloader.ReadableGroupFS` - File systems for overlay support

## Core Functions

### Configuration Loading
- **LoadAliases(f ...fileloader.ReadableGroupFS)**: Loads keyword configuration
  - Loads base keywords.yaml file from data directory
  - Registers additional file systems for overlay support
  - Processes and validates all alias data
  - Panics on loading failures

### Alias Resolution
- **TryCommandAlias(input string) string**: Resolves command aliases
  - Case-insensitive lookup of command shortcuts
  - Returns original input if no alias found
  - Supports both explicit command aliases and direction aliases

- **TryHelpAlias(input string) string**: Resolves help topic aliases
  - Case-insensitive lookup of help shortcuts
  - Returns original input if no alias found
  - Enables abbreviated help commands

- **TryDirectionAlias(input string) string**: Resolves direction shortcuts
  - Converts direction abbreviations to full directions
  - Case-insensitive processing
  - Returns original input if no alias found

### Data Retrieval
- **GetAllHelpTopics() []string**: Returns sorted list of all help topics
- **GetAllHelpTopicInfo() []HelpTopic**: Returns detailed help topic information
- **GetAllCommandAliases() map[string]string**: Returns copy of all command aliases
- **GetAllHelpAliases() map[string]string**: Returns copy of all help aliases
- **GetAllLegendAliases(area ...string) map[rune]string**: Returns map legend overrides
  - Supports global (`*`) and area-specific legend customization
  - Returns rune-to-string mapping for map symbols

## Configuration Structure

### Help System Organization
```yaml
help:
  commands:
    character: ["alignment", "experience", "stats"]
    combat: ["attack", "defend", "flee"]
  skills:
    magic: ["cast", "enchant", "scribe"]
  admin:
    server: ["shutdown", "reload", "backup"]
```

### Alias Definitions
```yaml
help-aliases:
  alignment: ["align", "al"]
  experience: ["exp", "xp"]

command-aliases:
  look: ["l", "examine", "ex"]
  inventory: ["i", "inv"]

direction-aliases:
  n: "north"
  s: "south"
  ne: "northeast"
```

### Map Legend Customization
```yaml
legend-overrides:
  "*":  # Global overrides
    "#": "wall"
    ".": "floor"
  frostfang:  # Area-specific overrides
    "^": "mountain peak"
    "~": "frozen river"
```

## Overlay System Support

### File System Integration
- **Multi-Source Loading**: Supports base configuration plus overlays
- **Modular Content**: Enables plugin-based keyword extensions
- **Merge Strategy**: Combines multiple alias sources into unified system
- **Validation**: Processes and validates merged configurations

### Overlay Processing
- **Data Merging**: Combines base and overlay alias definitions
- **Override Support**: Later sources can override earlier definitions
- **Error Tolerance**: Continues processing if individual overlays fail
- **Flexible Sources**: Supports various file system implementations

## Data Processing and Validation

### Validation Process
- **Merge Processing**: Combines base and overlay configurations
- **Case Normalization**: Converts all keys to lowercase for consistency
- **Structure Building**: Creates optimized lookup structures
- **Cross-Reference**: Links aliases to their target commands/topics

### Processed Data Structures
- **helpTopics**: Fast lookup of help topic information
- **helpAliases**: Direct alias-to-topic mapping
- **commandAliases**: Direct alias-to-command mapping
- **mapLegendOverrides**: Area-specific map symbol definitions

## Dependencies

### Internal Dependencies
- `internal/configs`: For accessing file path configurations
- `internal/fileloader`: For configuration file loading and overlay support
- `internal/util`: For utility functions like file path construction

### External Dependencies
- `gopkg.in/yaml.v2`: For YAML configuration parsing
- Standard library: `sort`, `strings`

## Usage Patterns

### Command Processing
```go
// Resolve user input through aliases
userInput := "l"  // User types "l"
actualCommand := keywords.TryCommandAlias(userInput)  // Returns "look"

// Process direction shortcuts
direction := "n"  // User types "n"
fullDirection := keywords.TryDirectionAlias(direction)  // Returns "north"
```

### Help System Integration
```go
// Resolve help aliases
helpQuery := "exp"  // User types "help exp"
actualTopic := keywords.TryHelpAlias(helpQuery)  // Returns "experience"

// Get all available help topics
topics := keywords.GetAllHelpTopics()
for _, topic := range topics {
    // Display available help topics
}
```

### Map Legend Customization
```go
// Get map symbols for specific area
legends := keywords.GetAllLegendAliases("frostfang")
for symbol, description := range legends {
    // Use custom symbol descriptions for map rendering
}
```

## Integration Points

### Command System
- **Input Processing**: Resolves user input through command aliases
- **Direction Handling**: Converts direction shortcuts to full commands
- **Extensibility**: Supports adding new command aliases dynamically

### Help System
- **Topic Organization**: Categorizes help topics by type and category
- **Access Control**: Supports admin-only help topics
- **Alias Support**: Enables abbreviated help commands

### Mapping System
- **Symbol Customization**: Allows area-specific map symbol definitions
- **Global Defaults**: Provides fallback symbol definitions
- **Dynamic Legends**: Supports runtime legend customization

### Plugin System
- **Overlay Support**: Enables plugins to add custom aliases
- **Modular Extensions**: Supports independent alias packages
- **Configuration Merging**: Combines multiple alias sources seamlessly

## Performance Considerations

### Lookup Efficiency
- **O(1) Lookups**: Hash map-based alias resolution
- **Preprocessed Data**: All aliases processed into optimized structures
- **Memory Efficiency**: Separate structures for different alias types

### Loading Strategy
- **Startup Loading**: All aliases loaded at server startup
- **Merge Processing**: Efficient combination of multiple sources
- **Validation Caching**: Processed data cached for runtime use

## Error Handling

### Graceful Degradation
- **Missing Aliases**: Returns original input if no alias found
- **Invalid Configuration**: Continues processing with partial data
- **Overlay Failures**: Handles individual overlay loading failures
- **Case Sensitivity**: Normalizes case for consistent behavior

### Validation
- **Structure Validation**: Ensures proper YAML structure
- **Data Integrity**: Validates alias mappings and references
- **Error Reporting**: Provides clear error messages for configuration issues
