# Playtest Environment Supervisor Context

## Purpose

`internal/playtestenv` is the local-only ephemeral Docker playtest supervisor.
It turns one selected DOGMud checkout into one isolated, lease-bound, verified
Docker server endpoint and cleans up only that run's resources.

It does **not** create players, drive `mudagent`, evaluate goals, accept a
remote/production target, mount the checkout into the server, export a volume
back into source, or commit anything. Runtime admin/builder mutations stay in a
disposable volume that is discarded with the run.

## Files

- **types.go** — lifecycle `State`, options, `Result`, failure categories,
  artifact/git records, and other shared domain types.
- **manifest.go** — run reservation, legal state transitions, and atomic
  manifest persistence under the run advisory lock.
- **lock.go** — OS-native per-run advisory locking.
- **command.go** — `CommandSpec`, `Runner`, and deadlock-safe subprocess
  execution (`ExitError`).
- **docker.go** — forced-local Docker context preflight, Compose version gate,
  and context-scoped command construction.
- **checkout.go** — checkout validation, ignore checks, path fingerprinting,
  VERSION pin, and metadata-only Git baseline.
- **compose.go** — embedded Compose policy materialization, writable control
  overrides, and project-scoped Compose commands.
- **compose.playtest.yml** — trusted server/volume/network policy embedded into
  the binary (`EmbeddedComposePolicy`).
- **lifecycle.go** — `Supervisor`, `New`, and `Start` orchestration with
  failure cleanup.
- **readiness.go** — logs/process/port/TCP compound readiness.
- **report.go** — non-secret failed-environment Markdown reports.
- **operations.go** — `Status`, `Logs`, `Renew`, and `Stop`.
- **reaper.go** — conservative expired-run discovery and deletion diagnostics.
- **\*_test.go** — unit and opt-in Docker integration coverage.

## Core Types

```go
type State string
// validating | building | starting | ready | stopping | stopped | failed

type Endpoint struct {
    Host string `json:"host"`
    Port int    `json:"port"`
}

type FailureCategory string
// invalid_checkout | docker_unavailable | build_failure | container_exited |
// boot_panic | listener_creation_failure | port_publication_failure |
// non_loopback_publication | readiness_timeout | connection_probe_failure |
// manifest_failure | cleanup_failure | lock_busy | abandoned_run |
// identity_mismatch

type FailureRecord struct {
    Category  FailureCategory `json:"category"`
    Phase     State           `json:"phase"`
    Summary   string          `json:"summary"`
    Retryable bool            `json:"retryable,omitempty"`
}

type ArtifactPaths struct {
    Manifest  string `json:"manifest"`
    BuildLog  string `json:"build_log"`
    ServerLog string `json:"server_log"`
    Inspect   string `json:"inspect,omitempty"`
    Compose   string `json:"compose"`
    Config    string `json:"config"`
    Creds     string `json:"creds,omitempty"` // host path; body written by server
    Report    string `json:"report,omitempty"`
}

type CleanupResult struct {
    Complete  bool          `json:"complete"`
    Leftovers []ResourceRef `json:"leftovers,omitempty"`
    Summary   string        `json:"summary,omitempty"`
}

type ResourceRef struct {
    Kind string `json:"kind"`
    ID   string `json:"id"`
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
    Git                 GitBaseline          `json:"git"`
    Failure             *FailureRecord       `json:"failure,omitempty"`
    Cleanup             *CleanupResult       `json:"cleanup,omitempty"`
}

type Result struct {
    Operation string         `json:"operation"`
    RunID     string         `json:"run_id,omitempty"`
    Project   string         `json:"project,omitempty"`
    State     State          `json:"state,omitempty"`
    Endpoint  *Endpoint      `json:"endpoint,omitempty"`
    Manifest  string         `json:"manifest,omitempty"`
    ServerLog string         `json:"server_log,omitempty"`
    Report    string         `json:"report,omitempty"`
    Artifacts *ArtifactPaths `json:"artifacts,omitempty"`
    Cleanup   *CleanupResult `json:"cleanup,omitempty"`
    Failure   *FailureRecord `json:"failure,omitempty"`
}

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

`Manifest` is persisted at `tools/playtest/.run/<run-id>/manifest.json` under
the run's advisory lock. Immutable Docker labels must agree with the manifest
identity before destructive cleanup.

## Public API

### Construction

```go
func New() *Supervisor
func EmbeddedComposePolicy() []byte
```

`New` returns a `Supervisor` with production dependencies.
`EmbeddedComposePolicy` returns a copy of the embedded Compose policy for
test/inspection only.

### Lifecycle operations

```go
func (s *Supervisor) Start(ctx context.Context, opts StartOptions) (Result, error)
func (s *Supervisor) Status(ctx context.Context, opts RunOptions) (Result, error)
func (s *Supervisor) Logs(ctx context.Context, opts LogsOptions) (Result, error)
func (s *Supervisor) Renew(ctx context.Context, opts RenewOptions) (Result, error)
func (s *Supervisor) Stop(ctx context.Context, opts RunOptions) (Result, error)
func (s *Supervisor) Reap(ctx context.Context, checkoutPath string) ([]Result, error)
```

Option types:

```go
type StartOptions struct {
    Checkout         string
    Lease            time.Duration
    ReadinessTimeout time.Duration
    Profiles         []ProfileRequest // 0.3b: explicit synthetic profiles
}

type ProfileRequest struct {
    Profile   string
    StartRoom int
    Overlays  ProfileOverlays
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
```

Defaults: `DefaultLease` (2h), `DefaultReadinessTimeout` (90s),
`CleanupTimeout` (45s), `DefaultLockWait` (5s), `ReaperLockWait` (250ms).

### Synthetic profiles (0.3b)

When `StartOptions.Profiles` is non-empty, `Start` writes
`control/profiles-manifest.yaml`, sets `Playtest.ProfilesDir` /
`Playtest.ProfilesManifest` in `config-overrides.yaml`, and after ready
surfaces `Artifacts.Creds` as the host path to `control/creds.json`
(written by `internal/playtestprofiles`). Empty/omitted `Profiles` is
creation-flow (no manifest override). Failure reports may list the Creds
**path** only — never embed `creds.json` bodies. Runner image must
`COPY tools/playtest/profiles` → `/app/playtest/profiles`. CLI:
`playtestenv start --profile id:start_room` (repeatable).

### Errors and exit wrapping

Exported sentinel errors include checkout validation failures, Docker/Compose
local-context guards (`ErrDockerHostOverride`, `ErrDockerContextNotLocal`,
`ErrComposeVersionTooOld`), `ErrLockBusy`, `ErrLeaseExpired`, `ErrRunNotReady`,
`ErrIllegalTransition`, `ErrResourceIdentityMismatch`, and related run-ID /
control-directory errors. See `go doc ./internal/playtestenv`.

```go
type ExitError struct {
    Name     string
    Args     []string
    ExitCode int
    // unexported underlying *exec.ExitError
}
func (e *ExitError) Error() string
func (e *ExitError) Unwrap() error
```

## Gotchas

- **Local-only.** Nonempty `DOCKER_HOST` is rejected. The selected named Docker
  context must resolve to a local transport (`npipe://` on Windows, `unix://`
  on Linux). There is no remote-target, host, Compose-file, or context override
  API.
- **Compose >= 2.20.0** is required (`ErrComposeVersionTooOld` otherwise).
- **Checkout VERSION pin.** `main.go` must declare a package-level string
  `VERSION` constant; the supervisor writes that value into disposable control
  overrides so the ephemeral image boots current-data for that version.
- **Writable control dir.** Run control overrides must be writable
  (`ErrControlDirNotWritable`); they are not source mounts.
- **Lock then reread.** Operations acquire the run advisory lock
  (`ErrLockBusy` / `lock_busy` when contended) and re-read the final lease and
  identity under that lock before mutating or deleting.
- **Identity agreement.** Cleanup/reap only delete resources whose live labels
  unambiguously match the manifest. Ambiguous leftovers are reported, never
  broadly deleted.
- **Abandoned recovery.** `Stop`/`Reap` treat manifests abandoned in
  `validating`, `building`, `starting`, or `stopping` as `abandoned_run`, then
  resume exact-resource cleanup.
- **No source mutation path.** The checkout is never mounted; disposable volume
  state cannot be exported or committed through this package.
- **Opt-in integration.** Real Docker tests require
  `DOGMUD_PLAYTESTENV_INTEGRATION=1` and a long timeout; unit tests alone do not
  exercise the production runner image.

## Dependencies

- Docker Engine (Linux containers) + Docker Compose v2 (>= 2.20.0)
- Git (checkout root / ignore / status baseline; metadata only)
- `github.com/gofrs/flock`, `github.com/natefinch/atomic`, `gopkg.in/yaml.v3`

## Consumers

- `cmd/playtestenv` — thin CLI over `Supervisor`
- Future Chunks 0.3b–0.3d — consume the verified loopback AI `Endpoint` and
  lifecycle commands; they must not accept production/`targets.yaml` prod
  endpoints through this boundary
