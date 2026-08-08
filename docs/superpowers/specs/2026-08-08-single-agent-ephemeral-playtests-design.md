# Single-Agent Ephemeral Playtests — Design

**Chunk:** 0.3c (Adversarial Review Remediation Roadmap)  
**Status:** Revised after adversarial spec review 2026-08-08 (awaiting plan;
**not** an implementation green light)  
**Depends on:** 0.3a (playtestenv), 0.3b (playtestprofiles / creds)  
**Feeds:** 0.3d (multi-agent ephemeral scenarios)

## Goal

Wire local single-agent playtests onto the ephemeral supervisor: a goals
file binds a synthetic profile (or an explicit creation-flow), Go starts
the disposable server and enforces a wall-clock budget plus cleanup,
Claude drives mudagent and writes the gameplay report, and incomplete
stops are never misreported as gameplay success.

This chunk ends when `/playtest local --checkout <path> <personality>
<goals-file>` always uses an ephemeral environment for that named checkout,
with a session sidecar, a living wall-clock watchdog, guaranteed stop/reap,
and a gameplay report that states which tree was tested. It does **not**
implement multi-agent scenarios or a Go-hosted LLM brain.

## Non-goals

Owned by **0.3d** or later — not this design:

- Multi-agent / `ptorch` scenario rosters and combined reports
- Hard token-budget kill switch in Go (tokens remain soft driver guidance;
  Claude may still mark the gameplay report incomplete on API/token stop)
- A separate max-turns kill switch in `playtestrun` (session backstop =
  wall-clock; per-round spam = in-engine `AICommandsPerRound`)
- Pinning `--ref` / requiring clean tree equal to a specific SHA
- Bulk migration of every file under `tools/playtest/goals/`
- Retiring or ephemeral-izing `prod` playtests
- Replacing mudagent or moving command judgment into Go
- Production or remote targeting of any kind
- Reading `_archive/prod-users` (or any archive) at runtime
- Dead-code cleanup / deletion of the pre-0.3c local playtest path
  (old `targets.yaml`-based local wiring, unused bridge helpers, etc.).
  0.3c **rewires** local onto `playtestrun` and stops using that path; it
  deliberately leaves the old code in place until the new path is
  stress-tested. Removal is a later slice.

## Decisions (locked in brainstorm + review)

| Topic | Choice |
|-------|--------|
| Orchestration | **Hybrid:** Go owns env, wall-clock, cleanup, sidecar; Claude + mudagent play and write the gameplay report |
| Go surface | Thin CLI/package **`playtestrun`** composing `playtestenv` |
| Goal→profile | Explicit in goals YAML (`ephemeral.profile` + `start_room` [+ overlays]) |
| Creation-flow | `creation_flow: true` + non-empty `creation_rationale` only |
| `local` meaning | Always ephemeral; **does not** use `targets.yaml` for endpoint/creds |
| Goals required | Ephemeral `local` requires a goals file with `ephemeral:` |
| Goals migration | Schema + few exemplars only |
| Budgets | Go hard-enforces **wall-clock only** (default 30m) |
| Spam pacing | Existing `Network.AICommandsPerRound` (not a session turn budget) |
| Reports | Claude gameplay markdown; Go sidecar; env failures → `environment-failed` |
| Report templates | Mined templates; ship core set in 0.3c |
| Checkout | Explicit `--checkout` on CLI and `/playtest local` (no silent cwd) |
| IDs | Use playtestenv **`run_id`** everywhere (not a parallel “session id”) |
| Human docs | Verbose `context.md` invocation guide required |

## Architecture

```text
/playtest local --checkout PATH <personality> <goals>
        │
        ▼
  playtestrun run   (blocking supervisor for the wall-clock window)
        │  parse goals ephemeral: (fail closed, KnownFields)
        │  playtestenv.Start(+Profiles|empty) with Lease ≈ wall_clock+buffer
        │  write sidecar under tools/playtest/.run/<run_id>/
        │  print ready JSON (endpoint, creds path|null, run_id, git, deadline)
        │
        ├──────────────► Claude drives mudagent
        │                 bridge: .run/<run_id>/bridge/{commands.txt,events.jsonl}
        │                      │
        │                      ▼
        │               gameplay report.md
        │
        ▼
  at deadline or driver stop signal → playtestenv.Stop / reap
  sidecar: ready | incomplete_wallclock | stopped | environment_failed
```

| Piece | Responsibility |
|-------|----------------|
| `playtestrun` | Binding parse, start/stop env, wall-clock watchdog, sidecar, cleanup |
| `playtestenv` | Unchanged 0.3a/0.3b env + profile materialization |
| `/playtest` | Personality play, mudagent I/O, gameplay report |
| Goals YAML | `ephemeral:` profile path **or** creation-flow rationale |

### Wall-clock enforcement (required contract)

`playtestrun run` is the primary entrypoint: it **blocks** until (a) the
driver signals stop via a control file / `playtestrun stop`, (b) wall-clock
`deadline_at` is reached, or (c) the container/env fails.

At deadline: set sidecar `status=incomplete_wallclock`, call
`playtestenv.Stop`, exit non-zero with a machine-readable result. Claude
polls sidecar `status` (or watches process exit) and must not claim gameplay
success after `incomplete_wallclock`.

Optional thin wrappers `start` / `status` / `stop` may exist for debugging,
but the driver contract for `/playtest local` is **`run`** (or
`start --watch` equivalent that holds the watchdog). Silent “start and
exit leaving only the 2h lease” is **not** sufficient enforcement.

**Lease coupling:** `playtestenv.Start` lease MUST be set to at least
`wall_clock + cleanup_buffer` (buffer ≥ `CleanupTimeout`, suggest 5m) so the
lease does not outlive the intended session by hours without a Stop, and
does not expire before the wall-clock watchdog finishes cleanup.

## Goals binding schema

Top-level `ephemeral:` block. Unknown keys under `ephemeral:` fail closed
(KnownFields), matching 0.3b manifest strictness.

### Profile path

```yaml
ephemeral:
  profile: veteran
  start_room: 5455
  overlays:                 # optional; 0.3b ProfileOverlays shape
    grant_spells: { heal: 1 }
  budgets:
    wall_clock: 30m
```

### Creation-flow path

```yaml
ephemeral:
  creation_flow: true
  creation_rationale: >
    Brand-new character required: this run grades whether the game itself
    teaches a first-time player without a pre-seeded kit.
  budgets:
    wall_clock: 30m
```

When `creation_flow: true`, `profile` / `start_room` / `overlays` must be
absent. Empty `Profiles` list → no `creds.json`; Claude drives `new`.

### Fail-closed rules (before Docker start)

| Case | Behavior |
|------|----------|
| Missing `--checkout` | Error |
| Missing goals file / unreadable | Error |
| No `ephemeral:` block | Error |
| Unknown key under `ephemeral:` | Error |
| `profile` set without positive `start_room` | Error |
| Unknown `profile` id | Error |
| `creation_flow: true` without non-empty `creation_rationale` | Error |
| Both `profile` and `creation_flow: true` | Error |
| Neither `profile` nor `creation_flow` | Error |

### Legacy goals fields

`session.max_rounds` / similar soft hints in older goals files are
**ignored by Go**. Claude may treat them as soft pacing notes. They do not
create a second kill switch.

### Exemplars in 0.3c

- `newbie-naive.yaml` → `creation_flow: true` + rationale (matches content)  
- `corpse-looting.yaml` (or similar mid-game file whose objectives assume an
  existing character) → `profile` + `start_room` — **do not** pick
  `shop-economy.yaml` without rewriting its create-character objectives  

### Creds login selection

When `creds.json` is present: select the player whose `profile` field
matches `ephemeral.profile`. If no match, or if multiple matches exist,
fail closed before mudagent login. Creation-flow: `creds` path is null.

## `playtestrun` CLI

```text
playtestrun run    --checkout PATH --goals PATH --personality NAME
                   [--wall-clock DURATION]
playtestrun status --checkout PATH --run ID
playtestrun stop   --checkout PATH --run ID
```

- Flags use **`--run`** (playtestenv `run_id`), not `--session`.  
- `--personality` is required when invoked from `/playtest` (driver always
  has one); CLI may still accept it for sidecar/report echo.  
- stdout on ready: one JSON object for Claude (endpoint, creds path|null,
  run_id, checkout, commit, dirty, deadline_at, sidecar path).

### Sidecar

Path: `tools/playtest/.run/<run_id>/session.json`

Minimum fields:

- `run_id`, `checkout`, `commit`, `dirty`, `goals_path`, `personality`  
- `endpoint` `{host,port}`, `creds` path or null  
- `profile` or `creation_flow` + rationale excerpt  
- `budgets.wall_clock`, `started_at`, `deadline_at`  
- `status`: `starting` | `ready` | `incomplete_wallclock` | `stopped` |
  `environment_failed`  
- `environment_report` path when status is `environment_failed`

**On env/materialize failure:** write/update sidecar to
`environment_failed` with the playtestenv report path (if any), then exit.
`/playtest` must not drive mudagent unless status is `ready`.

### Mudagent bridge isolation

Bridge files move under the run directory:

`tools/playtest/.run/<run_id>/bridge/commands.txt`  
`tools/playtest/.run/<run_id>/bridge/events.jsonl`

No flat shared `tools/playtest/.run/commands.txt` for ephemeral local
(avoids collisions with concurrent runs / playtestenv artifacts).

### Budgets

| Knob | Default | Role |
|------|---------|------|
| Wall-clock | 30m | Go session backstop → Stop + `incomplete_wallclock` |
| Tokens | soft | Claude may stop and mark gameplay report incomplete |
| `AICommandsPerRound` | 2 (shipped) | Per-round spam drop inside the MUD — **not** a session command budget |

## `/playtest` driver changes

For **`local`**:

1. Usage: `/playtest local --checkout <path> <personality> <goals-file>`  
   (exact flag parsing in the command file; checkout required, no cwd).  
2. Goals file required; must contain valid `ephemeral:`.  
3. Call `playtestrun run …` (watchdog held for the session).  
4. Do **not** read `targets.yaml` for host/port/user/password. Endpoint +
   creds come only from playtestrun ready JSON / sidecar.  
5. Mudagent bridge paths under `.run/<run_id>/bridge/`.  
6. Login: match creds player to `ephemeral.profile`, or creation-flow.  
7. On finish / incomplete / abort → ensure stop; write gameplay report with
   required header (checkout, commit, dirty, run_id, binding, wall-clock).  

For **`prod`**: unchanged (`targets.yaml`, no playtestrun).

### SOP / docs deliverables (in scope)

- Update `.claude/commands/playtest.md` for the new local contract.  
- Update CLAUDE.md / TESTING_GUIDE AI-testing sections: ephemeral local
  requires checkout + goals with `ephemeral:`; provide a default adversarial
  exemplar goals file for SOP “bug-finder” style runs (or document the
  required goals path).  
- Verbose `internal/playtestrun/context.md` human invocation section.

## Reports and templates

| Artifact | Owner |
|----------|--------|
| `session.json` | `playtestrun` |
| `*-environment-failed.md` | `playtestenv` |
| Gameplay `*.md` | Claude `/playtest` |

Gameplay report must include checkout/commit/dirty, run_id, binding,
wall-clock elapsed vs budget, sidecar status. Never passwords (paths only).
`incomplete_wallclock` or driver-declared token/API stop ⇒ outcome
**incomplete**, not success.

Templates under `tools/playtest/report-templates/`, mined from current
goals, `tools/playtest/reports/` harness-era gameplay reports, and
`tools/_archive/testing-pre-harness/testing/reports/`. Ship at least
newbie/creation-flow and bug-finder/sweep templates in 0.3c; remaining
classes are required if mining is cheap, else plan follow-up.

## Failure modes

| Failure | Behavior |
|---------|----------|
| Missing checkout / goals / ephemeral binding | Exit before Docker |
| Env / materialize fail | Sidecar `environment_failed` + environment-failed.md |
| Wall-clock exceeded | Sidecar `incomplete_wallclock`; Stop/reap; incomplete gameplay report |
| Soft token/API stop (Claude) | Gameplay report incomplete; still `playtestrun stop` |
| Driver crash | Lease/reap + documented recovery; may leave non-terminal sidecar briefly |
| Secret leakage | Forbidden in markdown reports |

## Testing

- **Unit:** ephemeral parse matrix (incl. unknown keys); checkout required;
  wall-clock deadline → incomplete status; creation vs profile exclusion;
  creds player match helper.  
- **Opt-in Docker:** profile exemplar + creation-flow exemplar through
  `playtestrun run` (or start+watch) → sidecar ready → stop/cleanup.  
- **Driver contract smoke:** scripted check that ready JSON exposes
  endpoint/creds/run_id and that bridge paths are run-scoped (full Claude
  session optional).  
- No Done without Docker evidence + adversarial **implementation** review
  after plan approval.

## Process gates

1. Adversarial **spec** review → revise → user approves spec  
2. Implementation **plan** → adversarial plan review → user approves plan  
3. Only then TDD implementation  

## Handoff to 0.3d

Reuse `playtestrun` shell, sidecar, run-scoped bridge, wall-clock watchdog;
extend binding to a roster of profiles.

## Adversarial spec review (2026-08-08)

**Verdict:** Request changes → amendments applied in-file + roadmap/0.3b
handoff reconciled.

| Severity | Finding | Resolution |
|----------|---------|------------|
| Blocking | Roadmap token/turn vs wall-clock-only | Roadmap 0.3c rewritten to wall-clock + AI rate limit + soft tokens |
| Blocking | 0.3b handoff “token exhaustion” ownership | 0.3b handoff amended; soft incomplete via Claude allowed |
| Blocking | Watchdog unspecified after start-exit | Primary `playtestrun run` blocking supervisor + lease coupling |
| Blocking | `/playtest` missing checkout | `--checkout` in driver usage + deliverables |
| Important | Flat mudagent bridge collision | Bridge under `.run/<run_id>/bridge/` |
| Important | `--session` vs `--run` | Standardized on `run_id` / `--run` |
| Important | Lease 2h vs wall-clock 30m | Lease ≥ wall_clock + cleanup buffer |
| Important | SOP free-form local break | Docs + adversarial exemplar goals deliverable |
| Important | `local` still using targets.yaml | Explicit: local skips targets.yaml |
| Important | shop-economy as profile exemplar | Prefer corpse-looting-style match |
| Important | Sidecar on env failure | `environment_failed` + report path |
| Important | Legacy max_rounds | Ignored by Go |
| Important | Creds multi-player select | Match `ephemeral.profile` |
| Important | AICommandsPerRound ≠ turn budget | Reworded |

## Brainstorm record (2026-08-08)

Hybrid C; explicit goals binding; wall-clock only; creation-flow with
rationale; `local` always ephemeral; exemplar-only goals migration; no
free-form local; thin `playtestrun`; explicit checkout + loud git baseline;
mined report templates; verbose human `context.md`.
