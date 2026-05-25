# GoMud Configuration Management System Context

## Overview

The GoMud configuration system provides comprehensive, type-safe configuration management with YAML-based storage, runtime overrides, environment variable support, and validation. It supports hierarchical configuration structures, dot-notation access, and secure handling of sensitive data through a sophisticated type system and validation framework.

## Architecture

The configuration system is built around a centralized `Config` struct with several key components:

### Core Components

**Configuration Structure:**
- Hierarchical configuration organized into logical subsections (Server, Network, GamePlay, etc.)
- Type-safe configuration values using custom types (`ConfigString`, `ConfigInt`, `ConfigBool`, etc.)
- Automatic validation and default value enforcement
- Thread-safe access with read-write mutex protection

**Override System:**
- Runtime configuration overrides stored separately from base configuration
- Dot-notation path support for nested configuration access
- Persistent override storage in YAML format
- Automatic path correction and fuzzy matching for configuration keys

**Type System:**
- Custom configuration types with string conversion and validation
- Special `ConfigSecret` type that redacts sensitive values in output
- Automatic type inference and conversion from string values
- Support for complex types including slices and nested structures

**Validation Framework:**
- Per-subsection validation with range checking and defaults
- Locked configuration properties that cannot be changed at runtime
- Banned name patterns for user input validation
- Environment variable integration with automatic assignment

## Key Features

### 1. **Type-Safe Configuration**
- Custom configuration types prevent type errors and provide consistent interfaces
- Automatic validation and range checking for all configuration values
- Support for integers, floats, booleans, strings, slices, and secret values
- Type inference from string input for dynamic configuration updates

### 2. **Hierarchical Structure**
- Logical organization into subsections (Server, Network, GamePlay, FilePaths, etc.)
- Dot-notation access for nested configuration properties
- Automatic path resolution and correction for typos or partial matches
- Flattening and unflattening of nested structures for override management

### 3. **Runtime Configuration Management**
- Live configuration updates without server restart
- Persistent override storage separate from base configuration
- Thread-safe configuration access with mutex protection
- Configuration change validation and rollback on errors

### 4. **Security Features**
- `ConfigSecret` type automatically redacts sensitive values in logs and output
- Environment variable support for secure credential injection
- Locked configuration properties to prevent unauthorized changes
- Validation of user input against banned patterns

### 5. **Override System**
- Dot-notation configuration overrides (e.g., `Server.MaxCPUCores`)
- Persistent storage of overrides in separate YAML file
- Automatic path correction and fuzzy matching
- Override precedence over base configuration values

## Configuration Subsections

### Server Configuration
```yaml
Server:
  MudName: "My MUD Server"
  CurrentVersion: "1.0.0"
  Seed: "random-seed-value"        # ConfigSecret type
  MaxCPUCores: 4
  OnLoginCommands: ["look", "who"]
  Motd: "Welcome to the game!"
  NextRoomId: 1000
  Locked: ["Seed"]                 # Locked properties
```

### Network Configuration
```yaml
Network:
  MaxTelnetConnections: 50
  TelnetPort: ["4000", "4001"]
  LocalPort: 4002
  HttpPort: 80
  HttpsPort: 443
  HttpsRedirect: true
  AfkSeconds: 300
  MaxIdleSeconds: 1800
  TimeoutMods: false
  ZombieSeconds: 60
  LogoutRounds: 10
```

### GamePlay Configuration
```yaml
GamePlay:
  AllowItemBuffRemoval: true
  Death:
    CorpseDecayRounds: 100
  ShopRestockRate: "1h"
  ContainerSizeMax: 50
  MaxAltCharacters: 5
  ConsistentAttackMessages: true
  PVP: "limited"                   # enabled, disabled, limited, off
  PVPMinimumLevel: 10
  XPScale: 100.0
  MobConverseChance: 25
```

### File Paths Configuration
```yaml
FilePaths:
  DataFiles: "_datafiles"
  PublicHtml: "_datafiles/html/public"
  AdminHtml: "_datafiles/html/admin"
  HttpsCertFile: "cert.pem"
  HttpsKeyFile: "key.pem"
  WebCDNLocation: "/static"
  CarefulSaveFiles: true
```

## Configuration Types

### Basic Types
```go
type ConfigInt int           // Integer values with validation
type ConfigUInt64 uint64     // Unsigned 64-bit integers
type ConfigString string     // String values
type ConfigSecret string     // Automatically redacted strings
type ConfigFloat float64     // Floating-point values
type ConfigBool bool         // Boolean values
type ConfigSliceString []string  // String arrays
```

### Type Interface
```go
type ConfigValue interface {
    String() string    // String representation
    Set(string) error  // Set value from string
}
```

### Secret Type Behavior
```go
// ConfigSecret automatically redacts in output
func (c ConfigSecret) String() string {
    return `*** REDACTED ***`
}

// Access actual value through helper function
func GetSecret(v ConfigSecret) string {
    return string(v)
}
```

## Configuration Access Patterns

### Reading Configuration
```go
// Get complete configuration
config := configs.GetConfig()

// Access subsections
serverConfig := configs.GetServerConfig()
networkConfig := configs.GetNetworkConfig()
gameplayConfig := configs.GetGamePlayConfig()

// Access specific values
mudName := config.Server.MudName.String()
maxConnections := int(config.Network.MaxTelnetConnections)
pvpEnabled := config.GamePlay.PVP.String() == "enabled"
```

### Setting Configuration Values
```go
// Set configuration value by dot path
err := configs.SetVal("Server.MudName", "New Server Name")
err := configs.SetVal("Network.HttpPort", "8080")
err := configs.SetVal("GamePlay.PVP", "enabled")

// Configuration is automatically validated and saved
if err != nil {
    log.Printf("Configuration error: %v", err)
}
```

### Environment Variable Integration
```go
// Configuration fields can be populated from environment variables
type Server struct {
    DatabaseURL ConfigSecret `yaml:"DatabaseURL" env:"DATABASE_URL"`
    APIKey      ConfigSecret `yaml:"APIKey" env:"API_KEY"`
}

// Values are automatically loaded from environment on startup
```

## Override System

### Override File Format
```yaml
# config-overrides.yaml
Server:
  MudName: "Development Server"
  MaxCPUCores: 8
Network:
  HttpPort: 8080
  HttpsPort: 8443
GamePlay:
  PVP: "enabled"
  XPScale: 150.0
```

### Dot-Notation Access
```go
// All configuration paths support dot notation
allConfig := config.AllConfigData()
// Returns map with keys like:
// "Server.MudName" -> "My MUD Server"
// "Network.HttpPort" -> 80
// "GamePlay.PVP" -> "limited"

// Set values using dot notation
configs.SetVal("Server.MudName", "New Name")
configs.SetVal("GamePlay.Death.EquipmentDropChance", "0.1")
```

### Path Resolution and Correction
```go
// Automatic path correction for typos
fullPath, typeName := configs.FindFullPath("mudname")
// Returns: "Server.MudName", "configs.ConfigString"

fullPath, typeName := configs.FindFullPath("httpport")
// Returns: "Network.HttpPort", "configs.ConfigInt"

// Supports partial matches and case-insensitive lookup
```

## Validation System

### Per-Subsection Validation
```go
func (s *Server) Validate() {
    if s.Seed == `` {
        s.Seed = `Mud` // default value
    }
    
    if s.MaxCPUCores < 0 {
        s.MaxCPUCores = 0 // enforce minimum
    }
}

func (n *Network) Validate() {
    if n.MaxTelnetConnections < 1 {
        n.MaxTelnetConnections = 50 // default
    }
    
    if n.HttpPort < 0 {
        n.HttpPort = 0 // disable if negative
    }
}
```

### Banned Name Validation
```go
// Check if a name matches banned patterns
bannedPattern, isBanned := config.IsBannedName("testname")
if isBanned {
    return fmt.Errorf("Name matches banned pattern: %s", bannedPattern)
}
```

### Locked Configuration Properties
```go
// Some properties cannot be changed at runtime
func isEditAllowed(configPath string) bool {
    serverConfig := configs.GetServerConfig()
    for _, lockedPath := range serverConfig.Locked {
        if configPath == lockedPath {
            return false
        }
    }
    return true
}
```

## Configuration Loading and Persistence

### Startup Process
1. **Load Base Configuration**: Read `_datafiles/config.yaml`
2. **Load Overrides**: Read `config-overrides.yaml` if it exists
3. **Apply Environment Variables**: Set values from environment
4. **Validate Configuration**: Run all validation functions
5. **Build Lookup Tables**: Create path and type lookup maps

### Runtime Updates
```go
// Configuration changes are immediately persisted
err := configs.SetVal("Server.MudName", "New Name")
// This automatically:
// 1. Validates the new value
// 2. Updates the override file
// 3. Reloads the configuration
// 4. Validates the complete configuration
```

### Configuration Reload
```go
// Reload configuration from files
err := configs.ReloadConfig()
if err != nil {
    log.Printf("Failed to reload config: %v", err)
}
```

## Integration Examples

### User Command Integration
```go
// Server configuration command
func server_Config(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
    args := util.SplitButRespectQuotes(rest)
    
    if len(args) >= 2 {
        configName := strings.ToLower(args[0])
        configValue := strings.Join(args[1:], ` `)
        
        if err := configs.SetVal(configName, configValue); err != nil {
            user.SendText(fmt.Sprintf("Config error: %s=%s (%s)", configName, configValue, err))
            return true, nil
        }
        
        user.SendText(fmt.Sprintf("Configuration updated: %s=%s", configName, configValue))
        return true, nil
    }
    
    // Show current configuration
    allConfigData := configs.GetConfig().AllConfigData()
    // Display configuration options...
}
```

### Web Interface Integration
```go
// Template access to configuration
templateData := map[string]any{
    "CONFIG": configs.GetConfig(),
    "STATS":  GetStats(),
}

// In templates:
// {{.CONFIG.Server.MudName}}
// {{.CONFIG.Network.HttpPort}}
// {{.CONFIG.GamePlay.PVP}}
```

### Plugin Configuration
```go
// Plugin-specific configuration
func (p *PluginConfig) Set(name string, val any) {
    configs.SetVal(fmt.Sprintf(`Modules.%s.%s`, p.pluginName, name), fmt.Sprintf(`%v`, val))
}

func (p *PluginConfig) Get(name string) any {
    m := configs.Flatten(configs.GetModulesConfig())
    return m[fmt.Sprintf(`%s.%s`, p.pluginName, name)]
}
```

## Performance Considerations

### Configuration Caching
- Configuration is cached in memory with read-write mutex protection
- Changes trigger validation and persistence but don't require full reload
- Lookup tables provide O(1) access to configuration paths and types

### Thread Safety
```go
var configDataLock sync.RWMutex

func GetConfig() Config {
    configDataLock.RLock()
    defer configDataLock.RUnlock()
    
    if !configData.validated {
        configData.Validate()
    }
    return configData
}
```

### Validation Optimization
- Validation only runs when configuration changes
- Cached validation state prevents redundant validation calls
- Subsection validation allows targeted updates

## Security Features

### Secret Management
```go
// Secrets are automatically redacted in logs and output
type Server struct {
    DatabasePassword ConfigSecret `yaml:"DatabasePassword" env:"DB_PASSWORD"`
    APIKey          ConfigSecret `yaml:"APIKey" env:"API_KEY"`
}

// Access actual values securely
dbPassword := configs.GetSecret(config.Server.DatabasePassword)
```

### Configuration Locking
```go
// Prevent runtime changes to sensitive configuration
Server:
  Locked: ["Seed", "DatabasePassword", "APIKey"]
```

### Input Validation
```go
// Validate user input against banned patterns
Validation:
  BannedNames: ["admin*", "root", "system", "*test*"]
```

## Error Handling and Logging

### Configuration Errors
- Type conversion errors with detailed messages
- Path resolution errors with suggestions for correct paths
- Validation errors with specific constraint information
- File I/O errors with full context

### Logging Integration
```go
// Configuration changes are logged
mudlog.Info("SetVal", "path", propertyPath, "value", newVal, "success", true)
mudlog.Error("SetVal", "path", propertyPath, "error", err)
```

## Dependencies

- `gopkg.in/yaml.v2` - YAML parsing and generation
- `internal/mudlog` - Logging and monitoring
- `internal/util` - File operations and utilities
- `sync` - Thread-safe access control
- `reflect` - Dynamic configuration introspection
- `os` - Environment variable access and file operations

## Usage Examples

### Dynamic Configuration Updates
```go
// Update server settings at runtime
configs.SetVal("Server.MudName", "Production Server")
configs.SetVal("Network.MaxTelnetConnections", "100")
configs.SetVal("GamePlay.XPScale", "75.0")

// Changes are immediately validated and persisted
```

### Configuration Validation
```go
// Custom validation in subsections
func (g *GamePlay) Validate() {
    if g.XPScale <= 0 {
        g.XPScale = 100.0 // default
    }
    
    if g.PVP != "enabled" && g.PVP != "disabled" && g.PVP != "limited" {
        g.PVP = "limited" // default
    }
    
    if g.MaxAltCharacters < 0 {
        g.MaxAltCharacters = 5 // default
    }
}
```

### Environment Variable Integration
```go
// Automatic environment variable loading
os.Setenv("DATABASE_URL", "postgres://localhost/muddb")
os.Setenv("API_KEY", "secret-api-key")

// Configuration automatically loads these values
config := configs.GetConfig()
dbURL := configs.GetSecret(config.Server.DatabaseURL)
apiKey := configs.GetSecret(config.Server.APIKey)
```

This configuration system provides a robust foundation for managing all aspects of GoMud server configuration with type safety, validation, security, and runtime flexibility.

---

## DOGMud Balance Configuration Knobs

The `Balance` subsection of the config holds all gameplay-tuning constants.
Additions by chunk are documented below.

### Weapon Reach (chunk 4c)

Three knobs control how harshly long weapons are penalised in grapples.
All live under `Balance` in `_datafiles/config.yaml`.

| Knob | Default | Effect |
|------|---------|--------|
| `ReachStandingGrappleRadius` | 0.5 | Effective radius (m) for Clinch / BackStanding. Weapons longer than this are penalised. |
| `ReachGroundGrappleRadius` | 0.3 | Effective radius (m) for ground grapple states (Mount, Guard, etc.). Tighter than standing. |
| `ReachUtilityFloor` | 0.15 | Minimum damage multiplier from the reach curve. Prevents long weapons from doing literal zero damage. |

See `internal/combat/context.md` "Weapon Reach Utility" for the full
formula and exempt subtype list.

### Submission System (chunk 4d)

Six knobs control when submission windows open and how tiers are
resolved. All live under `Balance` in `_datafiles/config.yaml`.
The `SubSkillWeight` default is 1.5 (higher than the initial T3 plan
value of 1.0 — adjusted at validation time in
`config.balance.combat.go`).

| Knob | Default | Effect |
|------|---------|--------|
| `SubmissionAttemptAlpha` | 1.0 | Min drift-margin z-score (absolute) for a sub window to open on either side of the grapple. |
| `SubmissionAttemptCritZ` | 2.0 | Defender drift z >= this opens a bottom-sub window regardless of margin (the crit shortcut). |
| `SubSkillWeight` | 1.5 | Unarmed-combat skill contribution multiplier in the sub roll. Higher = more skill-dependent outcomes. |
| `SubBadZThreshold` | -1.0 | Sub-roll z-score below which the bad tier fires (attempter falls Prone, grapple breaks). |
| `SubCritZThreshold` | 2.0 | Sub-roll z-score at or above which the crit tier fires (recipient gets 1-round Stunned buff on mercy policy). |
| `SubGoldLossFraction` | 0.20 | Fraction of defender's carried gold transferred to attacker on subdue/cripple outcomes. |

See `internal/combat/context.md` "Submission System (chunk 4d)" for the
full resolution flow and policy matrix.

### NPC Schedules (chunk 3.2)

| Knob | Default | Effect |
|------|---------|--------|
| `ScheduleMaxPathRetries` | 20 | After N consecutive failed `pathto` attempts, a scheduled mob falls back to `pathto home`. Chunk 3.2. Also governs patrol path retries (chunk 3.4 reuses the same threshold; no separate knob). |
| `SleepRegenMultiplier` | 5.0 | HP/SP/CP per-round regen multiplier when bearer has the Sleeping buff. Chunk 3.3. |
| `ScheduleWakeGraceRounds` | 50 | After a forced wake during a sleep segment, suppress re-sleep for N rounds (~200 sec real-time at default tick rate). Chunk 3.3. |