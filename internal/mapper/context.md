# Mapping System Context

## Overview

The `internal/mapper` package provides a comprehensive mapping and pathfinding system for the GoMud game engine. It generates ASCII-based maps, calculates optimal paths between locations, handles different terrain types, and provides navigation assistance with support for secrets, locks, and dynamic room layouts.

## Key Components

### Core Files
- **mapper.go**: Main mapping functionality and map generation
- **mapper.config.go**: Configuration structures and settings
- **mapper.map.go**: Map data structures and management
- **mapper.node.go**: Node structures for pathfinding algorithms
- **mapper.path.go**: Pathfinding algorithms and route calculation
- **mapper.path_test.go**: Unit tests for pathfinding functionality
- **mapper_test.go**: Comprehensive mapping system tests

### Key Structures

#### MapConfig
```go
type MapConfig struct {
    Width       int
    Height      int
    ShowSecrets bool
    ShowLocked  bool
    CenterRoom  int
}
```
Configuration for map generation including dimensions, visibility options, and center point.

#### MapNode
```go
type MapNode struct {
    RoomId    int
    X, Y, Z   int
    Symbol    rune
    Exits     map[string]*MapNode
    Visited   bool
}
```
Represents a room node in the mapping system with position, visual representation, and connections.

#### PathResult
```go
type PathResult struct {
    Path      []int
    Distance  int
    Found     bool
    Error     error
}
```
Result structure for pathfinding operations containing route information and success status.

### Constants
- **defaultMapSymbol**: `'•'` - Default symbol for rooms
- **SecretSymbol**: `'?'` - Symbol for secret or unknown areas
- **LockedSymbol**: `'⚷'` - Symbol for locked rooms or passages

### Global State
- **compassDirections**: Map of valid directional movement commands
- **posDeltas**: Position delta calculations for different directions with connection symbols

## Core Functions

### Map Generation
- **GenerateMap(userId int, centerRoomId int, config MapConfig) ([]string, error)**: Creates ASCII map
  - Generates visual representation of game world around specified center room
  - Supports configurable map dimensions and visibility options
  - Handles room symbols, connections, and terrain representation
  - Integrates with user preferences for personalized mapping

### Pathfinding
- **FindPath(startRoomId, endRoomId int, userId int) PathResult**: Calculates optimal route
  - Uses advanced pathfinding algorithms (A*, Dijkstra's algorithm variants)
  - Considers room accessibility, locks, and user permissions
  - Handles multi-level navigation with up/down movement
  - Returns complete path with distance calculations

- **FindNearestRoom(startRoomId int, targetType string, userId int) PathResult**: Locates nearest room of specific type
  - Searches for rooms matching specific criteria (shops, guilds, etc.)
  - Uses breadth-first search for optimal distance calculation
  - Considers user accessibility and room availability
  - Returns path to closest matching room

### Navigation Assistance
- **GetDirections(path []int) []string**: Converts room path to movement commands
  - Translates room ID sequence into directional commands
  - Handles complex routing with multiple direction changes
  - Provides step-by-step navigation instructions
  - Optimizes route descriptions for clarity

### Map Analysis
- **AnalyzeConnectivity(roomId int, maxDepth int) ConnectivityResult**: Analyzes room connections
  - Examines reachability from specified starting point
  - Identifies isolated areas and connection bottlenecks
  - Provides statistics on world connectivity
  - Supports depth-limited analysis for performance

## Mapping Features

### Visual Representation
- **ASCII Art Maps**: Text-based visual maps using Unicode characters
- **Room Symbols**: Customizable symbols for different room types
- **Connection Lines**: Visual representation of exits and passages
- **Multi-Level Support**: Handling of vertical movement (up/down)
- **Terrain Indication**: Different symbols for various terrain types

### Dynamic Elements
- **Secret Areas**: Conditional display of secret rooms and passages
- **Locked Content**: Visual indication of locked or inaccessible areas
- **User Permissions**: Personalized maps based on user access levels
- **Real-Time Updates**: Maps reflect current world state and accessibility

### Pathfinding Algorithms
- **Optimal Routing**: Shortest path calculation using advanced algorithms
- **Cost Considerations**: Weighted pathfinding considering movement costs
- **Accessibility**: Routing respects locks, permissions, and requirements
- **Multi-Criteria**: Pathfinding with multiple optimization criteria

### Navigation Tools
- **Step-by-Step Directions**: Clear movement instructions for complex routes
- **Landmark Recognition**: Integration with notable locations and landmarks
- **Alternative Routes**: Multiple path options for flexibility
- **Distance Estimation**: Accurate distance and travel time calculations

## Dependencies

### Internal Dependencies
- `internal/mudlog`: For logging mapping operations and errors
- `internal/rooms`: For accessing room data and world structure
- `internal/users`: For user preferences and permission checking

### External Dependencies
- Standard library: `errors`, `fmt`, `math`, `strconv`, `strings`, `time`, `unicode`

## Usage Patterns

### Basic Map Generation
```go
// Generate map centered on player's current room
config := MapConfig{
    Width:       21,
    Height:      15,
    ShowSecrets: false,
    ShowLocked:  true,
    CenterRoom:  playerRoomId,
}

mapLines, err := mapper.GenerateMap(userId, centerRoomId, config)
if err != nil {
    // Handle mapping error
}

// Display map to user
for _, line := range mapLines {
    sendToUser(line)
}
```

### Pathfinding Usage
```go
// Find path to specific destination
pathResult := mapper.FindPath(startRoom, destinationRoom, userId)
if pathResult.Found {
    directions := mapper.GetDirections(pathResult.Path)
    for _, direction := range directions {
        sendToUser(direction)
    }
} else {
    sendToUser("No path found to destination")
}
```

### Navigation Assistance
```go
// Find nearest shop
shopResult := mapper.FindNearestRoom(playerRoom, "shop", userId)
if shopResult.Found {
    directions := mapper.GetDirections(shopResult.Path)
    sendToUser(fmt.Sprintf("Nearest shop is %d rooms away:", shopResult.Distance))
    for _, direction := range directions {
        sendToUser(direction)
    }
}
```

## Integration Points

### Room System
- **World Data**: Direct integration with room data structures
- **Exit Information**: Uses room exit data for pathfinding and mapping
- **Room Properties**: Incorporates room types, terrain, and special properties
- **Dynamic Updates**: Responds to changes in world structure

### User System
- **Permissions**: Respects user access levels and permissions
- **Preferences**: Incorporates user mapping preferences and settings
- **Exploration**: Tracks user exploration and visited areas
- **Accessibility**: Considers user-specific accessibility requirements

### Command System
- **Map Commands**: Integration with user commands for map display
- **Navigation Commands**: Pathfinding integration with movement commands
- **Administrative Tools**: Mapping tools for world builders and administrators
- **Help Integration**: Context-sensitive mapping assistance

### Game Mechanics
- **Movement**: Integration with character movement and travel systems
- **Exploration**: Support for exploration mechanics and discovery
- **Quests**: Pathfinding assistance for quest objectives
- **Transportation**: Integration with teleportation and fast travel systems

## Performance Considerations

### Algorithm Optimization
- **Efficient Pathfinding**: Optimized algorithms for large world maps
- **Caching**: Intelligent caching of pathfinding results
- **Pruning**: Search space pruning for improved performance
- **Heuristics**: Advanced heuristics for faster path calculation

### Memory Management
- **Node Pooling**: Efficient memory management for pathfinding nodes
- **Map Caching**: Cached map generation for frequently accessed areas
- **Garbage Collection**: Minimal allocation during pathfinding operations
- **Resource Limits**: Configurable limits to prevent resource exhaustion

### Scalability
- **Large Worlds**: Efficient handling of massive game worlds
- **Concurrent Access**: Thread-safe operations for multiple simultaneous users
- **Distributed Processing**: Support for distributed pathfinding calculations
- **Load Balancing**: Balanced processing of mapping requests

## Advanced Features

### Multi-Level Mapping
- **3D Navigation**: Full three-dimensional pathfinding and mapping
- **Level Transitions**: Handling of stairs, elevators, and teleporters
- **Cross-Level Routing**: Pathfinding across multiple world levels
- **Vertical Visualization**: Visual representation of multi-level structures

### Dynamic World Support
- **Changing Topology**: Adaptation to dynamic world changes
- **Temporary Obstacles**: Handling of temporary barriers and blockages
- **Conditional Passages**: Support for time-based or condition-based access
- **Real-Time Updates**: Live updates as world structure changes

### Specialized Pathfinding
- **Weighted Routing**: Pathfinding with movement cost considerations
- **Constraint-Based**: Routing with specific constraints and requirements
- **Multi-Objective**: Optimization for multiple criteria simultaneously
- **Probabilistic**: Pathfinding with uncertainty and probability factors

## Future Enhancements

### Enhanced Visualization
- **Graphical Maps**: Support for graphical map generation
- **Interactive Maps**: Web-based interactive mapping interfaces
- **3D Visualization**: Three-dimensional world visualization
- **Augmented Reality**: AR integration for immersive navigation

### Advanced Navigation
- **AI-Assisted Routing**: Machine learning for optimal path selection
- **Predictive Navigation**: Anticipatory pathfinding based on user patterns
- **Social Navigation**: Pathfinding considering other player locations
- **Dynamic Optimization**: Real-time route optimization during travel

### World Analysis Tools
- **Connectivity Analysis**: Advanced world connectivity and flow analysis
- **Bottleneck Detection**: Identification of world design bottlenecks
- **Balance Assessment**: Analysis of world balance and accessibility
- **Usage Analytics**: Player movement pattern analysis and optimization

### Integration Enhancements
- **External Maps**: Integration with external mapping services
- **Mobile Apps**: Mobile application integration for offline maps
- **Voice Navigation**: Voice-guided navigation assistance
- **Accessibility Tools**: Enhanced accessibility for users with disabilities

## Security and Validation

### Access Control
- **Permission Validation**: Strict validation of user access permissions
- **Information Security**: Protection of sensitive world information
- **Exploration Limits**: Enforcement of exploration boundaries and limits
- **Anti-Cheating**: Prevention of mapping-based cheating and exploitation

### Data Integrity
- **World Validation**: Validation of world data consistency and integrity
- **Path Verification**: Verification of calculated paths and routes
- **Error Detection**: Detection and handling of world data errors
- **Recovery Mechanisms**: Automatic recovery from mapping errors

## Administrative Tools

### World Building Support
- **Design Validation**: Tools for validating world design and connectivity
- **Balance Analysis**: Analysis tools for world balance and flow
- **Visualization Tools**: Advanced visualization for world designers
- **Import/Export**: Tools for importing and exporting world map data

### Monitoring and Analytics
- **Usage Tracking**: Monitoring of mapping system usage and performance
- **Performance Metrics**: Analysis of pathfinding performance and efficiency
- **Error Reporting**: Comprehensive error reporting and analysis
- **Optimization Recommendations**: Automated suggestions for world optimization

## Cartesian Consistency Engine

The mapper enforces geometric coherence between room coordinates and
declared exits. The entry point is `ValidateZoneConsistency()`, called
at the tail of `PreCacheMaps()` and gated by the config knob
`GamePlay.MapConsistencyEnforce` (`off` | `warn` (default) | `panic`).
The same logic is exposed on demand via the `cartcheck [zone]` admin
command (`internal/usercommands/admin.cartcheck.go`).

### Core Method

```go
(*mapper).CheckConsistency(zone string, nonCartesian bool) []Finding
```

Scans `crawledRooms` (the `map[int]*mapNode` populated during BFS
crawl) for the zone and returns a slice of findings. `RoomGrid` is a
separate struct over a `[][][]*mapNode` 3D slice used for rendering, not
for consistency checks. The `nonCartesian` flag is sourced from the
zone's `zone-config.yaml` field `non_cartesian: true`; when set, only
`longcrossing` warnings are emitted (the hard checks are skipped).

### Exit Kind Classification

```go
func classifyKind(nominal, actual positionDelta) ExitKind
```

Exit-kind classifier used by the map snapshot/render layer (consumed in
later tasks). Compares the nominal compass delta (what the exit direction
implies) to the actual coordinate delta (the difference between the two
rooms' positions). Returns one of four `ExitKind` values:

**Note:** `CheckConsistency` does NOT call `classifyKind`. Detection of
`deltamismatch` and `longcrossing` is performed via inline delta
comparisons (`samePos`) directly inside `CheckConsistency`.

| Kind       | Meaning                                                     |
|------------|-------------------------------------------------------------|
| `normal`   | Nominal == actual (standard single-cell step)               |
| `long`     | Same direction, magnitude > 1 (multi-cell connector)        |
| `vertical` | One axis is Z only (up/down exits)                          |
| `wrap`     | Nominal and actual differ — toroidal or maze-style crossing |

**Important:** `classifyKind` uses the helper `samePos(a, b positionDelta)`,
which compares only the `x`, `y`, `z` fields. Never compare
`positionDelta` values with `==` directly — the struct also carries an
`arrow` display field that will cause false mismatches.

### Finding Kinds

| Kind            | Severity | Description                                              |
|-----------------|----------|----------------------------------------------------------|
| `collision`     | error    | Two distinct rooms share the same (x,y,z) coordinate.   |
|                 |          | Detected by grouping `crawledRooms` nodes by their       |
|                 |          | `(x,y,z)` Pos. Scanning `crawledRooms` (not `RoomGrid`)  |
|                 |          | is required because `RoomGrid` is a 3D slice: when two   |
|                 |          | rooms land on the same cell, the slice assignment keeps  |
|                 |          | only the last writer, making the collision invisible     |
|                 |          | there — every room is present in `crawledRooms`.         |
| `noreciprocal`  | error    | A spatial exit has no matching return exit in the        |
|                 |          | opposite direction, and the exit is not marked           |
|                 |          | `oneway: true`.                                          |
| `deltamismatch` | error    | Exit's compass direction does not match the actual       |
|                 |          | coordinate delta between the two rooms — a wrap exit     |
|                 |          | inside a Cartesian zone.                                 |
| `longcrossing`  | warning  | A long-connector exit's straight span passes through     |
|                 |          | another room's occupied cell. Always emitted regardless  |
|                 |          | of `non_cartesian` setting.                              |

### Exemptions

The engine automatically skips:
- **Non-compass exits** (portals, named exits): filtered via the
  `getMapNode` `mapdirection→name→skip` rule; they are not spatial edges.
- **Ephemeral/instance rooms**: checked via `rooms.IsEphemeralRoomId`.
- **`oneway: true` exits**: exempt from `noreciprocal`; still
  collision-checked.
- **`non_cartesian: true` zones**: exempt from `collision`,
  `noreciprocal`, and `deltamismatch`; their wrap exits render as
  edge stubs in the web mapper.

### Authoring Primitives

- **`oneway: true`** on an exit YAML field — marks an intentional
  one-way passage; suppresses the reciprocity check for that exit.
- **`non_cartesian: true`** in a zone's `zone-config.yaml` — marks the
  entire zone as toroidal/maze geometry; skips the three hard checks
  zone-wide.

### Known Limitation: Cross-Zone Crawl

`CheckConsistency` operates on the BFS-populated `crawledRooms` map,
which follows all exits and can cross zone boundaries. This is mitigated
at the reporting layer by `FilterFindingsToZone`, which drops any finding
whose room's owning `zone:` field does not match the zone being checked.
Both `ValidateZoneConsistency` (startup) and `CartCheck` (admin command)
apply this filter, so findings are correctly scoped to their owning zone.

## Fog-of-War Snapshot (Web Map)

### File

`mapper.snapshot.go`

### Types

```go
type SnapshotRoom struct {
    RoomId int            `json:"num"`
    X      int            `json:"x"`
    Y      int            `json:"y"`
    Z      int            `json:"z"`
    Symbol string         `json:"symbol"`
    Biome  string         `json:"biome"`
    Exits  []SnapshotExit `json:"exits"`
}

type SnapshotExit struct {
    ToRoomId int      `json:"to"`
    DX       int      `json:"dx"`
    DY       int      `json:"dy"`
    DZ       int      `json:"dz"`
    Kind     ExitKind `json:"kind"`
}
```

### Method

```go
func (r *mapper) Snapshot(visited map[int]struct{}) []SnapshotRoom
```

Iterates `crawledRooms` and returns only rooms whose ID is present in
`visited`. For each included room, exits are also filtered: an exit is
included only if its destination room is in `visited` (fog of war — the
client never learns about rooms the player hasn't been to). Each exit's
`Kind` is set by `classifyKind(nominal, actual)` — the same classifier
used by the consistency engine.

This method is the sole output consumed by the `gmcp.Zone` module
(`modules/gmcp/gmcp.Zone.go`), which builds the `Zone.Map` GMCP payload
and sends it to the web client on every room change. The web client
renderer (`RoomGridSVG` in `gmcp.js`) uses the `kind` field to route
each exit to its correct visual treatment: connector line (`normal`/
`long`), teal edge-stub with chevron (`wrap`), or ▲/▼ tick (`vertical`).

### SnapshotExit Extended Fields

`SnapshotExit` carries additional flags that inform per-exit visual
styling on the client:

```go
type SnapshotExit struct {
    ToRoomId int      `json:"to"`
    DX       int      `json:"dx"`
    DY       int      `json:"dy"`
    DZ       int      `json:"dz"`
    Kind     ExitKind `json:"kind"`
    Locked   bool     `json:"locked,omitempty"`
    Secret   bool     `json:"secret,omitempty"`
    OneWay   bool     `json:"oneway,omitempty"`
    Gate     bool     `json:"gate,omitempty"`
    Stub     bool     `json:"stub,omitempty"`
    ToZone   string   `json:"tozone,omitempty"`
}
```

- `Locked` — exit has a key requirement (room has a lock or door key set).
- `Secret` — exit is normally hidden from plain sight.
- `OneWay` — exit is flagged `oneway: true`; no return exit expected.
- `Gate` — set when the exit's `ExitMessage != ""`; indicates a barrier
  or door with flavor text (portcullises, heavy doors, etc.).
- `Stub` — the destination room is **not** in the visited set (unvisited)
  or is in a different zone. Stub exits are now emitted by `Snapshot`
  instead of being dropped, so the client can render a visual hint that
  a passage continues beyond the fog boundary.
- `ToZone` — populated on cross-zone stub exits; contains the destination
  zone name. Allows the client to label or style zone-boundary exits
  distinctly.

**Prior behavior:** `Snapshot` dropped exits whose destination was not in
the visited set. After this change it emits them as `Stub: true` entries,
giving the client enough information to draw a "passage continues"
indicator without revealing the destination room's details.

### nodeExit.Gate

`nodeExit` (the internal per-exit node built during BFS crawl) gained a
`Gate bool` field set from `exit.ExitMessage != ""`. This propagates to
`SnapshotExit.Gate` during snapshot construction.

## Map Consumers

The mapper data is consumed by two independent rendering paths:

### (a) In-Game ASCII `map` Command

`internal/usercommands/skill.map.go` calls `GetLimitedMap` and
`GetLegend` to render a terminal-width ASCII map scaled by the player's
Perception skill. Symbol legend:

| Symbol | Meaning              |
|--------|----------------------|
| `@`    | You (current room)   |
| `☺`    | Player / Party / NPC |
| `☠`    | Hostile mob          |
| `☹`    | Friendly NPC         |
| *(biome/mapsymbol)* | Room terrain glyph |

Detail level (visible radius, secret/locked display) scales with
Perception. This path is text-only and has no awareness of the
`SnapshotExit` extended flags or `Zone.Map.Party`.

### (b) Web Leather Map (GMCP `Zone.Map`)

The `Zone.Map` GMCP snapshot (`Snapshot`) feeds the browser client.
The `Zone.Map` payload now includes a `Party []int` field — a list of
room IDs currently occupied by party members — enabling the client to
render party-member position markers on the map.

The web renderer (`RoomGridSVG` in
`_datafiles/html/public/static/js/gmcp.js`) presents an **antique
tooled-leather** themed map: a fixed leather-textured SVG surface holds
a nested pannable `worldSvg` containing the room grid. Connection
styling is per-exit-type:

- **Biome roads/trails/water** — line color/style derived from room biome.
- **Locked** — distinctive styling for keyed doors.
- **Secret** — rendered as a dimmed or dashed connector.
- **One-way** — directional arrow or asymmetric line weight.
- **Gate** — styled to suggest a barrier (portcullis texture or color).
- **Stairs** — ▲/▼ ticks on the room node for `vertical` exits.
- **Cross-zone stubs** — short labeled stubs at the zone boundary.
- **Fog stubs** — dim stubs for unvisited exits (stub exits in the
  snapshot).

Party markers are small figures drawn on the room node for each room ID
in `Zone.Map.party`. The current player's room is rendered with a raised
(drop-shadow) treatment to distinguish it from adjacent rooms.

Visual source of truth: `docs/superpowers/specs/2026-06-06-mapper-leather-mockups/`.

Connection-type styling and party markers are **web-only** — the ASCII
`map` command does not reflect these.