# Reproducible Full-Test Baseline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Provide one Docker Compose command that runs DOGMud's ordinary Go
package suite freshly under Linux with the race detector and authoritative
pass/fail output.

**Architecture:** Add an independent Debian/glibc `test` stage before the
existing final Alpine `runner`, select it through an isolated Compose file, and
protect both builds with a Dockerfile-specific context filter. Tests run as the
container command with `-v -count=1`; they never run in a cacheable Docker build
layer.

**Tech Stack:** Docker Engine, Docker Compose v2, Go 1.25.0, Go race detector,
PowerShell or a POSIX shell for verification.

**Specification:**
`docs/superpowers/specs/2026-08-07-reproducible-full-test-baseline-design.md`

**Status:** Implemented and verified 2026-08-07

---

## File map

- Modify `provisioning/Dockerfile` — add the independent Linux test stage while
  leaving the production stage graph intact.
- Create `provisioning/Dockerfile.dockerignore` — exclude host secrets,
  runtime persistence, binaries, and scratch state from this Dockerfile's
  shared build context.
- Create `compose.test.yml` — expose one isolated test service and canonical
  local command.
- Create `docs/guides/TESTING_GUIDE.md` — document prerequisites, command,
  guarantees, limitations, and diagnostics.
- Modify `docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md` — advance
  Chunk 0.2 status and record verification evidence after review.

## Execution constraints

- Docker Desktop's CLI is installed, but its Linux engine was not running when
  this plan was written. Start it before the first image build; do not weaken
  Windows security settings.
- Preserve all unrelated working-tree changes, including authored room YAML and
  the existing review/roadmap documents.
- Do not commit unless the user explicitly authorizes commits. Commit
  checkpoints below are conditional reminders, not authorization.
- Never build the pre-filter `COPY . .` context as a red test: doing so could
  bake ignored secrets and player state into an image layer.
- `DOGMUD_BOOT_SMOKE` tests remain opt-in. This chunk does not change their CI
  ownership.

### Task 1: Establish the safe Docker context contract

**Files:**
- Create: `provisioning/Dockerfile.dockerignore`

- [ ] **Step 1: Reconfirm the package-local runtime-tree invariant**

Run:

```powershell
git ls-files ":(glob)internal/**/_datafiles/**"
```

Expected: no output. If tracked files appear, stop and preserve them with
specific negations before using the broad `internal/**/_datafiles/` rule.

- [ ] **Step 2: Record the failing precondition without building**

Run:

```powershell
Test-Path "provisioning/Dockerfile.dockerignore"
```

Expected before implementation: `False`.

This static red check replaces an unsafe pre-filter image build.

- [ ] **Step 3: Create the Dockerfile-specific ignore file**

Create `provisioning/Dockerfile.dockerignore` with:

```dockerignore
# Source-control and worktree metadata
.git
.git/**
.worktrees
.worktrees/**
.codegraph
.codegraph/**
.superpowers
.superpowers/**
.vscode
.vscode/**
vendor
vendor/**
_archive
_archive/**
book_build
book_build/**
.icongen
.icongen/**

# Machine-local credentials and settings
.mcp.json
private-notes.txt
server.crt
server.key
tools/playtest/targets.yaml
.claude/settings.local.json
.claude/scheduled_tasks.lock
.claude/worktrees
.claude/worktrees/**
.claude/projects
.claude/projects/**
.aider*

# Compiled and generated host artifacts
bin
bin/**
*.exe
**/*.exe
coverage.out
coverage_check.out
cov_*.out
tmp-gomud.exe
go-mud-server
GoMud.exe
GoMud.exe~
dogmud
dogmud-t3-review.exe
*.m4b
__debug*
__pycache__
**/__pycache__/**
/*.png
/*.jpg
/*.jpeg
/*.txt

# Logs, reports, and local tool scratch
startup.log
server_boot.log
boot_err.txt
boot_out.txt
smoke_out.txt
bug_log.txt
moons_feedback.md
post_stage18_thoughts.txt
novel/moons_feedback.md
novel/wtmk_cover_concept.png
tools/testing/reports/*.md
tools/playtest/.run*
tools/playtest/.run*/**
tools/playtest/.d2.log
tools/playtest/reports/*
!tools/playtest/reports/.gitkeep
!tools/playtest/reports/2026-06-12-local-fresh-feel-tester-newbie-naive.md
tools/bridge_*.txt
tools/bridge_*.log
tools/mud_*.txt
tools/mud_*.log
tools/run_bridge.bat

# Root world runtime and persistent living state
**/config-overrides.yaml
**/.roundcount
**/rooms.instances
**/rooms.instances/**
**/mobs.instances
**/mobs.instances/**
_datafiles/**/users/*
!_datafiles/world/default/users/1.yaml
!_datafiles/world/dogmud/users/.gitkeep
_datafiles/**/plugin-data/*
!_datafiles/world/default/plugin-data/README.md
!_datafiles/world/dogmud/plugin-data/.gitkeep
_datafiles/**/shops/*
!_datafiles/world/dogmud/shops/.gitkeep
_datafiles/**/guilds
_datafiles/**/guilds/**
_datafiles/**/warehouses
_datafiles/**/warehouses/**
_datafiles/**/caravans
_datafiles/**/caravans/**
_datafiles/**/foragers
_datafiles/**/foragers/**
_datafiles/**/crates
_datafiles/**/crates/**
_datafiles/**/bounties.yaml
_datafiles/**/factions.rep
_datafiles/**/factions.crimes
_datafiles/**/knowledge
_datafiles/**/knowledge/**
_datafiles/**/opinions
_datafiles/**/opinions/**
_datafiles/**/facts.awareness
_datafiles/**/facts.awareness/**
_datafiles/**/goals/*
!_datafiles/world/dogmud/goals/.gitkeep
_datafiles/**/economy/snapshots
_datafiles/**/economy/snapshots/**
_datafiles/logs
_datafiles/logs/**
_datafiles/feedback/*
!_datafiles/feedback/bugs-prod.txt
!_datafiles/feedback/suggestions-prod.txt
_datafiles/world/dogmud/moderation
_datafiles/world/dogmud/moderation/**

# Package-local test/runtime residue; no tracked files exist here as of 2026-08-07
internal/**/_datafiles
internal/**/_datafiles/**
internal/language/testdata/reload-test/*
!internal/language/testdata/reload-test/.gitignore
```

- [ ] **Step 4: Verify the file now exists**

Run:

```powershell
Test-Path "provisioning/Dockerfile.dockerignore"
```

Expected: `True`. Read the file directly and compare it with Step 3. Do not
stage it merely to make ordinary `git diff` display an untracked file.

- [ ] **Step 5: Conditional commit checkpoint**

If and only if the user has authorized commits:

```powershell
git add provisioning/Dockerfile.dockerignore
git commit -m "build: constrain Docker build context"
```

### Task 2: Add the test image stage and Compose entry point

**Files:**
- Modify: `provisioning/Dockerfile:1-26`
- Create: `compose.test.yml`

- [ ] **Step 1: Demonstrate that the Compose contract is absent**

Run:

```powershell
docker compose -f compose.test.yml config
```

Expected before implementation: nonzero with a missing-file error. This command
does not require a running Docker daemon.

- [ ] **Step 2: Add the independent test stage**

Insert the following stage after the existing Alpine `builder` and before the
existing `FROM alpine:latest AS runner` line:

```dockerfile
FROM golang:1.25.0-bookworm AS test

ENV CGO_ENABLED=1
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go generate ./...

CMD ["go", "test", "-v", "-count=1", "-timeout", "300s", "-race", "./..."]
```

Do not alter the existing builder or runner instructions. `runner` must remain
the final stage.

- [ ] **Step 3: Create the isolated Compose service**

Create `compose.test.yml` with:

```yaml
services:
  test:
    build:
      context: .
      dockerfile: provisioning/Dockerfile
      target: test
```

Do not add ports, volumes, dependencies, fixed container names, or an override
for the image's default command.

- [ ] **Step 4: Validate Compose configuration**

Run:

```powershell
docker compose -f compose.test.yml config
```

Expected: exit `0`; rendered service `test` has context `.`, Dockerfile
`provisioning/Dockerfile`, and target `test`.

- [ ] **Step 5: Review the stage graph statically**

Read `provisioning/Dockerfile` and verify in order:

1. `builder` remains Alpine;
2. `test` is Debian-based and independent;
3. `runner` remains last; and
4. `runner` copies only from `builder`.

- [ ] **Step 6: Conditional commit checkpoint**

If and only if the user has authorized commits:

```powershell
git add provisioning/Dockerfile compose.test.yml
git commit -m "build: add isolated Linux race-test target"
```

### Task 3: Verify context filtering, status propagation, and production isolation

**Files:**
- Temporary: `docs/docker-context-include-probe.txt`
- Temporary: `_datafiles/world/dogmud/users/docker-context-exclude-probe.yaml`

- [ ] **Step 1: Confirm Docker is ready**

Run:

```powershell
docker version --format "client={{.Client.Version}} server={{.Server.Version}}"
```

Expected: both client and server versions are nonempty. If the server value is
missing, stop and ask the user to start Docker Desktop's Linux engine.

- [ ] **Step 2: Add harmless build-context sentinels**

Using file-edit tools, create:

`docs/docker-context-include-probe.txt`

```text
included authored-tree probe
```

`_datafiles/world/dogmud/users/docker-context-exclude-probe.yaml`

```text
excluded runtime-state probe
```

Do not touch any existing user save or `.mcp.json`.

- [ ] **Step 3: Build the test target**

Run:

```powershell
docker build --target test -t dogmud-test-baseline:context -f provisioning/Dockerfile .
```

Expected: exit `0`; the Debian test image builds and `go generate ./...`
completes.

- [ ] **Step 4: Inspect context inclusion and exclusion**

Run:

```powershell
docker run --rm --entrypoint sh dogmud-test-baseline:context -c "test -f /src/docs/docker-context-include-probe.txt && test ! -e /src/_datafiles/world/dogmud/users/docker-context-exclude-probe.yaml && test ! -e /src/.mcp.json && test ! -e /src/tools/playtest/targets.yaml && test ! -e /src/_archive && test ! -e /src/vendor && test -f /src/_datafiles/world/dogmud/users/.gitkeep && test -f /src/_datafiles/world/default/users/1.yaml && test -f /src/_datafiles/world/default/plugin-data/README.md && test -f /src/_datafiles/world/dogmud/rooms/thornwall_city/484.yaml && test -f /src/internal/language/testdata/localize/de.yaml && test -f /src/internal/language/testdata/reload-test/.gitignore && test ! -e /src/internal/language/testdata/reload-test/de.yaml"
```

Expected: exit `0`. This proves authored source and required placeholders enter
the image while representative runtime state, archives, vendored host
dependencies, and the local MCP secret do not.

Then verify that every tracked path except the explicitly excluded
credential-bearing playtest target survived the filter:

```powershell
git -c core.quotePath=false ls-files | docker run --rm -i --entrypoint sh dogmud-test-baseline:context -c 'missing=0; while IFS= read -r path; do if [ "$path" = "tools/playtest/targets.yaml" ]; then continue; fi; if [ ! -e "/src/$path" ]; then echo "missing tracked path: $path"; missing=1; fi; done; exit $missing'
```

Expected: exit `0` and no `missing tracked path` lines.

Finally, prove the image contains the current modified room rather than merely
some file at that path:

```powershell
$hostHash = (Get-FileHash "_datafiles/world/dogmud/rooms/thornwall_city/484.yaml" -Algorithm SHA256).Hash.ToLowerInvariant()
$imageHash = ((docker run --rm --entrypoint sha256sum dogmud-test-baseline:context /src/_datafiles/world/dogmud/rooms/thornwall_city/484.yaml) -split '\s+')[0].ToLowerInvariant()
if ($hostHash -ne $imageHash) { throw "working-tree room content was not copied into the test image" }
```

Expected: the hashes match.

Also verify one currently untracked authored room, if it remains in the
working tree at execution time:

```powershell
if (Test-Path "_datafiles/world/dogmud/rooms/pothole_coulee/6468.yaml") {
    $untrackedHostHash = (Get-FileHash "_datafiles/world/dogmud/rooms/pothole_coulee/6468.yaml" -Algorithm SHA256).Hash.ToLowerInvariant()
    $untrackedImageHash = ((docker run --rm --entrypoint sha256sum dogmud-test-baseline:context /src/_datafiles/world/dogmud/rooms/pothole_coulee/6468.yaml) -split '\s+')[0].ToLowerInvariant()
    if ($untrackedHostHash -ne $untrackedImageHash) { throw "untracked authored room was not copied into the test image" }
}
```

Expected with the current working tree: the file exists and its hashes match.
If the user intentionally removes it before execution, record that change and
use another user-owned untracked authored source file if one exists; do not
recreate deleted user content for this probe.

Verify downloaded modules are available without network access:

```powershell
docker run --rm --network none dogmud-test-baseline:context go test -count=1 ./internal/parser
```

Expected: exit `0`.

- [ ] **Step 5: Remove both host sentinels**

Delete only:

- `docs/docker-context-include-probe.txt`
- `_datafiles/world/dogmud/users/docker-context-exclude-probe.yaml`

Confirm neither remains. Perform this cleanup even if Steps 3 or 4 fail.
The execution controller must treat these two deletions as a `finally` action:
after either Docker command returns—successfully or unsuccessfully—delete the
two exact probe paths before interpreting the result or dispatching more work.
At the beginning of any resumed execution, delete stale copies of these two
known probes before recreating them.

- [ ] **Step 6: Verify exact exit-status propagation**

Run:

```powershell
docker compose -f compose.test.yml run --rm test sh -c "exit 7"
$probeExit = $LASTEXITCODE
if ($probeExit -ne 7) { throw "expected exit 7, got $probeExit" }
```

Expected: the container reports failure and the assertion itself exits cleanly,
proving Compose preserved status `7`.

- [ ] **Step 7: Build and inspect the default production output**

Run:

```powershell
docker build -t dogmud-runner-baseline:verify -f provisioning/Dockerfile .
docker image inspect --format "{{json .Config.Entrypoint}}" dogmud-runner-baseline:verify
docker run --rm --entrypoint sh dogmud-runner-baseline:verify -c "grep -q '^ID=alpine' /etc/os-release && test ! -d /src"
```

Expected:

- untargeted build exits `0`;
- entrypoint is `["/bin/sh","-c","./${BIN}"]`; and
- the inspection container exits `0`, proving the default output is Alpine and
  does not contain the Debian test stage's `/src`.

- [ ] **Step 8: Conditional commit checkpoint**

No files should remain from this verification task. Do not commit probe files or
local images.

### Task 4: Write the testing guide

**Files:**
- Create: `docs/guides/TESTING_GUIDE.md`

- [ ] **Step 1: Confirm no existing guide owns this path**

Run:

```powershell
Test-Path "docs/guides/TESTING_GUIDE.md"
```

Expected before implementation: `False`.

- [ ] **Step 2: Write the guide**

Create `docs/guides/TESTING_GUIDE.md` with these exact sections and contracts:

```markdown
# Testing DOGMud

## Focused development tests

Use native `go test` commands for fast feedback on a package or named test.
Native Windows execution may still be affected by antivirus handling of
generated Go test binaries.

## Reproducible Linux race baseline

Prerequisites:

- Docker Desktop or another running Linux-container Docker engine;
- Docker Compose v2;
- a host architecture supported by Go's Linux race detector;
- sufficient disk and memory for a race build; and
- registry and Go-module network access on the first uncached image build.

Downloaded module dependencies are baked into the test image. Individual
application tests may still use networking when their behavior requires it.

From the repository root, run:

```text
docker compose -f compose.test.yml run --build --rm test
```

The build copies the filtered current working tree into an image. Compose then
starts a fresh container and runs:

```text
go test -v -count=1 -timeout 300s -race ./...
```

Exit zero is the success contract. `-count=1` prevents successful Go test
results from being reused. The timeout applies to each package test binary, not
to the aggregate Docker build and run.

Tests requiring `DOGMUD_BOOT_SMOKE` remain visible as skips. Selected smoke
tests run separately in pull-request CI; assigning the remaining opt-in gates
belongs to Roadmap Chunk 1.1.

The local baseline omits CI's coverage artifact, formatting, vet, lint, and
other workflow gates. It is not complete CI parity.

## Failure categories

- Docker daemon unavailable: start the Linux-container engine.
- Image pull or build failure: inspect the failing Dockerfile step.
- Generation failure: fix the reported `go generate ./...` error.
- Compilation or test failure: use the named package and test output.
- Timeout: identify the package whose test binary exceeded 300 seconds.
- Race report: treat it as a failing test and preserve the detector output.

Do not disable antivirus, sandboxing, or other security controls to force a
native Windows run to pass.

## Reproducibility boundary

The Go version is aligned to `go.mod`, but the Docker base image is selected by
a mutable version tag rather than a digest. This is a repeatable,
version-aligned Linux environment, not a bit-for-bit hermetic build.
```

- [ ] **Step 3: Check guide accuracy**

Verify every command matches `compose.test.yml` and the Dockerfile `CMD`.
Search the guide for claims of complete CI parity or command-wide timeout and
remove any such claim.

- [ ] **Step 4: Conditional commit checkpoint**

If and only if the user has authorized commits:

```powershell
git add docs/guides/TESTING_GUIDE.md
git commit -m "docs: document Linux race-test baseline"
```

### Task 5: Run the authoritative baseline and close the chunk

**Files:**
- Modify: `docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md`

- [ ] **Step 1: Run the canonical baseline**

Run:

```powershell
docker compose -f compose.test.yml run --build --rm test
$suiteExit = $LASTEXITCODE
if ($suiteExit -ne 0) { throw "Linux race baseline failed with exit $suiteExit" }
```

Expected: exit `0`, every ordinary package reports success, deliberate
`DOGMUD_BOOT_SMOKE` skips are printed, and no race report appears.

If the result is red, preserve the package/test/race evidence. Fix only a defect
in this baseline mechanism; route an unrelated product or test failure into a
separate remediation chunk.

- [ ] **Step 2: Run final structural checks**

Run:

```powershell
docker compose -f compose.test.yml config
git status --short
git diff --check
```

Expected: Compose exits `0`; only owned files plus pre-existing user changes
appear; `git diff --check` reports no whitespace errors in tracked changes.
Remember that untracked plan/spec files do not appear in ordinary `git diff`.

Use the workspace search tool with pattern `[ \t]+$` against each owned new
file, then read each owned new file in full. Expected: no trailing-whitespace
matches, no truncated files, and no placeholder text. Provide the independent
reviewer an explicit owned-file inventory so untracked deliverables are not
omitted from review.

- [ ] **Step 3: Run an adversarial implementation review**

Give an independent reviewer:

- the approved specification;
- this plan;
- the complete owned-file diff;
- the canonical suite output and exit code;
- the context-probe result;
- the status-7 propagation result; and
- the production-image inspection result.

Require the reviewer to seek omitted files, leaked host state, cached or skipped
tests, Docker-stage regressions, inaccurate documentation, and unproven
completion claims. Resolve actionable findings and rerun affected checks.

- [ ] **Step 4: Update roadmap status and evidence**

Only after the adversarial implementation review is clear, set Chunk 0.2 to
`Done` in the progress tracker. Add a concise evidence line to the Chunk 0.2
section naming:

- the canonical command;
- its successful exit;
- the Docker context and production isolation checks; and
- the implementation-review disposition.

Do not change Chunk 0.3 beyond the separately captured roadmap entry.

- [ ] **Step 5: Conditional final commit**

If and only if the user explicitly authorizes commits:

```powershell
git add provisioning/Dockerfile provisioning/Dockerfile.dockerignore compose.test.yml docs/guides/TESTING_GUIDE.md docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md docs/superpowers/specs/2026-08-07-reproducible-full-test-baseline-design.md docs/superpowers/plans/2026-08-07-reproducible-full-test-baseline.md
git commit -m "build: add reproducible Linux race-test baseline"
```

Before committing, inspect staged files and remove any unrelated user work.

## Suggested subagents

- **Tasks 1–4 — one general-purpose implementation agent using GPT-5.6 Sol
  Medium.** The Dockerfile, ignore contract, Compose service, probes, and guide
  share state and should stay with one agent. This model is strong enough for
  Docker and cross-platform shell reasoning without the cost of a highest-tier
  model. Do not fan these tasks out.
- **Task 5 review — one independent general-purpose reviewer using Claude
  Sonnet 5 Thinking High.** Use the stronger independent pass once, after all
  evidence exists, because context leakage and false-green verification are the
  primary risks.
- **No separate documentation agent.** The guide is short and coupled directly
  to the implementation contract; another context would add cost without useful
  independence.
