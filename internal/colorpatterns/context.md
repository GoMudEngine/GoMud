# Color Patterns System Context

## Overview

The `internal/colorpatterns` package provides a sophisticated color pattern application system for the GoMud game engine. It enables dynamic text colorization using predefined color patterns with multiple application methods, supporting ANSI color codes and preserving existing formatting.

## Key Components

### Core Files
- **colorpatterns.go**: Complete color pattern management and application system

### Key Enumerations

#### ColorizeStyle
```go
type ColorizeStyle uint8
```
Defines different methods for applying color patterns:
- **Default** (0): Per-character coloring with reversing pattern
- **Words** (1): Per-word coloring
- **Once** (2): Progressive coloring that stops at final color
- **Stretch** (3): Stretches pattern across entire string length

### Global State
- **numericPatterns**: `map[string][]int` - Loaded color patterns with numeric color codes
- **ShortTagPatterns**: `map[string][]string` - Compiled short tag versions of patterns
- **colorsCompiled**: `bool` - Compilation status flag

## Core Functions

### Pattern Management
- **LoadColorPatterns()**: Loads color patterns from `color-patterns.yaml` file
  - Clears existing patterns and recompiles
  - Logs loading time and pattern count
  - Displays test output for each loaded pattern
  - Panics on file read or YAML parsing errors

- **CompileColorPatterns()**: Converts numeric patterns to short tag format
  - Creates `{colorcode}` format tags from numeric values
  - Sets compilation flag to prevent duplicate work
  - Populates ShortTagPatterns map

- **GetColorPatternNames() []string**: Returns sorted list of available pattern names
  - Ensures patterns are compiled before returning
  - Provides alphabetically sorted pattern list

- **IsValidPattern(pName string) bool**: Validates pattern name existence

### Pattern Application
- **ApplyColorPattern(input, pattern string, method ...ColorizeStyle) string**: Main pattern application function
  - Applies named color pattern to input text
  - Supports optional colorization method
  - Returns original text if pattern doesn't exist

- **ApplyColors(input string, patternValues []int, method ...ColorizeStyle) string**: Core colorization engine
  - Handles all four colorization methods
  - Preserves existing ANSI tags through tokenization
  - Manages pattern progression and reversal logic

## Colorization Methods

### Default Method
- **Character-by-Character**: Colors each non-space character individually
- **Reversing Pattern**: Reaches end of pattern then reverses direction
- **Space Handling**: Spaces don't advance pattern position
- **Bidirectional**: Creates wave-like color effects

### Words Method
- **Word-by-Word**: Changes color at word boundaries (spaces)
- **Cycling Pattern**: Loops through pattern colors for each word
- **Continuous**: Wraps around pattern when reaching end

### Once Method
- **Progressive Coloring**: Advances through pattern once
- **Final Color Lock**: Stays on final pattern color
- **Linear Progression**: No reversal or cycling

### Stretch Method
- **Distributed Pattern**: Spreads pattern across entire string length
- **Calculated Intervals**: Uses string width to determine color change points
- **Proportional**: Adjusts pattern application based on text length

## ANSI Tag Preservation

### Tokenization System
- **Regex Detection**: Identifies existing `<ansi ...>...</ansi>` tags
- **Temporary Replacement**: Replaces tags with unique numeric placeholders
- **Pattern Application**: Applies colors without affecting existing tags
- **Tag Restoration**: Restores original tags after colorization

### Placeholder Handling
- **Format**: Uses `:number` format for temporary placeholders
- **Collision Avoidance**: Incremental counter prevents conflicts
- **Safe Processing**: Skips colorization of placeholder content

## Dependencies

### Internal Dependencies
- `internal/configs`: For accessing file path configurations
- `internal/mudlog`: For logging pattern loading operations
- `github.com/GoMudEngine/ansitags`: For ANSI tag parsing and display

### External Dependencies
- `gopkg.in/yaml.v2`: For YAML configuration parsing
- `github.com/mattn/go-runewidth`: For accurate string width calculation
- `github.com/pkg/errors`: For error wrapping
- Standard library: `fmt`, `math`, `os`, `regexp`, `sort`, `strconv`, `strings`, `time`

## Configuration File Format

### color-patterns.yaml Structure
```yaml
pattern_name:
  - 196  # Red
  - 208  # Orange
  - 226  # Yellow
  - 46   # Green
rainbow:
  - 196
  - 202
  - 208
  - 214
  - 220
  - 226
  - 190
  - 154
  - 118
  - 82
  - 46
```

## Usage Patterns

### Basic Pattern Application
```go
// Apply default colorization
coloredText := colorpatterns.ApplyColorPattern("Hello World", "rainbow")

// Apply specific method
coloredText := colorpatterns.ApplyColorPattern("Hello World", "rainbow", colorpatterns.Words)
```

### Direct Color Application
```go
// Apply colors directly
colors := []int{196, 208, 226, 46}
coloredText := colorpatterns.ApplyColors("Hello World", colors, colorpatterns.Stretch)
```

### Pattern Validation
```go
if colorpatterns.IsValidPattern("rainbow") {
    // Apply the pattern
    result := colorpatterns.ApplyColorPattern(text, "rainbow")
}
```

## Performance Considerations

### Memory Management
- **Pattern Caching**: All patterns loaded into memory at startup
- **Compilation Caching**: Compiled patterns cached to avoid recompilation
- **String Building**: Efficient string concatenation using strings.Builder

### Processing Efficiency
- **Lazy Compilation**: Patterns compiled only when first accessed
- **Regex Optimization**: Single regex compile for ANSI tag detection
- **Width Calculation**: Accurate text width for stretch method

## Integration Points

### Game Engine Integration
- **Text Formatting**: Used throughout game for colorful text output
- **User Interface**: Enhances visual appeal of game messages
- **Customization**: Players can choose color patterns for various elements

### Template System
- **Dynamic Coloring**: Integration with template rendering
- **Conditional Application**: Pattern application based on context
- **User Preferences**: Respects user color settings

## Error Handling

### Graceful Degradation
- **Missing Patterns**: Returns original text if pattern not found
- **Invalid Input**: Handles empty or malformed input gracefully
- **File Errors**: Panics on critical configuration loading failures

### Validation
- **Pattern Existence**: Validates pattern names before application
- **Method Validation**: Handles invalid colorization methods
- **Input Sanitization**: Safely processes user input

## Testing and Debugging

### Pattern Testing
- **Automatic Testing**: Displays test output for all patterns on load
- **Multiple Methods**: Tests each pattern with different colorization methods
- **Visual Verification**: Uses ansitags.Parse for immediate visual feedback

### Debug Information
- **Loading Metrics**: Logs pattern count and loading time
- **Pattern Listing**: Provides sorted list of available patterns
- **Compilation Status**: Tracks compilation state
