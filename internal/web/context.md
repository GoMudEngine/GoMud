# GoMud Web Server and Admin Interface Context

## Overview

The GoMud web system provides a comprehensive HTTP/HTTPS server with both public web client functionality and secure administrative interfaces. It supports WebSocket connections for real-time game clients, template-based HTML rendering, plugin integration, and extensive admin tools for managing game content including rooms, items, mobs, races, and mutators.

## Architecture

The web system is built around Go's standard `net/http` package with several key components:

### Core Components

**HTTP Server Management:**
- Dual HTTP/HTTPS server support with configurable ports
- TLS certificate validation and configuration
- Automatic HTTPS redirect capability
- WebSocket upgrade handling for real-time clients
- Graceful shutdown with timeout management

**Template System:**
- Go `text/template` based HTML rendering
- Automatic inclusion of `_*.html` template files
- Plugin template override capability
- Custom template functions for formatting and logic
- Dynamic navigation menu generation

**Authentication and Security:**
- HTTP Basic Authentication for admin areas
- Role-based access control (admin/user roles)
- Authentication caching (30-minute sessions)
- Game state mutex locking for concurrent access protection
- Directory traversal protection

**Plugin Integration:**
- `WebPlugin` interface for module web extensions
- Dynamic navigation link management
- Custom request handling and template data injection
- Static file serving for plugin assets

## Key Features

### 1. **Public Web Interface**
- Static file serving from configurable public HTML directory
- Template processing with game data injection
- WebSocket endpoint for real-time game clients
- Automatic `.html` extension handling
- Custom 404 error page support

### 2. **Administrative Interface**
- Comprehensive admin panels for game content management
- Real-time game data editing and visualization
- Secure authentication with role verification
- Mutex-protected operations to prevent data corruption
- HTMX-powered dynamic interfaces

### 3. **WebSocket Support**
- Real-time bidirectional communication for game clients
- Automatic connection upgrade from HTTP
- Integration with game connection handling system
- Cross-origin request support for development

### 4. **Template Engine**
- Dynamic content generation with game state data
- Custom template functions for formatting and calculations
- Automatic template file discovery and inclusion
- Plugin template override and extension capability

### 5. **Security Features**
- HTTP Basic Authentication with user database integration
- Role-based access control for admin functions
- Authentication result caching to reduce database load
- Request logging and monitoring
- Game state protection through mutex locking

## Admin Interface Components

### Room Administration (`/admin/rooms/`)
- Zone-based room browsing and filtering
- Room property editing (title, description, exits)
- Container and mutator management
- Map visualization and navigation
- Bulk operations and zone management
- Note: Zone-level mob autoscaling was removed in Phase 21; mob difficulty is now per-mob via `statpool`

### Item Administration (`/admin/items/`)
- Item type and subtype filtering
- Item property editing (stats, descriptions, values)
- Buff and effect management
- Item usage and restriction configuration
- Bulk import/export capabilities

### Mob Administration (`/admin/mobs/`)
- NPC template editing and management
- AI behavior configuration
- Stat and skill assignment
- Loot table management
- Spawn location configuration

### Race Administration (`/admin/races/`)
- Player race configuration
- Racial stat bonuses and penalties
- Special ability assignment
- Appearance and description management

### Mutator Administration (`/admin/mutators/`)
- Room and zone modifier management
- Effect configuration and testing
- Conditional application rules
- Performance impact monitoring

## Template System

### Available Template Variables
```html
<!-- Standard template variables -->
{{.REQUEST}}  <!-- HTTP request object -->
{{.PATH}}     <!-- Current request path -->
{{.CONFIG}}   <!-- Game configuration -->
{{.STATS}}    <!-- Server statistics -->
{{.NAV}}      <!-- Navigation menu items -->

<!-- Plugin-provided variables -->
{{.PLUGIN_DATA}}  <!-- Custom data from plugins -->
```

### Custom Template Functions
```html
<!-- String manipulation -->
{{pad 20 "text"}}           <!-- Center pad to width -->
{{lpad 20 "text"}}          <!-- Left pad to width -->
{{rpad 20 "text"}}          <!-- Right pad to width -->
{{join .Items ", "}}        <!-- Join array with separator -->
{{uc "text"}}               <!-- Title case -->
{{lc "TEXT"}}               <!-- Lower case -->
{{escapehtml .UserInput}}   <!-- HTML escape -->

<!-- Numeric operations -->
{{add .Count 1}}            <!-- Addition -->
{{sub .Total .Used}}        <!-- Subtraction -->
{{mul .Base .Multiplier}}   <!-- Multiplication -->
{{intRange 1 10}}           <!-- Generate number range -->

<!-- Comparisons -->
{{if lte .Level 5}}         <!-- Less than or equal -->
{{if gte .Health 100}}      <!-- Greater than or equal -->
{{if lt .Mana 50}}          <!-- Less than -->

<!-- Configuration access -->
{{getconfig}}               <!-- Access game configuration -->
```

### Template File Structure
```
_datafiles/html/public/
├── _header.html           # Included in all pages
├── _footer.html           # Included in all pages
├── index.html             # Homepage
├── webclient.html         # Game client interface
├── online.html            # Player list
├── viewconfig.html        # Configuration viewer
└── static/
    ├── css/
    ├── js/
    └── images/

_datafiles/html/admin/
├── _header.html           # Admin header
├── _footer.html           # Admin footer
├── index.html             # Admin dashboard
├── rooms/
│   ├── index.html         # Room listing
│   └── roomdata.html      # Room editor
├── items/
├── mobs/
├── races/
└── mutators/
```

## Plugin Integration

### WebPlugin Interface
```go
type WebPlugin interface {
    // Return navigation links for menu integration
    NavLinks() map[string]string
    
    // Handle custom web requests
    WebRequest(r *http.Request) (html string, templateData map[string]any, ok bool)
}
```

### Plugin Registration
```go
// In main.go
web.SetWebPlugin(plugins.GetPluginRegistry())

// Plugin implementation
func (p *MyPlugin) NavLinks() map[string]string {
    return map[string]string{
        "My Feature": "/myplugin/dashboard",
        "Settings":   "/myplugin/config",
    }
}

func (p *MyPlugin) WebRequest(r *http.Request) (string, map[string]any, bool) {
    if strings.HasPrefix(r.URL.Path, "/myplugin/") {
        html := "<h1>Custom Plugin Page</h1>"
        data := map[string]any{
            "PluginData": "Custom content",
        }
        return html, data, true
    }
    return "", nil, false
}
```

## Security Implementation

### Authentication Flow
1. **Request Interception**: Admin routes protected by `doBasicAuth` middleware
2. **Cache Check**: Authentication results cached for 30 minutes
3. **Credential Validation**: Username/password verified against user database
4. **Role Verification**: User must have admin or higher role
5. **Access Granted**: Request proceeds to handler with mutex protection

### Game State Protection
```go
// All admin operations wrapped with mutex
func RunWithMUDLocked(next http.HandlerFunc) http.HandlerFunc {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        util.LockMud()
        defer util.UnlockMud()
        next.ServeHTTP(w, r)
    })
}
```

## Configuration and Setup

### Network Configuration
```yaml
network:
  http_port: 80              # HTTP server port (0 to disable)
  https_port: 443            # HTTPS server port (0 to disable)
  https_redirect: true       # Redirect HTTP to HTTPS

file_paths:
  public_html: "_datafiles/html/public"
  admin_html: "_datafiles/html/admin"
  https_cert_file: "cert.pem"
  https_key_file: "key.pem"
```

### Server Startup Process
1. **Configuration Validation**: Check ports and certificate files
2. **Route Registration**: Set up all HTTP handlers and middleware
3. **HTTPS Server**: Start with TLS configuration if certificates available
4. **HTTP Server**: Start with optional HTTPS redirect
5. **WebSocket Handler**: Register upgrade endpoint for game clients

## API Endpoints

### Public Endpoints
- `GET /` - Homepage and static content
- `GET /ws` - WebSocket upgrade for game clients. Inbound frames are capped at
  `wsMaxMessageBytes` (64 KiB) via `conn.SetReadLimit` right after the upgrade —
  gorilla's default is unlimited and `ReadMessage` buffers a whole frame, so
  without the cap an unauthenticated client can force an arbitrary allocation.
  The handler passed in (`main.HandleWebSocketConnection`) owns removing the
  connection from the `connections` registry on every exit path.
- `GET /favicon.ico` - Favicon redirect
- `GET /<path>` - Template-processed HTML pages

### Admin Endpoints (Authentication Required)
- `GET /admin/` - Admin dashboard
- `GET /admin/static/*` - Admin static assets
- `GET /admin/rooms/` - Room management interface
- `GET /admin/rooms/roomdata/` - Room data API
- `GET /admin/items/` - Item management interface
- `GET /admin/items/itemdata/` - Item data API
- `GET /admin/mobs/` - Mob management interface
- `GET /admin/mobs/mobdata/` - Mob data API
- `GET /admin/races/` - Race management interface
- `GET /admin/races/racedata/` - Race data API
- `GET /admin/mutators/` - Mutator management interface
- `GET /admin/mutators/mutatordata/` - Mutator data API

## Performance Considerations

### Template Caching
- Templates are parsed on each request for development flexibility
- Include files (`_*.html`) are automatically discovered and loaded
- Plugin templates can override default behavior

### Authentication Caching
- Successful authentications cached for 30 minutes
- Reduces database load for frequent admin operations
- Automatic cache expiration and cleanup

### Mutex Protection
- All admin operations protected by game state mutex
- Prevents concurrent modification of game data
- May cause brief delays during heavy admin usage

### Static File Serving
- Efficient static file serving for assets
- Proper MIME type detection and headers
- Directory listing protection

## Error Handling and Logging

### Request Logging
```go
// All requests logged with details
mudlog.Info("Web", 
    "ip", r.RemoteAddr,
    "ref", r.Header.Get("Referer"),
    "file path", fullPath,
    "file extension", fileExt,
    "file source", source,
    "size", fmt.Sprintf("%.2fk", float64(fSize)/1024)
)
```

### Error Responses
- Custom 404 error page with template processing
- Proper HTTP status codes for all error conditions
- Template parsing errors logged with full context
- Authentication failures logged with security details

## Integration with Game Systems

### Real-time Data Access
- Direct access to game configuration and statistics
- Live player information and server status
- Real-time room, item, and mob data

### WebSocket Game Client
- Seamless integration with game connection system
- Real-time bidirectional communication
- Support for both telnet and web-based clients

### Plugin System Integration
- Dynamic content injection from modules
- Custom navigation and routing
- Template data extension and override

## Dependencies

- `net/http` - HTTP server and routing
- `github.com/gorilla/websocket` - WebSocket upgrade and handling
- `text/template` - HTML template processing
- `crypto/tls` - HTTPS certificate management
- `internal/configs` - Configuration management
- `internal/users` - Authentication and user management
- `internal/mudlog` - Logging and monitoring
- `internal/util` - Game state mutex protection
- `internal/plugins` - Plugin system integration

## Usage Examples

### Custom Admin Page Template
```html
{{template "_header.html" .}}

<div class="admin-content">
    <h1>{{.CONFIG.ServerName}} Administration</h1>
    
    <div class="stats-grid">
        <div class="stat-card">
            <h3>Online Players</h3>
            <span class="stat-value">{{len .STATS.OnlineUsers}}</span>
        </div>
        
        <div class="stat-card">
            <h3>Total Rooms</h3>
            <span class="stat-value">{{.ROOM_COUNT}}</span>
        </div>
    </div>
    
    <table class="data-table">
        <thead>
            <tr>
                <th>Player</th>
                <th>Level</th>
                <th>Location</th>
            </tr>
        </thead>
        <tbody>
            {{range .STATS.OnlineUsers}}
            <tr>
                <td>{{.CharacterName}}</td>
                <td>{{.Level}}</td>
                <td>{{.RoomTitle}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</div>

{{template "_footer.html" .}}
```

### Plugin Web Integration
```go
func (p *MyPlugin) WebRequest(r *http.Request) (string, map[string]any, bool) {
    switch r.URL.Path {
    case "/myplugin/dashboard":
        return p.renderDashboard(r)
    case "/myplugin/api/data":
        return p.handleAPIRequest(r)
    }
    return "", nil, false
}

func (p *MyPlugin) renderDashboard(r *http.Request) (string, map[string]any, bool) {
    html := `
    {{template "_header.html" .}}
    <h1>{{.PLUGIN_NAME}} Dashboard</h1>
    <div class="plugin-content">
        {{range .PLUGIN_ITEMS}}
        <div class="item">{{.Name}}: {{.Value}}</div>
        {{end}}
    </div>
    {{template "_footer.html" .}}
    `
    
    data := map[string]any{
        "PLUGIN_NAME":  "My Custom Plugin",
        "PLUGIN_ITEMS": p.getPluginData(),
    }
    
    return html, data, true
}
```

This web system provides a robust foundation for both player-facing web interfaces and comprehensive administrative tools, with strong security, plugin extensibility, and real-time game integration capabilities.
## Files

| File | Purpose |
|------|---------|
| `web.go` | Server setup, routing, static files |
| `auth.go` | Admin authentication |
| `template_func.go` | Template helper functions |
| `stats.go` | Public stats endpoints |
| `build.go` | The admin world-building tool backend |
| `admin.go` | Admin routing and shared handlers |
| `admin.rooms.go` | Room editor |
| `admin.mobs.go` | Mob editor |
| `admin.items.go` | Item editor |
| `admin.species.go` | Species editor |
| `admin.mutators.go` | Mutator editor |
| `admin.progression.go` | Progression tuning views |
| `admin.combatstats.go` | Combat statistics dashboard |
| `admin.economyhealth.go` | The economy health dashboard (`internal/economy/health`) |

Plugin-supplied pages (achievements, leaderboards, help) are **not** here —
modules register their own via `WebConfig.WebPage`, and this package serves
them through the plugin registry's `fs.ReadFileFS` implementation.

Note: filesystem-touching tests in this package chdir to the repo root and call
`configs.ReloadConfig()` — Go test binaries run with the CWD set to their own
package directory. `auth_test.go` is the precedent.
