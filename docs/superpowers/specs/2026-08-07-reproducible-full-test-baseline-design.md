# Reproducible Full-Test Baseline Design

**Roadmap chunk:** 0.2 — Establish a reproducible full-test baseline
**Status:** Approved 2026-08-07

## Problem

The adversarial review could not obtain an authoritative local full-suite
result on Windows because Microsoft Defender blocked execution of a generated
Go test binary. Disabling antivirus or weakening test coverage would turn the
environmental failure into a safety regression rather than establish a useful
baseline.

Pull-request CI already executes the ordinary package suite with the race
detector on Ubuntu:

```text
go test -timeout 300s -race -coverprofile=coverage.out ./...
```

That command honors tests' normal skip conditions. Selected tests gated behind
`DOGMUD_BOOT_SMOKE` are invoked separately by CI, while other tests sharing
that gate currently have no dedicated CI invocation. None are part of this
default race-enabled package-suite baseline.

What is missing is a documented local command that exercises the same package
pattern, per-package timeout, and race detector in a controlled Linux
environment without relying on Windows test-binary execution.

## Goals

1. Provide one documented, cross-platform Docker command for executing the
   ordinary Go package suite with the race detector in Linux.
2. Preserve authoritative exit status for image-build, generation,
   compilation, timeout, test, and race failures.
3. Keep test execution outside Docker image-build layers so a cached image
   cannot masquerade as a fresh passing test run.
4. Keep the existing production Alpine builder and runner behavior unchanged.
5. Document prerequisites, expected output, and common failure categories.
6. Produce fresh local evidence for whether the current ordinary package suite
   passes.

## Non-Goals

- Disabling or weakening Microsoft Defender, antivirus, or sandbox controls.
- Replacing native focused tests during ordinary development.
- Unifying all local and CI validation; that belongs to Roadmap Chunk 1.1.
- Adding local lint, formatting, coverage-threshold, generated-file-cleanliness,
  JavaScript, or real-data boot gates.
- Changing pull-request, master, tag, or release workflows.
- Repairing unrelated test or product failures discovered by the baseline.
- Running tests that deliberately require `DOGMUD_BOOT_SMOKE`. Selected gates
  retain their existing dedicated CI execution; inventorying and assigning the
  uncovered gates belongs to Chunk 1.1. All remain visibly skipped by this
  baseline.
- Testing only committed source. The container intentionally includes relevant
  uncommitted source and content changes from the current working tree.

## Design

### 1. Add an independent Linux test stage

Add a `test` stage to `provisioning/Dockerfile`. The stage uses the
Go-version-aligned `golang:1.25.0-bookworm` image, matching the Go version
declared by `go.mod` while providing the Debian/glibc compiler environment
needed by the Go race detector. The tag is not digest-pinned; the guide must
describe the environment as version-aligned, not bit-for-bit reproducible.

The test stage is independent of the Alpine production `builder` stage. It:

1. sets `CGO_ENABLED=1`;
2. uses `/src` as its working directory;
3. copies `go.mod` and `go.sum` and downloads the complete module graph in a
   cacheable layer;
4. copies the filtered current working tree into the image;
5. runs `go generate ./...`; and
6. declares `go test -v -count=1 -timeout 300s -race ./...` as its default
   runtime command.

The stage is placed before the final `runner` stage. The production runner
therefore remains the Dockerfile's default final stage, and an ordinary
production build still copies only from the existing Alpine `builder`.
Debian test dependencies cannot enter the runner image.

Generation may be cached when neither its inputs nor the Dockerfile have
changed. The test command must not be a Dockerfile `RUN` instruction: it runs
in a newly created container on every canonical invocation. `-count=1`
independently disables Go's successful-test result cache.

### 2. Constrain the shared Docker build context

Add `provisioning/Dockerfile.dockerignore`, the Dockerfile-specific ignore file
for builds that use `provisioning/Dockerfile`.

The ignore contract excludes host-only or sensitive inputs that must not enter
reusable image layers, including:

- `.git`, worktrees, local code indexes, and agent scratch state;
- `.mcp.json`, TLS private keys, and machine-local settings;
- compiled binaries, coverage files, logs, reports, and temporary output; and
- player saves and other mutable runtime persistence ignored by Git.

The patterns must preserve all tracked source, schemas, authored world data,
tracked `.gitkeep` placeholders, and package test fixtures. Direct
`git ls-files ":(glob)internal/**/_datafiles/**"` verification currently
returns no tracked files, so package-local `_datafiles` runtime/test residue may
be excluded as a tree. Recheck that invariant before implementing the pattern;
if tracked fixtures are introduced later, preserve them explicitly.

One tracked file is deliberately excluded:
`tools/playtest/targets.yaml`, because it currently contains operational
credentials that must not enter reusable image layers. Its removal from source
control and credential rotation belong to a separate security chunk. Guild
persistence under `_datafiles/**/guilds/` is also excluded even when a local
guild directory exists.

The implementation verifies representative exclusions and inclusions using a
Docker build-context probe before relying on the file for either test or
production builds. Adding this file hardens the context shared by both targets;
it must not alter files intentionally copied into the existing production
runner.

### 3. Add an isolated Compose test service

Add `compose.test.yml` with one service named `test`. The service:

- builds the repository root with `provisioning/Dockerfile`;
- selects the `test` target;
- uses the test image's default command;
- does not bind-mount the host source tree;
- does not publish ports or depend on game-server services; and
- does not alter `compose.yml`.

The canonical command, run from the repository root, is:

```text
docker compose -f compose.test.yml run --build --rm test
```

`--build` ensures current source and Dockerfile inputs are considered before
the run. Docker may reuse valid build layers, but Compose starts a new test
container and executes the full test command each time. `--rm` removes that
container afterward; reusable image layers remain available.

No GNU Make installation or platform-specific wrapper script is required.

### 4. Preserve unambiguous pass/fail behavior

The canonical command succeeds only when:

1. the test image is available or builds successfully;
2. repository generation succeeds;
3. every package compiles under the race-enabled Linux toolchain;
4. each package test binary completes within its 300-second timeout;
5. every test selected by the ordinary suite passes, while deliberate skips
   remain visible in Go test output; and
6. the race detector reports no race.

Docker build failures and the test process's exit status propagate through
Compose. The design introduces no retry, package exclusion, failure suppression,
or success-text parser. Terminal output from `go test` remains the evidence.

The local command deliberately omits CI's `-coverprofile=coverage.out` because
its purpose is the ordinary race-enabled package baseline, not coverage
artifact production. It adds `-v` so deliberate skips are visible and
`-count=1` to make fresh execution explicit. It otherwise uses the same package
pattern, per-package timeout, and race setting as the current pull-request CI
command. Coverage and opt-in gate ownership remain in Chunk 1.1.

### 5. Document the workflow

Add `docs/guides/TESTING_GUIDE.md` containing:

1. prerequisites: Docker Desktop or another Docker engine with Linux-container
   support, Docker Compose v2, an architecture supported by Go's Linux race
   detector, sufficient local disk and memory for a race build, and registry
   and Go-module network access for an uncached first image build;
2. the repository-root canonical command;
3. an explanation that source is copied at image-build time and tests run in a
   fresh container;
4. the success contract: process exit zero and complete `go test` output;
5. common failure categories:
   - Docker daemon unavailable;
   - image pull or build failure;
   - `go generate` failure;
   - compilation or test failure;
   - test timeout;
   - race-detector report;
6. guidance that native `go test` remains appropriate for focused feedback;
7. the relationship to Ubuntu pull-request CI; and
8. an explicit warning not to disable security controls to make native Windows
   execution pass;
9. the fact that `DOGMUD_BOOT_SMOKE` tests remain skipped and are outside this
   baseline; and
10. the mutable-image-tag limitation: Go-version-aligned, not bit-for-bit
    reproducible.

The guide must not describe the container command as complete CI parity.

## Failure Handling

### Docker is installed but unavailable

Docker Compose returns nonzero. The guide directs the developer to start the
Docker engine and ensure Linux containers are enabled. The repository does not
attempt to start Docker Desktop or modify its configuration automatically.

### Image construction or generation fails

The canonical command stops before tests and returns nonzero. The build output
identifies the failing Dockerfile step. Such a result is not reported as a test
failure or a successful baseline.

### A package fails, times out, or races

The canonical command returns nonzero and retains the Go test output in the
terminal. The 300-second timeout applies independently to each package test
binary, not to the aggregate Compose invocation or image build. Chunk 0.2
records the failing package and diagnostic. It does not exclude the package,
increase the timeout reflexively, or repair unrelated behavior merely to turn
the baseline green.

If the failure is an environmental defect in this design, revise the design or
implementation narrowly and rerun. If it is a product or test defect, create or
assign a separate remediation chunk before declaring a green safety net.

## Verification Strategy

### Static configuration checks

Run:

```text
docker compose -f compose.test.yml config
```

This must resolve the service, build context, Dockerfile, and `test` target
without error.

Review the Dockerfile stage graph to confirm:

- `runner` is still the final stage;
- `runner` still copies only from the Alpine `builder`;
- the Debian test stage is independent of `runner`; and
- the full test command is a runtime command, not a build-layer command.

Place harmless sentinel files at representative ignored paths, build the test
image, and inspect `/src` to confirm those sentinels are absent. Also inspect
representative authored data, `.gitkeep` placeholders, and package fixtures to
confirm they are present. Verify every tracked path except the intentionally
excluded credential-bearing playtest target is present. Remove the sentinels
after the check.

### Exit-status negative control

After building the test image, run:

```text
docker compose -f compose.test.yml run --rm test sh -c "exit 7"
```

In PowerShell, assert `$LASTEXITCODE -eq 7`. In a POSIX shell, capture `$?`
immediately and assert it equals `7`. This proves that the canonical execution
path does not convert a container failure into apparent success.

The negative control is implementation verification, not a permanent test in
the Go suite.

### Ordinary package-suite baseline

With the Docker engine running, execute:

```text
docker compose -f compose.test.yml run --build --rm test
```

Capture the command's exit code and full terminal result. A green baseline
requires exit zero from this fresh invocation of
`go test -v -count=1 -timeout 300s -race ./...`. Deliberately
environment-gated tests remain visible as skips. A red result is still useful
diagnostic evidence but does not satisfy the green-baseline completion
criterion.

### Production build regression check

Build the Dockerfile once without `--target` and once with `--target test`.
Inspect the untargeted image to confirm it has the existing production
entrypoint, reports Alpine in `/etc/os-release`, and has no `/src` tree. This
proves that `runner` remains the default final target and that the Debian test
filesystem was not selected as the production output.

### Documentation check

Follow `docs/guides/TESTING_GUIDE.md` from the repository root without relying
on GNU Make or an unstated host Go installation. Every documented command and
failure interpretation must match observed behavior.

## Completion Criteria

Chunk 0.2 is complete when:

1. `provisioning/Dockerfile` has an independent Debian/glibc test stage while
   preserving the Alpine production builder and final runner.
2. `provisioning/Dockerfile.dockerignore` excludes representative secrets and
   runtime state while preserving required source, authored data, placeholders,
   and test fixtures; the tracked credential-bearing playtest target is the
   documented exception.
3. `compose.test.yml` provides the single canonical local ordinary-suite
   command.
4. Tests execute in a new container with `-count=1` rather than as a cached
   Docker build layer or Go test result.
5. Compose configuration validation passes.
6. A deliberate exit status of `7` propagates unchanged to the caller.
7. The canonical command freshly completes with exit zero for
   `go test -v -count=1 -timeout 300s -race ./...`, with deliberate
   `DOGMUD_BOOT_SMOKE` skips disclosed.
8. An untargeted Docker build still produces the Alpine production runner,
   which has no `/src` test-stage tree and remains isolated from test-only
   dependencies.
9. `docs/guides/TESTING_GUIDE.md` accurately documents prerequisites, command,
   guarantees, limitations, and failures.
10. No security control, package, ordinary-suite test, race check, or timeout is
    weakened to obtain the passing baseline.

## Risks and Mitigations

### Docker cache hides test execution

Keep `go test` in the test image's runtime command and pass `-count=1`. Docker
may cache dependency, copy, and generation layers, but every Compose run creates
a container and freshly executes the ordinary suite.

### The test stage changes the production image

Keep `runner` last, retain its existing dependency on the Alpine `builder`, and
make the Debian stage independent. Verify the production target separately.

### Local output is mistaken for complete CI parity

Document the exact overlap and differences. Chunk 0.2 establishes the ordinary
race-enabled package-suite baseline only; Chunk 1.1 owns broader validation
parity.

### Host files bypass image isolation

Do not bind-mount source into the service. Use the Dockerfile-specific ignore
contract to omit sensitive and runtime-only host state, then let `--build`
snapshot the remaining current working tree into the test image.

### A genuine failure expands this chunk indefinitely

Preserve the failing evidence and classify it. Repair only defects in the test
baseline mechanism here; route unrelated product or test defects to their own
remediation work.
