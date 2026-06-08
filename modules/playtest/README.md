# playtest module

Server-side policy for the GoMud AI playtest harness. It compiles into a GoMud
server and applies its behaviors to any client **connected on the AI port** —
per-round beacons, a structural safe mode, and death protection. It does **not**
create, provision, or own an account: the AI agent logs into a character (or
creates one via the normal new-player flow) exactly like a real player.

## Requirements (read before installing)

The module registry has no dependency field, so these prerequisites are listed
here — installing without them will fail to build or silently lose features:

- **A GoMud with the AI-port primitives** (the AI listener + `connections`
  AI-connection type). These landed via the engine AI-port PR; on a GoMud
  release/commit that predates it, the module **will not compile**.
- **The bundled `gmcp` module** — required for the `Playtest.Round` beacons. If
  it's absent, beacons disable gracefully (startup warning), but you lose the
  per-round pacing/state tick the harness relies on.

There is **no account/password to configure** — see "What it does".

## What it does

Everything keys off whether a live session is **on the AI port** (a client on
the AI port is a tester by definition — no account flag or provisioning needed):

- **Per-round beacons** (`beacons.go`): emits a `Playtest.Round` GMCP package to
  every live AI-port session at the end of each round (see Beacons).
- **Structural safe mode** (`safemode.go`): when a `SandboxZoneTag` is set, an
  AI-port tester that leaves a room carrying that tag is **snapped back** (a
  `RoomChange` listener — the event fires post-move, so this is a fail-closed
  snap-back, not a pre-move veto). Keeps the tester away from live players.
- **Death protection** (`safemode.go`): on AI-port spawn, grants the character a
  high `ExtraLives` count so it survives within GoMud's global permadeath setting.
- **Admin commands** (`commands.go`): `ai-flag <username> [on|off]` and `ai-list`
  (admin-only) — manage the engine's `IsAI` account flag (e.g. for leaderboard
  exclusion). Optional; the module's own behaviors don't depend on it.

## Configuration

Keys live under `Modules.playtest.*`:

| Key | Default | Meaning |
|-----|---------|---------|
| `Enabled` | `true` | Master switch for the module. |
| `SafeMode` | `true` | Enable confinement (needs `SandboxZoneTag`). |
| `SandboxZoneTag` | `""` | Room tag to confine the tester to. Empty = no confinement. |
| `DeathProtection` | `true` | Grant AI-port characters a high `ExtraLives`. |
| `Beacons` | `true` | Emit a `Playtest.Round` GMCP beacon each round to AI-port sessions (requires the `gmcp` module). |

> **How to set these:** edit this module's own defaults file,
> **`files/data-overlays/config.yaml`**, then restart the server. Do **not** add
> a `Modules:` → `playtest:` block to the server's `config.yaml` /
> `config-overrides.yaml`: the module's overlay default overrides a hand-edited
> base `config.yaml`, and a nested `Modules.*` block in `config-overrides.yaml`
> does not merge (it leaves the value empty). The admin web config UI / config
> API is the intended long-term path (being finalized). Full operator
> walkthrough + troubleshooting:
> <https://github.com/GoMudEngine/GoMud-Module-Playtest-Harness/blob/main/docs/usage/playtest-module.md>

## Beacons

`Beacons: true` (default) emits a `Playtest.Round` GMCP package to every client
on the AI port at the end of each game round. The payload is:

```json
{"round": 42, "hp": 30, "hp_max": 50, "sp": 10, "sp_max": 20, "room_id": 1001}
```

This gives the agent a reliable per-round pacing tick plus a compact state
snapshot it can use to score goal progress between commands.

Requires the bundled `gmcp` module. If `gmcp` is absent (its `SendGMCPEvent`
export is not in the plugin registry), the beacon disables gracefully with a
startup warning — no crash.

## Notes

- **"…not flagged as AI but connected on the AI port"** may appear in the
  server log / client when a tester logs in. It's **benign** — the module keys
  off the connection, not the `IsAI` flag, so it doesn't pre-flag accounts.
  Beacons and safe mode work regardless. Use `ai-flag <name>` if you want the
  account excluded from a leaderboard.
- **NoCombat buff is deferred.** Combat damage is applied synchronously, so a
  module can't cancel an attack mid-flight. Safety is delivered by sandbox
  confinement + death protection. Without a `SandboxZoneTag`, the tester roams
  the live world as a death-protected character.

## Development

This module only compiles inside a GoMud checkout. Source of truth is the
`GoMud-Module-Playtest-Harness` repo (`module/playtest/`); develop by placing it
under a GoMud checkout's `modules/playtest/`, then `go generate && go build`.
