# Multi-Agent Ephemeral Scenarios Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.
>
> **Do not start this plan until the user explicitly approves it** (after
> adversarial plan review). Spec approval alone is not enough.

**Goal:** One shared ephemeral env runs a scenario roster of actors (each with
own goals file + loadout), with per-actor bridges, file blackboard, scenario
wall-clock, `actor_id` creds, rewritten `/playtest-scenario`, and a pre-merge
~10m mixed-party live smoke.

**Architecture:** Extend `playtestrun` with `scenario` subcommand composing
`playtestenv`. Driver (Claude or implementer) keeps the watchdog alive, spawns
N concurrent mudagents, coordinates via in-game + file blackboard, writes one
combined report. Soft per-actor tokens; hard cut = scenario wall-clock.

**Tech Stack:** Go 1.25, playtestenv/playtestprofiles/playtestrun, yaml.v3,
testify, Docker opt-in + live smoke.

**Approved design:**
`docs/superpowers/specs/2026-08-08-multi-agent-ephemeral-scenarios-design.md`
(revised after adversarial spec review 2026-08-08).

---

## Execution constraints

- Branch: `feature/stage-0.3d-multi-agent-ephemeral-scenarios` from current
  `master` (0.3c already merged).
- Do not stage protected dirty room / adversarial / invalidated-0.1 files.
- Stage exact owned paths only; no `git add .`.
- No ptorch; no hard per-actor token kill; no admin in multi; PvP scenarios
  deferred (refuse).
- Do not claim Done without Docker evidence + pre-merge live smoke + adversarial
  implementation review.

## File map

### New

- `internal/playtestrun/scenario.go` (+ `_test.go`) — parse scenario YAML
- `internal/playtestrun/scenario_run.go` (+ `_test.go`) — scenario supervisor
- `internal/playtestprofiles/prod_identity_denylist.go` (+ tests) — stems +
  close-variant matcher
- `internal/playtestprofiles/identity.go` — `ForbiddenIdentity(name) error`
- `tools/playtest/goals/scenarios/party-formation/{leader,joiner}.yaml`
- `tools/playtest/goals/scenarios/parallel-coverage/{explorer,shopper}.yaml`
- `tools/playtest/goals/scenarios/feel-pothole-newbie-veteran/{newbie,veteran}.yaml`
- `tools/playtest/report-templates/multi-agent-combined.md` (optional if format
  doc is enough)

### Modify

- `internal/playtestprofiles/types.go` — `ActorID` on `PlayerCreds`
- `internal/playtestprofiles/materialize.go` / persist path — stamp `actor_id`
- `internal/playtestenv` — pass actor ids through Profiles/manifest if needed
- `cmd/playtestrun/main.go` (+ tests) — `scenario` subcommand
- `internal/playtestrun/context.md` — Human invocation for scenario + blackboard
- `.claude/commands/playtest-scenario.md` — full rewrite off ptorch
- `tools/playtest/multi-agent-report-format.md` — 0.3d headers
- `tools/playtest/scenarios/*.yaml` — migrate (except PvP defer)
- `docs/guides/TESTING_GUIDE.md`, `CLAUDE.md` AI Testing — scenario pointers
- Roadmap 0.3d status when done

### Do not modify for scope

- Production `compose.yml`
- Pre-0.3c local dead-code cleanup
- Full PvP ephemeral config override system (defer)

## Core contracts

```go
type ScenarioFile struct {
    Name         string
    Mode         string // opaque
    Summary      string
    OnActorStop  string // continue|abort
    WallClock    time.Duration
    Requires     map[string]any // opaque to Go except pvp refuse
    Roster       []ScenarioActor
    GroupGoals   []yaml.Node // opaque
}

type ScenarioActor struct {
    ID           string
    Personality  string
    GoalsPath    string // resolved
    Binding      EphemeralBinding
}

func ParseScenario(scenarioPath, playtestRoot string, opts ScenarioParseOpts) (ScenarioFile, error)
// ScenarioParseOpts.Force bypasses MaxAIConnections size check only.
func RunScenario(ctx context.Context, p ScenarioParams) error
func ForbiddenIdentity(name string) error // playtestprofiles
```

Ready JSON / sidecar per approved spec (`actor_id` on creds; per-actor bridges).

---

### Task 0: Branch hygiene

- [ ] Create `feature/stage-0.3d-multi-agent-ephemeral-scenarios` from master.
- [ ] If dirty room/adversarial files present, do not stage them.
- [ ] Cherry/checkout approved 0.3d spec+plan onto the branch if needed.

---

### Task 1: Prod-identity denylist + helper (TDD)

**Files:** `internal/playtestprofiles/prod_identity_denylist.go`, `identity.go`,
tests; wire into `SanitizeTemplate` + credential generation character names.

- [ ] Write failing tests for: exact stem; `pt_`+stem; digits around stem;
      Levenshtein-1 (len≥4); substring (len≥5); allowed unrelated names.
- [ ] Generate/check in denylist stems from `_archive/prod-users` offline (script
      or one-shot); commit `prod_identity_denylist.go`.
- [ ] Implement `ForbiddenIdentity`; call from sanitize + materialize naming.
- [ ] `go test ./internal/playtestprofiles -count=1`
- [ ] Commit: `feat(playtest): expand prod-identity denylist and matcher`

---

### Task 2: `actor_id` on creds (TDD)

**Files:** `playtestprofiles` types/materialize; `playtestenv` manifest if
needed; tests.

- [ ] Failing tests: materialize two `early` entries with distinct actor ids →
      creds players carry `actor_id`; lookup by actor_id succeeds; profile-only
      ambiguous still errors for multi helper.
- [ ] Add `ActorID string \`json:"actor_id,omitempty"\`` to `PlayerCreds`.
- [ ] Stamp from manifest/roster when materializing.
- [ ] Add `SelectCredsByActorID(credsPath, actorID)`.
- [ ] Commit: `feat(playtest): stamp actor_id on playtest creds`

---

### Task 3: Scenario parse (TDD)

**Files:** `internal/playtestrun/scenario.go`, `scenario_test.go`

- [ ] Failing tests: happy party-formation shape; unknown scenario key; unknown
      roster key; duplicate id; empty roster; invalid roster `id` format;
      admin profile reject; legacy target/role/inline goals reject; missing
      goals file; `on_actor_stop` default continue; unknown `on_actor_stop` →
      error; `requires.pvp` → error; `requires.foo` (non-pvp) parses OK;
      roster size > MaxAIConnections → error; `--force` bypass for size only;
      default `budgets.wall_clock` 45m when omitted; actor wall_clock parsed
      but ignored by scenario supervisor later.
- [ ] Implement `ParseScenario` with KnownFields; resolve goals paths; call
      `ParseGoalsEphemeral` per actor; reject admin.
- [ ] Commit: `feat(playtest): parse multi-agent scenario YAML`

---

### Task 4: `playtestrun scenario` supervisor (TDD with fakes)

**Files:** `scenario_run.go`, `cmd/playtestrun`, tests

- [ ] Failing tests: missing checkout; binding error no Start; Start failure →
      environment_failed; ready JSON actors + blackboard_dir + bridges;
      scenario sidecar fields per spec (`scenario_path`, `on_actor_stop`,
      `actors[]` with per-actor paths/creds/username); default wall_clock 45m;
      CLI `--wall-clock` overrides scenario file; lease = wall_clock + ≥5m;
      wall-clock → `incomplete_wallclock` non-zero; SIGINT → `interrupted`;
      stop → `stopped`; scenario-aware `status` output; abort path documents
      driver-set `incomplete_abort` + peer `aborted_peer` (unit-test helper or
      documented driver contract in tests).
- [ ] Implement RunScenario: parse → Profiles in roster order with actor ids →
      Start → mkdir `actors/<id>/bridge` + `blackboard/` → write scenario
      sidecar → ready JSON → wait deadline/stop → Stop.
- [ ] Add scenario sidecar types (`StatusIncompleteAbort`, per-actor status
      enum) — extend or parallel `SessionSidecar` in `sidecar.go` /
      `scenario_run.go`.
- [ ] Wire CLI flags (`--scenario`, `--wall-clock`, `--force`).
- [ ] Commit: `feat(playtest): add playtestrun scenario supervisor`

---

### Task 5: Migrate scenarios + per-actor goals

**Files:** scenarios + `goals/scenarios/...`

- [ ] Migrate `party-formation.yaml` (mixed personalities/loadouts; no admin).
- [ ] Migrate `parallel-coverage.yaml` **and**
      `feel-pothole-newbie-veteran.yaml` as shared-env concurrency exemplars.
- [ ] Leave `adversarial-contest.yaml` with a header comment: deferred PvP;
      ParseScenario/driver refuse if invoked with pvp require.
- [ ] Prefer distinct templates/overlays across roster (diversity guidance).
- [ ] Commit: `feat(playtest): migrate scenarios to ephemeral roster binds`

---

### Task 6: Driver + docs + report format

**Files:** `.claude/commands/playtest-scenario.md`,
`tools/playtest/multi-agent-report-format.md`, `internal/playtestrun/context.md`,
`CLAUDE.md`, `TESTING_GUIDE.md`

- [ ] Rewrite `/playtest-scenario`: checkout required; `playtestrun scenario`;
      ready-gate; long-lived watchdog; N bridges; blackboard file I/O with
      temp + atomic rename; no ptorch; `on_actor_stop`; combined report
      checklist.
- [ ] Driver: creation-flow RoleUser-only; `ForbiddenIdentity` pre-check on
      `new` names before accept; mudagent spawn failure honors `on_actor_stop`.
- [ ] Document when to use 0.3c `playtestrun run` vs `scenario` (non-shared
      parallel stays multiple 0.3c runs) in TESTING_GUIDE / CLAUDE.md.
- [ ] Rewrite multi-agent report format (0.3d headers; drop ptorch bb dump).
- [ ] Verbose Human invocation for `scenario` + blackboard in context.md.
- [ ] Commit: `docs(playtest): wire /playtest-scenario to playtestrun scenario`

---

### Task 7: Docker + live smoke + roadmap + impl review

- [ ] **Driver-contract smoke (no LLM):** `playtestrun scenario` → parse ready
      JSON → verify per-actor `bridge_dir`, `blackboard_dir`, sidecar paths;
      `playtestrun stop`. No mudagents required.
- [ ] Opt-in Docker: `DOGMUD_PLAYTESTRUN_INTEGRATION=1` (or dedicated flag)
      two-actor mixed profiles ready+stop through `playtestrun scenario`.
- [ ] **Pre-merge live smoke:** ~10m party-formation (or dedicated smoke
      scenario) with mixed loadouts/personalities; implementer drives mudagents
      via ready JSON bridges; produce combined report with checklist; confirm
      invite/accept (or equivalent). Cap wall-clock ~10–15m.
- [ ] Unit tests green; gofmt.
- [ ] Roadmap 0.3d → Done with evidence.
- [ ] Adversarial **implementation** review; fix Blocking/Important.
- [ ] Commit: `test(playtest): cover multi-agent scenario integration`

---

## Suggested subagents

Prefer subagents when a stronger/non-quota model is available. **If only
Grok/Composer (or parent-only) is available, execute Tasks 0–7 inline on the
parent agent** — same TDD order and commit cadence; skip fan-out rather than
blocking.

- **Task 0 — shell:** branch hygiene.
- **Tasks 1–4 — Sonnet/generalPurpose:** TDD core (identity, creds, parse, run).
- **Task 5 — Sonnet:** scenario/goals migration (read existing scenarios first).
- **Task 6 — Sonnet:** driver + docs.
- **Task 7 — Sonnet:** Docker + live smoke.
- **Task 7 — generalPurpose (review):** adversarial implementation review
  (independent from implementer when possible; otherwise parent self-review
  against the spec checklist with fresh eyes).

## Spec coverage checklist

- [x] Shared env + concurrent bridges — Task 4
- [x] Per-actor goals + ephemeral — Tasks 3, 5
- [x] actor_id creds — Task 2
- [x] Blackboard dirs — Task 4; contract in docs Task 6
- [x] Scenario wall-clock sole hard cut — Task 4
- [x] Admin ban — Task 3
- [x] Prod-identity helper shared — Task 1
- [x] on_actor_stop continue/abort — Tasks 4, 6 (abort + driver docs)
- [x] Driver rewrite off ptorch — Task 6
- [x] PvP defer — Tasks 3, 5
- [x] MaxAIConnections — Task 3
- [x] Driver-contract smoke — Task 7
- [x] Docker + live smoke — Task 7
- [x] No hard per-actor token kill — constraints

## Plan process note

Adversarial **plan** review 2026-08-08: Request changes → amended; re-review
**Approve** (parent-agent execution fallback noted). Obtain **explicit user
approval to implement** before Task 0.
