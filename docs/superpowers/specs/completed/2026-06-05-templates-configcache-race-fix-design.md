# Fix: templates.Process concurrent-map-write crash

**Status:** Design approved — pending implementation
**Type:** Standalone hotfix (branch off `master`, independent of mob-aliveness 6.4)
**Date:** 2026-06-05
**Severity:** High — whole-server fatal panic when two telnet connections collide

## Problem

`internal/templates/templates.go` keeps a per-user render-config cache:

```go
templateConfigCache = make(map[int]templateConfig)   // line 56
```

It is accessed from three sites, none of which safely synchronizes the map:

| Site | Access | Lock held |
|------|--------|-----------|
| `Process()` ~140 | read `templateConfigCache[userId]` | `ansiLock.RLock()` |
| `Process()` ~153 | write `templateConfigCache[userId] = …` | `ansiLock.RLock()` (a **read** lock) |
| `ClearTemplateConfigCache()` 114 | `delete(templateConfigCache, userId)` | **none** |

`ansiLock` exists to protect the **ANSI alias configuration** — `LoadAliases()`
takes `ansiLock.Lock()` (write) while `Process()`/`AnsiParse()` take `RLock()`
to read it. It was never intended to guard `templateConfigCache`. Because
`Process()` holds only a *read* lock, many `Process()` calls execute
concurrently; when two of them write the map at once — e.g. two fresh telnet
connections both rendering `login/connect-splash` with `userId == 0` (main.go:652,
before login assigns a real id) — Go's runtime raises
`fatal error: concurrent map writes` and the **entire server process dies**
(exit status 2). `ClearTemplateConfigCache`'s lock-free `delete` is even more
exposed.

### Discovery

Found 2026-06-05 during the mob-aliveness 6.4 baseline capture: a scripted
telnet poller that opened overlapping connections crashed the idle server twice.
A single strictly-sequential connection never triggers it. Latent for real
players (who rarely connect within the same millisecond) but a genuine hazard
for any burst of simultaneous connects: bot swarms, parallel AI testers,
reconnect storms after a restart, or load tests. See memory
`project_templates_configcache_concurrent_write_panic`.

## Fix

Replace `templateConfigCache` with a `sync.Map`, whose `Load`/`Store`/`Delete`
are each individually atomic — eliminating the entire data-race class with no
manual locking and no coupling to `ansiLock` (which keeps doing its real job).

### Changes

1. **Declaration (line ~56):**
   ```go
   templateConfigCache sync.Map // key: int userId -> value: templateConfig
   ```
   (Remove it from the `make(...)` initializer list; a zero `sync.Map` is ready
   to use.)

2. **Extract a helper for testability.** Lift the current build-or-cache block
   (lines ~140–153) out of `Process()` into:
   ```go
   // getTemplateConfig returns the cached render config for userId, building
   // and caching it on first use. Safe for concurrent use.
   func getTemplateConfig(userId int) templateConfig {
       if v, ok := templateConfigCache.Load(userId); ok {
           return v.(templateConfig)
       }
       cfg := templateConfig{}
       if userId > 0 {
           if tmpU := users.GetByUserId(userId); tmpU != nil {
               cfg.ScreenReader = tmpU.ScreenReader
           }
       } else if userId == ForceScreenReaderUserId {
           cfg.ScreenReader = true
       }
       templateConfigCache.Store(userId, cfg)
       return cfg
   }
   ```
   `Process()` then calls `tplConfig := getTemplateConfig(userId)` in place of the
   inline block. The build happens outside any cache lock; on a simultaneous miss
   two goroutines may both build and `Store` — harmless, the value is
   deterministic (identical config for a given userId).

3. **`ClearTemplateConfigCache` (line 114):**
   ```go
   func ClearTemplateConfigCache(userId int) {
       templateConfigCache.Delete(userId)
   }
   ```

4. **`internal/templates/context.md`:** update the `templateConfigCache` line
   (currently "`map[int]templateConfig` - Per-user configuration cache") to note
   it is now a `sync.Map` for concurrency-safety, accessed via `getTemplateConfig`.

### What stays unchanged

- `ansiLock` and its protection of the ANSI alias config (`LoadAliases`,
  `AnsiParse`, the `forceAnsiFlags` reads in `Process`). `Process` still takes
  `ansiLock.RLock()` for that purpose; the cache ops just no longer rely on it.
- `templateCache` (compiled-template cache, `map[string]cacheEntry`) — already
  guarded by its own `cacheLock sync.Mutex`. Out of scope.

## Out of scope (observed, not fixed here)

- `SetAnsiFlag()` writes the package global `forceAnsiFlags` with no lock while
  `Process`/`AnsiParse` read it under `RLock`. A separate minor race, but
  `SetAnsiFlag` is a boot-time configuration call (set once before traffic), so
  it does not crash under normal operation. Noted for a future tidy; not part of
  this crash-fix.

## Testing

TDD, race-detector-driven:

1. **Failing/▶ regression test** `internal/templates/templates_concurrency_test.go`:
   spawn N goroutines (e.g. 50) that concurrently call `getTemplateConfig(0)`,
   `getTemplateConfig(i)` for varied ids, and `ClearTemplateConfigCache(...)` in
   a tight loop. Run with `go test -race ./internal/templates/`. Against the old
   `map` this reproduces `fatal error: concurrent map writes` / a race report;
   against the `sync.Map` fix it passes clean.
   - `getTemplateConfig(0)` and small ids without a seeded user record take the
     `userId == 0` / `GetByUserId == nil` paths, so the test needs no users
     fixture.
2. `go build ./...` clean; `go test ./internal/templates/` green; full
   `go test ./...` green.
3. Boot smoke (after the 6.4 capture frees the server): server starts clean and
   a normal login still renders the connect-splash and prompts.

## Files touched

- Edit: `internal/templates/templates.go` (declaration, extract `getTemplateConfig`,
  `Process` call-site, `ClearTemplateConfigCache`).
- New: `internal/templates/templates_concurrency_test.go`.
- Edit: `internal/templates/context.md`.

## Branch / landing

Branch `fix/templates-configcache-race` off `master` (independent of the 6.4
work). Conventional commits; merge `--no-ff`. On landing, update the
`project_templates_configcache_concurrent_write_panic` memory to resolved.
