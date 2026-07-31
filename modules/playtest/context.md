# Playtest Module Context

## Purpose

`modules/playtest` is the in-server half of the AI playtest harness. It does
three things for an AI-driven tester connection: emits a structured per-round
**beacon** so the driver can pace itself and verify state, provides admin
commands to flag a character as an AI tester, and applies **safe mode** so a
tester cannot wander somewhere that invalidates the run or die pointlessly.

The driver side lives outside the repo (`mudagent`, located via
`GOMUD_HARNESS_DIR`); the DOGMud overlay lives in `tools/playtest/`.

## Files

- **playtest.go** — `PlaytestModule`, registration, `isAIConnection`.
- **beacons.go** — the per-round `Playtest.Round` GMCP beacon.
- **commands.go** — the AI-flag admin commands.
- **safemode.go** — sandbox containment and death protection.
- **config.go** — module config, read through the plugin config bag.

## Beacons

`onNewRound` emits a `Playtest.Round` GMCP payload carrying **hp/sp/cp with
their maxima, and the current room**. That is what lets the driver wait for a
real round to elapse rather than sleeping, and lets a scenario assert on state
instead of scraping prose.

Beacons are opt-in via `Modules.playtest.*` config.

## Safe mode

```go
shouldSnapBack(isAITester bool, sandboxTag string, destTags []string) bool
```

A tester is confined to rooms carrying the configured sandbox tag; a move to an
untagged destination snaps them back. `applyDeathProtection` keeps a tester
alive so a run is not ended by an unlucky fight.

Both apply **only** to connections identified by `isAIConnection`. A normal
player is never affected.

## Gotchas

- **Everything here is gated on the AI flag.** A test that seems to show safe
  mode leaking into normal play is much more likely to be a mis-flagged
  character.
- **Beacons are config-gated and off by default.** A driver that hangs waiting
  for one is usually looking at a server with `Modules.playtest` unset.
- **The AI port strips colour.** Verifying ANSI output requires a real telnet
  connection on 33333, not the harness.
- **Testers observe; they do not fix.** Findings come back as reports — nothing
  in this module should acquire the ability to mutate world content.
- **Config is read through the plugin config bag** (`asString`/`asBool` on
  `any`), so a mistyped key yields a zero value rather than an error.

## Dependencies

`plugins`, `events`, `users`, `rooms`, `mudlog`, and the GMCP module.

## Consumers

The external `mudagent` driver, via GMCP and the AI port.
