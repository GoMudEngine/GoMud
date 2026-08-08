# Synthetic Playtest Profiles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.
>
> **Do not start this plan until the user explicitly approves it.**

**Goal:** Materialize run-scoped, AI-flagged offline players from six tracked
sanitized profiles (plus optional overlays) after world load and before
listeners, with per-run credentials for later 0.3c orchestration.

**Architecture:** Package `internal/playtestprofiles` owns template load,
manifest parse (KnownFields), overlay apply, sanitizer, credential generation,
and offline persist. `main.go` calls it on normal boot when
`Playtest.ProfilesManifest` is set. `playtestenv` writes the manifest into
`control/`, sets overrides, and surfaces `creds.json`. Goal→profile picking
and mudagent remain 0.3c.

**Tech Stack:** Go 1.25, `gopkg.in/yaml.v3`, existing `UserRecord.SetPassword`,
Docker playtestenv (0.3a), testify.

**Approved design:**
`docs/superpowers/specs/2026-08-08-synthetic-playtest-profiles-design.md`
(revised after adversarial review 2026-08-08).

---

## Execution constraints

- Branch: `feature/stage-0.3b-synthetic-playtest-profiles`.
- **Task 0 is mandatory:** discard premature implementation commits from the
  process skip; restart TDD from the approved spec + this plan only.
- Preserve pre-existing uncommitted room / adversarial / invalidated-0.1 files;
  never stage them.
- Stage exact owned paths only; no `git add .`.
- Never open `_archive/prod-users` from runtime load paths; archive is offline
  authoring reference only and must not be committed.
- Never call `users.CreateUser` for materialization.
- Do not implement goal→profile selection, mudagent, or token budgets.
- Do not claim Done without Docker integration evidence and adversarial
  implementation review.

## File map

### New

- `internal/playtestprofiles/` — `types.go`, `sanitize.go`, `template.go`,
  `manifest.go`, `overlay.go`, `credentials.go`, `persist.go`,
  `materialize.go`, `room_bridge.go`, `context.md`, `*_test.go`, `testdata/`
- `tools/playtest/profiles/{fresh,early,mid,veteran,specialist-caster,admin}.yaml`
- `tools/playtest/profiles/README.md`

### Modify

- `internal/configs/config.playtest.go` (new) + wire into `Config` / `Validate`
- `_datafiles/config.yaml` — `Playtest:` section (empty `ProfilesManifest`)
  if skip-worktree allows; otherwise rely on Validate defaults + overrides
- `main.go` — materialize after user-index rebuild, before plugins; skip
  copyover; exit on error
- `provisioning/Dockerfile` — runner `COPY` profiles to `/app/playtest/profiles`
- Confirm `Dockerfile.dockerignore` does not exclude `tools/playtest/profiles/**`
- `internal/playtestenv/` — `StartOptions.Profiles`, manifest write, overrides,
  `Artifacts.Creds`, tests
- `cmd/playtestenv/` — optional flags for explicit profile requests
- `docs/guides/TESTING_GUIDE.md`
- `docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md` — 0.3b status

## Core contracts

```go
type Playtest struct {
	ProfilesDir      ConfigString `yaml:"ProfilesDir"`
	ProfilesManifest ConfigString `yaml:"ProfilesManifest"`
}

type Manifest struct {
	Entries []ManifestEntry `yaml:"entries"`
}

type ManifestEntry struct {
	Profile   string   `yaml:"profile"`
	StartRoom int      `yaml:"start_room"`
	Overlays  Overlays `yaml:"overlays,omitempty"`
}

type Overlays struct {
	GrantSpells    map[string]int    `yaml:"grant_spells,omitempty"`
	GrantSkills    map[string]int    `yaml:"grant_skills,omitempty"`
	GrantItems     []int             `yaml:"grant_items,omitempty"`
	Equip          map[string]int    `yaml:"equip,omitempty"`
	SetQuestTokens []string          `yaml:"set_quest_tokens,omitempty"`
	SetQuestFlags  map[string]string `yaml:"set_quest_flags,omitempty"`
	SetGold        *int              `yaml:"set_gold,omitempty"`
}

type CredsFile struct {
	RunID   string        `json:"run_id,omitempty"`
	Players []PlayerCreds `json:"players"`
}

type PlayerCreds struct {
	Profile  string `json:"profile"`
	Username string `json:"username"`
	Password string `json:"password"`
	UserID   int    `json:"user_id"`
	RoomID   int    `json:"room_id"`
}
```

Manifest/overlays decode with `KnownFields(true)`. Template IDs exact:
`fresh`, `early`, `mid`, `veteran`, `specialist-caster`, `admin`.

Overlay semantics and fail-closed rules: see approved spec sections
“Overlay semantics” and “Failure modes”.

---

### Task 0: Discard premature implementation; keep approved docs

**Why:** Code was landed before plan approval. Restart TDD cleanly.

**Do not `git reset --hard` while tracked worktree files are dirty** (e.g.
`thornwall_city/484.yaml`). That would destroy unrelated user edits.

- [ ] **Step 1: Record tip SHA and dirty inventory**

```powershell
git status --short
git rev-parse HEAD
git log --oneline master..HEAD
```

- [ ] **Step 2: Create a clean feature branch from master and restore only docs**

```powershell
$docsTip = git rev-parse HEAD   # tip that has revised spec + this plan
git switch master
git pull --ff-only origin master
git switch -c feature/stage-0.3b-synthetic-playtest-profiles-v2
git checkout $docsTip -- `
  docs/superpowers/specs/2026-08-08-synthetic-playtest-profiles-design.md `
  docs/superpowers/plans/2026-08-08-synthetic-playtest-profiles.md
git add `
  docs/superpowers/specs/2026-08-08-synthetic-playtest-profiles-design.md `
  docs/superpowers/plans/2026-08-08-synthetic-playtest-profiles.md
git commit -m "docs(playtest): approve 0.3b profiles spec and plan baseline"
```

Leave the old premature-code branch tip unmerged (delete later only if the
user asks). Working-tree dirty room/0.1 files carry over on switch when they
do not conflict — confirm with `git status` they remain unstaged.

- [ ] **Step 3: Verify tree**

```powershell
git diff --name-only master...HEAD
```

Expected: only the two docs paths. No `internal/playtestprofiles`, no
`main.go` materializer hook yet.

- [ ] **Step 4: Set active work to the v2 branch for all later tasks**

---

### Task 1: Config + manifest types (TDD)

**Files:** `internal/configs/config.playtest.go`, `configs.go` Validate wire,
`internal/playtestprofiles/{types,manifest}.go` + tests.

- [ ] **Step 1: Failing tests** — empty `ProfilesManifest` valid after Validate;
  `ParseManifest` rejects unknown overlay key; rejects unknown profile id;
  accepts happy veteran entry.

- [ ] **Step 2: Implement** `Playtest` defaults (`ProfilesDir` =
  `tools/playtest/profiles`), `ParseManifest` / `LoadManifest` with
  `KnownFields(true)`.

- [ ] **Step 3: Commit**

```text
feat(playtest): add profiles manifest config and types
```

---

### Task 2: Sanitizer + template load (TDD)

**Files:** `sanitize.go`, `template.go`, `testdata/`, tests.

Forbidden: nonempty password, inbox, emailaddress, admin role on non-`admin`
id, missing character/name.

- [ ] Failing tests for each forbidden case + successful load of fixture
  `testdata/fresh.yaml`.
- [ ] Implement `SanitizeTemplate` + `LoadTemplate(dir, id)`.
- [ ] Commit: `feat(playtest): sanitize and load synthetic profile templates`

---

### Task 3: Overlays, credentials, offline persist (TDD)

**Files:** `overlay.go`, `credentials.go`, `persist.go`, `materialize.go`,
`room_bridge.go`, tests with injectable `WorldChecks`.

- [ ] Tests covering exact overlay table from spec; unknown spell/room/item
  fail; `GenerateCredentials` → `pt-<profile>-*` + bcrypt match;
  `PersistOfflineUser` preserves `role: admin`, sets `IsAI`, does not register
  online connection maps; `Materialize` runs `Character.Validate` per entry
  before creds write; writes `creds.json` mode 0600; `MaterializeFromConfig`
  no-op when manifest empty; second entry failure still returns error
  (partial volume users OK per spec).
- [ ] Persist order: `SaveUser` then index `Create` if missing else `AddUser`.
- [ ] Commit: `feat(playtest): apply overlays and persist offline playtesters`

---

### Task 4: Boot hook

**Files:** `main.go` after user-index rebuild, before `plugins.Load`.

```go
if creds, err := playtestprofiles.MaterializeFromConfig(); err != nil {
    mudlog.Error("playtestprofiles.MaterializeFromConfig()", "error", err)
    os.Exit(1)
} else if len(creds) > 0 {
    mudlog.Info("playtestprofiles", "materialized", len(creds))
}
```

Only inside `if !isCopyover { ... }`. Never log passwords.

- [ ] Commit: `feat(playtest): materialize synthetic profiles before listeners`

---

### Task 5: Author six templates

**Files:** `tools/playtest/profiles/*.yaml`, `README.md`.

| ID | Intent |
|----|--------|
| `fresh` | No gear/inventory/spells; newbie start room via manifest |
| `early` | Basic kit past first lessons |
| `mid` | Mixed skills/spells |
| `veteran` | Sanitized Meirok-class kit (offline authoring from archive) |
| `specialist-caster` | Casting-focused |
| `admin` | `role: admin` |

- [ ] Offline: read `_archive/prod-users` only on the authoring machine; never
  commit archive paths.
- [ ] Test: each template loads via sanitizer; Validate against loaded world
  data (package test or compose.test).
- [ ] Commit: `feat(playtest): add six sanitized synthetic profile templates`

---

### Task 6: playtestenv + Dockerfile

**Files:** `compose.go` / `lifecycle.go` / `types.go` / tests;
`provisioning/Dockerfile`; optional CLI flags.

- [ ] `StartOptions.Profiles []ProfileRequest` with overlays.
- [ ] Write `control/profiles-manifest.yaml`; set
  `Playtest.ProfilesManifest` + `Playtest.ProfilesDir` overrides.
- [ ] Dockerfile runner COPY to `/app/playtest/profiles`.
- [ ] Assert `provisioning/Dockerfile.dockerignore` does not exclude
  `tools/playtest/profiles` (add a test or checklist grep in this task).
- [ ] After ready: `Artifacts.Creds` = host path to control `creds.json`.
- [ ] Unit tests: manifest written; failure reports never embed passwords or
  creds JSON bodies (paths only).
- [ ] Commit: `feat(playtest): wire profile manifests through playtestenv`

---

### Task 7: Integration, docs, review

- [ ] Docker integration: fresh ready+creds+AI login; veteran+spell overlay;
  bad room fails; empty profiles OK; prod runner smoke clean.
- [ ] `context.md`, TESTING_GUIDE, roadmap 0.3b evidence.
- [ ] `gofmt`, focused tests, `compose.test.yml` suite, Windows Docker
  integration.
- [ ] Adversarial **implementation** review; fix Blocking/Important.
- [ ] Commit: `docs(playtest): document synthetic profile materialization`

## Suggested subagents

- **Task 0 — shell:** branch hygiene only.
- **Tasks 1–4 — generalPurpose / Sonnet:** TDD core package + boot hook.
- **Task 5 — Sonnet sequential:** template authoring (YAML ID collision risk).
- **Task 6 — Sonnet:** playtestenv safety.
- **Task 7 — Sonnet:** integration + adversarial implementation review.

## Handoff to 0.3c

Ready AI endpoint + `creds.json` + explicit `Profiles` lists. 0.3c owns
goal→profile binding, mudagent, budgets, API/token incomplete reports.

## Plan adversarial review (2026-08-08)

**Verdict:** Revise then pass (amendments applied in-file).

| Severity | Finding | Resolution |
|----------|---------|------------|
| Blocking | Task 0 `reset --hard` would destroy dirty `484.yaml` | Replaced with master→v2 branch + checkout docs paths only |
| Important | `Character.Validate` timing underspecified in Task 3 | Explicit per-entry Validate before creds write |
| Important | dockerignore exclusion risk not checked | Task 6 grep/assert step |
| Important | Failure-report secret leakage | Task 6: paths only, no creds bodies |
| Minor | Partial multi-entry failure | Task 3 notes error return; volume leftovers OK per spec |

Remaining checklist:

- [x] Spec overlay table has matching Task 3 tests
- [x] Spec CreateUser ban has Task 3 assertion
- [x] Spec Dockerfile COPY has Task 6 step
- [x] Spec creds 0600 / no secrets in reports has Task 6/7 coverage
- [x] Spec copyover skip has Task 4 constraint
- [x] Premature code discarded via safe Task 0
- [x] No goal/mudagent/token scope creep
