# Generalized MUD Playtest Harness — Design

**Date:** 2026-06-05
**Status:** Approved design, pending implementation plan
**Author:** Calabe Davis (with Claude)

## Summary

Generalize DOGMud's internal "smoketester" framework into an
engine-agnostic, agent-agnostic AI playtest harness that **any** GoMud
server operator can use and **any** AI agent framework can drive. The
deliverable is published as a community module via the
[GoMud-Modules registry](https://github.com/GoMudEngine/GoMud-Modules),
exercising both the *creation* and *consumption* sides of the module
system the upstream owner asked contributors to test.

The work is delivered as three artifacts in a **single standalone
repository**, developed against the existing vanilla GoMud checkout at
`~/GoMud` as the live test target.

## Motivation

DOGMud already has a working AI smoketester (`tools/mud_bridge.py`,
`tools/testing/roles/*.md`, `tools/testing/goals/*.yaml`,
`.claude/commands/test-mud.md`). It works, but it is:

- **Claude-Code-specific** — driven by a file-poll loop
  (`mud_cmd.txt` / `mud_output.txt`) and a `sleep 2 && sleep 2` cadence
  baked into a slash command.
- **DOGMud-specific** — personalities and goals reference DOGMud
  content, commands, and zones.
- **Brittle** — goal verification scrapes cleaned ANSI text rather than
  reading structured state.

The upstream GoMud owner asked the community to test creating modules
and using the module registry. This project both serves that request
and produces a genuinely reusable tool for every GoMud server.

## Goals

- Any AI agent framework can connect to a GoMud server on `:55555`, run
  a set of goals using a standard set of personalities, and produce a
  structured report.
- Engine-agnostic: works against vanilla GoMud, not just DOGMud.
- Robust goal verification via structured state, not text scraping.
- Distributed through the official module registry.

## Non-Goals

- Replacing DOGMud's existing internal testing assets (they remain;
  this is a parallel, generalized tool).
- Building an AI agent / LLM runner. The harness exposes a contract;
  bring-your-own agent.
- An in-game admin command to launch runs (considered and **cut** —
  external agents don't need it; revisit only if demand appears).

## Architecture — three artifacts

### A. `playtest` server module (the registry citizen)

A GoMud module (Go) that compiles into the server. Registered the
standard way: blank-import in `modules/all-modules.go`
(`github.com/GoMudEngine/GoMud/modules/playtest`), shipped as a
tarball + sha256 referenced from `module-registry.yaml`. Capabilities:

1. **Test-account auto-provisioning.** At boot, idempotently ensure a
   configurable AI-test account exists, flagged as AI/test (so it is
   auto-excluded from leaderboards and similar player-facing surfaces).
   This removes the brittle login/create dance from the adapter.
2. **Safe / sandbox mode.** Flagged test accounts:
   - cannot harm live (non-test) players,
   - optionally have permadeath disabled,
   - can optionally be confined to a tagged sandbox zone.
   Gates **fail closed**: if a confinement/safety guarantee cannot be
   honored, the harmful action is refused rather than silently allowed.
3. **GMCP test-beacon enrichment.** A small sub-package that emits
   structured `Playtest.*` GMCP messages (e.g. command-acknowledgment,
   goal-relevant state markers) so the adapter scores goals from
   structured data instead of ANSI scraping. **Depends on the existing
   `gmcp` module** — it does not reimplement GMCP. (Phase-2 capability.)

Module config defaults and help files ship in the module's `datafiles/`
directory, per GoMud module conventions.

### B. `mudagent` reference adapter (the "any agent" contract)

A **Go single static binary** — matches the engine, cross-compiles to
one dependency-free executable per OS, and ships naturally alongside the
module. Responsibilities:

- Connect to telnet `:55555`, handle all IAC / GMCP negotiation and
  login (using the provisioned test account), so the agent never
  touches sockets or the login flow.
- Expose a **line-in / JSON-line-out stdio protocol** (see below). Any
  agent framework spawns it as a subprocess.
- Stream structured events: cleaned game text, GMCP state snapshots
  (vitals / room / inventory, from the `gmcp` module), `Playtest.*`
  beacon events, round-tick boundaries, and connection status.

### C. Engine-agnostic framework content + spec

- **Personality schema** plus the standard three personalities
  (`bug-finder`, `feature-tester`, `feel-tester`), rewritten with no
  DOGMud specifics.
- **Goals schema** — game-agnostic YAML describing session objectives.
- **Report-format spec** — the structured markdown report shape.
- **Reference Claude Code driver** (a slash command) demonstrating one
  agent consuming the adapter end-to-end. This proves the contract; it
  is not the only supported consumer.

## Data Flow

```
agent runner ──spawn──▶ mudagent --target host:55555 --manifest run.yaml
   ▲  │                      │
   │  │ JSON events (stdout) │ telnet + GMCP
   │  ▼                      ▼
 decide cmd ──stdin──▶  [GoMud :55555 + playtest module + gmcp module]
```

1. The agent runner spawns `mudagent`, pointing it at the target and a
   run manifest (active personality + goals).
2. The adapter connects, logs in to the provisioned test account, and
   streams JSON events to stdout.
3. The agent reads events, decides the next command, and writes it to
   the adapter's stdin.
4. The adapter sends the command, waits one round, and streams the
   response.
5. The agent verifies goals against beacon / GMCP events plus text, and
   on completion writes a report per the spec.

## JSON Protocol

**Events** — one JSON object per line on stdout:

```json
{"type":"output","text":"<cleaned>","raw":"<ansi>"}
{"type":"gmcp","package":"Char.Vitals","data":{ }}
{"type":"beacon","event":"command_ack","data":{ }}
{"type":"status","state":"connected"}
{"type":"error","message":"..."}
```

`status.state` is one of `connected | logged_in | disconnected`.

**Commands** — one per line on stdin:

- A plain text line is sent verbatim to the MUD.
- `{"control":"quit"}` (and future control verbs) drive the adapter
  itself rather than the game.

## Error Handling

- **Adapter:** reconnect with backoff; surface disconnects as a
  `status` event; exit non-zero on a fatal/unrecoverable condition.
- **Module:** provisioning is idempotent (safe to run every boot);
  safe-mode gates fail closed.

## Testing Strategy

- **Module:** Go unit tests for provisioning idempotency and safe-mode
  gating; the standard GoMud boot test (server starts cleanly past
  data-file loading with the module registered).
- **Adapter:** Go unit tests for protocol encode/decode; an integration
  test against a mock telnet server.
- **End-to-end:** run `mudagent` against the local `~/GoMud` server with
  each personality and a smoke goals file; confirm a report is produced.

## Phasing

1. **Phase 1 — Core harness, text + standard GMCP.**
   `mudagent` adapter + generalized content/spec + the module's
   account-provisioning and safe-mode (no beacon yet). Get an
   end-to-end run green against `~/GoMud`.
2. **Phase 2 — Structured goal verification.**
   GMCP test-beacon enrichment in the module + adapter beacon plumbing +
   goal auto-scoring from structured state.
3. **Phase 3 — Publish.**
   Tarball packaging, sha256, registry PR to `module-registry.yaml`,
   and documentation. Includes a validation pass confirming the harness
   works against a clean GoMud server (the `~/GoMud` checkout already
   provides this).

## Repository & Development Setup

- **Published source of truth:** a new **standalone repository**
  (engine-agnostic; clean release tags and tarball URLs for the
  registry; its own README / issues / license / CI). Holds the adapter,
  the framework content + spec, and the `playtest` module source.
- **Compile / run host:** the existing vanilla GoMud checkout at
  `~/GoMud` (master, recent). The `playtest` module is a registry-style
  package (`package playtest`, importing core GoMud packages); it only
  compiles when wired into a GoMud checkout. During development, copy or
  symlink the module source into `~/GoMud/modules/playtest` and add its
  blank-import to `modules/all-modules.go` (or run `go generate`).
- **Why vanilla GoMud, not DOGMud:** developing against `~/GoMud` proves
  the "works on any GoMud server" claim from day one and keeps the tool
  free of DOGMud coupling. DOGMud is out of this project's loop.

## Naming

- Server module: `playtest`
- Reference adapter binary: `mudagent`

(Working names; trivially renameable before publish.)

## Open Questions / Deferred

- Final standalone-repo name and hosting org.
- Whether the beacon GMCP package warrants its own registry entry or
  stays bundled in `playtest` (lean: bundled).
- Additional personalities beyond the standard three (deferred — start
  with three, add on demand).
