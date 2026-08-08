# Ephemeral Playtest Server Supervisor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> superpowers:subagent-driven-development (recommended) or
> superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a local-only Go supervisor that turns one selected DOGMud
checkout into one isolated, lease-bound, verified Docker server endpoint and
cleans up only that run's resources.

**Architecture:** `cmd/playtestenv` is a thin CLI over
`internal/playtestenv.Supervisor`. The package embeds the trusted Compose
policy, invokes Docker through a testable command boundary pinned to a validated
named local context, stores an atomic run manifest under the selected checkout,
and separates startup/readiness, ordinary lifecycle operations, and stale-run
reaping. Docker-backed tests exercise the real production runner image while
unit tests drive failure and race cases through fakes.

**Tech Stack:** Go 1.25, Docker Engine with Linux containers, Docker Compose v2,
`github.com/gofrs/flock`, `github.com/natefinch/atomic`,
`gopkg.in/yaml.v3`, `testify`, GitHub Actions.

**Approved design:**
`docs/superpowers/specs/2026-08-07-ephemeral-playtest-supervisor-design.md`

**Status:** Implemented 2026-08-08

---

## Execution constraints

- Work on `feature/stage-0.3a-ephemeral-playtest-supervisor`.
- Preserve all pre-existing room, review, invalidated-plan, and invalidated-spec
  changes. Never stage them.
- Stage exact owned paths; never use `git add .`, `git clean`, reset, checkout,
  or broad deletion.
- Use TDD for every behavior: failing focused test, minimal implementation,
  passing focused test, then relevant package tests.
- Use `go get github.com/gofrs/flock@latest
  github.com/natefinch/atomic@latest`; do not invent dependency versions.
- Do not expose a remote-target, arbitrary-host, Compose-file, Docker-context,
  source-mount, or volume-export option.
- Test-only Compose substitutions stay in `_test.go`; production constructors
  always use the embedded policy.
- Docker integration tests may delete only resources bearing the exact
  test-created identity and complete supervisor labels.
- Do not claim completion from unit tests alone. Fresh Windows Docker, Linux
  Docker, full-suite, production-image boot, and adversarial review evidence
  are required.

## File map

### New production files

- `cmd/playtestenv/main.go` — subcommand parsing, signal context, output mode,
  and exit-code mapping.
- `internal/playtestenv/context.md` — verified package contract and safety
  notes.
- `internal/playtestenv/types.go` — states, options, results, failure
  categories, manifest records, and dependency types.
- `internal/playtestenv/manifest.go` — run reservation, legal transitions, and
  atomic manifest persistence.
- `internal/playtestenv/lock.go` — OS-native per-run advisory locking.
- `internal/playtestenv/command.go` — deadlock-safe subprocess execution.
- `internal/playtestenv/docker.go` — forced-local Docker command construction,
  context preflight, inspection, and label parsing.
- `internal/playtestenv/checkout.go` — checkout validation, ignore validation,
  path fingerprinting, and metadata-only Git baseline.
- `internal/playtestenv/compose.go` — embedded Compose policy, run control-file
  materialization, and project-scoped commands.
- `internal/playtestenv/compose.playtest.yml` — trusted server/volume/network
  policy embedded into the binary.
- `internal/playtestenv/lifecycle.go` — start, failure cleanup, and shared
  supervisor orchestration.
- `internal/playtestenv/readiness.go` — logs/process/port/TCP compound
  readiness.
- `internal/playtestenv/report.go` — non-secret failed-environment Markdown.
- `internal/playtestenv/operations.go` — status, logs, renew, and stop.
- `internal/playtestenv/reaper.go` — conservative expired-run discovery and
  deletion.

### New test files

- `cmd/playtestenv/main_test.go`
- `internal/playtestenv/manifest_test.go`
- `internal/playtestenv/lock_test.go`
- `internal/playtestenv/command_test.go`
- `internal/playtestenv/docker_test.go`
- `internal/playtestenv/checkout_test.go`
- `internal/playtestenv/compose_test.go`
- `internal/playtestenv/lifecycle_test.go`
- `internal/playtestenv/readiness_test.go`
- `internal/playtestenv/report_test.go`
- `internal/playtestenv/operations_test.go`
- `internal/playtestenv/reaper_test.go`
- `internal/playtestenv/integration_test.go`
- `.github/workflows/playtestenv-integration.yml` — Linux Docker integration
  gate for supervisor-related pull requests and manual runs.

### Existing files to modify

- `go.mod`, `go.sum` — add the latest `gofrs/flock` and `natefinch/atomic`.
- `docs/guides/TESTING_GUIDE.md` — local supervisor test commands, scope, and
  artifact/cleanup contract.
- `docs/superpowers/specs/2026-08-07-ephemeral-playtest-supervisor-design.md` —
  approved design and evidence-backed implementation clarifications.
- `docs/superpowers/plans/2026-08-07-ephemeral-playtest-supervisor.md` — this
  reviewed execution record.
- `docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md` — move 0.3a through
  implementation and review status; mark Done only after final evidence.

No `.gitignore` or `provisioning/Dockerfile.dockerignore` edit is expected.
Existing rules already ignore `tools/playtest/.run*/` and generated reports and
exclude both from Docker contexts. Task 3 proves this before relying on it.

## Core contracts

Define these contracts in Task 1 and keep later tasks consistent:

```go
type State string

const (
	StateValidating State = "validating"
	StateBuilding   State = "building"
	StateStarting   State = "starting"
	StateReady      State = "ready"
	StateStopping   State = "stopping"
	StateStopped    State = "stopped"
	StateFailed     State = "failed"
)

type Endpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type FailureCategory string

type FailureRecord struct {
	Category FailureCategory `json:"category"`
	Phase    State           `json:"phase"`
	Summary  string          `json:"summary"`
	Retryable bool           `json:"retryable,omitempty"`
}

type ResourceRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type CleanupResult struct {
	Complete  bool          `json:"complete"`
	Leftovers []ResourceRef `json:"leftovers,omitempty"`
	Summary   string        `json:"summary,omitempty"`
}

type ReadinessObservation struct {
	ContainerRunning bool      `json:"container_running"`
	ServerReady      bool      `json:"server_ready"`
	PanicSeen        bool      `json:"panic_seen"`
	ListenerError    bool      `json:"listener_error"`
	PortMappings     int       `json:"port_mappings"`
	Endpoint         *Endpoint `json:"endpoint,omitempty"`
	TCPConnected     bool      `json:"tcp_connected"`
	ObservedAt       time.Time `json:"observed_at"`
}

type ArtifactPaths struct {
	Manifest string `json:"manifest"`
	BuildLog string `json:"build_log"`
	ServerLog string `json:"server_log"`
	Compose  string `json:"compose"`
	Config   string `json:"config"`
	Report   string `json:"report,omitempty"`
}

type GitEntry struct {
	Status   string `json:"status"`
	Path     string `json:"path"`
	OrigPath string `json:"orig_path,omitempty"`
}

type GitBaseline struct {
	Commit  string     `json:"commit,omitempty"`
	Entries []GitEntry `json:"entries,omitempty"`
}

type StartOptions struct {
	Checkout         string
	Lease            time.Duration
	ReadinessTimeout time.Duration
}

type RunOptions struct {
	Checkout string
	RunID    string
}

type LogsOptions struct {
	Checkout string
	RunID    string
	Follow   bool
	Output   io.Writer
}

type RenewOptions struct {
	Checkout string
	RunID    string
	Lease    time.Duration
}

type Result struct {
	Operation    string          `json:"operation"`
	RunID       string          `json:"run_id,omitempty"`
	Project     string          `json:"project,omitempty"`
	State       State           `json:"state,omitempty"`
	Endpoint    *Endpoint       `json:"endpoint,omitempty"`
	Manifest    string          `json:"manifest,omitempty"`
	ServerLog   string          `json:"server_log,omitempty"`
	Report      string          `json:"report,omitempty"`
	Artifacts   *ArtifactPaths  `json:"artifacts,omitempty"`
	Cleanup     *CleanupResult  `json:"cleanup,omitempty"`
	Failure     *FailureRecord  `json:"failure,omitempty"`
}

type Supervisor struct { /* unexported injected dependencies */ }

func New() *Supervisor
func (s *Supervisor) Start(context.Context, StartOptions) (Result, error)
func (s *Supervisor) Status(context.Context, RunOptions) (Result, error)
func (s *Supervisor) Logs(context.Context, LogsOptions) (Result, error)
func (s *Supervisor) Renew(context.Context, RenewOptions) (Result, error)
func (s *Supervisor) Stop(context.Context, RunOptions) (Result, error)
func (s *Supervisor) Reap(context.Context, string) ([]Result, error)
```

Tests in package `playtestenv` may use an unexported `newSupervisor(deps)`
constructor. The CLI and other packages use only `New`.

---

### Task 1: Establish manifest, state, and lock foundations

**Files:**
- Create: `internal/playtestenv/types.go`
- Create: `internal/playtestenv/manifest.go`
- Create: `internal/playtestenv/manifest_test.go`
- Create: `internal/playtestenv/lock.go`
- Create: `internal/playtestenv/lock_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

- [x] **Step 1: Add the cross-platform advisory-lock dependency**

Run:

```powershell
go get github.com/gofrs/flock@latest github.com/natefinch/atomic@latest
```

Expected: exit `0`; `go.mod` and `go.sum` add the package selected by the Go
toolchain.

- [x] **Step 2: Write failing manifest and transition tests**

Cover:

```go
func TestReserveRunCreatesUniqueDirectoryAndInitialManifest(t *testing.T)
func TestReserveRunRetriesGeneratedIDCollision(t *testing.T)
func TestProjectNameDerivationIsComposeSafe(t *testing.T)
func TestWriteManifestUsesAtomicReplacement(t *testing.T)
func TestManifestRejectsIllegalTransitions(t *testing.T)
func TestManifestRoundTripPreservesEndpointFailureAndCleanup(t *testing.T)
```

Use a deterministic injected ID sequence (`collision`, then `fresh`) and clock.
Assert the first durable state is `validating`, schema version is `1`, and no
temporary file remains after replacement.

Expected first run:

```powershell
go test ./internal/playtestenv -run "TestReserveRun|TestProjectName|TestWriteManifest|TestManifest"
```

Expected: FAIL because the package/contracts do not exist.

- [x] **Step 3: Implement domain records and atomic manifest persistence**

`Manifest` must include:

```go
type Manifest struct {
	SchemaVersion       int                  `json:"schema_version"`
	RunID               string               `json:"run_id"`
	Project             string               `json:"project"`
	Checkout            string               `json:"checkout"`
	CheckoutFingerprint string               `json:"checkout_fingerprint"`
	State               State                `json:"state"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
	LeaseExpiresAt      time.Time            `json:"lease_expires_at"`
	Image               string               `json:"image"`
	Service             string               `json:"service"`
	ContainerID         string               `json:"container_id,omitempty"`
	Network             string               `json:"network"`
	Volume              string               `json:"volume"`
	Endpoint            *Endpoint            `json:"endpoint,omitempty"`
	Readiness           ReadinessObservation `json:"readiness"`
	Artifacts           ArtifactPaths        `json:"artifacts"`
	Git                 GitBaseline           `json:"git"`
	Failure             *FailureRecord        `json:"failure,omitempty"`
	Cleanup             *CleanupResult        `json:"cleanup,omitempty"`
}
```

Implement `reserveRun`, `readManifest`, `writeManifest`, and
`transitionManifest`. Marshal to bytes and pass a reader to
`atomic.WriteFile`, which writes a same-directory temporary file, syncs and
closes it, then uses a platform-specific atomic replace
(`MoveFileEx(REPLACE_EXISTING)` on Windows). Never implement overwrite as
remove-then-rename or reuse `util.SafeSave`, which uses a fixed `.new` path,
world-writable mode, and no file sync.

Legal state transitions:

```text
validating -> building | failed
building   -> starting | failed
starting   -> ready | failed
ready      -> stopping | failed
stopping   -> stopped | failed
failed     -> failed
stopped    -> stopped
```

- [x] **Step 4: Write failing OS-lock contention and release tests**

Cover same-process contention and a helper child process that acquires the lock
and exits without calling `Unlock`. The parent must acquire after child exit,
proving OS release rather than PID-file cleanup. A held lock with an expired
injected wait must return `ErrLockBusy`.

Expected first run:

```powershell
go test ./internal/playtestenv -run "TestRunLock"
```

Expected: FAIL because `runLock` is not implemented.

- [x] **Step 5: Implement the lock wrapper**

Use:

```go
type runLock struct {
	file *flock.Flock
}

func acquireRunLock(
	ctx context.Context,
	path string,
	wait time.Duration,
) (*runLock, error) {
	lockCtx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	f := flock.New(path)
	ok, err := f.TryLockContext(lockCtx, 50*time.Millisecond)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			return nil, ErrLockBusy
		}
		return nil, err
	}
	if !ok {
		return nil, ErrLockBusy
	}
	return &runLock{file: f}, nil
}

func (l *runLock) Close() error { return l.file.Unlock() }
```

Do not record or inspect PIDs and do not infer staleness from lock-file age.
Use named, injected wait durations: five seconds for direct run operations and
250 milliseconds per reaper candidate. Tests use millisecond waits. A held lock
must produce `ErrLockBusy`, never an unbounded wait.

- [x] **Step 6: Run focused tests and commit**

Run:

```powershell
gofmt -w internal/playtestenv/types.go internal/playtestenv/manifest.go internal/playtestenv/manifest_test.go internal/playtestenv/lock.go internal/playtestenv/lock_test.go
go test ./internal/playtestenv -run "TestReserveRun|TestProjectName|TestWriteManifest|TestManifest|TestRunLock"
```

Expected: PASS.

Stage only the files listed in this task and commit:

```text
feat(playtest): add run manifest and locking
```

---

### Task 2: Build the subprocess boundary and force local Docker

**Files:**
- Create: `internal/playtestenv/command.go`
- Create: `internal/playtestenv/command_test.go`
- Create: `internal/playtestenv/docker.go`
- Create: `internal/playtestenv/docker_test.go`

- [x] **Step 1: Write failing command-runner tests**

Define and test:

```go
type CommandSpec struct {
	Name   string
	Args   []string
	Dir    string
	Env    []string
	Stdout io.Writer
	Stderr io.Writer
}

type Runner interface {
	Run(context.Context, CommandSpec) error
}
```

Tests must start the test binary as a helper process that emits more than one
OS pipe buffer to stdout and stderr concurrently. Assert both streams complete
without deadlock. Also assert context cancellation terminates the child and
nonzero exit status remains inspectable through `ExitError`.

- [x] **Step 2: Run the command tests to verify RED**

Run:

```powershell
go test ./internal/playtestenv -run "TestExecRunner"
```

Expected: FAIL because `execRunner` is missing.

- [x] **Step 3: Implement deadlock-safe execution**

Use `exec.CommandContext`, assign stdout and stderr writers before `Start`, and
call `Wait` only after both streams are attached. Do not call `StdoutPipe` and
drain streams serially. Copy `os.Environ()` before filtering so the caller's
environment is never mutated.

- [x] **Step 4: Write failing local-context tests**

Cover:

```go
func TestDockerCommandAlwaysUsesValidatedLocalContext(t *testing.T)
func TestDockerContextRejectsDockerHostOverride(t *testing.T)
func TestDockerContextAcceptsLocalSelectedContext(t *testing.T)
func TestDockerCommandScrubsAmbientOverridesAfterResolution(t *testing.T)
func TestDockerPreflightAcceptsWindowsNamedPipe(t *testing.T)
func TestDockerPreflightAcceptsLinuxUnixSocket(t *testing.T)
func TestDockerPreflightRejectsTCPSSHAndMalformedEndpoints(t *testing.T)
```

After preflight, the fake runner must observe commands beginning:

```text
docker --context <validated-local-context> ...
```

and an environment without `DOCKER_HOST`, `DOCKER_CONTEXT`,
`DOCKER_TLS_VERIFY`, or `DOCKER_CERT_PATH`.

- [x] **Step 5: Implement Docker command construction and preflight**

Reject any non-empty `DOCKER_HOST` before invoking Docker; tell the caller to
select a named local Docker context instead. Resolve `DOCKER_CONTEXT` when
present, otherwise run `docker context show` with override variables scrubbed.
Then run:

```text
docker --context <candidate> context inspect <candidate>
  --format {{json .Endpoints.docker.Host}}
docker --context <candidate> compose version --short
docker --context <candidate> version --format {{.Server.Version}}
```

Decode the output as a JSON string. Accept only `npipe://` on Windows and
`unix://` on Linux. Reject `tcp://`, `ssh://`, empty, or platform-mismatched
endpoints before any build/create command. Require Compose
`>= 2.20.0` and a non-empty Docker server version. Inject platform and version
parsing in unit tests; accept either `2.20.0` or `v2.20.0` version text.

Keep the four scrubbed variables absent from every Docker/Compose child. Never
expose a Docker context parameter outside `docker.go`; all later commands use
the context name returned by this preflight. This accepts local rootless and
Docker Desktop named contexts while refusing remote transports.

- [x] **Step 6: Run focused tests and commit**

Run:

```powershell
gofmt -w internal/playtestenv/command.go internal/playtestenv/command_test.go internal/playtestenv/docker.go internal/playtestenv/docker_test.go
go test ./internal/playtestenv -run "TestExecRunner|TestDocker"
```

Expected: PASS.

Commit exact task files:

```text
feat(playtest): enforce local Docker execution
```

---

### Task 3: Validate checkouts and materialize the embedded Compose policy

**Files:**
- Create: `internal/playtestenv/checkout.go`
- Create: `internal/playtestenv/checkout_test.go`
- Create: `internal/playtestenv/compose.go`
- Create: `internal/playtestenv/compose_test.go`
- Create: `internal/playtestenv/compose.playtest.yml`

- [x] **Step 1: Write failing checkout tests**

Cover:

```go
func TestValidateCheckoutRequiresRepositoryRootGoModAndDockerfile(t *testing.T)
func TestValidateCheckoutNormalizesWindowsGitPath(t *testing.T)
func TestValidateCheckoutRejectsArchivePathComponent(t *testing.T)
func TestValidateCheckoutRequiresIgnoredRunAndReportPaths(t *testing.T)
func TestCheckoutFingerprintIsStableForCanonicalPath(t *testing.T)
func TestCheckoutVersionRequiresValidNonzeroStringLiteralVERSION(t *testing.T)
func TestGitBaselineStoresOnlyPathAndStatusMetadata(t *testing.T)
func TestGitBaselineExcludesCredentialsArchiveAndSupervisorArtifacts(t *testing.T)
func TestGitBaselineNeverInvokesDiffOrReadsFiles(t *testing.T)
```

Use a fake Git runner. Require `git --no-optional-locks` and only `rev-parse`,
`check-ignore`, and `status --short -z --untracked-files=all`. Include
rename/copy two-path records, spaces, Unicode, and malicious newline/Markdown
path text. The stored baseline must contain no patch, file bytes, password, or
credential path.

- [x] **Step 2: Implement checkout validation and baseline parsing**

Canonicalize with `filepath.Abs`, `filepath.EvalSymlinks`, and
`filepath.Clean`. Require `git rev-parse --show-toplevel` to resolve to that
same path after applying `filepath.FromSlash`, `filepath.EvalSymlinks`, and
`filepath.Clean` to Git's output. Compare with `strings.EqualFold` on Windows
and exact equality on Linux. Reject any canonical path component equal to
`_archive` case-insensitively.

Before reserving a run, probe these future files with `git check-ignore`:

```text
tools/playtest/.run/<probe>/manifest.json
tools/playtest/reports/<probe>-environment-failed.md
```

Run all Git reads with `git --no-optional-locks` so status does not refresh or
lock the user's index. Parse
`git status --short -z --untracked-files=all` without shell quoting.
Record status and sanitized relative path only. Filter
`tools/playtest/targets.yaml`, any `_archive` path component, basenames starting
`.env`, basenames containing `credential` or `secret` case-insensitively,
basenames ending `.key`, `tools/playtest/.run*/**`, and
`tools/playtest/reports/**`.

- [x] **Step 3: Write failing embedded-policy tests**

In `TestEmbeddedComposePolicy`, assert that production code reads the `go:embed`
value, not
`<checkout>/compose.playtest.yml`. Parse the embedded YAML with
`gopkg.in/yaml.v3` and assert:

- one `server` service;
- runner target and direct `/app/go-mud-server` entrypoint;
- no privileged, host network/PID/IPC, devices, Docker socket, or source bind;
- `restart: "no"`;
- explicit validated-checkout build context rather than Compose-file-relative
  `.`;
- only container port `55555` declared with `host_ip: 127.0.0.1` and no
  `published` value, letting Docker assign the host port;
- one `_datafiles` named volume;
- one writable bind of the run's ignored `control/` directory at
  `/run/dogmud`, with no bind of the checkout or run manifest directory;
- generated override pins the parsed checkout version and remains writable;
- `LOG_NOCOLOR=1` keeps readiness logs free of ANSI escapes;
- custom immutable labels on image, container, network, and volume; and
- no fixed container/network name.

Add two tests gated by `DOGMUD_COMPOSE_POLICY_TEST=1`:

- `TestDockerComposePolicyRenders` materializes the embedded policy, supplies
  representative manifest environment, and runs
  `docker --context <validated> compose ... config`; it asserts the rendered
  absolute build context, exact control bind, and sole dynamic-loopback
  declaration.
- `TestDockerDynamicLoopbackPublication` starts a tiny test-only BusyBox service
  using the same no-`published` port stanza, inspects
  `.NetworkSettings.Ports["55555/tcp"]`, requires one nonzero loopback host port,
  and removes the exact GUID-named project in `t.Cleanup`.
- `TestDockerContextExcludesSensitiveAndRunState` creates one exact ignored
  run-state sentinel with `t.Cleanup`, builds the real `builder` target under a
  GUID tag, and runs a shell assertion that `/src/tools/playtest/targets.yaml`,
  `/src/_archive`, and the sentinel are absent. It removes only that image tag.

- [x] **Step 4: Add the embedded Compose policy**

Use this shape, with complete labels repeated for build, service, network, and
volume:

```yaml
services:
  server:
    image: "dogmud-playtest:${DOGMUD_RUN_ID}"
    build:
      context: "${DOGMUD_CHECKOUT}"
      dockerfile: provisioning/Dockerfile
      target: runner
      labels: &run-labels
        dogmud.playtest.managed: "true"
        dogmud.playtest.run-id: "${DOGMUD_RUN_ID}"
        dogmud.playtest.project: "${DOGMUD_PROJECT}"
        dogmud.playtest.checkout: "${DOGMUD_CHECKOUT_FINGERPRINT}"
        dogmud.playtest.schema: "1"
        dogmud.playtest.created-at: "${DOGMUD_CREATED_AT}"
    labels: *run-labels
    restart: "no"
    entrypoint: ["/app/go-mud-server"]
    environment:
      CONFIG_PATH: "/run/dogmud/config-overrides.yaml"
      LOG_NOCOLOR: "1"
    volumes:
      - data:/app/_datafiles
      - type: bind
        source: "${DOGMUD_CONTROL_DIR}"
        target: /run/dogmud
    ports:
      - target: 55555
        host_ip: 127.0.0.1
        protocol: tcp

volumes:
  data:
    labels: *run-labels

networks:
  default:
    labels: *run-labels
```

If Compose rejects YAML anchors across these locations, repeat literal label
maps; do not weaken label coverage.

The preferred dynamic-port form omits `published`. If the Task 3 runtime probe
shows that a supported Compose version drops `host_ip` in that form, the only
sanctioned fallback is documented short syntax
`"127.0.0.1::55555"`, followed by the same real inspect assertion. Do not probe
a free port in user space and do not use undocumented `published: "0"`.

- [x] **Step 5: Implement control-file materialization**

Embed with:

```go
//go:embed compose.playtest.yml
var trustedCompose []byte
```

Write `compose.resolved.yml` from the embedded bytes and generate:

```yaml
Server:
  CurrentVersion: "<parsed checkout VERSION>"
Network:
  AIPort: 55555
Logging:
  LogToFile: false
```

as `control/config-overrides.yaml`. Parse the selected checkout's `main.go` with
the Go AST and require a string-literal package constant named `VERSION`; do not
use regex or a hard-coded version. Validate it with `internal/version.Parse` and
reject an invalid or zero version during checkout validation rather than
starting a misleading container. Pinning `Server.CurrentVersion` prevents the
ephemeral copy of current authored data from replaying all historical migrations
and backing itself up on every run. The writable control bind still allows
legitimate `configs.SetVal` calls during later admin playtests, but all writes
remain ignored and disposable.

Build the Compose environment from manifest values and validated checkout/control
paths only. Materialize both files before build; use absolute paths normalized
with `filepath.ToSlash` for Compose interpolation on Windows. Never read a
Compose file from the selected checkout.

All Compose commands use:

```text
docker --context <validated-local-context> compose
  --project-directory <checkout>
  -f <run>/compose.resolved.yml
  -p <project>
```

- [x] **Step 6: Verify repository ignore and Compose rendering**

Run:

```powershell
git check-ignore "tools/playtest/.run/probe/manifest.json"
git check-ignore "tools/playtest/reports/probe-environment-failed.md"
$env:DOGMUD_COMPOSE_POLICY_TEST = "1"
go test -v ./internal/playtestenv -run "^TestDocker(ComposePolicyRenders|DynamicLoopbackPublication|ContextExcludesSensitiveAndRunState)$" -timeout 30m
Remove-Item Env:\DOGMUD_COMPOSE_POLICY_TEST
```

Expected: both ignore checks exit `0`; Compose config exits `0`, contains only
the loopback AI publication, and resolves build context to the selected
checkout; Docker assigns a nonzero loopback port; and the builder image contains
neither the tracked playtest credential, archive, nor run sentinel. The tests
must use the production local-context and Compose-environment builders and leave
no test-owned container, network, volume, or image.

- [x] **Step 7: Run focused tests and commit**

Run:

```powershell
gofmt -w internal/playtestenv/checkout.go internal/playtestenv/checkout_test.go internal/playtestenv/compose.go internal/playtestenv/compose_test.go
go test ./internal/playtestenv -run "TestValidateCheckout|TestCheckout|TestGitBaseline|TestEmbedded|TestCompose"
```

Expected: PASS.

Commit exact task files:

```text
feat(playtest): materialize isolated compose runs
```

---

### Task 4: Implement startup, readiness, and failure evidence

**Files:**
- Create: `internal/playtestenv/lifecycle.go`
- Create: `internal/playtestenv/lifecycle_test.go`
- Create: `internal/playtestenv/readiness.go`
- Create: `internal/playtestenv/readiness_test.go`
- Create: `internal/playtestenv/report.go`
- Create: `internal/playtestenv/report_test.go`

- [x] **Step 1: Write failing startup-order and cancellation tests**

The fake runner records each event. Assert this strict order:

```text
validate checkout
reserve and lock run
write validating manifest
local Docker preflight
write control files
transition building
compose build server
transition starting
compose up --detach --no-build server
resolve container
readiness
transition ready
unlock
```

Also assert validation/manifest failures invoke no Docker command, and
cancellation during build or startup records `failed`, captures evidence,
cleans exact resources, and returns the original failure even when cleanup
succeeds. Every error after reservation must return a `Result` populated with
run ID, project, manifest, and known artifact paths so integration cleanup can
recover.

- [x] **Step 2: Write failing readiness tests**

Drive a table over:

- running + exact `Server Ready` + one loopback mapping + TCP success;
- stopped container;
- `panic:` and structured `PANIC`;
- `Error creating server`;
- missing/duplicate/malformed/non-loopback port mapping;
- TCP refusal;
- timeout; and
- container exit after an initially positive observation.

Use an injected dial function and clock/ticker. Require the final container
running check after TCP success before returning ready.

- [x] **Step 3: Implement compound readiness**

Resolve the container with `compose ps -q server`, then use structured
`docker inspect` JSON for state, labels, and:

```text
.NetworkSettings.Ports["55555/tcp"]
```

Require exactly one `{HostIp, HostPort}` entry. Normalize `0.0.0.0` as unsafe,
accept loopback only, parse port in `1..65535`, and probe with
`net.Dialer.DialContext`.

Poll logs through `docker logs <container-id>`. Detect exact `Server Ready`,
`Error creating server`, `panic:`, and structured `PANIC` events. The
90-second readiness timeout begins after `compose up` returns, not during build.

- [x] **Step 4: Write failing failure-report tests**

Cover every `FailureCategory`, hostile Markdown/path content, missing build or
server logs, cleanup success/failure, and report-name collision. Assert no
password, config body, production endpoint, or Git diff appears.

- [x] **Step 5: Implement start and failure reporting**

Use explicit categories:

```go
const (
	FailureInvalidCheckout      FailureCategory = "invalid_checkout"
	FailureDockerUnavailable    FailureCategory = "docker_unavailable"
	FailureBuild                FailureCategory = "build_failure"
	FailureContainerExited      FailureCategory = "container_exited"
	FailureBootPanic            FailureCategory = "boot_panic"
	FailureListenerCreation     FailureCategory = "listener_creation_failure"
	FailurePortPublication      FailureCategory = "port_publication_failure"
	FailureNonLoopback          FailureCategory = "non_loopback_publication"
	FailureReadinessTimeout     FailureCategory = "readiness_timeout"
	FailureConnectionProbe      FailureCategory = "connection_probe_failure"
	FailureManifest             FailureCategory = "manifest_failure"
	FailureCleanup              FailureCategory = "cleanup_failure"
	FailureLockBusy             FailureCategory = "lock_busy"
	FailureAbandonedRun         FailureCategory = "abandoned_run"
)
```

Map `ErrLockBusy` to retryable `FailureLockBusy` in every human/JSON operation
result.

Build output goes concurrently to terminal diagnostics and `build.log`.
Capture server logs and inspect evidence before cleanup. Write the Markdown
stub under `tools/playtest/reports/` with checkout, run, phase, category,
artifact paths, and cleanup result only.

Evidence capture and cleanup after cancellation use:

```go
cleanupBase := context.WithoutCancel(callerCtx)
cleanupCtx, cancelCleanup := context.WithTimeout(cleanupBase, 45*time.Second)
defer cancelCleanup()
```

Never pass the already-cancelled caller context to Docker logs, inspect, stop,
down, or image removal. Unit tests must assert cleanup commands receive a live,
bounded context.

Validation failures before the checkout's ignored report path and run identity
are proven safe return structured stderr/JSON only. They must not write a
manifest or report into an invalid checkout.

- [x] **Step 6: Run focused tests and commit**

Run:

```powershell
gofmt -w internal/playtestenv/lifecycle.go internal/playtestenv/lifecycle_test.go internal/playtestenv/readiness.go internal/playtestenv/readiness_test.go internal/playtestenv/report.go internal/playtestenv/report_test.go
go test ./internal/playtestenv -run "TestStart|TestReadiness|TestFailureReport"
```

Expected: PASS.

Commit exact task files:

```text
feat(playtest): supervise ephemeral server startup
```

---

### Task 5: Add status, logs, renewal, and idempotent stop

**Files:**
- Create: `internal/playtestenv/operations.go`
- Create: `internal/playtestenv/operations_test.go`

- [x] **Step 1: Write failing operation tests**

Cover:

```go
func TestStatusReportsManifestDockerAgreement(t *testing.T)
func TestStatusReportsMissingOrMismatchedResources(t *testing.T)
func TestLogsRefreshesStoredLog(t *testing.T)
func TestLogsFollowTeesToCallerAndServerLog(t *testing.T)
func TestRenewHoldsLockAndRejectsExpiredStoppedOrAmbiguousRun(t *testing.T)
func TestStopCapturesLogsUsesGraceThenForceAndRemovesExactResources(t *testing.T)
func TestStopRecoversAbandonedPreterminalRun(t *testing.T)
func TestStopFinishesCleanupAfterCallerCancellation(t *testing.T)
func TestStopIsIdempotentForStoppedAndCleanedFailedRuns(t *testing.T)
func TestStopIssuesNoPruneOrCacheDestroyingCommand(t *testing.T)
```

Use a blocking fake lock/runner to prove concurrent renew and stop cannot pass
the critical section together.
`TestStopRecoversAbandonedPreterminalRun` covers both `building` and `stopping`,
including a prior cleanup timeout with exact leftovers.

- [x] **Step 2: Implement live identity reconciliation**

Every operation:

1. canonicalizes and validates checkout;
2. resolves and validates the selected local Docker context;
3. validates run ID format `[a-z0-9-]+`;
4. acquires `<run>/.lock` using the bounded direct-operation wait;
5. reads manifest;
6. inspects live resources; and
7. requires every live resource label to match managed, run, project,
   checkout-fingerprint, schema, and creation-time values.

Status reports mismatch; renew and destructive operations reject it.
Tests assert no resource inspection or mutation occurs before context preflight
succeeds.

- [x] **Step 3: Implement logs and renew**

Non-follow logs replace `server.log` atomically from `docker logs`. Follow mode
uses `docker logs --follow`, tees to the caller and `server.log`, and returns on
context cancellation. JSON mode is rejected by the CLI before calling follow.

Renew must reread the manifest under lock, require `ready` and unexpired lease,
set `LeaseExpiresAt = now + requestedLease`, update `UpdatedAt`, and write
atomically. Docker labels remain unchanged.

- [x] **Step 4: Implement stop and cleanup**

Transition `ready` runs to `stopping`; a failed run remains `failed` while its
cleanup record is updated. A run abandoned in `validating`, `building`,
`starting`, or `stopping` first transitions legally to `failed` with
`FailureAbandonedRun`, then uses the failed-run cleanup path. Capture final logs.

Once the first destructive action begins, create a 45-second bounded context
from `context.WithoutCancel(callerCtx)` and use it for the complete sequence.
Caller cancellation before that point performs no mutation; cancellation after
that point cannot strand a half-removed run. If the validated container is
running, send exact-container SIGTERM:

```text
docker --context <validated-local-context> kill --signal=TERM <container-id>
```

Poll structured inspect for up to ten seconds. If it is still running, remove
only that validated ID with `docker rm --force`. Then run project-scoped:

```text
docker --context <validated-local-context> compose ... down
  --volumes --remove-orphans
docker --context <validated-local-context> image rm dogmud-playtest:<run-id>
```

Never run prune. Remove `control/` and `compose.resolved.yml` only after resource
cleanup and final evidence. If the writable bind left any root-owned file or
subdirectory that cannot be removed, record its exact host path as a
`cleanup_failure` leftover. Preserve manifest, logs, report, and BuildKit cache.

If the bounded cleanup context expires, synchronously record exact leftovers
and transition any nonterminal state to `failed` before returning. A repeated
`stop` can immediately resume cleanup; it does not wait for lease expiry.

If no live labelled resources remain and manifest is terminal (`stopped` or
`failed`), return success; failed remains the run's historical state. Record any
exact leftovers and return nonzero on cleanup failure.

- [x] **Step 5: Run focused/package tests and commit**

Run:

```powershell
gofmt -w internal/playtestenv/operations.go internal/playtestenv/operations_test.go
go test ./internal/playtestenv -run "TestStatus|TestLogs|TestRenew|TestStop"
go test ./internal/playtestenv
```

Expected: PASS.

Commit:

```text
feat(playtest): manage ephemeral server lifecycle
```

---

### Task 6: Implement conservative stale-run reaping

**Files:**
- Create: `internal/playtestenv/reaper.go`
- Create: `internal/playtestenv/reaper_test.go`

- [x] **Step 1: Write failing reaper safety tests**

Cover:

- active unexpired run untouched;
- expired run with complete matching manifest/labels removed;
- expired runs abandoned in `building` and `stopping` transitioned to failed,
  cleaned, and reported;
- lease renewed while reap waits for lock;
- final lease reread under lock preventing deletion;
- malformed manifest;
- absent manifest;
- partial labels;
- mismatched run/project/checkout/schema/creation labels;
- labelled decoy;
- resource from another checkout;
- lock acquisition failure; and
- one run's cleanup failure not broadening or hiding later diagnostics.

The fake runner must fail the test if any wildcard delete, unfiltered Docker
query, `docker system prune`, or resource name not sourced from the validated
manifest is used.

- [x] **Step 2: Implement candidate discovery**

Enumerate directories immediately under `tools/playtest/.run/`. Do not discover
deletion candidates from Docker alone. Resolve and validate the local Docker
context once before any candidate inspection. For each candidate:

1. validate run ID and manifest;
2. acquire its OS-native lock with the 250-millisecond reaper wait;
3. reread manifest;
4. require lease strictly before `now`;
5. inspect exact manifest resource identities and labels; and
6. transition an abandoned `validating`, `building`, `starting`, or `stopping`
   state to failed; and
7. call the same cleanup primitive used by `Stop`.

Manifest-less labelled Docker resources are diagnostic-only and remain
untouched. Once a candidate's first destructive action begins, use the same
`context.WithoutCancel` plus 45-second bounded cleanup context as `Stop`; caller
cancellation cannot interrupt a partially completed candidate cleanup. Later
candidates are not started after caller cancellation.

- [x] **Step 3: Implement Docker orphan diagnostics**

Use filtered queries only:

```text
docker --context <validated-local-context> ps -a
  --filter label=dogmud.playtest.managed=true
  --format {{json .}}
```

and equivalent image/network/volume queries. Compare returned run labels to
known manifests solely to report labelled resources that cannot be safely
reaped. Do not pass those discovered names to deletion.

- [x] **Step 4: Run focused/package tests and commit**

Run:

```powershell
gofmt -w internal/playtestenv/reaper.go internal/playtestenv/reaper_test.go
go test ./internal/playtestenv -run "TestReap"
go test ./internal/playtestenv
```

Expected: PASS.

Commit:

```text
feat(playtest): reap expired playtest servers safely
```

---

### Task 7: Add the agent-facing CLI and machine-readable output

**Files:**
- Create: `cmd/playtestenv/main.go`
- Create: `cmd/playtestenv/main_test.go`

- [x] **Step 1: Write failing CLI tests**

Test `run(ctx, args, stdout, stderr, supervisor)` directly with a fake
supervisor interface. Cover:

- every subcommand and default checkout;
- default two-hour lease and 90-second readiness timeout;
- required `--run`/`--lease` flags;
- unknown flags/subcommands;
- `logs --follow --json` rejection;
- lock-busy JSON rendering with `category: "lock_busy"` and
  `retryable: true`;
- JSON stdout containing exactly one object and no subprocess text;
- human output sent to stdout and diagnostics to stderr;
- context cancellation from `SIGINT`/`SIGTERM`;
- exit `0` success, `1` operation failure, and `2` usage failure; and
- absence/rejection of `--host`, `--target`, `--context`, `--compose-file`,
  source-mount, and export flags.

- [x] **Step 2: Run CLI tests to verify RED**

Run:

```powershell
go test ./cmd/playtestenv
```

Expected: FAIL because the command does not exist.

- [x] **Step 3: Implement subcommands with `flag.FlagSet`**

Support exactly:

```text
playtestenv start  [--checkout PATH] [--lease 2h] [--json]
playtestenv status --checkout PATH --run ID [--json]
playtestenv logs   --checkout PATH --run ID [--follow] [--json]
playtestenv renew  --checkout PATH --run ID --lease DURATION [--json]
playtestenv stop   --checkout PATH --run ID [--json]
playtestenv reap   [--checkout PATH] [--json]
```

Use `signal.NotifyContext(context.Background(), os.Interrupt,
syscall.SIGTERM)`. `main` calls `os.Exit(run(...))`; tests never invoke `main`.

Define a narrow CLI-facing interface matching the six Supervisor methods.
Render `Result` with `json.Encoder` in JSON mode. Errors must expose category,
run ID, manifest/log/report paths, and cleanup leftovers without secrets.

- [x] **Step 4: Run CLI and package tests; build the command**

Run:

```powershell
gofmt -w cmd/playtestenv/main.go cmd/playtestenv/main_test.go
go test ./cmd/playtestenv ./internal/playtestenv
go build ./cmd/playtestenv
```

Expected: all exit `0`.

- [x] **Step 5: Commit**

Commit exact command files:

```text
feat(playtest): add environment supervisor CLI
```

---

### Task 8: Add real Docker lifecycle integration and Linux CI

**Files:**
- Create: `internal/playtestenv/integration_test.go`
- Create: `.github/workflows/playtestenv-integration.yml`

- [x] **Step 1: Write an opt-in integration test harness**

At test entry:

```go
if os.Getenv("DOGMUD_PLAYTESTENV_INTEGRATION") != "1" {
	t.Skip("set DOGMUD_PLAYTESTENV_INTEGRATION=1")
}
```

Use `t.Cleanup` from the moment a run ID is reserved. Cleanup may call only
`Stop`/`Reap` with that run's checkout and ID; if they fail, print exact
leftovers and fail. Snapshot host Git status before each case and compare after,
excluding known ignored run/report artifacts. These test-side Git reads also
use `git --no-optional-locks`.

`Start` must return run ID, project, manifest, and artifact paths on every error
after reservation. Register `t.Cleanup` from that partial `Result` even when
`err != nil`; pre-reservation errors have no Docker resources by contract.

Test-only constructors may replace the embedded Compose bytes to induce
publication or forced-stop failures. They must be declared only in `_test.go`
and cannot be reached from `New` or the CLI.

- [x] **Step 2: Implement successful and concurrent real-Docker cases**

Cover in one cached test process:

1. start/status/log/renew/stop;
2. two simultaneous starts with distinct project, port, image, volume, network,
   manifest, and report identities;
3. two temporary detached Git worktrees: one modifies a tracked harmless web
   asset and the other adds a distinct untracked harmless probe under
   `_datafiles/html/public/static/`, proving each runtime image contains its own
   selected checkout state;
4. Docker-assigned loopback publication;
5. authored `_datafiles` copied into the volume;
6. `docker exec` mutation inside the volume followed by unchanged host file
   hash and Git status;
7. run image removed after stop; and
8. `control/` fully removed on Windows and Linux; and
9. a second build showing BuildKit cache reuse in build output.

Create each with `git worktree add --detach <exact-temp-path> HEAD`. Register
`git worktree remove --force <exact-temp-path>` in `t.Cleanup` before temporary
directory teardown. Record the exact administrative entry and assert it is gone;
do not use broad `git worktree prune`.

- [x] **Step 3: Implement real-Docker failure and reaper cases**

Cover:

- invalid Dockerfile build failure in a temporary worktree;
- malformed authored YAML causing pre-ready boot panic/exit;
- deliberately tiny readiness timeout;
- test-only no-port/non-loopback policy rejected;
- hostile `DOCKER_HOST` and a named remote-context fixture rejected before
  build, while a named local Unix/npipe context is honored;
- cancellation after container creation;
- graceful stop and a test-only PID-1 that ignores TERM, requiring exact force;
- repeated stop;
- expired run reaped;
- labelled decoy/ambiguous resource left untouched;
- build/server logs and failed-environment report retained; and
- no run container, network, volume, or image tag after each cleanup.

Create the remote context fixture under a GUID name with a reserved
documentation-only TCP address, register exact `docker context rm <guid>` in
`t.Cleanup` immediately, and never make a daemon call through it. Do not change
the user's active Docker context.

- [x] **Step 4: Run the integration suite on Windows Docker Desktop**

Run:

```powershell
$env:DOGMUD_PLAYTESTENV_INTEGRATION = "1"
go test -v -run "^TestDockerIntegration$" -timeout 30m ./internal/playtestenv
Remove-Item Env:\DOGMUD_PLAYTESTENV_INTEGRATION
```

Expected: PASS with two simultaneous ready endpoints, all intentional failure
cases classified, no race for host ports, and no test-owned Docker resources
remaining. Record cold and cached boot-to-ready durations; if either exceeds the
90-second readiness default, stop and revise the default with evidence rather
than weakening readiness. Assert writable `CONFIG_PATH` operations do not emit
migration/config-save errors.

- [x] **Step 5: Add the Linux integration workflow**

Create a workflow triggered by `workflow_dispatch` and pull requests touching
the supervisor, Dockerfile, Docker context filter, config, or workflow:

```yaml
name: Playtest Environment Integration

on:
  workflow_dispatch:
  pull_request:
    paths:
      - "cmd/playtestenv/**"
      - "internal/playtestenv/**"
      - "provisioning/Dockerfile"
      - "provisioning/Dockerfile.dockerignore"
      - "_datafiles/config.yaml"
      - "go.mod"
      - "go.sum"
      - ".github/workflows/playtestenv-integration.yml"

permissions:
  contents: read

jobs:
  docker-integration:
    runs-on: ubuntu-latest
    timeout-minutes: 35
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.mod
      - name: Show Docker versions
        run: docker version && docker compose version
      - name: Run supervisor integration
        env:
          DOGMUD_PLAYTESTENV_INTEGRATION: "1"
        run: go test -v -run "^TestDockerIntegration$" -timeout 30m ./internal/playtestenv
      - name: Build and boot ordinary production runner
        shell: bash
        run: |
          tag="dogmud-prod-smoke:${GITHUB_RUN_ID}"
          name="dogmud-prod-smoke-${GITHUB_RUN_ID}"
          cleanup() {
            docker rm -f "$name" >/dev/null 2>&1 || true
            docker image rm "$tag" >/dev/null 2>&1 || true
          }
          trap cleanup EXIT
          docker build -f provisioning/Dockerfile --target runner -t "$tag" .
          docker run -d --name "$name" "$tag"
          ready=0
          for _ in $(seq 1 45); do
            logs="$(docker logs "$name" 2>&1 || true)"
            if printf '%s' "$logs" | grep -Eq 'panic:|PANIC|Error creating server'; then
              printf '%s\n' "$logs"
              exit 1
            fi
            if printf '%s' "$logs" | grep -q 'Server Ready'; then
              ready=1
              break
            fi
            sleep 2
          done
          test "$ready" -eq 1
```

- [x] **Step 6: Commit**

Commit:

```text
test(playtest): cover Docker lifecycle integration
```

Do not claim Linux verification until the workflow has actually passed.

---

### Task 9: Document, verify, and adversarially review the implementation

**Files:**
- Create: `internal/playtestenv/context.md`
- Modify: `docs/guides/TESTING_GUIDE.md`
- Modify: `docs/superpowers/specs/2026-08-07-ephemeral-playtest-supervisor-design.md`
- Modify: `docs/superpowers/plans/2026-08-07-ephemeral-playtest-supervisor.md`
- Modify: `docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md`

- [x] **Step 1: Write `context.md` from verified symbols**

Document:

- package purpose and local-only boundary;
- one line per actual package file;
- exact public types and method signatures from `go doc`;
- lifecycle/state contract;
- immutable-label and manifest agreement;
- lock requirement and final lease reread;
- readiness evidence;
- artifact paths;
- no source mount/export/reset/commit behavior;
- Docker/Compose/Git dependencies; and
- CLI and future 0.3b–0.3d consumers.

Before writing, run:

```powershell
go doc ./internal/playtestenv
```

Do not document any symbol that command does not show.

- [x] **Step 2: Update the testing guide**

Add exact commands for:

```powershell
go test ./cmd/playtestenv ./internal/playtestenv
docker compose -f compose.test.yml run --build --rm test
$env:DOGMUD_PLAYTESTENV_INTEGRATION = "1"
go test -v -run "^TestDockerIntegration$" -timeout 30m ./internal/playtestenv
Remove-Item Env:\DOGMUD_PLAYTESTENV_INTEGRATION
```

Explain opt-in scope, Windows/Linux requirements, validated-local-context guard,
ignored artifacts, lease/renew/reap behavior, conservative non-deletion on
ambiguity, and how to inspect exact leftovers. State explicitly that this tool
cannot target production and runtime admin changes are discarded with the
volume. Document Compose `>= 2.20.0`, writable disposable control overrides,
and why the selected checkout's `VERSION` is pinned for ephemeral current-data
boots.

- [x] **Step 3: Run formatting, static, focused, and full-suite verification**

Run:

```powershell
gofmt -w cmd/playtestenv internal/playtestenv
go test ./cmd/playtestenv ./internal/playtestenv
go vet ./cmd/playtestenv ./internal/playtestenv
docker compose -f compose.test.yml run --build --rm test
```

Expected: every command exits `0`; Docker suite contains no failed package,
panic, or race report. Do not substitute the unreliable native-Windows
full-suite invocation for the canonical Chunk 0.2 Linux-container baseline.

- [x] **Step 4: Repeat real Docker verification and inspect residue**

Run the Task 8 Windows integration command again, then:

```powershell
if ($env:DOCKER_HOST) { throw "unset DOCKER_HOST and select a named local context" }
$localContext = (docker context show).Trim()
$dockerEndpoint = docker --context $localContext context inspect $localContext --format '{{json .Endpoints.docker.Host}}' | ConvertFrom-Json
if ($dockerEndpoint -notlike "npipe://*") { throw "Windows verification requires a local npipe Docker context" }
docker --context $localContext ps -a --filter "label=dogmud.playtest.managed=true"
docker --context $localContext network ls --filter "label=dogmud.playtest.managed=true"
docker --context $localContext volume ls --filter "label=dogmud.playtest.managed=true"
docker --context $localContext image ls --filter "label=dogmud.playtest.managed=true"
```

Expected: no test-owned run resources. Labelled decoys intentionally retained
by a test must be removed by that test's exact `t.Cleanup`, not a broad command.

- [x] **Step 5: Verify an ordinary production runner still boots**

Build without targeting the supervisor:

```powershell
if ($env:DOCKER_HOST) { throw "unset DOCKER_HOST and select a named local context" }
$localContext = (docker context show).Trim()
$dockerEndpoint = docker --context $localContext context inspect $localContext --format '{{json .Endpoints.docker.Host}}' | ConvertFrom-Json
if ($dockerEndpoint -notlike "npipe://*") { throw "Windows smoke requires a local npipe Docker context" }
$suffix = [guid]::NewGuid().ToString("N")
$smokeImage = "dogmud-prod-smoke:$suffix"
$smokeContainer = "dogmud-prod-smoke-$suffix"
$imageBuilt = $false
$containerCreated = $false
try {
    docker --context $localContext build -f provisioning/Dockerfile --target runner -t $smokeImage .
    if ($LASTEXITCODE -ne 0) { throw "production image build failed" }
    $imageBuilt = $true

    docker --context $localContext run -d --name $smokeContainer $smokeImage
    if ($LASTEXITCODE -ne 0) { throw "production container start failed" }
    $containerCreated = $true

    $deadline = (Get-Date).AddSeconds(90)
    do {
        $bootLog = docker --context $localContext logs $smokeContainer 2>&1 | Out-String
        # Use -cmatch: PowerShell -match is case-insensitive and false-positives
        # on MapConsistencyEnforce value=panic / mode="panic".
        if ($bootLog -cmatch 'panic:|Error creating server') {
            throw "production smoke emitted a panic or listener error`n$bootLog"
        }
        if ($bootLog -match "Server Ready") { break }
        Start-Sleep -Seconds 2
    } while ((Get-Date) -lt $deadline)
    if ($bootLog -notmatch "Server Ready") {
        throw "production smoke did not become ready`n$bootLog"
    }
} finally {
    if ($containerCreated) {
        docker --context $localContext stop --time 10 $smokeContainer
        docker --context $localContext rm -f $smokeContainer
    }
    if ($imageBuilt) {
        docker --context $localContext image rm $smokeImage
    }
}
```

Expected: clean boot and exact cleanup. Do not wipe shop, guild, moderation, or
other host persistence; the image uses its own data.

Evidence (2026-08-08): PASS on `desktop-linux` / `npipe:////./pipe/dockerDesktopLinuxEngine`;
`Server Ready` observed; smoke image/container removed in `finally`.

- [ ] **Step 6: Confirm Linux workflow evidence**

Push only when the user requests it. After a PR or manual workflow exists,
require the `Playtest Environment Integration` job to pass on Ubuntu. Record
its URL or run ID in implementation evidence. If no remote run is authorized,
run the same test on a documented native Linux environment and record command,
OS, Docker/Compose versions, and exit output; do not substitute Windows Docker
for the Linux-host requirement.

Deferred: no push/PR authorized in this session. Chunk 0.2 Linux-container
full suite PASS is recorded separately and does not satisfy this step.

- [x] **Step 7: Run an adversarial implementation review**

Give the reviewer:

- approved spec and this plan;
- complete branch diff against `master`;
- unit/full/Docker/production boot outputs;
- Windows and Linux evidence;
- current `git status`; and
- exact pre-existing unrelated-file inventory.

Mandate review of remote-context escape, Compose policy trust, command
injection, label/manifest ambiguity, renew/reap races, cancellation, Windows
locking/replacement, subprocess deadlock, secrets, source mutation, Docker
residue, and claimed test coverage. Resolve every blocking/important finding or
document why it is non-actionable with evidence.

Review evidence (2026-08-08, inline after subagent dispatch limit):
**Verdict: Approve with follow-ups.** No Blocking code findings.

| Severity | Area | Finding |
|----------|------|---------|
| Important (process) | Linux evidence | Step 6 still open: Ubuntu workflow not run without push/PR |
| Important (docs) | Prod smoke script | PowerShell `-match` + `PANIC` false-positived on `MapConsistencyEnforce=panic`; plan script corrected to `-cmatch 'panic:\|Error creating server'`. Bash workflow grep remains case-sensitive and OK |
| Non-actionable | Fingerprint | Fixed in `0d499c39a`; compose.test suite PASS |
| Non-actionable | Owned files | Branch commits match plan ownership; protected room/0.1 files remain unstaged |

Covered without Blocking gaps: DOCKER_HOST/context endpoint npipe\|unix gate, scrubbed compose env, embedded compose policy, identity mismatch / abandoned recovery + WithoutCancel cleanup, flock+atomic write, WaitDelay tee, failure-report escaping (paths/summaries only), checkout not mounted, residue filters empty after verification.

- [x] **Step 8: Update roadmap and commit docs**

Only after all verification and review pass, set Chunk 0.3a to `Done` and add a
dated evidence paragraph naming Windows/Linux integration, full race suite,
production boot, and final review.

The roadmap was clean when this feature branch was created; its current 0.3
decomposition/status diff is owned by this design cycle. Before staging, inspect
`git diff master -- docs/roadmaps/ADVERSARIAL_REVIEW_REMEDIATION_ROADMAP.md`.
If any unrelated delta appears, stop and ask the user instead of staging the
shared file.

Set this plan's status to implemented/reviewed and retain the approved spec.
Confirm the spec's Failure Evidence lists retryable `lock_busy` and
`abandoned_run`, and its Lifecycle section matches recovery from all four
abandoned nonterminal states.
Commit the exact guide, roadmap, spec, plan, and `context.md` paths:

```text
docs(playtest): document environment supervisor
```

- [x] **Step 9: Final owned-file and unrelated-work audit**

Run:

```powershell
git status --short
git diff --check
git diff --stat master...HEAD
git diff --name-only master...HEAD
```

Expected: only this plan's owned files appear in branch commits. The user's
pre-existing room files, adversarial report, and invalidated 0.1 documents
remain uncommitted and unchanged unless the user separately authorizes them.

## Suggested subagents

Keep execution sequential because Tasks 1–8 share package state and later
contracts depend on earlier ones.

- **Tasks 1–3 — `generalPurpose`, Claude Sonnet 5 Thinking High:** Manifest
  replacement, cross-platform locking, command execution, Git parsing, and
  embedded Compose policy are safety-sensitive and benefit from stronger
  reasoning. Ask the agent to verify direct source and current dependency APIs.
- **Tasks 4–6 — `generalPurpose`, Claude Sonnet 5 Thinking High:** Startup,
  readiness, cleanup, and reaping carry the highest destructive/race risk.
  Keep these with one strong implementation agent or three sequential fresh
  agents with explicit contract handoff.
- **Task 7 — `generalPurpose`, Composer 2.5 Fast:** The CLI is a narrow,
  mechanical adapter once the Supervisor API is stable; the cheaper model is
  sufficient, followed by parent review.
- **Task 8 — `generalPurpose`, Claude Sonnet 5 Thinking High:** Real Docker
  concurrency and failure injection require judgment and careful cleanup.
- **Task 9 documentation — `generalPurpose`, Composer 2.5 Fast:** This is
  mostly verified-symbol transcription and command documentation; parent must
  check every named API and test result.
- **Final adversarial review — `generalPurpose`, Claude Opus 5 Thinking High:**
  Use an independent fresh context for destructive-cleanup, concurrency,
  security-boundary, and evidence review. A stronger model is warranted here;
  do not ask an implementation agent to approve its own work.

Minimize fan-out by grouping adjacent tasks as above. Do not parallelize agents
that edit `internal/playtestenv` or run the real Docker integration suite.
