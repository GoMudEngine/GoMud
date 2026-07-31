# GoMud - Summary

GoMud is an open source Multi-user Dungeon (MUD) game engine and server written in Go. It provides a complete MUD framework with a default fantasy world, web-based administration tools, telnet and web client support, and extensive JavaScript-based scripting capabilities for creating custom game content. The project serves both as a playable MUD and as a library/engine for building custom MUD worlds.

## Technology Stack

**Primary Language:** Go 1.24+ (requires Go 1.24 minimum as specified in go.mod)

**Key Dependencies:**
- `github.com/dop251/goja` - JavaScript runtime for game scripting (spells, mobs, rooms, items)
- `github.com/gorilla/websocket` - WebSocket support for web client connectivity
- `github.com/GoMudEngine/ansitags` - ANSI color/formatting library for terminal output
- `github.com/natefinch/lumberjack` - Log rotation and management
- `github.com/nicksnyder/go-i18n/v2` - Internationalization support
- `github.com/stretchr/testify` - Testing framework
- `gopkg.in/yaml.v2` and `gopkg.in/yaml.v3` - YAML configuration parsing

**Frontend Technologies:**
- HTMX 2.0.3 for dynamic web admin interface
- Pure CSS/HTML for web client and admin panels
- JavaScript ES6 for client-side functionality (JSHint configured)

**Infrastructure:**
- Docker with multi-stage builds (Alpine Linux base)
- Docker Compose for orchestration
- GitHub Actions for CI/CD
- Telnet server for classic MUD client connections
- HTTP/HTTPS web server for browser-based access

## Project Structure

```
_datafiles/                 # Game world data and configuration files
├── config.yaml            # Main server configuration (22K+ lines of settings)
├── guides/                 # Documentation for players, builders, and operators
├── html/                   # Web interface templates and static assets
│   ├── admin/             # Administrative web interface (HTMX-based)
│   └── public/            # Public web client and assets
├── localize/              # Translation files (en.yaml, zh.yaml)
├── sample-scripts/        # Example JavaScript scripts for spells and mobs
└── world/default/         # Default game world content
    ├── biomes/            # Environment definitions
    ├── buffs/             # Status effects with JS logic
    ├── conversations/     # NPC dialogue trees
    ├── items/             # Game items (weapons, armor, consumables)
    ├── mobs/              # Non-player characters with AI scripts
    ├── quests/            # Quest definitions and progression
    ├── races/             # Character races and stats
    └── rooms/             # Game world locations and connections

cmd/generate/              # Code generation utilities
internal/                  # Core engine packages (Go internal convention)
├── characters/           # Player/NPC character system
├── rooms/                # Room management and world state
├── usercommands/         # Player command implementations
├── mobcommands/          # NPC AI command system
├── scripting/            # JavaScript runtime integration
├── web/                  # HTTP server and admin interface
├── configs/              # Configuration management
└── [30+ other packages]  # Modular architecture with clear separation

modules/                   # Plugin system for extending functionality
├── gmcp/                 # Generic MUD Communication Protocol
├── auctions/             # Player auction system
├── follow/               # Player following mechanics
└── [other modules]       # Extensible module architecture

provisioning/             # Docker deployment configuration
```

## Development Workflow

**Essential Commands:**

```bash
# Build the project (includes code generation)
make build

# Run development server locally
make run

# Run with fresh world state (deletes instance data)
make run-new

# Run comprehensive tests
make test

# Run JavaScript linting
make js-lint

# Format Go code
make fmt

# Validate code (format check + vet)
make validate

# Generate code (required before building)
go generate ./...

# Build for specific platforms
make build_linux64        # Linux AMD64
make build_win64          # Windows AMD64
make build_rpi_zero2w     # Raspberry Pi ARM64

# Docker development
make run-docker           # Run in Docker container
make client               # Connect telnet client to Docker instance
```

**Testing Strategy:**
- Unit tests throughout codebase with `*_test.go` files
- Table-driven tests following Go conventions
- Test coverage reporting via `make coverage`
- JavaScript linting with JSHint (ES6 configuration)
- GitHub Actions CI/CD on all pull requests and master branch pushes
- Multi-platform build testing (Linux, macOS, Windows, ARM)

**Development Environment Setup:**
1. Go 1.24+ required (specified in go.mod)
2. Docker and Docker Compose for containerized development
3. Make for build automation
4. Node.js for JavaScript linting (via Docker)
5. Environment variables for configuration overrides:
   - `CONFIG_PATH` - Custom config.yaml path
   - `LOG_PATH` - Log file location
   - `LOG_LEVEL` - Logging verbosity (LOW/MEDIUM/HIGH)
   - `LOG_NOCOLOR` - Disable colored logging

## Code Conventions

**Go Code Style:**
- Standard Go formatting enforced via `go fmt`
- Package organization follows Go internal conventions
- Extensive use of interfaces for modularity (e.g., `UserCommand` function signature)
- Event-driven architecture with typed event system
- Comprehensive error handling with wrapped errors
- Structured logging throughout

**File Naming Patterns:**
- `*_test.go` - Unit tests
- `admin.*.go` - Administrative functionality
- `skill.*.go` - Player skill implementations
- `config.*.go` - Configuration structures
- Package names are singular and lowercase

**JavaScript Integration:**
- ES6 JavaScript for game scripting via Goja runtime
- Scripts located in `_datafiles/` with `.js` extensions
- Global functions exposed from Go to JavaScript runtime
- Timeout protection for runaway scripts (configurable)
- Separate contexts for different script types (spells, mobs, rooms, items)

**Architecture Patterns:**
- Event-driven system with typed events and listeners
- Plugin/module system for extensibility
- Command pattern for user and mob actions
- Repository pattern for data access
- Singleton pattern for global managers (rooms, users, etc.)

**Error Handling:**
- Wrapped errors with context using `github.com/pkg/errors`
- Structured logging with consistent field names
- Graceful degradation for non-critical failures
- Panic recovery at main goroutine level

## Important Notes

**Configuration Management:**
- Main config in `_datafiles/config.yaml` (22K+ lines of comprehensive settings)
- Override configs via `CONFIG_PATH` environment variable
- Locked configurations prevent runtime modification of critical settings
- YAML-based configuration with extensive validation

**Networking:**
- Default telnet ports: 33333, 44444
- Default HTTP port: 80
- Default HTTPS port: disabled (configurable)
- WebSocket support for modern web clients
- Localhost-only port 9999 for administrative access

**Game Engine Specifics:**
- Turn-based processing with configurable timing (50ms turns, 4-second rounds by default)
- Room-based world with dynamic loading/unloading for memory efficiency
- Character persistence with automatic saving
- Comprehensive buff/debuff system with JavaScript scripting
- Quest system with progress tracking
- Combat system with customizable damage calculations

**Scripting System:**
- JavaScript runtime for game logic (spells, mob AI, room interactions)
- Timeout protection (1000ms load timeout, 50ms room script timeout)
- Extensive API exposed to JavaScript for game world manipulation
- Sample scripts provided in `_datafiles/sample-scripts/`

**Performance Considerations:**
- Memory management with configurable thresholds for room/mob unloading
- Automatic cleanup of idle game objects
- Configurable CPU core usage
- Log rotation to prevent disk space issues

**Common Pitfalls:**
- Always run `go generate ./...` before building (required for module imports)
- JavaScript scripts must not exceed timeout limits or they will be killed
- Room instance data is automatically generated - delete `rooms.instances` directories for fresh world state
- Configuration changes in locked sections require config file modification, not runtime commands
- Docker builds require copying `_datafiles` directory to include game content

**Deployment:**
- Multi-stage Docker builds with Alpine Linux
- Automated GitHub releases with cross-platform binaries
- Raspberry Pi support with ARM builds
- Environment-specific configuration via environment variables
- Log aggregation and rotation built-in

**Development Workflow Integration:**
- EditorConfig for consistent formatting across editors
- GitHub Actions for automated testing and releases
- Docker Compose for local development environment
- Make-based build system for cross-platform compatibility
- Modular architecture allows focused development on specific game systems

## Code Context Documentation

### Core Engine Components
- **Characters System**: `internal/characters/context.md` - Player/NPC character system with stats, equipment, combat mechanics, and character states
- **Rooms System**: `internal/rooms/context.md` - World management system with dynamic loading, biomes, spawning, and ephemeral room creation
- **User Commands System**: `internal/usercommands/context.md` - Complete player command system with 100+ commands for gameplay, combat, skills, and administration
- **Mob AI Commands System**: `internal/mobcommands/context.md` - Sophisticated NPC AI system with autonomous behaviors, combat intelligence, and social interactions
- **Combat System**: `internal/combat/context.md` - Turn-based combat engine with damage calculations, attack types, and battle mechanics
- **Mobs System**: `internal/mobs/context.md` - NPC management with AI behaviors, spawning, pathfinding, and lifecycle management
- **Items System**: `internal/items/context.md` - Game item system with equipment, consumables, containers, and item interactions
- **Scripting System**: `internal/scripting/context.md` - JavaScript runtime integration for spells, mobs, rooms, and dynamic game content
- **Events System**: `internal/events/context.md` - Event-driven architecture with typed events, listeners, and game state management
- **Buffs System**: `internal/buffs/context.md` - Status effects system with JavaScript scripting, duration management, and effect stacking
- **Spells System**: `internal/spells/context.md` - Magic system with spell casting, targeting, cooldowns, and JavaScript-based spell effects
- **Skills System**: `internal/skills/context.md` - Player skill progression system with experience, ranks, and skill-based actions
- **Quests System**: `internal/quests/context.md` - Quest management with progress tracking, completion validation, and reward distribution
- **Stats System**: `internal/stats/context.md` - Character statistics system with primary stats, derived stats, and stat modifications
- **Game Time System**: `internal/gametime/context.md` - In-game time management with calendar system, day/night cycles, and temporal events

### Infrastructure and Utilities
- **Configuration System**: `internal/configs/context.md` - Comprehensive configuration management with YAML loading, validation, and hot-reloading
- **Web Interface**: `internal/web/context.md` - HTTP server with admin interface, WebSocket support, and HTMX-based dynamic content
- **User Management**: `internal/users/context.md` - Player account system with authentication, character management, and user data persistence
- **Connections System**: `internal/connections/context.md` - Network connection management for telnet, WebSocket, and client protocol handling
- **Prompt System**: `internal/prompt/context.md` - Dynamic prompt generation with customizable formats, color support, and real-time updates
- **Hooks System**: `internal/hooks/context.md` - Event hook system for game loop integration, automated processes, and system event handling
- **Utility Functions**: `internal/util/context.md` - Core utility functions for string processing, data validation, formatting, and common operations

### Supporting Systems
- **Audio System**: `internal/audio/context.md` - Audio configuration management for sound effects and music file handling
- **Bad Input Tracker**: `internal/badinputtracker/context.md` - Thread-safe tracking system for invalid user commands and usage analytics
- **Clans System**: `internal/clans/context.md` - Guild/clan system with membership management, ranks, territory control, and financial systems
- **Color Patterns**: `internal/colorpatterns/context.md` - Advanced text colorization system with pattern application and ANSI tag preservation
- **Exit System**: `internal/exit/context.md` - Room exit management with locks, secret passages, temporary portals, and custom exit messages
- **Command Flags**: `internal/flags/context.md` - Command-line argument processing for version display and port availability scanning
- **Game Locks**: `internal/gamelock/context.md` - Locking mechanism with difficulty-based security, automatic relocking, and trap systems
- **Keywords and Aliases**: `internal/keywords/context.md` - Comprehensive alias system for commands, help topics, directions, and map legends
- **File Loader**: `internal/fileloader/context.md` - Comprehensive YAML file loading system with validation, batch operations, and concurrent processing
- **Language System**: `internal/language/context.md` - Internationalization support with YAML-based translations and multi-language message handling
- **Logging System**: `internal/mudlog/context.md` - Structured logging with colored output, file rotation, and in-game log display integration
- **Version Management**: `internal/version/context.md` - Semantic version management with comparison, validation, and compatibility checking
- **UUID System**: `internal/uuid/context.md` - Custom UUID generation with embedded timestamps, type classification, and high-performance ID creation
- **Party System**: `internal/parties/context.md` - Group management system with leadership, tactical positioning, combat coordination, and invitation management
- **Pet System**: `internal/pets/context.md` - Comprehensive companion system with feeding mechanics, combat abilities, inventory management, and stat modifications
- **Race System**: `internal/races/context.md` - Character race system with racial characteristics, abilities, stat bonuses, size classifications, and behavioral traits
- **Stat Modifications**: `internal/statmods/context.md` - Centralized system for managing character statistic modifications from items, buffs, racial bonuses, and skills
- **Suggestions System**: `internal/suggestions/context.md` - Simple suggestion cycling system for command completion, auto-suggestions, and help hints
- **Template System**: `internal/templates/context.md` - Comprehensive template processing with ANSI color support, markdown processing, and multi-source file integration
- **Conversations System**: `internal/conversations/context.md` - Dynamic NPC conversation system with scripted dialogues, participant matching, and turn-based execution
- **Input Handlers**: `internal/inputhandlers/context.md` - Comprehensive input processing system with authentication, login flows, system commands, and terminal protocol handling
- **Mapping System**: `internal/mapper/context.md` - ASCII-based mapping and pathfinding system with navigation assistance, terrain support, and route optimization
- **Plugin System**: `internal/plugins/context.md` - Comprehensive plugin architecture for extending the game with custom commands, events, web interfaces, and modular functionality
- **Mutators System**: `internal/mutators/context.md` - Dynamic world modification system for temporary and permanent changes to game elements with time-based evolution
- **Terminal System**: `internal/term/context.md` - Comprehensive terminal protocol handling with telnet negotiation, ANSI processing, and cross-platform compatibility

### Plugin Architecture
- **Modules System**: `modules/context.md` - Comprehensive plugin architecture with event integration, web interfaces, scripting, and extensible functionality