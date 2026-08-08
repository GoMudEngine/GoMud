# Ephemeral Playtest Server Supervisor Design

**Roadmap chunk:** 0.3a — Build the ephemeral server supervisor
**Status:** Implemented 2026-08-08

## Problem

DOGMud's current local playtest flow assumes that a developer has already
started one server at `localhost:55555`. The ordinary `compose.yml` cannot safely
provide isolated playtest environments: it uses fixed container names, fixed
host ports, a fixed network name, and does not publish the dedicated AI port.

That makes it difficult to test a particular branch or dirty worktree, run
independent tests concurrently, distinguish their runtime state, collect boot
evidence, or guarantee cleanup after an interrupted agent session.

Chunk 0.2 established a filtered Docker build context and a repeatable Linux
test image. Chunk 0.3a builds the next enabling layer: a local-only supervisor
that turns one selected checkout into one disposable, verified server endpoint.

## Goals

1. Provide one cross-platform Go command for creating, inspecting, renewing,
   logging, stopping, and reaping ephemeral DOGMud servers.
2. Build the explicitly selected checkout, including relevant modified and
   untracked authored files that survive the Docker context filter.
3. Isolate every run's container, network, image tag, host port, writable game
   data, logs, manifest, and lease.
4. Let Docker assign the host port and publish it on loopback only.
5. Declare a server ready only after log, process, listener, and connection
   checks all pass.
6. Preserve actionable build and server evidence when startup fails.
7. Remove only resources that can be proven to belong to the requested or
   expired run.
8. Ensure server-side admin, builder, and runtime mutations cannot modify the
   selected checkout or enter a commit.
9. Preserve reusable Docker BuildKit cache while removing run-specific
   resources and image tags.
10. Expose stable machine-readable output for later single- and multi-agent
    orchestration.

## Non-Goals

- Connecting to production or any other remote server.
- Accepting an arbitrary target host or port.
- Running `mudagent`, choosing gameplay commands, or evaluating goals.
- Creating users or materializing synthetic player profiles.
- Writing successful gameplay reports.
- Coordinating multi-agent scenarios.
- Reading archived production users.
- Exporting a disposable data volume back into authored `_datafiles`.
- Changing `compose.yml` or the production Docker runner.
- Replacing the Chunk 0.2 package-test command.
- Implementing a long-running local daemon.
- Automatically committing any file.

## Local-Only Security Boundary

The supervisor always builds a local checkout and starts a new local Docker
server. It has no `--target`, `--host`, caller-supplied context flag, SSH, or
production mode, and it rejects any selected context whose resolved daemon
transport is remote.

The published AI endpoint must satisfy both conditions:

- Docker reports host IP `127.0.0.1` or `::1`.
- The supervisor rejects any discovered non-loopback address before declaring
  readiness.

Later commands in Chunks 0.3c and 0.3d consume only the endpoint returned by
this supervisor. They do not accept the existing `prod` target from
`tools/playtest/targets.yaml`.

The supervisor never reads `_archive`, production-user exports, or tracked
playtest credentials. The Docker context contract from Chunk 0.2 excludes those
paths from image layers.

## Architecture

### Components

Add:

```text
cmd/playtestenv/
  main.go

internal/playtestenv/
  compose.playtest.yml
  context.md
  command.go
  compose.go
  lifecycle.go
  manifest.go
  readiness.go
  reaper.go
  report.go
  *_test.go
```

File boundaries may be adjusted during planning, but the package must keep
these responsibilities separate:

- CLI parsing and rendering;
- subprocess execution;
- Compose project and port management;
- manifest/state persistence;
- readiness classification;
- failure evidence;
- exact-run cleanup; and
- stale-run discovery and cleanup.

Because `internal/playtestenv` is a new package, it must ship a `context.md`
describing its verified public API, files, dependencies, consumers, and
cleanup hazards.

### Subprocess boundary

The supervisor invokes the installed Docker and Docker Compose CLIs through
`os/exec.CommandContext`. It does not import a Docker SDK.

Arguments are passed as an argument slice. Checkout paths, run IDs, project
names, and other values are never concatenated into shell source. The package
depends on a narrow command-runner interface so unit tests can assert exact
commands, output, exit status, cancellation, and timeouts without invoking
Docker.

The supervisor rejects ambient `DOCKER_HOST`, resolves the selected/current
named Docker context, and inspects its daemon endpoint before creating
resources. It accepts only a local Windows named pipe or Linux Unix socket;
TCP, SSH, and other remote transports are rejected. Every later command uses
that validated context name explicitly while child processes receive no
inherited `DOCKER_HOST`, `DOCKER_CONTEXT`, `DOCKER_TLS_VERIFY`, or Docker
certificate-path override. This supports local rootless and Docker Desktop
contexts without allowing a remote daemon.

Subprocess capture drains stdout and stderr concurrently, or intentionally
combines them into one ordered stream, before waiting. Large build output must
not be able to deadlock the supervisor.

### Selected checkout

`start` accepts an explicit checkout path, resolves it to an absolute canonical
path, and verifies at minimum:

- the path exists and is a directory;
- `go.mod` and `provisioning/Dockerfile` exist beneath it;
- Git identifies the path itself as a checkout/worktree root; and
- Git confirms the planned `.run` and generated-report paths are ignored;
- the supervisor's run directory can be created; and
- the resolved path is not under `_archive`.

The Compose policy is embedded in the supervisor binary from
`internal/playtestenv/compose.playtest.yml`; the supervisor never reads a
Compose definition from the selected checkout. It materializes the embedded
definition as a run-scoped control file. Its build context is the validated
absolute checkout path, never `.` relative to the materialized Compose file.
Normal Docker context behavior includes modified and untracked files unless
`provisioning/Dockerfile.dockerignore` excludes them.

The checkout is never mounted into the server container.

The selected checkout still contains executable Go source and a Dockerfile, so
this is isolation from accidental runtime/source-state contamination, not a
sandbox for hostile source code. The supervisor must not weaken its Compose
policy based on values from that checkout.

## Dedicated Compose Definition

The embedded `compose.playtest.yml` defines one service, `server`, with:

- validated absolute checkout build context;
- `provisioning/Dockerfile`;
- the existing production `runner` target;
- no `container_name`;
- no fixed image name;
- no fixed network name;
- `restart: "no"`;
- one project-scoped named volume mounted at `/app/_datafiles`;
- internal AI port `55555` published to a Docker-assigned host port on loopback;
- a direct `/app/go-mud-server` entrypoint so the server is PID 1; and
- immutable supervisor, run, project, checkout-fingerprint, schema-version, and
  creation-time labels supplied by supervisor-controlled substitution.

Only the AI port is published. Telnet, HTTP, and other listeners remain
container-internal.

The empty named volume uses Docker's copy-up behavior to receive the image's
authored `/app/_datafiles` tree on first mount. All server writes then remain in
that volume.

The supervisor bind-mounts only an ignored run-owned `control/` directory,
writable at `/run/dogmud`; it does not mount the checkout, manifest, reports, or
logs. `CONFIG_PATH` points to a generated override there. The override forces
the dedicated AI listener to port `55555`, disables file logging, and pins
`Server.CurrentVersion` to the string-literal `VERSION` parsed from the selected
checkout so current authored data does not replay historical migrations on
every ephemeral boot. Legitimate `configs.SetVal` calls can update this
disposable override. It contains no secrets and is deleted with run control
files after final evidence has been captured.

The service also sets `LOG_NOCOLOR=1`; readiness parsing therefore receives
stable plain-text log markers rather than ANSI-colored output.

The implementation must prove the selected Compose syntax actually requests a
Docker-assigned loopback port on Docker Compose `>= 2.20.0`. The long port
syntax omits `published` rather than using undocumented port zero. It may not
fall back to probing for a free host port and then racing another process to
bind it.

## CLI Contract

The command is available as:

```text
go run ./cmd/playtestenv <operation>
```

The eventual built binary has the same interface.

### Start

```text
playtestenv start --checkout <path> [--lease <duration>] [--json]
```

Defaults:

- checkout: current working directory;
- lease: two hours;
- boot-readiness timeout: 90 seconds.

`start` returns only after the environment is ready or cleanup has completed
after failure.

### Status

```text
playtestenv status --checkout <path> --run <run-id> [--json]
```

`status` reads the manifest and queries Docker. It reports disagreement rather
than trusting either source alone.

### Logs

```text
playtestenv logs --checkout <path> --run <run-id> [--follow] [--json]
```

Without `--follow`, `logs` refreshes the stored server log and prints its path.
Machine-readable mode never mixes human log text into JSON stdout.
`--follow` and `--json` are mutually exclusive.

### Renew

```text
playtestenv renew --checkout <path> --run <run-id> --lease <duration> [--json]
```

Docker resource labels are immutable after creation, so `renew` updates only
the atomically written manifest after revalidating the live immutable identity
labels. It rejects an already expired, stopped, or ambiguous run.

### Stop

```text
playtestenv stop --checkout <path> --run <run-id> [--json]
```

`stop` captures final logs, requests graceful termination for up to ten
seconds, force-removes only if necessary, performs project-scoped Compose
cleanup, removes the run-specific local image tag, and retains BuildKit cache.

Stopping an already-cleaned run succeeds and reports that no resources
remained.

### Reap

```text
playtestenv reap --checkout <path> [--json]
```

`reap` considers only resources bearing the supervisor's complete immutable
identity label set. It reconciles those labels with manifests below the
selected checkout. A run is eligible only when the manifest lease is expired
and its project/run/checkout identity is unambiguous.

Malformed, partial, mismatched, unlabeled, or manifest-less resources are
reported and left untouched. A missing manifest never falls back to a stale
creation-time label for destructive cleanup because the lease may have been
renewed after resource creation.

`renew`, `stop`, and each per-run reaper action acquire the same exclusive
cross-process run lock. This is an OS-native advisory lock, using a maintained
cross-platform Go implementation of Unix advisory locking and Windows
`LockFileEx`; the operating system releases it when a process exits. It is not a
PID file and uses no PID-liveness or age heuristic. The reaper rereads and
validates the manifest lease while holding that lock immediately before its
first destructive action. Lock waits are bounded—direct operations wait at most
five seconds and reaper candidates at most 250 milliseconds by default.
Failure to acquire the lock leaves resources untouched and produces a
diagnostic.

## Run Identity and Concurrency

Each start creates:

- a collision-resistant run ID;
- a Compose-safe project name derived from that ID;
- a project-scoped network;
- a project-scoped `_datafiles` volume;
- a unique local image tag;
- a Docker-assigned loopback host port;
- a run directory; and
- a lease.

Run IDs contain only lowercase ASCII letters, digits, and hyphens. User-supplied
run IDs are not accepted by `start`. It atomically creates the generated run
directory, acquires that run's OS-native lock, and writes the initial manifest
under the lock before invoking Docker. A generated collision causes a new ID to
be generated; it never reuses the existing directory.

No mutable package global stores active run state. Independent supervisor
processes and independent worktrees must be able to start concurrently.

## Manifest and Artifacts

Run root:

```text
tools/playtest/.run/<run-id>/
```

This path is relative to the selected checkout, regardless of where the
supervisor binary was built or invoked. It is a supervisor-owned, ignored
artifact path; no other checkout path may be written by the supervisor.

Required files:

```text
manifest.json
build.log
server.log
control/config-overrides.yaml
compose.resolved.yml
```

The versioned JSON manifest contains no secrets. It records:

- schema version;
- run ID and Compose project;
- canonical checkout path;
- lifecycle state;
- creation, update, and lease-expiry timestamps;
- image, service, network, and volume identities;
- discovered loopback endpoint;
- readiness observations;
- artifact paths;
- cleanup progress; and
- a structured non-secret failure category and summary.

Manifest updates use write-to-temporary-file plus atomic replacement. State
transitions are validated; an operation cannot move a stopped or failed run
back to ready.

The initial `validating` manifest, including the lease and planned immutable
identity, is closed successfully before the first Docker resource is created.
If that write fails, startup stops without invoking Docker.

Human output is concise. With `--json`, stdout contains exactly one JSON result
object. Diagnostics and subprocess output go to stderr and artifact files.

## Lifecycle

The state machine is:

```text
validating -> building -> starting -> ready -> stopping -> stopped
     |           |           |         |          |
     +-----------+-----------+---------+----------+-> failed
```

Failure during validation does not create Docker resources. Failure during
build or startup records evidence, transitions to `failed`, and then performs
cleanup. The final result remains failed even when cleanup succeeds.

Cancellation or interruption during `start` uses the same failure and cleanup
path. Evidence capture and cleanup detach from the cancelled caller through a
new bounded cleanup context so cancellation does not strand Docker resources.
Once `start` returns ready, the caller owns eventual `stop`; expired runs are
recovered by `reap`.

Stopping an already-cleaned failed run succeeds while preserving `failed` as
its historical state; a successful stop operation does not rewrite that run as
if startup had succeeded.

An explicit stop or expired-run reap treats a manifest abandoned in
`validating`, `building`, `starting`, or `stopping` as failed, records an
abandoned-run category, and resumes exact-resource cleanup. Cleanup timeout
records leftovers and leaves the run immediately recoverable by another
`stop`.

## Readiness Contract

`start` declares readiness only when all of the following are true:

1. Compose reports the server container running.
2. The container remains running through the readiness checks.
3. Logs contain the exact `Server Ready` event.
4. Logs contain no panic.
5. Logs contain no generic listener-creation failure.
6. Compose or Docker inspection reports exactly one published mapping for
   internal port `55555`.
7. The published host address is loopback.
8. A real TCP connection to the published endpoint succeeds.

The probe closes immediately and does not attempt login.

Process exit code alone is not authoritative. Existing server panic recovery
can allow startup failure to exit zero, and listener errors can be logged
without preventing `Server Ready`.

The current `TelnetListenOnPort` error does not identify which listener failed.
Accordingly, any `Error creating server` event fails startup and is categorized
as a generic listener-creation failure; the port mapping and TCP probe provide
the AI-listener-specific evidence.

The boot-readiness deadline starts after the image build and container-create
command complete. It defaults to 90 seconds and is configurable for tests. A
deadline failure records the last observed readiness facts.

## Failure Evidence

Failure categories include:

- invalid checkout or run ID;
- Docker or Compose unavailable;
- build failure;
- container exited;
- boot panic;
- listener-creation failure;
- missing or ambiguous port publication;
- non-loopback publication;
- readiness timeout;
- connection probe failure;
- manifest failure;
- retryable run-lock contention (`lock_busy`);
- abandoned preterminal run (`abandoned_run`); and
- cleanup failure.

Before deleting run resources, the supervisor captures build output, server
logs where available, inspection evidence, and the manifest.

Pre-gameplay failures after checkout validation and run reservation also create:

```text
tools/playtest/reports/<timestamp>-<run-id>-environment-failed.md
```

This report path is relative to the selected checkout.

Failures before a safe ignored report path and run identity can be established
return structured stderr/JSON only; they do not write into an invalid checkout.

The report names the checkout, run ID, lifecycle phase, failure category,
artifact paths, and cleanup outcome. It contains no credentials or production
data.

If cleanup also fails, the command remains nonzero and lists exact labelled
resources still present. It never broadens deletion in response.

## Cleanup and Reaping

Normal cleanup uses the exact Compose project and run labels from the validated
manifest. It:

1. captures final logs;
2. requests bounded graceful shutdown;
3. force-removes the run container only if needed;
4. runs project-scoped `down -v --remove-orphans`;
5. removes the run-specific local image tag; and
6. preserves BuildKit cache and artifacts.

Once cleanup's first destructive action begins, it uses a fresh bounded context
detached from caller cancellation. This applies to start-failure cleanup,
explicit stop, and each reaper candidate, preventing Ctrl-C from stranding a
half-removed run.

An expired or explicitly stopped run abandoned in `validating`, `building`, or
`starting` transitions to `failed` with an abandoned-run category before using
normal cleanup. A hard-killed supervisor therefore remains recoverable without
misrepresenting the run as successful.

The reaper does not use wildcard names, `docker system prune`, broad process
kills, or unfiltered label queries.

Every destructive action is preceded by identity checks against both manifest
and live Docker labels. Disagreement blocks deletion and produces a diagnostic.

## Source and Runtime Mutation Isolation

The pre-run manifest records a non-secret Git baseline:

- current commit, if any;
- path/status metadata for tracked modifications;
- path/status metadata for staged paths; and
- path/status metadata for untracked paths relevant to the selected checkout.

The baseline is collected from machine-readable Git status, never `git diff`,
patch text, or file contents. Git reads use `--no-optional-locks` so observation
does not refresh or lock the user's index. The collector does not open
`tools/playtest/targets.yaml`, environment/credential files, production-user
exports, or `_archive`; those paths and the supervisor's ignored artifact roots
are excluded from recorded metadata. Manifest and report renderers treat all
Git path text as untrusted data.

This baseline is evidence, not a cleanup recipe. The supervisor never resets,
checks out, cleans, stages, commits, or deletes source-tree files.

Server writes—including admin and builder commands in later chunks—occur only
inside the disposable `_datafiles` volume and ignored run-owned control
directory. There is no source bind mount and no volume export operation.
Cleanup destroys the volume and control directory.

Tester-facing integrations in Chunks 0.3c and 0.3d may write only run-scoped
commands, events, logs, manifests, and reports. Any unexpected checkout delta
observed after a run is classified as contamination and must be excluded from
commits. Path/status changes from the baseline are reported, but concurrent
author changes are not automatically attributed to the playtest. Pre-existing
and concurrently authored user changes are never automatically reverted.

An intended content correction discovered during playtesting must be
re-authored and reviewed in a later implementation step. Runtime admin state is
never promoted into source.

## Portability

Supported hosts are Windows and Linux systems with:

- Docker Engine using Linux containers;
- Docker Compose `>= 2.20.0`; and
- the Go version declared by `go.mod` when using `go run`.

The supervisor contains no Bash, `tail -f`, `pkill`, shell pipelines, or
PowerShell-specific lifecycle logic. Path handling uses Go filesystem APIs.

Subprocess cancellation, Windows process behavior, and Docker Desktop output
formats receive focused tests. Docker output parsing prefers structured JSON or
documented formatted fields over human-oriented text.

## Test Strategy

### Unit tests

Use a fake command runner and temporary directories to cover:

- run ID generation and validation;
- atomic run-directory reservation, generated-ID collision retry, and start
  locking;
- project-name derivation;
- canonical checkout validation;
- rejection of `_archive`;
- exact Docker/Compose argument construction;
- selected named local Docker context, scrubbed override environment, daemon
  connectivity, Compose-version floor, and rejection of remote endpoints;
- deadlock-free simultaneous large stdout/stderr capture;
- use of the embedded Compose policy rather than a checkout file;
- manifest schema, atomic writes, and legal state transitions;
- initial manifest persistence before any Docker invocation;
- human versus JSON output separation;
- rejection of `logs --follow --json`;
- port-inspection parsing and loopback enforcement;
- readiness success and every failure category;
- timeout and cancellation;
- idempotent stop;
- lease renewal;
- OS-native cross-process mutual exclusion among start, renew, stop, and reap,
  including automatic release after process exit;
- stale versus live classification;
- final lease reread under lock before reap;
- label/manifest mismatch refusal;
- exact-resource cleanup plans;
- cleanup failure reporting;
- failed-environment report generation;
- metadata-only Git baseline recording without credential/file-content reads or
  mutation; and
- secret-like value redaction from errors and artifacts.

### Docker-backed integration tests

Integration tests are opt-in and run from the host because the Chunk 0.2 test
container does not expose the host Docker socket.

They must cover:

1. one successful start, status, log capture, renewal, and stop;
2. two simultaneous starts with distinct project, port, volume, image, and
   artifact identities;
3. two worktrees whose modified or untracked authored probe differs, proving
   each image received its own checkout;
4. loopback-only publication selected by Docker, never by free-port probing;
5. authored `_datafiles` copy-up into the disposable volume;
6. a runtime mutation inside the volume followed by proof that the checkout and
   Git state did not change;
7. build failure;
8. boot panic or pre-ready exit;
9. readiness timeout;
10. listener or publication failure;
11. cancellation during startup;
12. graceful and forced stop;
13. repeated stop;
14. expired-run reaping;
15. a labelled decoy and an ambiguous resource that must not be deleted;
16. failure report/log preservation with no run resources left;
17. removal of the run image while BuildKit cache remains reusable; and
18. ambient remote Docker environment variables and context selection being
    rejected rather than honored; and
19. a normal production-image build and clean boot after the supervisor tests.

Tests record pre/post host Git state. Any playtest-created checkout delta fails
the test and is removed only when it is a test-owned sentinel with an exact
known path.

### Relevant broader verification

Run:

```text
docker compose -f compose.test.yml run --build --rm test
```

Also run the supervisor integration suite on Windows Docker Desktop and on
Linux CI or a documented Linux environment before completion.

## Completion Criteria

Chunk 0.3a is complete when:

1. The Go CLI implements `start`, `status`, `logs`, `renew`, `stop`, and `reap`
   with human and JSON output.
2. A dedicated Compose definition creates no fixed names, fixed host ports, or
   shared run state.
3. The selected dirty checkout is the build input while the checkout itself is
   never mounted into the server.
4. Docker assigns exactly one loopback AI endpoint and compound readiness proves
   it accepts a connection.
5. Two independent worktrees and two simultaneous runs remain isolated.
6. Build, startup, interruption, and cleanup failures produce nonzero outcomes
   plus useful manifests, logs, and failed-environment reports.
7. Stop is idempotent, and the reaper deletes only expired resources whose
   manifest and labels agree.
8. Containers, networks, volumes, and run image tags are gone after cleanup;
   BuildKit cache and reports remain.
9. Runtime/admin mutations occur only in the disposable volume or ignored
   run-owned control directory, are destroyed, and cannot enter Git.
10. The supervisor cannot accept or derive a production or remote target.
11. Unit, Docker integration, merged full-suite, Windows, Linux, and production
    boot verification pass with fresh evidence.
12. `internal/playtestenv/context.md` documents the verified package surface and
    cleanup hazards.
13. An adversarial implementation review finds no unresolved blocking or
    important issue.

## Risks and Mitigations

### A stale manifest points at unrelated Docker resources

Require matching run, project, checkout, and supervisor labels from live
resources before deletion. Treat disagreement as ambiguity and refuse cleanup.

### Server output claims readiness after a partial startup

Require the complete readiness contract, including a live container, no
panic/bind error, exact loopback publication, and a successful connection.

### A caller disappears after start

Use leases, renewal, labelled resources, and a conservative reaper. Do not add
a background daemon.

### A downstream playtester exhausts its token budget

Chunk 0.3a does not run or meter model calls, so it cannot distinguish token
exhaustion from another caller disappearance. Its lease and reaper still bound
the server lifetime and preserve environment evidence. Chunks 0.3c and 0.3d
must classify token exhaustion explicitly, retain partial transcripts/reports,
and trigger normal stop rather than reporting gameplay success.

### Concurrent builds or runs collide

Use collision-resistant run IDs, Compose project isolation, Docker-assigned
ports, project-scoped volumes, and run-scoped artifact paths.

### Playtest admin changes leak into authored content

Never mount the checkout or host `_datafiles`; destroy the disposable volume;
provide no export path; compare post-run Git state with the recorded baseline.

### Cleanup destroys unrelated resources

Use no wildcard deletion or global prune. Require exact manifest/label
agreement before each destructive action and test with labelled decoys.

### Cross-platform process behavior drifts

Keep lifecycle logic in Go, pass arguments without a shell, test Windows and
Linux, and use structured Docker output wherever available.
