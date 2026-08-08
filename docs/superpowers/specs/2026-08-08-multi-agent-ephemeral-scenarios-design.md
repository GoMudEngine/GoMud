# Multi-Agent Ephemeral Scenarios — Design

**Chunk:** 0.3d (Adversarial Review Remediation Roadmap)  
**Status:** Draft after brainstorm 2026-08-08 (awaiting adversarial spec
review + user approval; **not** an implementation green light)  
**Depends on:** 0.3a (playtestenv), 0.3b (profiles/creds), 0.3c (playtestrun)  
**Feeds:** later harness polish; not a new LLM runner

## Goal

Wire local **multi-agent** playtests onto one shared ephemeral server: a
scenario file lists a roster of actors; each actor has its **own goals file**
and **own character bind/loadout** (`ephemeral:`); Go starts one disposable
env, materializes all profile-bound actors, creates per-actor mudagent bridges
plus a file blackboard, and enforces a **scenario wall-clock**; Claude drives
**concurrent** mudagents and writes one **combined** gameplay report.

Independent non-interacting parallel work stays **multiple 0.3c
`playtestrun run` invocations** (often on separate checkouts/worktrees). 0.3d
is for scenarios that need a **shared world** (party, PvP, same-server
concurrency, coordinated group_goals).

## Non-goals

- Wiring or requiring the `ptorch` binary (Claude orchestrates; optional later)
- Hard per-actor token or turn kill switches in Go (soft driver guidance only)
- One Docker env per actor / fan-out to N independent `playtestrun run`s for
  interactive scenarios
- Go spawning or supervising mudagent processes
- Production or remote targeting; reading `_archive/prod-users` at runtime
- Bulk migration of every single-agent goals file
- Dead-code cleanup of the pre-0.3c local playtest path (still deferred)
- Replacing mudagent or moving gameplay judgment into Go

## Decisions (locked in brainstorm)

| Topic | Choice |
|-------|--------|
| Scope | Roadmap 0.3d multi-agent ephemeral scenarios |
| Topology | **One shared env** per scenario; N actors |
| Non-interacting parallel | Just invoke 0.3c multiple times — out of 0.3d |
| Orchestration | **Hybrid:** Go owns env/roster/wall-clock/cleanup/sidecar; Claude drives N mudagents + combined report |
| Mudagents | **Concurrent** (one process + bridge per actor) |
| Binding | Explicit per-actor `ephemeral:` via each actor’s **goals file**; migrate existing scenarios |
| Goals / loadout | Each actor has its **own goals file** and character build (profile+room[+overlays] or creation_flow) |
| Blackboard | Run-scoped **file** blackboard for Claude↔group orchestration |
| In-game coord | Prefer say/party/tell/etc. for character↔character when co-located |
| Actor early stop | `on_actor_stop: continue\|abort`; **default continue** |
| Per-actor budgets | Soft guidelines only; **scenario wall-clock** is the hard cut |
| Go surface | Extend **`playtestrun scenario`** (not a new binary) |
| Approach | Scenario supervisor in playtestrun (Approach 1) |

## Architecture

```text
/playtest-scenario --checkout PATH <scenario.yaml>
        │
        ▼
playtestrun scenario  (blocking wall-clock supervisor)
        │  parse scenario + each actor goals ephemeral:
        │  playtestenv.Start(Profiles=[…profile actors…])
        │  mkdir actors/<id>/bridge + blackboard/
        │  write scenario sidecar; print ready JSON
        ▼
Claude: N concurrent mudagents (one bridge each)
        │  in-game channels for character↔character
        │  file blackboard for Claude↔group signals
        │  soft per-actor token guidance
        ▼
combined gameplay report + playtestrun stop/cleanup
```

| Piece | Owner |
|--------|--------|
| Shared env, roster materialize, wall-clock, sidecar, bridge/blackboard dirs, cleanup | `playtestrun scenario` |
| Per-actor goals play + combined report | Claude `/playtest-scenario` |
| Character↔character timing | Prefer in-game comms |
| Group orchestration Claude must see | File blackboard |

## Budgets

| Budget | Enforcement | Notes |
|--------|-------------|--------|
| Scenario wall-clock | **Hard** (Go) | Default 45m unless scenario/CLI overrides; lease = wall_clock + ≥5m buffer |
| Per-actor tokens / pacing | **Soft** (Claude) | Guidelines only — do not hard-cut a nearly finished actor; wall-clock is the backstop |
| `AICommandsPerRound` | In-engine | Per-connection spam pacing, not a scenario turn budget |

At wall-clock: sidecar `incomplete_wallclock`, Stop env, combined report
**incomplete**. Soft token stop on one actor follows `on_actor_stop`.

## Scenario + per-actor binding

### Scenario YAML

```yaml
name: party-formation
mode: party                 # hint for Claude; Go need not interpret
on_actor_stop: continue     # continue | abort; default continue
budgets:
  wall_clock: 45m

roster:
  - id: leader              # unique; path segment for bridges
    personality: feature-tester
    goals: goals/party-leader.yaml
  - id: joiner
    personality: feature-tester
    goals: goals/party-joiner.yaml

group_goals:                # Claude-owned; Go ignores semantics
  - id: party-formed
    do: ...
    verify: ...
```

Goals paths resolve relative to `tools/playtest/` when not absolute.

### Per-actor goals file

Same 0.3c `ephemeral:` contract (KnownFields), plus that actor’s objectives /
verify blocks. Each actor brings its own loadout via profile overlays or
creation-flow:

```yaml
ephemeral:
  profile: early
  start_room: 5200
  overlays:
    set_gold: 100
goals:
  - ...
```

or:

```yaml
ephemeral:
  creation_flow: true
  creation_rationale: >
    This actor must exercise new-player onboarding.
```

### Fail-closed (before Docker)

| Case | Behavior |
|------|----------|
| Missing `--checkout` / scenario file | Error |
| Empty roster / duplicate roster `id` | Error |
| Missing or unreadable actor goals path | Error |
| Any actor fails `ParseGoalsEphemeral` | Error |
| Legacy `target:` / `onboarding:` without migrated bind | Error (fail closed for local ephemeral scenarios) |
| Unknown `on_actor_stop` | Error |
| Unknown keys under scenario root that Go parses | KnownFields / explicit allow-list — fail closed on unknown **Go-owned** keys; `group_goals` / `mode` / `summary` allowed as opaque Claude fields |

### Materialize + creds

- Profile-bound actors → one `playtestenv` `Profiles` list in **roster order**.
- Creation-flow actors → no profile entry; that actor’s ready `creds` is null.
- Duplicate template ids in one roster are allowed (e.g. two `early` actors).
  Ready JSON / scenario sidecar MUST map each roster `id` to exactly one
  username (and creds path). Implementation may stamp `actor_id` onto
  `creds.json` players and/or keep an ordered sidecar map built at Start —
  either way, `SelectCredsPlayer(profile)` alone is insufficient when two
  actors share a template.
- Never log or write passwords into markdown reports (paths only).

## Paths and IDs

- Run id: playtestenv `run_id` (no parallel “session id”).
- Scenario sidecar: `tools/playtest/.run/<run_id>/session.json` (scenario-shaped
  fields: roster summary, `on_actor_stop`, blackboard path, per-actor status).
- Actor bridge: `tools/playtest/.run/<run_id>/actors/<id>/bridge/`.
- Blackboard: `tools/playtest/.run/<run_id>/blackboard/` (signal files / small
  JSON; Claude read/write convention documented in context.md).

## CLI

```text
playtestrun scenario --checkout PATH --scenario PATH [--wall-clock 45m]
playtestrun status --checkout PATH --run ID
playtestrun stop --checkout PATH --run ID
```

`--wall-clock` overrides scenario `budgets.wall_clock`. `run` (single-agent)
remains unchanged from 0.3c.

### Ready JSON (one line)

Minimum fields: `run_id`, `endpoint`, `checkout`, `commit`, `dirty`,
`deadline_at`, `sidecar`, `blackboard_dir`, `on_actor_stop`,
`actors: [{id, personality, goals_path, bridge_dir, creds|null,
profile|creation_flow, …}]`.

## `/playtest-scenario` driver

1. Require `--checkout` + scenario file (no silent cwd; no `targets.yaml` for
   endpoint/creds).
2. Start `playtestrun scenario` (blocking watchdog held for the scenario).
3. **Ready-gate:** no mudagents unless ready; on `environment_failed` abort and
   point at env report.
4. Spawn **concurrent** mudagents on each actor `bridge_dir`; login via that
   actor’s creds or creation-flow.
5. Play: personalities + per-actor goals + `group_goals`; coordinate characters
   in-game when possible; use blackboard for Claude-level group orchestration.
6. Actor soft-stop / finished early → honor `on_actor_stop` (`continue` default).
7. Always `playtestrun stop`; write **one combined** gameplay report (optional
   per-actor sections). Wall-clock or scenario soft abort ⇒ outcome
   **incomplete**.

## Reports

| Artifact | Owner |
|----------|--------|
| Scenario `session.json` | `playtestrun` |
| `*-environment-failed.md` | `playtestenv` |
| Combined gameplay `*.md` | Claude `/playtest-scenario` |

Combined report header must include checkout, commit, dirty, run_id, roster
binding summary, wall-clock elapsed vs budget, sidecar status, per-actor
outcomes. Never passwords.

## Failure modes

| Failure | Behavior |
|---------|----------|
| Parse / bind errors | Exit before Docker |
| Env / materialize fail | Sidecar `environment_failed`; no mudagents |
| Wall-clock exceeded | `incomplete_wallclock`; Stop; incomplete combined report |
| Actor soft token/API stop + `continue` | Stop that mudagent; others continue; report notes it |
| Actor stop + `abort` | Stop scenario; incomplete combined report |
| SIGINT / cancel | `interrupted` + non-zero (parity with 0.3c hardening) |
| Driver crash | Lease/reap recovery; may leave non-terminal sidecar briefly |

## Migration (in scope)

Migrate existing `tools/playtest/scenarios/*.yaml` to the new roster shape and
split per-actor goals files under `tools/playtest/goals/` (or
`goals/scenarios/<name>/`) as needed so each actor has its own goals +
`ephemeral:` bind. Ship `/playtest-scenario` command docs + verbose Human
invocation updates in `internal/playtestrun/context.md`.

## Testing

- **Unit:** scenario parse matrix; duplicate ids; bad `on_actor_stop`; per-actor
  goals ephemeral failure; ready JSON actor list; bridge/blackboard paths;
  continue vs abort policy hooks (fakes).
- **Opt-in Docker:** ≥2 profile actors through `playtestrun scenario` → ready →
  stop; optional mixed creation-flow + profile if cheap.
- Driver-contract smoke for ready JSON + run-scoped actor bridges (no full
  Claude session required).
- No Done without Docker evidence + adversarial **implementation** review after
  plan approval.

## Process gates

1. Adversarial **spec** review → revise → user approves spec  
2. Implementation **plan** → adversarial plan review → user approves plan  
3. Only then TDD implementation  

## Brainstorm record (2026-08-08)

0.3d confirmed; per-actor budgets soft / wall-clock hard; one shared env;
non-interacting parallel = multiple 0.3c; Hybrid Claude+N mudagents; concurrent
agents; explicit per-actor ephemeral via own goals files; file blackboard +
in-game channels; `on_actor_stop` default continue; `playtestrun scenario`;
Approach 1.
