package playtestenv

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func sequenceIDSource(ids ...string) func() (string, error) {
	i := 0
	return func() (string, error) {
		id := ids[i]
		i++
		return id, nil
	}
}

// TestReserveRunCreatesUniqueDirectoryAndInitialManifest proves that
// reserving a run atomically creates the run directory under
// tools/playtest/.run/<id>, acquires that run's lock, and persists a schema 1
// validating manifest before returning.
func TestReserveRunCreatesUniqueDirectoryAndInitialManifest(t *testing.T) {
	checkout := t.TempDir()
	created := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	lease := 2 * time.Hour

	res, err := reserveRun(context.Background(), checkout, lease, time.Second, fixedClock(created), sequenceIDSource("fresh"))
	require.NoError(t, err)
	require.NotNil(t, res)
	t.Cleanup(func() { _ = res.Lock.Close() })

	require.Equal(t, "fresh", res.RunID)
	wantRunDir := filepath.Join(checkout, "tools", "playtest", ".run", "fresh")
	require.Equal(t, wantRunDir, res.RunDir)

	info, err := os.Stat(wantRunDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())

	require.NotNil(t, res.Manifest)
	require.Equal(t, 1, res.Manifest.SchemaVersion)
	require.Equal(t, "fresh", res.Manifest.RunID)
	require.Equal(t, res.Project, res.Manifest.Project)
	require.Equal(t, checkout, res.Manifest.Checkout)
	require.Equal(t, StateValidating, res.Manifest.State)
	require.True(t, created.Equal(res.Manifest.CreatedAt))
	require.True(t, created.Equal(res.Manifest.UpdatedAt))
	require.True(t, created.Add(lease).Equal(res.Manifest.LeaseExpiresAt))

	manifestPath := filepath.Join(wantRunDir, manifestFileName)
	onDisk, err := readManifest(manifestPath)
	require.NoError(t, err)
	require.Equal(t, res.Manifest.RunID, onDisk.RunID)
	require.Equal(t, res.Manifest.SchemaVersion, onDisk.SchemaVersion)
	require.Equal(t, res.Manifest.State, onDisk.State)
	require.True(t, res.Manifest.CreatedAt.Equal(onDisk.CreatedAt))
}

// TestReserveRunRetriesGeneratedIDCollision proves a generated run ID that
// collides with an existing run directory is discarded - never reused - and
// a freshly generated ID is retried instead.
func TestReserveRunRetriesGeneratedIDCollision(t *testing.T) {
	checkout := t.TempDir()
	runsRoot := filepath.Join(checkout, "tools", "playtest", ".run")
	collisionDir := filepath.Join(runsRoot, "collision")
	require.NoError(t, os.MkdirAll(collisionDir, 0o755))
	sentinel := filepath.Join(collisionDir, "sentinel.txt")
	require.NoError(t, os.WriteFile(sentinel, []byte("pre-existing"), 0o644))

	calls := 0
	genID := func() (string, error) {
		calls++
		if calls == 1 {
			return "collision", nil
		}
		return "fresh", nil
	}

	res, err := reserveRun(context.Background(), checkout, time.Hour, time.Second, fixedClock(time.Now()), genID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = res.Lock.Close() })

	require.Equal(t, 2, calls, "reserveRun must regenerate a new id rather than reuse the collided one")
	require.Equal(t, "fresh", res.RunID)

	// The pre-existing colliding directory and its contents must be
	// untouched: reserveRun never reuses an existing directory.
	data, err := os.ReadFile(sentinel)
	require.NoError(t, err)
	require.Equal(t, "pre-existing", string(data))
}

// TestReserveRunRemovesRunDirWhenLockAcquisitionFails proves that a run
// directory created by reserveRun is removed again if acquiring that run's
// advisory lock fails partway through reservation, rather than being left
// behind as permanent, manifest-less garbage (which the reaper deliberately
// never touches).
func TestReserveRunRemovesRunDirWhenLockAcquisitionFails(t *testing.T) {
	checkout := t.TempDir()
	errSimulatedLock := errors.New("simulated lock acquisition failure")
	failingAcquire := func(ctx context.Context, path string, wait time.Duration) (*runLock, error) {
		return nil, errSimulatedLock
	}

	res, err := reserveRunWithDeps(
		context.Background(), checkout, time.Hour, time.Second,
		fixedClock(time.Now()), sequenceIDSource("lock-fails"),
		failingAcquire, writeManifest,
	)
	require.Nil(t, res)
	require.Error(t, err)
	require.ErrorIs(t, err, errSimulatedLock)

	runDir := filepath.Join(checkout, "tools", "playtest", ".run", "lock-fails")
	_, statErr := os.Stat(runDir)
	require.True(t, os.IsNotExist(statErr),
		"runDir must be removed after a failed lock acquisition, got stat error: %v", statErr)
}

// TestReserveRunRemovesRunDirAndReleasesLockWhenManifestWriteFails proves
// that a run directory is removed, and its already-acquired advisory lock
// released first, when persisting the initial manifest fails partway
// through reservation. This test uses the real acquireRunLock (not an
// injected fake) so the lock file has a genuinely open OS handle at the
// moment of failure: on Windows, removing a directory while one of its
// files is still open fails with a sharing violation, so a successful
// RemoveAll here proves the lock was closed before removal on both
// platforms.
func TestReserveRunRemovesRunDirAndReleasesLockWhenManifestWriteFails(t *testing.T) {
	checkout := t.TempDir()
	errSimulatedWrite := errors.New("simulated manifest write failure")
	failingWrite := func(path string, m *Manifest) error {
		return errSimulatedWrite
	}

	res, err := reserveRunWithDeps(
		context.Background(), checkout, time.Hour, time.Second,
		fixedClock(time.Now()), sequenceIDSource("write-fails"),
		acquireRunLock, failingWrite,
	)
	require.Nil(t, res)
	require.Error(t, err)
	require.ErrorIs(t, err, errSimulatedWrite)

	runDir := filepath.Join(checkout, "tools", "playtest", ".run", "write-fails")
	_, statErr := os.Stat(runDir)
	require.True(t, os.IsNotExist(statErr),
		"runDir must be removed after a failed manifest write, got stat error: %v", statErr)
}

// TestProjectNameDerivationIsComposeSafe proves every derived Compose project
// name matches Docker Compose's project-name grammar regardless of the
// (already-validated) run ID's exact shape.
func TestProjectNameDerivationIsComposeSafe(t *testing.T) {
	cases := []string{
		"a",
		"fresh",
		"run-123",
		"0-leading-digit",
		"many-hyphens-in-a-row---ok",
	}

	seen := map[string]bool{}
	for _, runID := range cases {
		name := projectName(runID)
		require.Regexp(t, `^[a-z0-9][a-z0-9_-]*$`, name, "project name %q derived from run id %q is not Compose-safe", name, runID)
		require.False(t, seen[name], "project name %q collided across distinct run ids", name)
		seen[name] = true

		// Deterministic: deriving twice from the same run ID must agree.
		require.Equal(t, name, projectName(runID))
	}
}

// TestWriteManifestUsesAtomicReplacement proves manifest persistence replaces
// the target file atomically, leaving no temporary residue in the run
// directory, whether creating the file for the first time or replacing an
// existing one.
func TestWriteManifestUsesAtomicReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, manifestFileName)

	first := &Manifest{
		SchemaVersion: 1,
		RunID:         "run-a",
		Project:       "dogmud-playtest-run-a",
		Checkout:      dir,
		State:         StateValidating,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	require.NoError(t, writeManifest(path, first))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temporary files may remain after the initial write")
	require.Equal(t, manifestFileName, entries[0].Name())

	second := *first
	second.State = StateBuilding
	second.UpdatedAt = first.UpdatedAt.Add(time.Minute)
	require.NoError(t, writeManifest(path, &second))

	entries, err = os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "no temporary files may remain after replacing an existing manifest")
	require.Equal(t, manifestFileName, entries[0].Name())

	onDisk, err := readManifest(path)
	require.NoError(t, err)
	require.Equal(t, StateBuilding, onDisk.State)
}

// TestManifestRejectsIllegalTransitions proves transitionManifest enforces
// exactly the approved state machine and no other edge.
func TestManifestRejectsIllegalTransitions(t *testing.T) {
	allStates := []State{
		StateValidating, StateBuilding, StateStarting,
		StateReady, StateStopping, StateStopped, StateFailed,
	}

	legal := map[State]map[State]bool{
		StateValidating: {StateBuilding: true, StateFailed: true},
		StateBuilding:   {StateStarting: true, StateFailed: true},
		StateStarting:   {StateReady: true, StateFailed: true},
		StateReady:      {StateStopping: true, StateFailed: true},
		StateStopping:   {StateStopped: true, StateFailed: true},
		StateFailed:     {StateFailed: true},
		StateStopped:    {StateStopped: true},
	}

	for _, from := range allStates {
		for _, to := range allStates {
			m := &Manifest{State: from, UpdatedAt: time.Unix(0, 0)}
			now := time.Unix(100, 0)
			err := transitionManifest(m, to, now)

			if legal[from][to] {
				require.NoErrorf(t, err, "%s -> %s must be legal", from, to)
				require.Equal(t, to, m.State)
				require.True(t, now.Equal(m.UpdatedAt))
			} else {
				require.Errorf(t, err, "%s -> %s must be rejected", from, to)
				require.ErrorIs(t, err, ErrIllegalTransition)
				require.Equal(t, from, m.State, "rejected transition must not mutate state")
			}
		}
	}
}

// TestManifestRoundTripPreservesEndpointFailureAndCleanup proves the full
// manifest - including nested endpoint, readiness, artifact, git, failure,
// and cleanup records - survives a write/read round trip byte-for-byte in
// meaning.
func TestManifestRoundTripPreservesEndpointFailureAndCleanup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, manifestFileName)

	created := time.Date(2026, 8, 7, 9, 30, 0, 0, time.UTC)
	observed := created.Add(5 * time.Second)

	original := &Manifest{
		SchemaVersion:       1,
		RunID:               "run-b",
		Project:             "dogmud-playtest-run-b",
		Checkout:            dir,
		CheckoutFingerprint: "fingerprint-abc123",
		State:               StateFailed,
		CreatedAt:           created,
		UpdatedAt:           created.Add(time.Minute),
		LeaseExpiresAt:      created.Add(2 * time.Hour),
		Image:               "dogmud-playtest:run-b",
		Service:             "server",
		ContainerID:         "abc123containerid",
		Network:             "dogmud-playtest-run-b_default",
		Volume:              "dogmud-playtest-run-b_data",
		Endpoint: &Endpoint{
			Host: "127.0.0.1",
			Port: 54321,
		},
		Readiness: ReadinessObservation{
			ContainerRunning: true,
			ServerReady:      false,
			PanicSeen:        false,
			ListenerError:    true,
			PortMappings:     1,
			Endpoint: &Endpoint{
				Host: "127.0.0.1",
				Port: 54321,
			},
			TCPConnected: false,
			ObservedAt:   observed,
		},
		Artifacts: ArtifactPaths{
			Manifest:  path,
			BuildLog:  filepath.Join(dir, "build.log"),
			ServerLog: filepath.Join(dir, "server.log"),
			Compose:   filepath.Join(dir, "compose.resolved.yml"),
			Config:    filepath.Join(dir, "control", "config-overrides.yaml"),
			Report:    filepath.Join(dir, "report.md"),
		},
		Git: GitBaseline{
			Commit: "deadbeef",
			Entries: []GitEntry{
				{Status: "M", Path: "_datafiles/world/dogmud/rooms/thornwall_city/484.yaml"},
				{Status: "R100", Path: "new/path.yaml", OrigPath: "old/path.yaml"},
			},
		},
		Failure: &FailureRecord{
			Category:  FailureCategory("listener_creation_failure"),
			Phase:     StateStarting,
			Summary:   "Error creating server",
			Retryable: false,
		},
		Cleanup: &CleanupResult{
			Complete: false,
			Leftovers: []ResourceRef{
				{Kind: "container", ID: "abc123containerid"},
			},
			Summary: "graceful stop timed out",
		},
	}

	require.NoError(t, writeManifest(path, original))

	roundTripped, err := readManifest(path)
	require.NoError(t, err)

	require.Equal(t, original.SchemaVersion, roundTripped.SchemaVersion)
	require.Equal(t, original.RunID, roundTripped.RunID)
	require.Equal(t, original.Project, roundTripped.Project)
	require.Equal(t, original.Checkout, roundTripped.Checkout)
	require.Equal(t, original.CheckoutFingerprint, roundTripped.CheckoutFingerprint)
	require.Equal(t, original.State, roundTripped.State)
	require.True(t, original.CreatedAt.Equal(roundTripped.CreatedAt))
	require.True(t, original.UpdatedAt.Equal(roundTripped.UpdatedAt))
	require.True(t, original.LeaseExpiresAt.Equal(roundTripped.LeaseExpiresAt))
	require.Equal(t, original.Image, roundTripped.Image)
	require.Equal(t, original.Service, roundTripped.Service)
	require.Equal(t, original.ContainerID, roundTripped.ContainerID)
	require.Equal(t, original.Network, roundTripped.Network)
	require.Equal(t, original.Volume, roundTripped.Volume)

	require.NotNil(t, roundTripped.Endpoint)
	require.Equal(t, *original.Endpoint, *roundTripped.Endpoint)

	require.Equal(t, original.Readiness.ContainerRunning, roundTripped.Readiness.ContainerRunning)
	require.Equal(t, original.Readiness.ListenerError, roundTripped.Readiness.ListenerError)
	require.Equal(t, original.Readiness.PortMappings, roundTripped.Readiness.PortMappings)
	require.NotNil(t, roundTripped.Readiness.Endpoint)
	require.Equal(t, *original.Readiness.Endpoint, *roundTripped.Readiness.Endpoint)
	require.True(t, original.Readiness.ObservedAt.Equal(roundTripped.Readiness.ObservedAt))

	require.Equal(t, original.Artifacts, roundTripped.Artifacts)
	require.Equal(t, original.Git, roundTripped.Git)

	require.NotNil(t, roundTripped.Failure)
	require.Equal(t, *original.Failure, *roundTripped.Failure)

	require.NotNil(t, roundTripped.Cleanup)
	require.Equal(t, *original.Cleanup, *roundTripped.Cleanup)

	// The persisted bytes must be valid JSON with the documented schema
	// version, independent of our own struct decoding.
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var generic map[string]any
	require.NoError(t, json.Unmarshal(raw, &generic))
	require.Equal(t, float64(1), generic["schema_version"])
}
