# Multi-Agent Ephemeral Scenarios — Design

**Chunk:** 0.3d (Adversarial Review Remediation Roadmap)  
**Status:** Revised after adversarial spec review 2026-08-08 (awaiting plan;
**not** an implementation green light)  
**Depends on:** 0.3a (playtestenv), 0.3b (profiles/creds), 0.3c (playtestrun)  
**Feeds:** later harness polish; not a new LLM runner

## Goal

Wire local **multi-agent** playtests onto one shared ephemeral server: a
scenario file lists a roster of actors; each actor has its **own goals file**
and **own character bind/loadout** (`ephemeral:`); Go starts one disposable
env, materializes all profile-bound actors, creates per-actor mudagent bridges
plus a file blackboard, and enforces a **scenario wall-clock**; Claude (or an
implementing agent) drives **concurrent** mudagents and writes one **combined**
gameplay report.

Independent non-interacting parallel work that does **not** need a shared
server stays **multiple 0.3c `playtestrun run` invocations**. Scenarios that
need a **shared world** (party, PvP-when-supported, same-server AI concurrency
coverage, coordinated `group_goals`) are **0.3d** — including “parallel” mode
files that still share one env to exercise concurrent AI connections.

## Non-goals

- Wiring or requiring the `ptorch` binary (driver is rewritten off ptorch)
- Hard per-actor token or turn kill switches in Go (soft driver guidance only)
- One Docker env per actor / fan-out to N independent `playtestrun run`s for
  interactive / shared-world scenarios
- Go spawning or supervising mudagent processes
- Production or remote targeting; reading `_archive/prod-users` at **runtime**
- Bulk migration of every single-agent goals file
- Dead-code cleanup of the pre-0.3c local playtest path (still deferred)
- Replacing mudagent or moving gameplay judgment into Go
- AI admin characters in multi-agent scenarios (hard-banned)
- Migrating PvP scenarios that require manual production `config.yaml` PvP
  edits until an allow-listed ephemeral override exists (defer; see Migration)

## Decisions (locked in brainstorm + review)

| Topic | Choice |
|-------|--------|
| Scope | Roadmap 0.3d multi-agent ephemeral scenarios |
| Topology | **One shared env** per scenario; N actors |
| Non-shared parallel | Multiple 0.3c runs when shared world **not** required |
| Shared-server concurrency coverage | **In** 0.3d (one env, N actors) |
| Orchestration | **Hybrid:** Go owns env/roster/wall-clock/cleanup/sidecar; driver drives N mudagents + combined report |
| Mudagents | **Concurrent** (one process + bridge per actor) |
| Binding | Explicit per-actor `ephemeral:` via each actor’s **goals file** |
| Goals / loadout | Each actor has its **own goals file** and character build |
| Duplicate templates | **Allowed but mildly discouraged** — prefer mixed loadouts/personalities |
| Admin in multi | **Hard ban:** no `profile: admin` / admin-role actors |
| Prod identities | **Hard ban:** prod-user example names + close variants (shared helper) |
| Blackboard | Run-scoped **file** blackboard (contract below; no ptorch) |
| In-game coord | Prefer say/party/tell/etc. for character↔character |
| Actor early stop | `on_actor_stop: continue\|abort`; **default continue** |
| Per-actor budgets | Soft; **scenario/CLI wall-clock** is the sole Go hard cut |
| Go surface | Extend **`playtestrun scenario`** |
| Creds mapping | Mandatory `actor_id` on creds players; never profile-only in multi |
| Pre-merge smoke | ~10m mixed-party via real `playtestrun scenario` ready-gate + N bridges |

## Architecture

```text
/playtest-scenario --checkout PATH <scenario.yaml>
        │
        ▼
playtestrun scenario  (blocking wall-clock supervisor)
        │  parse scenario + each actor goals ephemeral:
        │  playtestenv.Start(Profiles=[…profile actors…])
        │  stamp actor_id on creds; mkdir actors/<id>/bridge + blackboard/
        │  write scenario sidecar; print ready JSON
        ▼
Driver: N concurrent mudagents (one bridge each)
        │  in-game channels for character↔character
        │  file blackboard for driver↔group signals
        │  soft per-actor token guidance
        ▼
combined gameplay report + playtestrun stop/cleanup
```

| Piece | Owner |
|--------|--------|
| Shared env, roster materialize, wall-clock, sidecar, bridge/blackboard dirs, cleanup | `playtestrun scenario` |
| Per-actor play + combined report | `/playtest-scenario` (or implementer driving the same contract) |
| Character↔character timing | Prefer in-game comms |
| Group orchestration the driver must see | File blackboard |

## Budgets

| Budget | Enforcement | Notes |
|--------|-------------|--------|
| Scenario wall-clock | **Hard** (Go) | Default **45m**; CLI `--wall-clock` overrides scenario `budgets.wall_clock`; lease = wall_clock + ≥5m buffer |
| Per-actor `ephemeral.budgets.wall_clock` | **Ignored by Go** in scenario mode | Soft Claude/driver hint only (same spirit as ignoring legacy `max_rounds`) |
| Per-actor tokens / pacing | **Soft** (driver) | Do not hard-cut a nearly finished actor |
| `AICommandsPerRound` | In-engine | Per-connection spam pacing |

At wall-clock: sidecar `incomplete_wallclock`, Stop env, combined report
**incomplete**. Soft token stop on one actor follows `on_actor_stop`.

## Scenario YAML schema (Go-owned)

**Go-parsed / KnownFields (fail closed on unknown among these peers):**

| Key | Required | Notes |
|-----|----------|--------|
| `name` | yes | Scenario id string |
| `roster` | yes | Non-empty |
| `on_actor_stop` | no | `continue` \| `abort`; default `continue` |
| `budgets.wall_clock` | no | Duration string; default 45m |
| `summary` | no | Opaque string |
| `mode` | no | Opaque hint for driver (`party`, `parallel`, …) |
| `group_goals` | no | Opaque list for driver |
| `requires` | no | **Driver preconditions** only — Go does not fatal on unknown requires keys; driver warns/confirms. Special: `requires.pvp` / equivalent → **refuse migration run** until overrides exist (see Migration) |

**Roster entry (Go-parsed):**

| Key | Required | Notes |
|-----|----------|--------|
| `id` | yes | Unique; `[a-zA-Z0-9_-]+`; path segment |
| `personality` | yes | e.g. `feature-tester` (migrated from legacy `role`) |
| `goals` | yes | Path to goals file (relative to `tools/playtest/` or absolute) |

**Rejected (fail closed):** inline roster `goals:` lists; `target:`; `onboarding:`;
`role:` (use `personality`); missing `ephemeral:` in any actor goals file;
**unknown keys on roster entries** (KnownFields on the roster struct).

### Example

```yaml
name: party-formation
mode: party
on_actor_stop: continue
budgets:
  wall_clock: 45m
roster:
  - id: leader
    personality: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
  - id: joiner
    personality: feel-tester
    goals: goals/scenarios/party-formation/joiner.yaml
group_goals:
  - id: party-formed
    do: ...
    verify: ...
```

### Per-actor goals file

Same 0.3c `ephemeral:` contract (KnownFields), plus that actor’s objectives.
Profile+room[+overlays] or creation_flow+rationale.

### Fail-closed (before Docker)

| Case | Behavior |
|------|----------|
| Missing `--checkout` / scenario file | Error |
| Empty roster / duplicate roster `id` / invalid `id` | Error |
| Missing or unreadable actor goals path | Error |
| Any actor fails `ParseGoalsEphemeral` | Error |
| Legacy `target` / `onboarding` / `role` / inline goals | Error |
| Unknown `on_actor_stop` | Error |
| Unknown Go-owned scenario key | Error |
| Any actor `profile: admin` | Error |
| `len(roster) > MaxAIConnections` (from checkout config; default 20) | Error (optional `--force` for local break-glass only) |
| Scenario declares PvP require without supported override | Error / driver refuse |

### Roster diversity (guidance)

Duplicate template ids are **allowed but mildly discouraged**. Prefer mixed
personalities and loadouts. Go does not reject duplicates.

### Admin hard ban (multi-agent only)

- Reject any roster goals with `profile: admin` before Docker.
- Creation-flow actors in multi-agent runs are **RoleUser-only**; driver must
  not run admin-enabling sequences. No supported AI admin in multi-testing.
- Single-agent `playtestrun run` may still bind `admin`.

### Prod-identity hard ban (mandatory shared helper)

Playtest agents must **never** use account or character names from the
`_archive/prod-users` design-reference set, nor **close variants**.

**v1 algorithm (mandatory):**

1. Build denylist stems at **authoring/CI time** from the archive’s account and
   character names. Checked-in list lives at
   `internal/playtestprofiles/prod_identity_denylist.go` (or `.txt` generated
   beside it) — **not** a runtime read of `_archive/`.
2. A candidate name is banned if, after lowercasing and stripping separators
   (`_`, `-`, spaces), it:
   - equals a stem, or
   - equals `pt` + stem / stem with leading or trailing digits stripped to a
     stem, or
   - has Levenshtein distance ≤ 1 from a stem (length ≥ 4), or
   - contains a stem as a contiguous substring when stem length ≥ 5.

Applies to templates, generated usernames/character names at materialize, and
creation-flow `new` names (driver pre-check + shared helper; materialize path
uses the same helper). **Required** for both `playtestrun run` and `scenario`
(shared gate; not optional).

### Materialize + creds (locked)

1. Walk roster in order. Profile-bound actors append to `Profiles []` in that
   order. Creation-flow actors skip Profiles.
2. After Start, every `creds.json` player for this run **must** carry
   `actor_id` equal to the roster `id` (materializer / playtestenv stamp —
   extend `PlayerCreds`).
3. Ready JSON / scenario sidecar map each roster `id` → `{username, creds path
   or null}`. Multi-agent login **must** select by `actor_id`, never by
   `profile` alone (`SelectCredsPlayer(profile)` is single-agent-only).
4. Creation-flow actors: `creds: null`, `username` empty until driver completes
   `new` (still subject to prod-identity helper before accept).
5. Never log or write passwords into markdown (paths only).

## Paths and IDs

- Run id: playtestenv `run_id`.
- Scenario sidecar: `tools/playtest/.run/<run_id>/session.json`.
- Actor bridge: `tools/playtest/.run/<run_id>/actors/<id>/bridge/`.
- Blackboard: `tools/playtest/.run/<run_id>/blackboard/`.

### Scenario sidecar schema

```json
{
  "run_id": "...",
  "checkout": "...",
  "commit": "...",
  "dirty": true,
  "scenario_path": "...",
  "on_actor_stop": "continue",
  "blackboard_dir": "...",
  "budgets": { "wall_clock": "45m0s" },
  "started_at": "...",
  "deadline_at": "...",
  "status": "ready",
  "actors": [
    {
      "id": "leader",
      "personality": "feature-tester",
      "goals_path": "...",
      "bridge_dir": "...",
      "creds": ".../creds.json",
      "username": "pt_early_abc",
      "profile": "early",
      "creation_flow": false,
      "status": "ready"
    }
  ]
}
```

Scenario-level `status`: `starting` | `ready` | `incomplete_wallclock` |
`interrupted` | `stopped` | `environment_failed` | `incomplete_abort`.

Per-actor `status`: `pending` | `ready` | `stopped` | `failed` |
`incomplete` | `aborted_peer`.

### Blackboard contract

Directory: `tools/playtest/.run/<run_id>/blackboard/`.

| Convention | Rule |
|------------|------|
| Signal file | `<signal-name>.json` (e.g. `joiner-ready.json`); `signal-name` ∈ `[a-zA-Z0-9_-]+` |
| Payload | `{"signal":"<name>","actor_id":"<id>","ts":"<RFC3339>","data":{...}}` |
| Write | Write temp file then atomic rename into `blackboard/` |
| Read | Driver polls for file existence / JSON parse; ignore malformed with log |
| Clear | Optional `playtestrun stop` may leave files; new run gets new `run_id` |
| No ptorch | Driver must not invoke `ptorch bb`; file I/O only |
| Per-actor status | Driver updates scenario sidecar actor statuses (or a small
  `actors/<id>/status.json` the supervisor merges) when an actor stops;
  Go owns scenario-level status transitions (ready / wall-clock / stop) |

Character-level coordination should prefer in-game channels; blackboard is for
driver-visible group orchestration.

## CLI

```text
playtestrun scenario --checkout PATH --scenario PATH [--wall-clock 45m] [--force]
playtestrun status --checkout PATH --run ID
playtestrun stop --checkout PATH --run ID
```

`--force` only bypasses the MaxAIConnections roster-size check (local
break-glass). Single-agent `run` unchanged.

### Ready JSON (one line)

`run_id`, `endpoint`, `checkout`, `commit`, `dirty`, `deadline_at`, `sidecar`,
`blackboard_dir`, `on_actor_stop`,
`actors: [{id, personality, goals_path, bridge_dir, creds|null, username,
profile|null, creation_flow, status}]`.

## `on_actor_stop` semantics

| Policy | Behavior |
|--------|----------|
| `continue` (default) | Stop that actor’s mudagent only; mark actor `stopped`/`incomplete` in sidecar if updated; env and other actors continue until wall-clock or explicit `playtestrun stop` |
| `abort` | Driver stops **all** mudagents, calls `playtestrun stop`, scenario sidecar `incomplete_abort`, peer actors `aborted_peer`, env torn down; combined report **incomplete** |

Go wall-clock path always Stops the env regardless of policy.

## `/playtest-scenario` driver (rewrite)

**Mandatory rewrite** of `.claude/commands/playtest-scenario.md` off ptorch:

1. Usage: `/playtest-scenario --checkout <abs> <scenario-file>` (checkout
   required; no silent cwd; no `targets.yaml` for endpoint/creds).
2. Start `playtestrun scenario` as a **long-lived process** (same pattern as
   0.3c): keep the watchdog alive for the scenario window; consume the **one
   ready JSON line** as soon as it appears, then proceed while the supervisor
   remains running (do not treat start-and-exit as success).
3. **Ready-gate:** parse ready JSON; no mudagents unless `ready`.
4. Spawn concurrent mudagents on each `actors[].bridge_dir`; login via
   `actor_id` mapping / creation-flow. If any mudagent fails to start after
   ready: honor `on_actor_stop` (`continue` → mark that actor failed and keep
   others; `abort` → full scenario stop).
5. Play with personalities + per-actor goals + `group_goals`; in-game coord +
   file blackboard.
6. Honor `on_actor_stop`; always `playtestrun stop`.
7. Write **one combined** gameplay report per
   `tools/playtest/multi-agent-report-format.md` (also rewritten).

Also update `internal/playtestrun/context.md`: Human invocation for
`scenario`, blackboard contract, blocking supervisor (do not “start and
exit”), fix any gotcha that says to background without a living watchdog.

## Reports

| Artifact | Owner |
|----------|--------|
| Scenario `session.json` | `playtestrun` |
| `*-environment-failed.md` | `playtestenv` |
| Combined gameplay `*.md` | Driver |

Combined report **required header:** checkout, commit, dirty, run_id, scenario
path, roster binding summary, wall-clock elapsed vs budget, sidecar status,
per-actor outcomes. Never passwords. Optional per-actor sections. Drop ptorch /
`bb dump` requirements from the format doc.

## Failure modes

| Failure | Behavior |
|---------|----------|
| Parse / bind errors | Exit before Docker |
| Env / materialize fail | `environment_failed`; no mudagents |
| Wall-clock exceeded | `incomplete_wallclock`; Stop; incomplete report |
| Actor soft stop + `continue` | That mudagent stops; others continue |
| Actor stop + `abort` | `incomplete_abort`; full Stop |
| SIGINT / cancel | `interrupted` + non-zero |
| Driver crash | Lease/reap recovery |

## Migration (in scope)

- Migrate `tools/playtest/scenarios/*.yaml` that can run without PvP config
  hacks (`party-formation`, `parallel-coverage`, `feel-pothole-newbie-veteran`
  as shared-env concurrency/UX). Split per-actor goals under
  `tools/playtest/goals/scenarios/<name>/`.
- **Defer** `adversarial-contest` (and any `requires.pvp`) until allow-listed
  ephemeral config overrides exist; driver/Go refuse with a clear error if
  invoked.
- Rewrite `.claude/commands/playtest-scenario.md` and
  `tools/playtest/multi-agent-report-format.md`.
- Expand checked-in prod-identity denylist + shared helper used by 0.3c and
  0.3d.

## Testing

- **Unit:** scenario parse matrix; duplicate roster ids; admin reject;
  MaxAIConnections reject; prod-identity / close-variant helper; `actor_id`
  creds mapping; continue vs abort; ready JSON actor list; bridge/blackboard
  paths.
- **Opt-in Docker:** ≥2 non-admin mixed profile actors → ready → stop; optional
  creation-flow + profile mix.
- **Driver-contract smoke:** ready JSON + run-scoped actor bridges + blackboard
  dir (no full LLM session required).
- **Pre-merge live smoke (merge gate):** ~**10 minutes** of party play with a
  **mixed** party (different loadouts + personalities). Must exercise the real
  path: `playtestrun scenario` → parse ready JSON → N mudagent bridges →
  invite/accept (or equivalent) → `playtestrun stop` → combined report with
  **checklisted** header fields (checkout, commit, dirty, run_id, wall-clock,
  sidecar status, per-actor outcomes, no passwords). The implementing agent may
  drive the mudagents (need not be a separate Claude session) but must not skip
  the ready-gate. Wall-clock for this smoke may be capped ~10–15m.
- No Done without Docker evidence + this live smoke + adversarial
  **implementation** review after plan approval.

## Process gates

1. Adversarial **spec** review → revise → user approves spec  
2. Implementation **plan** → adversarial plan review → user approves plan  
3. Only then TDD implementation  

## Adversarial spec review (2026-08-08)

**Verdict:** Request changes → amendments applied in-file.

| Severity | Finding | Resolution |
|----------|---------|------------|
| Blocking | Creds/actor mapping | Mandatory `actor_id` on PlayerCreds; select by actor_id |
| Blocking | Blackboard unspecified | Blackboard contract section |
| Blocking | ptorch driver conflict | Mandatory `/playtest-scenario` rewrite |
| Blocking | Scenario allow-list vs legacy | Full schema; `requires` driver-only; reject legacy fields |
| Important | Close-variant algorithm | v1 algorithm + CI-time denylist |
| Important | Admin creation-flow hole | RoleUser-only; no admin sequences |
| Important | Per-actor vs scenario wall-clock | Scenario/CLI sole hard cut; actor wall_clock ignored by Go |
| Important | Sidecar schema | ScenarioSidecar fields documented |
| Important | `on_actor_stop` semantics | continue vs abort table |
| Important | MaxAIConnections | Fail-closed + optional `--force` |
| Important | PvP requires | Defer adversarial-contest; refuse until overrides |
| Important | Pre-merge smoke | Ready-gate path + checklist; agent may drive mudagents |
| Important | Report format stale | Rewrite deliverable |
| Minor | Parallel vs 0.3c | Clarified shared-server concurrency is 0.3d |
| Minor | Shared denylist | Mandatory for run + scenario |

## Brainstorm record (2026-08-08)

0.3d; soft per-actor budgets; shared env; Hybrid; concurrent mudagents; per-actor
goals/loadout; file blackboard + in-game channels; `on_actor_stop` default
continue; `playtestrun scenario`; Approach 1; diversity guidance; prod-identity
ban; admin multi ban; ~10m mixed-party pre-merge smoke.
