# Single-Agent Ephemeral Playtests — Design

**Chunk:** 0.3c (Adversarial Review Remediation Roadmap)  
**Status:** Draft after brainstorm 2026-08-08 (awaiting adversarial spec
review + user approval; **not** an implementation green light)  
**Depends on:** 0.3a (playtestenv), 0.3b (playtestprofiles / creds)  
**Feeds:** 0.3d (multi-agent ephemeral scenarios)

## Goal

Wire local single-agent playtests onto the ephemeral supervisor: a goals
file binds a synthetic profile (or an explicit creation-flow), Go starts
the disposable server and enforces wall-clock / turn budgets plus cleanup,
Claude drives mudagent and writes the gameplay report, and incomplete
budget stops are never misreported as gameplay success.

This chunk ends when `/playtest local <personality> <goals-file>` always
uses an ephemeral environment for a named checkout, with a session
sidecar, guaranteed stop/reap, and a gameplay report that states which
tree was tested. It does **not** implement multi-agent scenarios or a
Go-hosted LLM brain.

## Non-goals

Owned by **0.3d** or later — not this design:

- Multi-agent / `ptorch` scenario rosters and combined reports
- Hard token-budget kill switch (tokens remain soft guidance for Claude)
- Pinning `--ref` / requiring clean tree equal to a specific SHA
- Bulk migration of every file under `tools/playtest/goals/`
- Retiring or ephemeral-izing `prod` playtests
- Replacing mudagent or moving command judgment into Go
- Production or remote targeting of any kind
- Reading `_archive/prod-users` (or any archive) at runtime

## Decisions (locked in brainstorm)

| Topic | Choice |
|-------|--------|
| Orchestration | **Hybrid:** Go owns env, budgets, cleanup, session sidecar; Claude + mudagent play and write the gameplay report |
| Go surface | New thin CLI/package **`playtestrun`** that **composes** `playtestenv` (does not fold mudagent into 0.3a) |
| Goal→profile | **Explicit** in goals YAML (`ephemeral.profile` + `start_room` [+ overlays]) |
| Creation-flow | Allowed only with `creation_flow: true` **and** non-empty `creation_rationale` explaining why a brand-new character is required |
| `local` meaning | **Always ephemeral** (playtestenv every time) |
| Goals required | Ephemeral `local` **requires** a goals file (no free-form local) |
| Goals migration | Schema + **few exemplars** only in 0.3c |
| Budgets | Go hard-enforces **wall-clock** + **max turns**; token budget soft only |
| Reports | Claude owns gameplay markdown; Go owns session sidecar; env failures stay on existing `environment-failed` path |
| Report templates | Suggested templates for broad scenarios, **mined** from past goals + old harness / pre-harness reports |
| Checkout | **`--checkout` required** (no silent cwd); sidecar + report header echo commit + dirty baseline |
| Human docs | Plan must include a **verbose** `context.md` invocation guide (options, examples) |

## Architecture

```text
/playtest local <personality> <goals>  +  explicit checkout
        │
        ▼
  playtestrun start
        │  parse goals ephemeral: binding (fail closed)
        │  playtestenv.Start(+Profiles) or empty Profiles (creation-flow)
        │  write session sidecar (endpoint, creds path|null, git, budgets)
        │
        ├──────────────► Claude drives mudagent (existing file bridge)
        │                      │
        │                      ▼
        │               gameplay report.md (Claude; scenario template)
        │
        ▼
  budget watchdog (wall-clock + turns) → playtestenv.Stop / reap
  sidecar status: ready | incomplete_wallclock | incomplete_turns | stopped
```

| Piece | Responsibility |
|-------|----------------|
| `playtestrun` | Goals binding parse, start/stop env, budgets, sidecar, cleanup guarantee |
| `playtestenv` | Unchanged 0.3a/0.3b env + profile materialization contract |
| `/playtest` (Claude) | Personality play, mudagent I/O, gameplay report body |
| Goals YAML | `ephemeral:` profile path **or** creation-flow rationale |

## Goals binding schema

Top-level `ephemeral:` block (existing `goals` / `name` / `summary` /
`objectives` shapes remain for the driver).

### Profile path

```yaml
ephemeral:
  profile: veteran          # one of: fresh, early, mid, veteran,
                            # specialist-caster, admin
  start_room: 5455          # required when profile is set
  overlays:                 # optional; 0.3b ProfileOverlays shape
    grant_spells: { heal: 1 }
  budgets:                  # optional overrides
    wall_clock: 30m
    max_turns: 200
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
    max_turns: 200
```

When `creation_flow: true`, `profile` / `start_room` / `overlays` **must be
absent**. `playtestrun` starts playtestenv with an empty `Profiles` list
(no `creds.json`); Claude drives the normal `new` character flow.

### Fail-closed rules (before Docker start)

| Case | Behavior |
|------|----------|
| Missing `--checkout` | Error |
| Missing goals file / unreadable | Error |
| No `ephemeral:` block | Error |
| `profile` set without positive `start_room` | Error |
| Unknown `profile` id | Error |
| `creation_flow: true` without non-empty `creation_rationale` | Error |
| Both `profile` and `creation_flow: true` | Error |
| Neither `profile` nor `creation_flow` | Error |

### Exemplars in 0.3c

Update a small set only, for example:

- `newbie-naive.yaml` (or equivalent) → creation-flow + rationale  
- One mid-game / feature goals file → `profile` + `start_room`  

All other goals remain until touched; they cannot run under ephemeral
`local` until migrated.

## `playtestrun` CLI

Proposed commands (exact flag names may refine in the plan):

```text
playtestrun start  --checkout PATH --goals PATH [--personality NAME]
                   [--wall-clock DURATION] [--max-turns N]
playtestrun status --checkout PATH --session ID
playtestrun stop   --checkout PATH --session ID
```

### `start`

1. Require `--checkout` (never default to cwd).  
2. Validate checkout via existing playtestenv rules.  
3. Parse + validate goals `ephemeral:` block.  
4. Call `playtestenv.Start` with `Profiles` populated from the goals file,
   or empty for creation-flow.  
5. Record Git baseline (commit + dirty entries) from playtestenv / git.  
6. Write **session sidecar** and print one JSON object on stdout for Claude.

### Session sidecar

Path (proposed): `tools/playtest/.run/<session-or-run-id>/session.json`

Minimum fields:

- `session_id` / `run_id`, `checkout`, `commit`, `dirty` (bool or entry
  summary), `goals_path`, `personality` (if provided)  
- `endpoint` `{host,port}`, `creds` path or null  
- `profile` or `creation_flow` + rationale excerpt  
- `budgets` `{wall_clock, max_turns}`, `turns_used`, `started_at`  
- `status`: `ready` | `incomplete_wallclock` | `incomplete_turns` |
  `stopped` | `environment_failed`

### Budgets

| Knob | Default | Enforcement |
|------|---------|-------------|
| Wall-clock | 30m | Go stops mudagent session ownership + `playtestenv.Stop` |
| Max turns | 200 | Go when turn counter reaches N |
| Tokens | n/a in Go | Soft note in `/playtest` / personality only |

**Turn counting:** Claude (or a minimal helper) updates a turn counter in
the sidecar / adjacent file after each mudagent command. Missing updates
do not count as success; wall-clock still fires. Go must not invent
turn=0 “complete” when the driver dies early.

### Cleanup

- Normal exit, budget incomplete, and explicit stop all call
  `playtestenv.Stop` (and rely on leases/reap if the driver vanishes).  
- Incomplete budget ⇒ sidecar `incomplete_*`, **not** gameplay success.  
- Env/materialize failure ⇒ existing `environment-failed.md`; no fake
  gameplay pass.

## `/playtest` driver changes

For **`local`**:

1. Require `<goals-file>` and an **explicit checkout** (no cwd guess).  
2. Invoke `playtestrun start` first; do not drive mudagent without sidecar
   `status=ready`.  
3. Header-echo in the eventual gameplay report: checkout, commit, dirty,
   session/run id, profile or creation rationale, budgets.  
4. Login via `creds.json` when present; creation-flow when `creds` is null.  
5. On finish / incomplete / abort → `playtestrun stop`, then write the
   gameplay report.  

For **`prod`**: unchanged long-lived target path (no `playtestrun`, no
ephemeral). Still not an ephemeral/prod hybrid.

## Reports and templates

### Ownership

| Artifact | Owner |
|----------|--------|
| `session.json` | `playtestrun` |
| `*-environment-failed.md` | `playtestenv` (unchanged) |
| Gameplay `*.md` report | Claude `/playtest` |

### Gameplay report requirements

- Required header block: checkout path, commit, dirty summary, session/run
  id, profile **or** creation-flow rationale, personality, goals file,
  wall-clock/turns used vs budget, final sidecar status.  
- Never embed passwords; creds **path** only.  
- If sidecar status is `incomplete_*`, report outcome must be
  **incomplete** (not success), with partial findings.

### Scenario report templates

Add `tools/playtest/report-templates/` with suggested markdown skeletons
for broad classes, **distilled from** (not invented without reading):

- Current `tools/playtest/goals/*.yaml` shapes and intent  
- Archived pre-harness reports under
  `tools/_archive/testing-pre-harness/testing/reports/`  
- Any retained harness-era reports under `tools/playtest/reports/`  

Initial template classes (adjust names in plan if mining suggests better
buckets):

1. Newbie / creation-flow  
2. Shop / economy  
3. Combat / corpse / parser stress  
4. Quest / dialogue / NPC  
5. Feel / exploratory town flavor  
6. Bug-finder / sweep  

`/playtest` picks a template by goals metadata or personality+goals
heuristics documented in the command; authors may override.

## Human documentation (plan requirement)

The implementation plan **must** require a verbose section in
`internal/playtestrun/context.md` (and a short pointer from
`docs/guides/TESTING_GUIDE.md`) covering:

- How a human invokes `playtestrun` and `/playtest local`  
- Required flags (`--checkout`, `--goals`) and optional budgets  
- Profile vs creation-flow goals examples  
- How to read sidecar + gameplay report + environment-failed  
- Worked examples (creation-flow newbie; mid-game profile run)  
- What fails loudly (missing checkout, missing ephemeral block, etc.)

This is not optional polish; it is part of the 0.3c deliverable.

## Failure modes

| Failure | Behavior |
|---------|----------|
| Missing checkout / goals / ephemeral binding | Exit before Docker; loud error |
| Env / materialize fail | `environment-failed.md`; no gameplay success |
| Wall-clock / max-turns | Sidecar `incomplete_*`; Stop/reap; incomplete gameplay report |
| Driver crash mid-run | Lease expiry + stop/reap; may leave sidecar non-terminal until reaped — document operational recovery |
| Secret leakage | Forbidden in all markdown reports (paths only) |

## Testing

- **Unit:** goals `ephemeral:` parse matrix; checkout required; budget
  status transitions; creation-flow vs profile mutual exclusion.  
- **Opt-in Docker:** one profile exemplar and one creation-flow exemplar
  through `playtestrun start` → sidecar ready → stop/cleanup.  
- **Driver smoke:** `/playtest local` against an exemplar goals file on an
  explicit checkout (manual or scripted agent session).  
- Do not claim Done without Docker evidence + adversarial **implementation**
  review after plan approval and TDD implementation.

## Process gates (explicit)

This design is **not** permission to implement.

Required sequence after this file exists:

1. Adversarial **spec** review → revise if needed → user approves spec  
2. Implementation **plan** (writing-plans) → adversarial plan review →
   user approves plan  
3. Only then: TDD implementation on a feature branch  

Jumping from approved brainstorm / draft spec straight to code is a
process failure (same class as the 0.3b premature-impl incident).

## Handoff to 0.3d

0.3d reuses `playtestrun`’s session shell and sidecar, extending binding
to a **roster** of profiles / start rooms and multi-agent choreography.
Single-agent budgets and report templates remain the baseline.

## Brainstorm record (2026-08-08)

- Orchestration C (hybrid); goals binding A (explicit); budgets A
  (wall-clock + turns); reports A + mined templates; `local` = always
  ephemeral; creation-flow with explicit rationale; exemplar-only goals
  migration; no free-form local; approach = thin `playtestrun`; explicit
  `--checkout` + loud commit/dirty in artifacts.
