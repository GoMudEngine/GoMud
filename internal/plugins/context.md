# Plugin Registry Context

## Purpose

`internal/plugins` is the registration surface that lets a `modules/*` package
add commands, web pages, embedded data files, and IAC handlers to the engine
without the engine importing it back. A module calls `plugins.New(name,
version)` from its `init()`, decorates the returned `*Plugin`, and the engine
later walks the registry.

**These are compile-time plugins, not loadable ones.** The header comment in
`plugins.go` says it plainly: a new plugin must be dropped into the modules
folder and the server recompiled. There is no dynamic loading, no `.so`, and no
sandbox.

This package is the reason `modules/` is invisible to grep from inside
`internal/` — the dependency arrow points one way only, and everything crosses
back through this registry.

## Files

- **plugins.go** — the registry, the `Plugin` type, `New`, `Load`, `Save`, and
  the per-plugin data-file/embed plumbing.
- **plugincallbacks.go** — `PluginCallbacks`, the `NetConnection` interface, and
  the `Set*` hook setters.
- **pluginconfig.go** — `PluginConfig`, a name-scoped `Get`/`Set` bag.
- **pluginfiles.go** — `PluginFiles`, the `fs.ReadFileFS` view over a plugin's
  embedded FS.
- **webconfig.go** — `WebConfig`/`WebPage`, nav links and page registration.

## The `Plugin` type

```go
type Plugin struct {
    name              string
    version           string
    dependencies      []dependency
    Callbacks         PluginCallbacks
    exportedFunctions map[string]any
    Config            PluginConfig
    files             PluginFiles
    Web               WebConfig
}
```

Only `Callbacks`, `Config`, and `Web` are exported — everything else is reached
through methods.

## Registration API

```go
func New(name string, version string) *Plugin   // nil once registration closes
func (p *Plugin) Requires(modname, modversion string)
func (p *Plugin) ExportFunction(stringId string, f any)   // panics on non-func
func (p *Plugin) AddUserCommand(command string, handlerFunc usercommands.UserCommand, allowWhenDowned, isAdminOnly bool)
func (p *Plugin) AddMobCommand(command string, handlerFunc mobcommands.MobCommand, allowWhenDowned bool)
func (p *Plugin) AttachFileSystem(f embed.FS) error
```

Callbacks:

```go
func (c *PluginCallbacks) SetIACHandler(f func(uint64, []byte) bool)
func (c *PluginCallbacks) SetOnLoad(f func())
func (c *PluginCallbacks) SetOnSave(f func())
func (c *PluginCallbacks) SetOnNetConnect(f func(NetConnection))
```

Web:

```go
func (w *WebConfig) NavLink(name, path string)
func (w *WebConfig) WebPage(name, path, file string, addToNav bool, dataFunc func(r *http.Request) map[string]any)
```

Per-plugin persistence (scoped to a `<name>-v<version>` folder):

```go
func (p *Plugin) WriteBytes(identifier string, bytes []byte) error
func (p *Plugin) ReadBytes(identifier string) ([]byte, error)
func (p *Plugin) WriteStruct(identifier string, in any) error
func (p *Plugin) ReadIntoStruct(identifier string, out any) error
```

## Engine-side API

```go
func Load(dataFilesPath string)
func Save()
func GetPluginRegistry() pluginRegistry
func OnNetConnect(n NetConnection)
func ReadFile(dfPath string) ([]byte, error)
```

`pluginRegistry` implements `fs.ReadFileFS` (`ReadFile`/`Open`/`Stat`) plus
`GetExportedFunction`, `NavLinks`, `HandleIAC`, `WebRequest`, and
`AllFileSubSystems` — that last set is what satisfies the `web.WebPlugin`
interface, so web pages can be served straight out of a module's embedded FS.

## Embedded file layout

`AttachFileSystem` walks the embed and builds a short-path → embed-path map,
recognising exactly two folder names:

- `datafiles/` — mapped with the prefix **stripped**, so a plugin's
  `datafiles/help/foo.md` is fetched as `help/foo.md`.
- `data-overlays/` — mapped with the prefix **kept**, because overlays are
  looked up by their full `data-overlays/...` path when merging onto existing
  engine data.

Paths use forward slashes unconditionally; `embed.FS` does, even on Windows.

## Gotchas

- **`New` returns `nil` after registration closes.** Registration is only open
  during startup — call it from `init()` or a module's early setup, never
  lazily. A `nil` return will panic at the first method call, which is the
  intended loud failure.
- **`ExportFunction` panics if handed a non-func.** Deliberate: a typo'd export
  should fail at boot rather than at first cross-module call.
- **Names are sanitised, not rejected.** `New` and `Requires` regex-replace
  anything outside `[a-zA-Z0-9_]` with `_`, so `my-plugin` and `my_plugin`
  collapse to the same name. Watch for accidental collisions.
- **Exported ids are a flat global namespace.** `GetExportedFunction` searches
  every plugin and returns the first match, so `ExportFunction` **panics on a
  duplicate id**, naming both plugins. Qualify your ids
  (`"weather.GetFront"`, not `"GetFront"`).
- **`Requires` IS enforced, at `Load`.** Every recorded dependency is checked
  against the registry; an unmet one logs a `mudlog.Error` and the plugin is
  dropped from the registry. Note the match is exact-string on **both** name and
  version, so requiring `"1.0"` fails against a plugin declaring `"1.0.0"` —
  the source carries a `// Later improve version matching.` TODO.
- **Plugin writes before `Load(dataFilesPath)` land in `os.TempDir()`** and
  will never be read back. `WriteBytes` warns loudly when this happens.

## Dependencies

`configs`, `mobcommands`, `mudlog`, `usercommands`, `util`, plus `embed`,
`io/fs`, `net/http`, `reflect`, and `gopkg.in/yaml.v2`.

Note the import of `usercommands` and `mobcommands`: this package sits *above*
the command packages and *below* `modules/`, which is what keeps the graph
acyclic.

## Consumers

Every module: `modules/achievements`, `auctions`, `cleanup`, `follow`, `gmcp`,
`leaderboards`, `playtest`, `time`, `weather`, `webhelp` — plus `internal/hooks`
on the engine side.
