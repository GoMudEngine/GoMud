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

---

## Execution constraints

- Branch: create `feature/stage-0.3c-single-agent-ephemeral-playtests` from
  current `master` (or from merged 0.3b if already on master). Do **not**
  continue feature work on the 0.3b v2 branch unless 0.3b is already merged
  and this plan is applied as a follow-on commit series there by explicit
  user request.
- Preserve pre-existing uncommitted room / adversarial / invalidated-0.1
  files; never stage them (`484.yaml`, pothole rooms, etc.).
- Stage exact owned paths only; no `git add .`.
- Do not implement multi-agent, hard token kill switch, or max-turns.
- Do not claim Done without Docker evidence + adversarial implementation
  review.

## File map

### New

- `internal/playtestrun/` — binding parse, sidecar, run supervisor, creds
  match helper, `context.md` (verbose human guide), `*_test.go`
- `cmd/playtestrun/main.go` (+ tests)
- `tools/playtest/report-templates/` — at least `newbie-creation.md`,
  `bug-finder-sweep.md` (mine before inventing)
- Exemplar goals updates (binding blocks only where content matches)

### Modify

- `.claude/commands/playtest.md` — local ephemeral contract
- `docs/guides/TESTING_GUIDE.md` — pointer + ephemeral local usage
- `CLAUDE.md` AI Testing section — checkout + goals required for local
- `docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md` — 0.3c status
  when done
- Goals exemplars: `tools/playtest/goals/newbie-naive.yaml`,
  `tools/playtest/goals/corpse-looting.yaml` (or chosen mid-game file)

### Do not modify for scope

- `internal/playtestenv` env policy (compose only) except calling its
  public `Start`/`Stop`/`Status` APIs
- Production `compose.yml`

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
// Session sidecar JSON
type SessionSidecar struct {
    RunID              string `json:"run_id"`
    Checkout           string `json:"checkout"`
    Commit             string `json:"commit"`
    Dirty              bool   `json:"dirty"`
    GoalsPath          string `json:"goals_path"`
    Personality        string `json:"personality,omitempty"`
    Endpoint           *playtestenv.Endpoint `json:"endpoint,omitempty"`
    Creds              string `json:"creds,omitempty"`
    Profile            string `json:"profile,omitempty"`
    CreationFlow       bool   `json:"creation_flow,omitempty"`
    CreationRationale  string `json:"creation_rationale,omitempty"`
    WallClock          string `json:"wall_clock"`
    StartedAt          time.Time `json:"started_at"`
    DeadlineAt         time.Time `json:"deadline_at"`
    Status             string `json:"status"` // starting|ready|incomplete_wallclock|stopped|environment_failed
    EnvironmentReport  string `json:"environment_report,omitempty"`
    BridgeDir          string `json:"bridge_dir"`
}
```

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
- [ ] Confirm dirty room/adversarial files are present and will not be staged.
- [ ] Commit nothing yet (or empty docs-only cherry of approved spec/plan if
      they live only on another branch — prefer checkout of the two approved
      doc paths onto the new branch).

---

### Task 1: Goals `ephemeral:` parse (TDD)

**Files:** `internal/playtestrun/binding.go`, `binding_test.go`

- [ ] Write failing tests for: profile+room happy; creation_flow+rationale
      happy; missing ephemeral; unknown key; both profile and creation_flow;
      neither; creation without rationale; unknown profile id; wall_clock
      parse; default 30m when budgets omitted.
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

- [ ] Failing tests: write/read sidecar; status transitions; SelectCredsPlayer
      matches profile; errors on missing/ambiguous players; never logs password.
- [ ] Implement atomic sidecar write under
      `tools/playtest/.run/<run_id>/session.json`.
- [ ] Implement `SelectCredsPlayer`.
- [ ] Commit: `feat(playtest): session sidecar and creds player selection`

---

### Task 3: `playtestrun run` supervisor (TDD with fakes)

**Files:** `internal/playtestrun/run.go`, `cmd/playtestrun/main.go`, tests

- [ ] Define a narrow `envSupervisor` interface matching
      `playtestenv.Supervisor` Start/Stop/Status used by run.
- [ ] Failing tests with fake supervisor:
      - missing `--checkout` → usage error before Start
      - binding error → no Start
      - Start failure → sidecar `environment_failed`, non-zero exit
      - ready path writes sidecar `ready`, bridge dir created,
        lease = wall_clock + ≥5m buffer
      - fake clock / short wall-clock → `incomplete_wallclock` then Stop
      - explicit stop → `stopped`
- [ ] Implement `run`: ParseGoals → Start(Profiles|empty, Lease, Checkout) →
      mkdir bridge → write sidecar → print one JSON line → wait on
      deadline **or** stop signal file `tools/playtest/.run/<run_id>/bridge/stop`
      → Stop → update sidecar.
- [ ] Wire `cmd/playtestrun` flags (`--run` not `--session`).
- [ ] Commit: `feat(playtest): add playtestrun wall-clock supervisor`

---

### Task 4: Exemplar goals + report templates

**Files:** goals exemplars; `tools/playtest/report-templates/*`

- [ ] Read `newbie-naive.yaml`, `corpse-looting.yaml`, and sample reports
      under `tools/playtest/reports/` +
      `tools/_archive/testing-pre-harness/testing/reports/` before authoring
      templates (do not invent structure blind).
- [ ] Add `ephemeral:` to `newbie-naive.yaml` (creation_flow + rationale).
- [ ] Add `ephemeral:` to `corpse-looting.yaml` with a real start_room and
      suitable profile (e.g. `early` or `mid` — pick by reading objectives;
      verify room exists via id inventory / world knowledge).
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
      - skip `targets.yaml` for local
      - bridge under `.run/<run_id>/bridge/`
      - creds match / creation-flow
      - incomplete wall-clock / soft token stop ⇒ incomplete report
      - required report header fields
- [ ] Update CLAUDE.md AI Testing bullets for ephemeral local SOP (point at
      an exemplar goals file for adversarial bug-finder runs).
- [ ] TESTING_GUIDE: short section pointing at playtestrun + examples.
- [ ] **Verbose** `internal/playtestrun/context.md` section **“Human
      invocation”** covering: options/flags, checkout rules, profile vs
      creation-flow, reading sidecar/reports, worked examples, loud failures.
      This section is a deliverable, not optional polish.
- [ ] Commit: `docs(playtest): wire /playtest local to ephemeral playtestrun`

---

### Task 6: Integration + verification + roadmap

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
- **Tasks 1–3 — generalPurpose / Sonnet:** TDD core playtestrun.
- **Task 4 — Sonnet:** goals/templates (read archives first).
- **Task 5 — Sonnet:** driver + verbose docs.
- **Task 6 — Sonnet:** Docker integration + adversarial impl review.

## Spec coverage checklist

- [x] Hybrid playtestrun composing playtestenv — Tasks 3
- [x] Explicit ephemeral binding + KnownFields — Task 1
- [x] Creation-flow rationale — Tasks 1, 4
- [x] Wall-clock blocking watchdog + lease buffer — Task 3
- [x] Sidecar statuses incl. environment_failed — Tasks 2–3
- [x] run_id / `--run` — Task 3
- [x] Run-scoped bridge — Task 3, 5
- [x] Creds profile match — Task 2, 5
- [x] local skips targets.yaml — Task 5
- [x] Checkout required — Tasks 3, 5
- [x] Exemplars match content — Task 4
- [x] Report templates mined — Task 4
- [x] Verbose human context.md — Task 5
- [x] SOP/CLAUDE/TESTING_GUIDE — Task 5
- [x] Docker evidence + impl review — Task 6
- [x] No max-turns / no hard token kill — constraints

## Plan process note

After this plan is written: run adversarial **plan** review, amend if needed,
then obtain **explicit user approval** before any implementation task.
