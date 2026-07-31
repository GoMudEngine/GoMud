# Language and Internationalization Context

## Overview

The `internal/language` package provides internationalization (i18n) support for the GoMud game engine. It manages translation bundles, localization, and multi-language message handling using the go-i18n library with YAML-based translation files.

## Key Components

### Core Files
- **language.go**: Translation system implementation and configuration
- **language_test.go**: Comprehensive unit tests for translation functionality

### Key Structures

#### BundleCfg
```go
type BundleCfg struct {
    DefaultLanguage language.Tag
    Language        language.Tag
    LanguagePaths   []string
}
```
Configuration structure for translation bundle setup:
- **DefaultLanguage**: Fallback language when translations are missing
- **Language**: Primary language for the system
- **LanguagePaths**: File paths to translation YAML files

#### Translation
```go
type Translation struct {
    bundle          *i18n.Bundle
    bundleCfg       BundleCfg
    localizerByLng  map[language.Tag]*i18n.Localizer
    defaultLanguage language.Tag
}
```
Main translation management structure containing:
- **bundle**: i18n bundle for message management
- **bundleCfg**: Configuration settings
- **localizerByLng**: Per-language localizer instances
- **defaultLanguage**: Fallback language tag

### Global State
- **trans**: `*Translation` - Global translation instance
- **ErrMessageFallback**: Error indicating fallback to default language

## Core Functions

### Initialization
- **InitTranslation(c BundleCfg)**: Initializes global translation system
  - Creates new Translation instance with provided configuration
  - Sets up global translation state for system-wide access

- **NewTranslation(c BundleCfg) *Translation**: Creates new translation instance
  - Initializes i18n bundle with default language
  - Registers YAML unmarshaling for translation files
  - Sets up localizer map for language-specific instances
  - Loads translation files from configured paths

### Translation Loading
- **LoadTranslation(c BundleCfg)**: Loads translation files
  - Processes all configured language paths
  - Loads YAML translation files into bundle
  - Creates localizer instances for each language
  - Handles missing files gracefully

## Dependencies

### Internal Dependencies
- `internal/configs`: For accessing configuration file paths
- `internal/mudlog`: For logging translation operations

### External Dependencies
- `github.com/nicksnyder/go-i18n/v2/i18n`: Core internationalization library
- `golang.org/x/text/language`: Language tag support and parsing
- `gopkg.in/yaml.v2`: YAML file parsing for translation files

## Translation File Format

### YAML Structure
Translation files use standard go-i18n YAML format:
```yaml
welcome:
  other: "hello"

welcomeWithName:
  other: "hello {{.name}}"

welcomeWithAge:
  other: "{{.age}} years old"

pluralExample:
  one: "{{.count}} item"
  other: "{{.count}} items"
```

### Template Support
- **Variable Substitution**: `{{.variableName}}` syntax for dynamic content
- **Pluralization**: Different messages for singular/plural forms
- **Conditional Content**: Context-aware message selection

## Usage Patterns

### System Initialization
```go
config := language.BundleCfg{
    DefaultLanguage: language.English,
    Language:        language.English,
    LanguagePaths:   []string{"localize/en.yaml", "localize/zh.yaml"},
}
language.InitTranslation(config)
```

### Message Translation
```go
// Simple message
message := trans.Translate(language.English, "welcome", nil)

// Message with variables
data := map[any]any{"name": "Player"}
message := trans.Translate(language.English, "welcomeWithName", data)

// Message with pluralization
data := map[any]any{"count": 5}
message := trans.Translate(language.English, "itemCount", data)
```

## Integration Points

### Configuration System
- **Language Paths**: Integration with file path configuration
- **Default Language**: Configurable fallback language selection
- **Multi-Language Support**: Support for multiple simultaneous languages

### Game Engine Integration
- **User Messages**: Localized game messages and notifications
- **Help System**: Multi-language help documentation
- **Error Messages**: Localized error and warning messages
- **Interface Text**: Translated UI elements and prompts

### User Preferences
- **Language Selection**: Per-user language preference support
- **Dynamic Switching**: Runtime language switching capabilities
- **Fallback Handling**: Graceful degradation to default language

## Error Handling

### Graceful Degradation
- **Missing Translations**: Falls back to default language
- **Missing Files**: Continues operation with available translations
- **Invalid Templates**: Provides error indication while maintaining functionality
- **Malformed YAML**: Logs errors but doesn't crash system

### Error Types
- **ErrMessageFallback**: Indicates fallback to default language occurred
- **Template Errors**: Variable substitution or formatting issues
- **File Loading Errors**: Translation file access problems

## Performance Considerations

### Caching Strategy
- **Localizer Caching**: Per-language localizer instances cached
- **Bundle Optimization**: Single bundle instance for all languages
- **Memory Efficiency**: Efficient storage of translation data

### Loading Strategy
- **Startup Loading**: All translations loaded at system initialization
- **Lazy Localization**: Localizer instances created on demand
- **Batch Processing**: Efficient loading of multiple translation files

## Testing Coverage

### Test Scenarios
- **Basic Translation**: Simple message translation verification
- **Variable Substitution**: Template variable replacement testing
- **Pluralization**: Singular/plural form handling
- **Missing Translations**: Fallback behavior validation
- **Invalid Data**: Error handling for malformed input

### Test Data
- **Sample Translations**: Test YAML files with various message types
- **Edge Cases**: Boundary conditions and error scenarios
- **Performance Tests**: Load testing with large translation sets

## Multi-Language Support

### Supported Languages
- **English**: Default language with full coverage
- **Chinese**: Secondary language support (zh.yaml)
- **Extensible**: Easy addition of new languages through YAML files

### Language Features
- **Unicode Support**: Full Unicode character support
- **Right-to-Left**: Support for RTL languages (future enhancement)
- **Cultural Adaptation**: Locale-specific formatting and conventions
