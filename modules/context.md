# Modules Context

## Purpose

`modules/` holds the engine's plugins: self-contained packages that add
commands, listen to events, serve web pages, and ship their own data files,
without the engine importing them.

The direction of dependency is the whole point. `internal/` **never** imports
`modules/`. A module registers itself with `plugins.New(name, version)` from its
`init()`, and the engine walks the registry. That is why a grep from inside
`internal/` cannot find module code, and why a feature you cannot locate may
well live here.

The registration API — `Plugin`, callbacks, config, embedded files, web pages —
is documented once, in [`internal/plugins/context.md`](../internal/plugins/context.md).
This file is the index of what exists.

**These are compile-time plugins.** Adding one means dropping a package in here
and rebuilding. There is no dynamic loading and no sandbox.

## The modules

| Module | What it does |
|--------|--------------|
| [`achievements`](achievements/context.md) | Event wiring, unlock recording, and the web page for `internal/achievements` |
| [`auctions`](auctions/context.md) | The global auction house, including the NPC bidder panel |
| [`cleanup`](cleanup/context.md) | `trash` and `bury`, in user and mob variants |
| [`follow`](follow/context.md) | `follow` for players and mobs, plus every link-breaking listener |
| [`gmcp`](gmcp/context.md) | The GMCP protocol layer — the web client's entire data feed |
| [`leaderboards`](leaderboards/context.md) | Ranked top-N tables for the command and the dashboard |
| [`playtest`](playtest/context.md) | AI-playtest beacons, tester flagging, and safe mode |
| [`time`](time/context.md) | The player-facing `time` command |
| [`weather`](weather/context.md) | Weather simulation: fronts, seasons, and zone climate |
| [`webhelp`](webhelp/context.md) | Help files rendered as web pages |

`weather` is itself a tree — [`content`](weather/content/context.md),
[`crawler`](weather/crawler/context.md), [`engine`](weather/engine/context.md),
[`seasons`](weather/seasons/context.md), [`sim`](weather/sim/context.md) — and
is **vendored from a standalone repository**. Its packages carry architecture
tests (`TestCrawlerPackageStaysPure`) that forbid engine imports; do not
"simplify" them by reaching into `internal/`.

## Writing a module

```go
func init() {
    p := plugins.New("mymodule", "1.0")
    p.AddUserCommand("mycmd", handler, false, false)
    p.Callbacks.SetOnLoad(load)
    p.Web.WebPage("My Page", "/mypage", "mypage.html", true, dataFunc)
    p.AttachFileSystem(embeddedFiles)
}
```

Conventions the existing modules follow:

- **Registration happens in `init()`.** `plugins.New` returns `nil` once
  registration closes, so a lazily-registered plugin panics at its first
  method call.
- **State lives on the module struct**, loaded and saved through the
  `SetOnLoad` / `SetOnSave` callbacks and the plugin's own
  `ReadBytes`/`WriteBytes` / `ReadIntoStruct`/`WriteStruct` — not in the
  engine's world save. `plugins.Save()` is driven from
  `internal/hooks/NewTurn_AutoSave.go`.
- **A command available to both players and mobs is registered twice**
  (`AddUserCommand` and `AddMobCommand` take different handler types). Keep the
  two implementations in step; the boot-time command-parity check warns about
  unpaired commands.
- **Embedded data goes under `datafiles/`** (the prefix is stripped on lookup)
  or `data-overlays/` (the prefix is kept, for merging onto engine data).
- **Cross-module calls go through `ExportFunction` / `GetExportedFunction`**,
  a flat global id namespace with no collision checking — qualify your ids.

## Event handling

```go
func (m *MyModule) handleNewRound(e events.Event) events.ListenerReturn {
    if evt, ok := e.(events.NewRound); ok {
        m.processRoundLogic(evt.RoundNumber)
    }
    return events.Continue
}
```

The type assertion is required — listeners receive the interface, not the
concrete event.

## Gotchas

- **There is no sandbox and no isolation.** A module runs in-process with full
  access to everything it imports. "Plugin" here means "registered at compile
  time," not "untrusted."
- **Returning the wrong `ListenerReturn` swallows the event** for every listener
  behind you. This is the most common module bug.
- **Module config is read through an `any`-typed bag**, so a mistyped key
  yields a zero value silently rather than failing.
- **Plugin file writes land in `os.TempDir()` until `plugins.Load` runs.**
- **`modules/` is invisible from `internal/`.** Before building a new
  index/graph/crawl, grep `modules/` as well — the thing may already exist.

## Consumers

`main.go` (which imports the module packages for their `init()` side effects)
and `internal/plugins`, which serves the registry back to the engine.
