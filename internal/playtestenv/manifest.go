package playtestenv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/natefinch/atomic"
)

// manifestFileName is the basename of the persisted run manifest within a
// run directory.
const manifestFileName = "manifest.json"

// runsDirName is the checkout-relative directory holding every run's
// ignored, supervisor-owned artifacts.
const runsDirName = "tools/playtest/.run"

// manifestSchemaVersion is the current on-disk manifest schema version.
const manifestSchemaVersion = 1

// runIDPattern matches the required run ID grammar: lowercase ASCII letters,
// digits, and hyphens only.
var runIDPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// projectNamePattern matches Docker Compose's project-name grammar.
var projectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ErrRunIDInvalid is returned when a generated or supplied run ID does not
// match the required lowercase letters/digits/hyphens grammar.
var ErrRunIDInvalid = errors.New("playtestenv: run id must contain only lowercase ascii letters, digits, or hyphens")

// ErrIllegalTransition is returned when a requested manifest state
// transition is not part of the approved lifecycle state machine.
var ErrIllegalTransition = errors.New("playtestenv: illegal manifest state transition")

// legalTransitions is the approved lifecycle state machine. A state missing
// from this map, or a destination missing from its set, is illegal.
var legalTransitions = map[State]map[State]bool{
	StateValidating: {StateBuilding: true, StateFailed: true},
	StateBuilding:   {StateStarting: true, StateFailed: true},
	StateStarting:   {StateReady: true, StateFailed: true},
	StateReady:      {StateStopping: true, StateFailed: true},
	StateStopping:   {StateStopped: true, StateFailed: true},
	StateFailed:     {StateFailed: true},
	StateStopped:    {StateStopped: true},
}

// Manifest is the versioned, secret-free JSON record of one run's identity,
// lifecycle state, and evidence. It is persisted at
// tools/playtest/.run/<run-id>/manifest.json under the run's advisory lock.
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

// lockAcquireFunc matches acquireRunLock's signature. reserveRunWithDeps
// takes it as a parameter so failure-path tests can inject a deterministic
// lock-acquisition failure without depending on real OS lock contention.
type lockAcquireFunc func(ctx context.Context, path string, wait time.Duration) (*runLock, error)

// manifestWriteFunc matches writeManifest's signature. reserveRunWithDeps
// takes it as a parameter for the same reason as lockAcquireFunc: a
// deterministic, cross-platform way to inject a manifest-persistence
// failure in tests.
type manifestWriteFunc func(path string, m *Manifest) error

// reservation is the result of successfully reserving a new run: its unique
// directory has been created, its advisory lock is held, and its initial
// validating manifest has been durably persisted.
type reservation struct {
	RunID    string
	Project  string
	RunDir   string
	Lock     *runLock
	Manifest *Manifest
}

// isValidRunID reports whether id matches the required run ID grammar.
func isValidRunID(id string) bool {
	return id != "" && runIDPattern.MatchString(id)
}

// projectName derives a Compose-safe project name from a run ID. It is a
// pure, deterministic function: the same run ID always derives the same
// project name, and distinct run IDs never collide.
func projectName(runID string) string {
	name := "dogmud-playtest-" + runID
	if !projectNamePattern.MatchString(name) {
		// Defensive only: every caller is expected to pass an
		// already-validated run ID, so this branch should be unreachable.
		name = projectNamePattern.FindString(name)
	}
	return name
}

// reserveRun atomically creates a new, uniquely-named run directory under
// <checkout>/tools/playtest/.run/, acquires that run's advisory lock, and
// persists its initial schema-1 validating manifest before returning.
//
// genID is called once per attempt to generate a candidate run ID. If the
// candidate directory already exists, that ID is discarded - never reused -
// and a freshly generated ID is tried instead. now supplies the manifest's
// creation/update/lease timestamps so tests can inject a deterministic
// clock. lockWait bounds how long lock acquisition waits before returning
// ErrLockBusy.
func reserveRun(
	ctx context.Context,
	checkout string,
	lease time.Duration,
	lockWait time.Duration,
	now func() time.Time,
	genID func() (string, error),
) (*reservation, error) {
	return reserveRunWithDeps(ctx, checkout, lease, lockWait, now, genID, acquireRunLock, writeManifest)
}

// reserveRunWithDeps is reserveRun with its lock-acquisition and
// manifest-persistence steps injected as parameters. reserveRun calls it
// with the real acquireRunLock and writeManifest; tests call it directly
// with a failing fake to deterministically exercise the cleanup paths that
// run when either step fails partway through a reservation, without relying
// on real OS lock contention or platform-specific permission bits.
//
// If either step fails after runDir has already been created, runDir is
// removed again before returning: Task 6's reaper deliberately never
// touches a manifest-less run directory, so a leaked runDir here would
// become permanent garbage. Any already-acquired lock is closed before
// removal so the directory delete is never blocked by an open lock-file
// handle (required for correctness on Windows). The original operation
// error is preserved; if cleanup itself also fails, the returned error
// retains both facts. Cleanup removes only the exact runDir just created -
// never a wildcard or broader deletion.
func reserveRunWithDeps(
	ctx context.Context,
	checkout string,
	lease time.Duration,
	lockWait time.Duration,
	now func() time.Time,
	genID func() (string, error),
	acquireLock lockAcquireFunc,
	persistManifest manifestWriteFunc,
) (*reservation, error) {
	runsRoot := filepath.Join(checkout, filepath.FromSlash(runsDirName))
	if err := os.MkdirAll(runsRoot, 0o755); err != nil {
		return nil, fmt.Errorf("playtestenv: create runs root: %w", err)
	}

	const maxAttempts = 100
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		id, err := genID()
		if err != nil {
			return nil, fmt.Errorf("playtestenv: generate run id: %w", err)
		}
		if !isValidRunID(id) {
			return nil, fmt.Errorf("%w: %q", ErrRunIDInvalid, id)
		}

		runDir := filepath.Join(runsRoot, id)
		if err := os.Mkdir(runDir, 0o755); err != nil {
			if errors.Is(err, fs.ErrExist) {
				lastErr = err
				continue
			}
			return nil, fmt.Errorf("playtestenv: create run directory: %w", err)
		}

		lock, err := acquireLock(ctx, filepath.Join(runDir, ".lock"), lockWait)
		if err != nil {
			return nil, cleanupFailedReservation(runDir, nil, fmt.Errorf("playtestenv: acquire run lock: %w", err))
		}

		createdAt := now()
		project := projectName(id)
		manifestPath := filepath.Join(runDir, manifestFileName)
		manifest := &Manifest{
			SchemaVersion:  manifestSchemaVersion,
			RunID:          id,
			Project:        project,
			Checkout:       checkout,
			State:          StateValidating,
			CreatedAt:      createdAt,
			UpdatedAt:      createdAt,
			LeaseExpiresAt: createdAt.Add(lease),
			Artifacts: ArtifactPaths{
				Manifest: manifestPath,
			},
		}

		if err := persistManifest(manifestPath, manifest); err != nil {
			return nil, cleanupFailedReservation(runDir, lock, fmt.Errorf("playtestenv: write initial manifest: %w", err))
		}

		return &reservation{
			RunID:    id,
			Project:  project,
			RunDir:   runDir,
			Lock:     lock,
			Manifest: manifest,
		}, nil
	}

	return nil, fmt.Errorf("playtestenv: exhausted run id generation attempts: %w", lastErr)
}

// cleanupFailedReservation removes a run directory that was created by a
// reservation attempt which then failed before it could be returned to the
// caller. It closes lock first, if non-nil, so directory removal is never
// blocked by an open lock-file handle. It returns operationErr unchanged
// when cleanup succeeds; if removal itself fails, it returns an error
// wrapping both operationErr and the removal failure so neither fact is
// silently dropped. It removes only the exact runDir passed in.
func cleanupFailedReservation(runDir string, lock *runLock, operationErr error) error {
	if lock != nil {
		_ = lock.Close()
	}
	if removeErr := os.RemoveAll(runDir); removeErr != nil {
		return fmt.Errorf("%w (cleanup of %s also failed: %w)", operationErr, runDir, removeErr)
	}
	return operationErr
}

// readManifest loads and decodes the manifest at path.
func readManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("playtestenv: read manifest: %w", err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("playtestenv: decode manifest: %w", err)
	}
	return &m, nil
}

// writeManifest persists m to path using write-to-temporary-file-plus-atomic-
// replace semantics (natefinch/atomic.WriteFile), which syncs and closes the
// temporary file before an OS-level atomic rename/replace. It never removes
// the destination file first and never leaves a temporary file behind,
// whether creating path for the first time or replacing an existing file.
func writeManifest(path string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("playtestenv: encode manifest: %w", err)
	}
	if err := atomic.WriteFile(path, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("playtestenv: write manifest: %w", err)
	}
	return nil
}

// transitionManifest validates and applies a state transition in place. It
// mutates m.State and m.UpdatedAt only when the transition is legal; an
// illegal transition leaves m unchanged and returns an error wrapping
// ErrIllegalTransition. Persisting the mutated manifest is the caller's
// responsibility via writeManifest.
func transitionManifest(m *Manifest, next State, now time.Time) error {
	allowed, ok := legalTransitions[m.State]
	if !ok || !allowed[next] {
		return fmt.Errorf("%w: %s -> %s", ErrIllegalTransition, m.State, next)
	}
	m.State = next
	m.UpdatedAt = now
	return nil
}
