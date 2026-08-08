# Single-Agent Ephemeral Playtests Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.
>
> **Do not start this plan until the user explicitly approves it** (after
> adversarial plan review). Spec approval alone is not enough.

**Goal:** Local `/playtest` runs always use ephemeral playtestenv with
goals-bound profiles (or explicit creation-flow), a blocking wall-clock
watchdog, run-scoped mudagent bridge, and incomplete-aware gameplay reports.

**Architecture:** New `internal/playtestrun` + `cmd/playtestrun` compose
`playtestenv`. Claude `/playtest` drives mudagent and writes gameplay
reports. Go owns binding parse, wall-clock, sidecar, cleanup.

**Tech Stack:** Go 1.25, existing playtestenv/playtestprofiles, yaml.v3,
testify, Docker opt-in integration.

**Approved design:**
`docs/superpowers/specs/2026-08-08-single-agent-ephemeral-playtests-design.md`
(revised after adversarial spec review 2026-08-08).

**Plan review:** Amended 2026-08-08 after adversarial plan review
(Request changes → address Blocking/Important gaps below).

---

## Execution constraints

- Branch: create `feature/stage-0.3c-single-agent-ephemeral-playtests` from
  current `master` (or from merged 0.3b if already on master). Do **not**
  continue feature work on the 0.3b v2 branch unless 0.3b is already merged
  and this plan is applied as a follow-on commit series there by explicit
  user request.
- If uncommitted room / adversarial / invalidated-0.1 files are present in
  the workspace, do not stage them (`484.yaml`, pothole rooms,
  `ADVERSARIAL_CODE_REVIEW_2026-08-07.md`, invalidated web-terminal docs,
  etc.).
- Stage exact owned paths only; no `git add .`.
- Do not implement multi-agent, hard token kill switch, or max-turns.
- Do not claim Done without Docker evidence + adversarial implementation
  review.

## File map

### New

- `internal/playtestrun/` — binding parse, sidecar, run supervisor, creds
  match helper, `context.md` (verbose human guide), `*_test.go`
- `cmd/playtestrun/main.go`, `cmd/playtestrun/main_test.go`
- `tools/playtest/report-templates/` — at least `newbie-creation.md`,
  `bug-finder-sweep.md` (mine before inventing)
- Exemplar goals updates (binding blocks only where content matches)

### Modify

- `.claude/commands/playtest.md` — local ephemeral contract
- `docs/guides/TESTING_GUIDE.md` — pointer + ephemeral local usage
- `CLAUDE.md` AI Testing section — checkout + goals required for local;
  cite bug-finder exemplar path by name
- `docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md` — 0.3c status
  when done
- Goals exemplars:
  - `tools/playtest/goals/newbie-naive.yaml`
  - `tools/playtest/goals/corpse-looting.yaml`
  - `tools/playtest/goals/2026-08-03-prepush-sweep.yaml` (adversarial /
    bug-finder SOP exemplar)

### Do not modify for scope

- `internal/playtestenv` env policy (compose only) except calling its
  public `Start`/`Stop`/`Status` APIs
- Production `compose.yml`
- Dead-code cleanup of the pre-0.3c local playtest path (leave unused
  helpers until a later slice after stress-testing; see spec Non-goals)

## Core contracts

```go
// internal/playtestrun/binding.go
type EphemeralBinding struct {
    Profile            string        // empty if creation flow
    StartRoom          int
    Overlays           playtestenv.ProfileOverlays
    CreationFlow       bool
    CreationRationale  string
    WallClock          time.Duration // default 30m
}

func ParseGoalsEphemeral(goalsPath string) (EphemeralBinding, error)
func SelectCredsPlayer(credsPath, profileID string) (username, password string, err error)
```

```go
// Session sidecar JSON — nest budgets to match the approved spec.
type SessionBudgets struct {
    WallClock string `json:"wall_clock"` // e.g. "30m"
}

type SessionSidecar struct {
    RunID              string         `json:"run_id"`
    Checkout           string         `json:"checkout"`
    Commit             string         `json:"commit"`
    Dirty              bool           `json:"dirty"`
    GoalsPath          string         `json:"goals_path"`
    Personality        string         `json:"personality,omitempty"`
    Endpoint           *playtestenv.Endpoint `json:"endpoint,omitempty"`
    Creds              string         `json:"creds,omitempty"`
    Profile            string         `json:"profile,omitempty"`
    CreationFlow       bool           `json:"creation_flow,omitempty"`
    CreationRationale  string         `json:"creation_rationale,omitempty"`
    Budgets            SessionBudgets `json:"budgets"`
    StartedAt          time.Time      `json:"started_at"`
    DeadlineAt         time.Time      `json:"deadline_at"`
    Status             string         `json:"status"` // starting|ready|incomplete_wallclock|stopped|environment_failed
    EnvironmentReport  string         `json:"environment_report,omitempty"`
    BridgeDir          string         `json:"bridge_dir"`
}
```

Ready JSON (one line on stdout when status becomes `ready`) MUST include at
least: `endpoint`, `creds` (path or null), `run_id`, `checkout`, `commit`,
`dirty`, `deadline_at`, sidecar path, and `bridge_dir` under
`.run/<run_id>/bridge/`.

CLI:

```text
playtestrun run --checkout PATH --goals PATH --personality NAME [--wall-clock 30m]
playtestrun status --checkout PATH --run ID
playtestrun stop --checkout PATH --run ID
```

---

### Task 0: Branch hygiene

- [ ] From clean integration base (`master` with 0.3b merged, or user-named
      base), create `feature/stage-0.3c-single-agent-ephemeral-playtests`.
- [ ] If uncommitted room/adversarial/invalidated-doc files are present,
      confirm they will not be staged.
- [ ] Commit nothing yet (or empty docs-only cherry of approved spec/plan if
      they live only on another branch — prefer checkout of the two approved
      doc paths onto the new branch).

---

### Task 1: Goals `ephemeral:` parse (TDD)

**Files:** `internal/playtestrun/binding.go`, `binding_test.go`

- [ ] Write failing tests for:
      - profile+room happy path
      - optional `overlays:` round-trip into `ProfileOverlays`
      - creation_flow+rationale happy
      - missing ephemeral
      - unknown key under ephemeral (KnownFields)
      - both profile and creation_flow
      - neither profile nor creation_flow
      - creation without rationale / empty rationale
      - creation_flow with forbidden `profile`, `start_room`, and/or
        `overlays` (must be absent per spec) → reject
      - `profile` without `start_room`; missing `start_room`; `start_room: 0`;
        negative `start_room`
      - unknown profile id
      - wall_clock parse; default 30m when budgets omitted
      - missing goals path / unreadable file → error
      - legacy `session.max_rounds` present alongside valid `ephemeral:` →
        parse succeeds (legacy soft hint ignored by Go)
- [ ] Implement `ParseGoalsEphemeral` with `yaml.v3` KnownFields on the
      `ephemeral` object.
- [ ] Reject unknown profile IDs using `playtestprofiles.IsKnownTemplateID`
      (or a duplicated allow-list of the six IDs if import cost is too high —
      prefer the real helper).
- [ ] `go test ./internal/playtestrun -count=1`
- [ ] Commit: `feat(playtest): parse ephemeral goals binding for playtestrun`

---

### Task 2: Sidecar + creds player match (TDD)

**Files:** `internal/playtestrun/sidecar.go`, `creds.go`, tests

- [ ] Failing tests: write/read sidecar (including nested `budgets.wall_clock`);
      status transitions; SelectCredsPlayer matches profile; errors on
      missing/ambiguous players; never logs password.
- [ ] Implement atomic sidecar write under
      `tools/playtest/.run/<run_id>/session.json`.
- [ ] Implement `SelectCredsPlayer`.
- [ ] Commit: `feat(playtest): session sidecar and creds player selection`

---

### Task 3: `playtestrun` supervisor + CLI (TDD with fakes)

**Files:**
- Create/Modify: `internal/playtestrun/run.go`, `*_test.go`
- Create: `cmd/playtestrun/main.go`, `cmd/playtestrun/main_test.go`

- [ ] Define a narrow `envSupervisor` interface matching
      `playtestenv.Supervisor` Start/Stop/Status used by run.
- [ ] Failing tests with fake supervisor / fake clock:
      - missing `--checkout` → usage error before Start
      - missing `--personality` → usage error before Start
      - binding error → no Start
      - Start failure → sidecar `environment_failed`, non-zero exit
      - ready path writes sidecar `ready`, bridge dir created,
        lease = wall_clock + ≥5m buffer; sidecar `personality` set;
        ready JSON contains required fields listed under Core contracts
      - `--wall-clock` CLI flag overrides binding default / goals budgets;
        test precedence (CLI wins when set)
      - fake clock / short wall-clock → status `incomplete_wallclock`,
        Stop called, **exit code != 0**
      - explicit stop signal → `stopped`
- [ ] Failing CLI tests in `cmd/playtestrun/main_test.go`:
      - `playtestrun stop --checkout PATH --run ID` writes
        `tools/playtest/.run/<run_id>/bridge/stop` (idempotent if already
        present)
      - `playtestrun status --checkout PATH --run ID` prints sidecar JSON
        to stdout (includes nested `budgets.wall_clock`)
- [ ] Implement `run`: ParseGoals → apply optional `--wall-clock` override →
      Start(Profiles|empty, Lease, Checkout) → mkdir bridge → write sidecar →
      print one ready JSON line → wait on deadline **or** stop signal file
      `tools/playtest/.run/<run_id>/bridge/stop` → Stop → update sidecar;
      return non-zero on `incomplete_wallclock` and `environment_failed`.
- [ ] Wire `cmd/playtestrun` subcommands `run` / `status` / `stop` and
      flags (`--run` not `--session`; `--wall-clock` on `run`).
- [ ] Commit: `feat(playtest): add playtestrun wall-clock supervisor`

---

### Task 4: Exemplar goals + report templates

**Files:** goals exemplars; `tools/playtest/report-templates/*`

- [ ] Read `newbie-naive.yaml`, `corpse-looting.yaml`,
      `2026-08-03-prepush-sweep.yaml`, and sample reports under
      `tools/playtest/reports/` +
      `tools/_archive/testing-pre-harness/testing/reports/` before authoring
      templates (do not invent structure blind).
- [ ] Add `ephemeral:` to `newbie-naive.yaml` (creation_flow + rationale).
- [ ] Add `ephemeral:` to `corpse-looting.yaml` with a real start_room and
      suitable profile (e.g. `early` or `mid` — pick by reading objectives;
      verify room exists via id inventory / world knowledge).
- [ ] Add `ephemeral:` to `2026-08-03-prepush-sweep.yaml` as the adversarial
      bug-finder SOP exemplar (creation_flow + non-empty rationale, or a
      suitable profile+room if objectives clearly assume an existing char —
      prefer creation_flow for broad adversarial sweeps unless the file
      content contradicts that).
- [ ] Add at least `newbie-creation.md` and `bug-finder-sweep.md` templates
      with required header placeholders (checkout, commit, dirty, run_id,
      binding, wall-clock, status).
- [ ] Commit: `feat(playtest): ephemeral exemplars and report templates`

---

### Task 5: `/playtest` + human docs

**Files:** `.claude/commands/playtest.md`, `CLAUDE.md` (AI Testing),
`docs/guides/TESTING_GUIDE.md`, `internal/playtestrun/context.md`

- [ ] Rewrite local path in `playtest.md`:
      - require `--checkout` and goals file
      - call `playtestrun run` (or document exact invocation)
      - **ready-gate:** do not start mudagent unless ready JSON / sidecar
        status is `ready`; on `environment_failed` abort and point at
        (or write) the environment-failed report from the sidecar path
      - skip `targets.yaml` for local
      - bridge under `.run/<run_id>/bridge/`
      - creds match / creation-flow
      - incomplete wall-clock / soft token stop ⇒ incomplete report
      - required report header fields
- [ ] Update CLAUDE.md AI Testing bullets for ephemeral local SOP. Cite the
      bug-finder exemplar by path:
      `tools/playtest/goals/2026-08-03-prepush-sweep.yaml` (not a generic
      “an exemplar”). Example form:
      `/playtest local bug-finder 2026-08-03-prepush-sweep.yaml`.
- [ ] TESTING_GUIDE: short section pointing at playtestrun + examples.
- [ ] **Verbose** `internal/playtestrun/context.md` section **“Human
      invocation”** covering: options/flags, checkout rules, profile vs
      creation-flow, reading sidecar/reports, worked examples, loud failures.
      This section is a deliverable, not optional polish.
- [ ] Commit: `docs(playtest): wire /playtest local to ephemeral playtestrun`

---

### Task 6: Driver-contract smoke + Docker + verification + roadmap

- [ ] **Driver-contract smoke** (no Claude session required): unit or opt-in
      test that a successful ready path exposes JSON with `endpoint`, `creds`,
      `run_id`, `checkout`, `commit`, `dirty`, `deadline_at`, sidecar path,
      and that `bridge_dir` is under `tools/playtest/.run/<run_id>/bridge/`.
      Prefer extending Task 3 fake-supervisor tests if they already assert
      this; otherwise add an explicit package test named for the driver
      contract.
- [ ] Opt-in Docker: with `DOGMUD_PLAYTESTENV_INTEGRATION=1` (or a dedicated
      `DOGMUD_PLAYTESTRUN_INTEGRATION=1` if cleaner), run profile exemplar and
      creation-flow exemplar through `playtestrun` against a real checkout —
      assert sidecar ready, creds null vs present, stop cleanup.
- [ ] Unit package tests green; gofmt.
- [ ] Update roadmap 0.3c status with evidence when done.
- [ ] Adversarial **implementation** review; fix Blocking/Important.
- [ ] Commit: `test(playtest): cover ephemeral playtestrun integration`

---

## Suggested subagents

- **Task 0 — shell:** branch hygiene.
- **Tasks 1–3 — generalPurpose / Sonnet:** TDD core playtestrun + CLI.
- **Task 4 — Sonnet:** goals/templates (read archives first).
- **Task 5 — Sonnet:** driver + verbose docs.
- **Task 6 — Sonnet:** smoke + Docker integration + adversarial impl review.

## Spec coverage checklist

- [x] Hybrid playtestrun composing playtestenv — Task 3
- [x] Explicit ephemeral binding + KnownFields — Task 1
- [x] Creation-flow rationale — Tasks 1, 4
- [x] Creation-flow forbids profile/start_room/overlays — Task 1
- [x] Wall-clock blocking watchdog + lease buffer — Task 3
- [x] CLI `--wall-clock` override precedence — Task 3
- [x] Non-zero exit on incomplete_wallclock — Task 3
- [x] Driver ready-gate / env-failed abort — Task 5
- [x] Sidecar statuses incl. environment_failed — Tasks 2–3
- [x] Nested `budgets.wall_clock` — Tasks 2–3
- [x] run_id / `--run` — Task 3
- [x] `playtestrun stop` / `status` CLI — Task 3
- [x] Run-scoped bridge — Tasks 3, 5, 6
- [x] Ready JSON / driver-contract smoke — Tasks 3, 6
- [x] Creds profile match — Tasks 2, 5
- [x] local skips targets.yaml — Task 5
- [x] Checkout + personality required — Tasks 3, 5
- [x] Fail-closed profile without start_room / bad room — Task 1
- [x] Missing/unreadable goals path — Task 1
- [x] Legacy max_rounds ignored — Task 1
- [x] Optional overlays passthrough — Task 1
- [x] Exemplars match content (incl. bug-finder SOP) — Task 4
- [x] Report templates mined — Task 4
- [x] Verbose human context.md — Task 5
- [x] SOP/CLAUDE cites prepush-sweep goals path — Task 5
- [x] Docker evidence + impl review — Task 6
- [x] No max-turns / no hard token kill — constraints

## Plan process note

After this plan is written: run adversarial **plan** review, amend if needed,
then obtain **explicit user approval** before any implementation task.
