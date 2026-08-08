package playtestenv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const opsTestCtx = "desktop-linux"

func opsFingerprint(canonical string) string {
	return checkoutFingerprint(canonical, runtime.GOOS)
}

func labelledInspectJSON(running bool, hostIP, hostPort string, labels map[string]string) string {
	type mapping struct {
		HostIp   string `json:"HostIp"`
		HostPort string `json:"HostPort"`
	}
	ports := map[string][]mapping{}
	if hostIP != "" || hostPort != "" {
		ports["55555/tcp"] = []mapping{{HostIp: hostIP, HostPort: hostPort}}
	}
	doc := map[string]any{
		"Id": "sha256:deadbeef",
		"State": map[string]any{
			"Running": running,
			"Status":  map[bool]string{true: "running", false: "exited"}[running],
		},
		"Config": map[string]any{
			"Labels": labels,
		},
		"NetworkSettings": map[string]any{
			"Ports": ports,
		},
		"Labels": labels,
	}
	b, _ := json.Marshal([]any{doc})
	return string(b)
}

func identityLabels(m *Manifest) map[string]string {
	return map[string]string{
		labelManaged:   labelManagedValue,
		labelRunID:     m.RunID,
		labelProject:   m.Project,
		labelCheckout:  m.CheckoutFingerprint,
		labelSchema:    labelSchemaValue,
		labelCreatedAt: m.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type seededRun struct {
	Checkout  string
	Canonical string
	RunID     string
	RunDir    string
	Manifest  *Manifest
	DC        dockerContext
	Now       time.Time
}

func seedRun(t *testing.T, runID string, state State, withResources bool) seededRun {
	t.Helper()
	checkout := t.TempDir()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
	writeMinimalCheckout(t, checkout, `const VERSION = "1.2.3"`)

	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	fp := opsFingerprint(canonical)
	project := projectName(runID)
	runDir := filepath.Join(canonical, filepath.FromSlash(runsDirName), runID)
	require.NoError(t, os.MkdirAll(filepath.Join(runDir, controlDirName), 0o755))
	composePath := filepath.Join(runDir, composeResolvedFileName)
	require.NoError(t, os.WriteFile(composePath, EmbeddedComposePolicy(), 0o644))
	configPath := filepath.Join(runDir, controlDirName, configOverridesFileName)
	require.NoError(t, os.WriteFile(configPath, []byte("Network:\n  AIPort: 55555\n"), 0o644))

	m := &Manifest{
		SchemaVersion:       manifestSchemaVersion,
		RunID:               runID,
		Project:             project,
		Checkout:            canonical,
		CheckoutFingerprint: fp,
		State:               state,
		CreatedAt:           now,
		UpdatedAt:           now,
		LeaseExpiresAt:      now.Add(2 * time.Hour),
		Image:               imageNamePrefix + runID,
		Service:             serviceName,
		Network:             project + "_default",
		Volume:              project + "_data",
		Artifacts: ArtifactPaths{
			Manifest:  filepath.Join(runDir, manifestFileName),
			BuildLog:  filepath.Join(runDir, buildLogName),
			ServerLog: filepath.Join(runDir, serverLogName),
			Inspect:   filepath.Join(runDir, inspectLogName),
			Compose:   composePath,
			Config:    configPath,
		},
	}
	if withResources {
		m.ContainerID = "cid-" + runID
		m.Endpoint = &Endpoint{Host: "127.0.0.1", Port: 54321}
	}
	require.NoError(t, writeManifest(m.Artifacts.Manifest, m))
	require.NoError(t, os.WriteFile(m.Artifacts.ServerLog, []byte("stale log\n"), 0o644))

	return seededRun{
		Checkout:  checkout,
		Canonical: canonical,
		RunID:     runID,
		RunDir:    runDir,
		Manifest:  m,
		DC:        dockerContext{name: opsTestCtx, env: []string{"PATH=/usr/bin"}},
		Now:       now,
	}
}

func newOpsSupervisor(t *testing.T, seed seededRun, runner *lifecycleRunner, clock *readinessFakeClock) *Supervisor {
	t.Helper()
	scriptGitValidateAndBaseline(runner, seed.Canonical)
	nowFn := func() time.Time { return seed.Now }
	afterFn := time.After
	if clock != nil {
		nowFn = clock.Now
		afterFn = clock.After
	}
	return newSupervisor(supervisorDeps{
		runner: runner,
		now:    nowFn,
		after:  afterFn,
		dial:   (&fakeDialer{}).DialContext,
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			return seed.DC, nil
		},
		lockWait: 200 * time.Millisecond,
	})
}

func scriptMatchingResources(r *lifecycleRunner, m *Manifest, containerRunning bool) {
	labels := identityLabels(m)
	containerJSON := labelledInspectJSON(containerRunning, "127.0.0.1", "54321", labels)
	otherJSON := labelledInspectJSON(false, "", "", labels)

	r.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		switch {
		case strings.Contains(joined, m.ContainerID):
			writeStdout(spec, containerJSON+"\n")
		case strings.Contains(joined, m.Network),
			strings.Contains(joined, m.Volume),
			strings.Contains(joined, m.Image):
			writeStdout(spec, otherJSON+"\n")
		default:
			return fmt.Errorf("unexpected inspect target: %s", joined)
		}
		return nil
	})
}

func TestStatusReportsManifestDockerAgreement(t *testing.T) {
	seed := seedRun(t, "run-status-ok", StateReady, true)
	runner := newLifecycleRunner(seed.Canonical)
	scriptMatchingResources(runner, seed.Manifest, true)
	var sawResourceInspectBeforeDocker bool
	resolveCalls := 0
	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    func() time.Time { return seed.Now },
		after:  time.After,
		dial:   (&fakeDialer{}).DialContext,
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			resolveCalls++
			// No inspect yet before preflight returns.
			for _, c := range runner.Calls() {
				joined := strings.Join(c.Spec.Args, " ")
				if strings.Contains(joined, "inspect") {
					sawResourceInspectBeforeDocker = true
				}
			}
			return seed.DC, nil
		},
		lockWait: time.Second,
	})
	scriptGitValidateAndBaseline(runner, seed.Canonical)

	res, err := s.Status(context.Background(), RunOptions{Checkout: seed.Checkout, RunID: seed.RunID})
	require.NoError(t, err)
	require.Equal(t, "status", res.Operation)
	require.Equal(t, seed.RunID, res.RunID)
	require.Equal(t, StateReady, res.State)
	require.NotNil(t, res.Endpoint)
	require.Equal(t, 54321, res.Endpoint.Port)
	require.Nil(t, res.Failure)
	require.False(t, sawResourceInspectBeforeDocker)
	require.Equal(t, 1, resolveCalls)
	require.NotEmpty(t, runner.Calls())
	var sawInspect bool
	for _, c := range runner.Calls() {
		if strings.Contains(strings.Join(c.Spec.Args, " "), "inspect") {
			sawInspect = true
			break
		}
	}
	require.True(t, sawInspect, "status must inspect live resources after preflight")
}

func TestStatusReportsMissingOrMismatchedResources(t *testing.T) {
	t.Run("missing container", func(t *testing.T) {
		seed := seedRun(t, "run-status-missing", StateReady, true)
		runner := newLifecycleRunner(seed.Canonical)
		scriptGitValidateAndBaseline(runner, seed.Canonical)
		labels := identityLabels(seed.Manifest)
		otherJSON := labelledInspectJSON(false, "", "", labels)
		runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
			joined := strings.Join(spec.Args, " ")
			if strings.Contains(joined, seed.Manifest.ContainerID) {
				return &ExitError{Name: "docker", Args: spec.Args, ExitCode: 1}
			}
			writeStdout(spec, otherJSON+"\n")
			return nil
		})
		s := newOpsSupervisor(t, seed, runner, nil)

		res, err := s.Status(context.Background(), RunOptions{Checkout: seed.Checkout, RunID: seed.RunID})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrResourceIdentityMismatch)
		require.Equal(t, "status", res.Operation)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureIdentityMismatch, res.Failure.Category)
		require.Contains(t, strings.ToLower(res.Failure.Summary), "missing")
	})

	t.Run("mismatched labels", func(t *testing.T) {
		seed := seedRun(t, "run-status-mismatch", StateReady, true)
		runner := newLifecycleRunner(seed.Canonical)
		scriptGitValidateAndBaseline(runner, seed.Canonical)
		bad := identityLabels(seed.Manifest)
		bad[labelCheckout] = "other-fingerprint"
		badJSON := labelledInspectJSON(true, "127.0.0.1", "54321", bad)
		goodJSON := labelledInspectJSON(false, "", "", identityLabels(seed.Manifest))
		runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
			joined := strings.Join(spec.Args, " ")
			if strings.Contains(joined, seed.Manifest.ContainerID) {
				writeStdout(spec, badJSON+"\n")
				return nil
			}
			writeStdout(spec, goodJSON+"\n")
			return nil
		})
		s := newOpsSupervisor(t, seed, runner, nil)

		res, err := s.Status(context.Background(), RunOptions{Checkout: seed.Checkout, RunID: seed.RunID})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrResourceIdentityMismatch)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureIdentityMismatch, res.Failure.Category)
		require.Contains(t, strings.ToLower(res.Failure.Summary), "mismatch")
	})

	t.Run("no resource inspect before failed docker preflight", func(t *testing.T) {
		seed := seedRun(t, "run-status-nodocker", StateReady, true)
		runner := newLifecycleRunner(seed.Canonical)
		scriptGitValidateAndBaseline(runner, seed.Canonical)
		runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
			t.Fatal("inspect must not run before docker preflight succeeds")
			return nil
		})
		s := newSupervisor(supervisorDeps{
			runner: runner,
			now:    func() time.Time { return seed.Now },
			after:  time.After,
			dial:   (&fakeDialer{}).DialContext,
			resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
				return dockerContext{}, ErrDockerHostOverride
			},
			lockWait: time.Second,
		})

		res, err := s.Status(context.Background(), RunOptions{Checkout: seed.Checkout, RunID: seed.RunID})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDockerHostOverride)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureDockerUnavailable, res.Failure.Category)
		for _, c := range runner.Calls() {
			require.NotContains(t, strings.Join(c.Spec.Args, " "), "inspect")
		}
	})
}

func TestLogsRefreshesStoredLog(t *testing.T) {
	seed := seedRun(t, "run-logs", StateReady, true)
	runner := newLifecycleRunner(seed.Canonical)
	scriptMatchingResources(runner, seed.Manifest, true)
	const fresh = "fresh docker log line\n"
	runner.on(" logs ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		require.Contains(t, joined, seed.Manifest.ContainerID)
		require.NotContains(t, joined, "--follow")
		writeStdout(spec, fresh)
		return nil
	})
	s := newOpsSupervisor(t, seed, runner, nil)

	res, err := s.Logs(context.Background(), LogsOptions{Checkout: seed.Checkout, RunID: seed.RunID})
	require.NoError(t, err)
	require.Equal(t, "logs", res.Operation)
	require.Equal(t, seed.Manifest.Artifacts.ServerLog, res.ServerLog)
	body, err := os.ReadFile(res.ServerLog)
	require.NoError(t, err)
	require.Equal(t, fresh, string(body))
	require.NotContains(t, string(body), "stale log")
}

func TestLogsFollowTeesToCallerAndServerLog(t *testing.T) {
	seed := seedRun(t, "run-logs-follow", StateReady, true)
	runner := newLifecycleRunner(seed.Canonical)
	scriptMatchingResources(runner, seed.Manifest, true)

	ctx, cancel := context.WithCancel(context.Background())
	var caller bytes.Buffer
	runner.on(" logs ", func(c context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		require.Contains(t, joined, "--follow")
		require.Contains(t, joined, seed.Manifest.ContainerID)
		_, _ = io.WriteString(spec.Stdout, "line-one\n")
		_, _ = io.WriteString(spec.Stdout, "line-two\n")
		cancel()
		<-c.Done()
		return c.Err()
	})
	s := newOpsSupervisor(t, seed, runner, nil)

	res, err := s.Logs(ctx, LogsOptions{
		Checkout: seed.Checkout,
		RunID:    seed.RunID,
		Follow:   true,
		Output:   &caller,
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "canceled") || strings.Contains(err.Error(), "cancelled"))
	require.Equal(t, "logs", res.Operation)
	require.Contains(t, caller.String(), "line-one\n")
	require.Contains(t, caller.String(), "line-two\n")
	body, readErr := os.ReadFile(seed.Manifest.Artifacts.ServerLog)
	require.NoError(t, readErr)
	require.Contains(t, string(body), "line-one\n")
	require.Contains(t, string(body), "line-two\n")
}

func TestRenewHoldsLockAndRejectsExpiredStoppedOrAmbiguousRun(t *testing.T) {
	t.Run("renew extends lease under lock", func(t *testing.T) {
		seed := seedRun(t, "run-renew-ok", StateReady, true)
		runner := newLifecycleRunner(seed.Canonical)
		scriptMatchingResources(runner, seed.Manifest, true)
		clock := newReadinessFakeClock(seed.Now.Add(30 * time.Minute))
		s := newOpsSupervisor(t, seed, runner, clock)

		res, err := s.Renew(context.Background(), RenewOptions{
			Checkout: seed.Checkout,
			RunID:    seed.RunID,
			Lease:    time.Hour,
		})
		require.NoError(t, err)
		require.Equal(t, "renew", res.Operation)
		require.Equal(t, StateReady, res.State)
		m, err := readManifest(seed.Manifest.Artifacts.Manifest)
		require.NoError(t, err)
		require.Equal(t, StateReady, m.State)
		require.Equal(t, clock.Now().Add(time.Hour), m.LeaseExpiresAt)
		require.Equal(t, clock.Now(), m.UpdatedAt)
		// Labels remain unchanged (renew touches only manifest).
		require.Equal(t, seed.Manifest.CreatedAt, m.CreatedAt)
		require.Equal(t, seed.Manifest.ContainerID, m.ContainerID)
	})

	t.Run("rejects expired ready run", func(t *testing.T) {
		seed := seedRun(t, "run-renew-expired", StateReady, true)
		seed.Manifest.LeaseExpiresAt = seed.Now.Add(-time.Minute)
		require.NoError(t, writeManifest(seed.Manifest.Artifacts.Manifest, seed.Manifest))
		runner := newLifecycleRunner(seed.Canonical)
		scriptMatchingResources(runner, seed.Manifest, true)
		s := newOpsSupervisor(t, seed, runner, nil)

		_, err := s.Renew(context.Background(), RenewOptions{
			Checkout: seed.Checkout,
			RunID:    seed.RunID,
			Lease:    time.Hour,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrLeaseExpired)
	})

	t.Run("rejects stopped run", func(t *testing.T) {
		seed := seedRun(t, "run-renew-stopped", StateStopped, false)
		runner := newLifecycleRunner(seed.Canonical)
		s := newOpsSupervisor(t, seed, runner, nil)
		runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
			return &ExitError{Name: "docker", Args: spec.Args, ExitCode: 1}
		})

		_, err := s.Renew(context.Background(), RenewOptions{
			Checkout: seed.Checkout,
			RunID:    seed.RunID,
			Lease:    time.Hour,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRunNotReady)
	})

	t.Run("rejects ambiguous mismatched labels", func(t *testing.T) {
		seed := seedRun(t, "run-renew-ambiguous", StateReady, true)
		runner := newLifecycleRunner(seed.Canonical)
		scriptGitValidateAndBaseline(runner, seed.Canonical)
		bad := identityLabels(seed.Manifest)
		bad[labelProject] = "wrong-project"
		runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
			writeStdout(spec, labelledInspectJSON(true, "127.0.0.1", "54321", bad)+"\n")
			return nil
		})
		s := newOpsSupervisor(t, seed, runner, nil)

		_, err := s.Renew(context.Background(), RenewOptions{
			Checkout: seed.Checkout,
			RunID:    seed.RunID,
			Lease:    time.Hour,
		})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrResourceIdentityMismatch)
	})

	t.Run("concurrent renew and stop cannot both enter critical section", func(t *testing.T) {
		seed := seedRun(t, "run-renew-lock", StateReady, true)
		runner := newLifecycleRunner(seed.Canonical)
		scriptMatchingResources(runner, seed.Manifest, true)

		lockHeld := make(chan struct{})
		releaseLock := make(chan struct{})
		var acquireCount int
		var mu sync.Mutex

		s := newSupervisor(supervisorDeps{
			runner: runner,
			now:    func() time.Time { return seed.Now },
			after:  time.After,
			dial:   (&fakeDialer{}).DialContext,
			resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
				return seed.DC, nil
			},
			acquireLock: func(ctx context.Context, path string, wait time.Duration) (*runLock, error) {
				mu.Lock()
				acquireCount++
				n := acquireCount
				mu.Unlock()
				if n == 1 {
					close(lockHeld)
					select {
					case <-releaseLock:
						return acquireRunLock(ctx, path, wait)
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}
				return nil, ErrLockBusy
			},
			lockWait: 50 * time.Millisecond,
		})
		scriptGitValidateAndBaseline(runner, seed.Canonical)

		renewErr := make(chan error, 1)
		go func() {
			_, err := s.Renew(context.Background(), RenewOptions{
				Checkout: seed.Checkout,
				RunID:    seed.RunID,
				Lease:    time.Hour,
			})
			renewErr <- err
		}()

		select {
		case <-lockHeld:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for renew to hold lock")
		}

		_, stopErr := s.Stop(context.Background(), RunOptions{Checkout: seed.Checkout, RunID: seed.RunID})
		require.ErrorIs(t, stopErr, ErrLockBusy)

		close(releaseLock)
		require.NoError(t, <-renewErr)
	})
}

func TestStopCapturesLogsUsesGraceThenForceAndRemovesExactResources(t *testing.T) {
	seed := seedRun(t, "run-stop-grace", StateReady, true)
	runner := newLifecycleRunner(seed.Canonical)
	scriptGitValidateAndBaseline(runner, seed.Canonical)
	labels := identityLabels(seed.Manifest)
	clock := newReadinessFakeClock(seed.Now)

	inspectCount := 0
	var mu sync.Mutex
	var commands []string

	runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		mu.Lock()
		commands = append(commands, "inspect "+joined)
		mu.Unlock()
		if strings.Contains(joined, seed.Manifest.ContainerID) {
			inspectCount++
			// Identity checks + grace polls: stay running until force path.
			writeStdout(spec, labelledInspectJSON(true, "127.0.0.1", "54321", labels)+"\n")
			return nil
		}
		writeStdout(spec, labelledInspectJSON(false, "", "", labels)+"\n")
		return nil
	})
	runner.on(" logs ", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, "final server log\n")
		return nil
	})
	runner.on(" kill ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		require.Contains(t, joined, "kill")
		require.Contains(t, joined, "--signal=TERM")
		require.Contains(t, joined, seed.Manifest.ContainerID)
		mu.Lock()
		commands = append(commands, joined)
		mu.Unlock()
		return nil
	})
	runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		require.Contains(t, joined, seed.Manifest.Image)
		mu.Lock()
		commands = append(commands, joined)
		mu.Unlock()
		return nil
	})
	runner.on(" rm --force ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		require.Contains(t, joined, "--force")
		require.Contains(t, joined, seed.Manifest.ContainerID)
		mu.Lock()
		commands = append(commands, joined)
		mu.Unlock()
		return nil
	})
	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		require.Contains(t, joined, "down")
		require.Contains(t, joined, "--volumes")
		require.Contains(t, joined, "--remove-orphans")
		require.Contains(t, joined, "-p")
		require.Contains(t, joined, seed.Manifest.Project)
		mu.Lock()
		commands = append(commands, joined)
		mu.Unlock()
		return nil
	})

	s := newOpsSupervisor(t, seed, runner, clock)
	res, err := s.Stop(context.Background(), RunOptions{Checkout: seed.Checkout, RunID: seed.RunID})
	require.NoError(t, err)
	require.Equal(t, "stop", res.Operation)
	require.Equal(t, StateStopped, res.State)
	require.NotNil(t, res.Cleanup)
	require.True(t, res.Cleanup.Complete)

	body, err := os.ReadFile(seed.Manifest.Artifacts.ServerLog)
	require.NoError(t, err)
	require.Contains(t, string(body), "final server log")
	require.FileExists(t, seed.Manifest.Artifacts.Inspect)

	joinedAll := strings.Join(commands, " || ")
	require.Contains(t, joinedAll, "kill")
	require.Contains(t, joinedAll, "rm")
	require.Contains(t, joinedAll, "down")
	require.Contains(t, joinedAll, "image rm")
	// Exact resources only — no decoy IDs.
	require.NotContains(t, joinedAll, "decoy")
	require.Greater(t, inspectCount, 1, "grace poll must re-inspect running container")

	m, err := readManifest(seed.Manifest.Artifacts.Manifest)
	require.NoError(t, err)
	require.Equal(t, StateStopped, m.State)
	require.NoFileExists(t, seed.Manifest.Artifacts.Compose)
	require.NoDirExists(t, filepath.Join(seed.RunDir, controlDirName))
}

func TestStopRecoversAbandonedPreterminalRun(t *testing.T) {
	for _, state := range []State{StateBuilding, StateStopping} {
		state := state
		t.Run(string(state), func(t *testing.T) {
			seed := seedRun(t, "run-abandon-"+string(state), state, true)
			// Prior cleanup timeout leftovers for the stopping case.
			if state == StateStopping {
				seed.Manifest.Cleanup = &CleanupResult{
					Complete: false,
					Leftovers: []ResourceRef{
						{Kind: "container", ID: seed.Manifest.ContainerID},
						{Kind: "image", ID: seed.Manifest.Image},
					},
					Summary: "cleanup timed out",
				}
				require.NoError(t, writeManifest(seed.Manifest.Artifacts.Manifest, seed.Manifest))
			}
			runner := newLifecycleRunner(seed.Canonical)
			scriptMatchingResources(runner, seed.Manifest, false)
			runner.on(" logs ", func(ctx context.Context, spec CommandSpec) error {
				writeStdout(spec, "abandoned log\n")
				return nil
			})
			runner.on(" kill ", func(ctx context.Context, spec CommandSpec) error { return nil })
			runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error { return nil })
			runner.on(" rm --force ", func(ctx context.Context, spec CommandSpec) error { return nil })
			runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error { return nil })
			s := newOpsSupervisor(t, seed, runner, nil)

			res, err := s.Stop(context.Background(), RunOptions{Checkout: seed.Checkout, RunID: seed.RunID})
			require.NoError(t, err)
			require.Equal(t, StateFailed, res.State)
			require.NotNil(t, res.Failure)
			require.Equal(t, FailureAbandonedRun, res.Failure.Category)
			require.NotNil(t, res.Cleanup)
			require.True(t, res.Cleanup.Complete)

			m, err := readManifest(seed.Manifest.Artifacts.Manifest)
			require.NoError(t, err)
			require.Equal(t, StateFailed, m.State)
			require.Equal(t, FailureAbandonedRun, m.Failure.Category)
			require.True(t, m.Cleanup.Complete)
			require.NoFileExists(t, seed.Manifest.Artifacts.Compose)
		})
	}
}

func TestStopFinishesCleanupAfterCallerCancellation(t *testing.T) {
	seed := seedRun(t, "run-stop-cancel", StateReady, true)
	runner := newLifecycleRunner(seed.Canonical)
	labels := identityLabels(seed.Manifest)

	ctx, cancel := context.WithCancel(context.Background())
	var cleanupCtxs []context.Context
	var mu sync.Mutex
	termSent := false

	runner.on(" inspect ", func(c context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		if strings.Contains(joined, seed.Manifest.ContainerID) {
			mu.Lock()
			running := !termSent
			mu.Unlock()
			writeStdout(spec, labelledInspectJSON(running, "127.0.0.1", "54321", labels)+"\n")
			return nil
		}
		writeStdout(spec, labelledInspectJSON(false, "", "", labels)+"\n")
		return nil
	})
	runner.on(" logs ", func(c context.Context, spec CommandSpec) error {
		writeStdout(spec, "pre-destructive log\n")
		return nil
	})
	runner.on(" kill ", func(c context.Context, spec CommandSpec) error {
		cancel() // cancel caller once destructive work begins
		mu.Lock()
		termSent = true
		cleanupCtxs = append(cleanupCtxs, c)
		mu.Unlock()
		require.NoError(t, c.Err(), "destructive cleanup must ignore caller cancel")
		return nil
	})
	runner.on(" image rm ", func(c context.Context, spec CommandSpec) error {
		mu.Lock()
		cleanupCtxs = append(cleanupCtxs, c)
		mu.Unlock()
		require.NoError(t, c.Err())
		return nil
	})
	runner.on(" rm --force ", func(c context.Context, spec CommandSpec) error {
		t.Fatal("force rm should not be needed when container stops after TERM")
		return nil
	})
	runner.on(" compose ", func(c context.Context, spec CommandSpec) error {
		mu.Lock()
		cleanupCtxs = append(cleanupCtxs, c)
		mu.Unlock()
		require.NoError(t, c.Err())
		_, ok := c.Deadline()
		require.True(t, ok)
		return nil
	})

	s := newOpsSupervisor(t, seed, runner, newReadinessFakeClock(seed.Now))
	res, err := s.Stop(ctx, RunOptions{Checkout: seed.Checkout, RunID: seed.RunID})
	require.NoError(t, err, "stop must finish cleanup despite caller cancel")
	require.Equal(t, StateStopped, res.State)
	require.NotEmpty(t, cleanupCtxs)
	require.True(t, res.Cleanup.Complete)
}

func TestStopIsIdempotentForStoppedAndCleanedFailedRuns(t *testing.T) {
	t.Run("stopped with no live resources", func(t *testing.T) {
		seed := seedRun(t, "run-stop-idem-stopped", StateStopped, true)
		runner := newLifecycleRunner(seed.Canonical)
		scriptGitValidateAndBaseline(runner, seed.Canonical)
		runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
			return &ExitError{Name: "docker", Args: spec.Args, ExitCode: 1}
		})
		s := newOpsSupervisor(t, seed, runner, nil)

		res, err := s.Stop(context.Background(), RunOptions{Checkout: seed.Checkout, RunID: seed.RunID})
		require.NoError(t, err)
		require.Equal(t, StateStopped, res.State)
		require.NotNil(t, res.Cleanup)
		require.True(t, res.Cleanup.Complete)
	})

	t.Run("failed cleaned remains failed", func(t *testing.T) {
		seed := seedRun(t, "run-stop-idem-failed", StateFailed, true)
		seed.Manifest.Failure = &FailureRecord{Category: FailureBuild, Phase: StateBuilding, Summary: "prior"}
		seed.Manifest.Cleanup = &CleanupResult{Complete: true, Summary: "resources removed"}
		require.NoError(t, writeManifest(seed.Manifest.Artifacts.Manifest, seed.Manifest))
		runner := newLifecycleRunner(seed.Canonical)
		scriptGitValidateAndBaseline(runner, seed.Canonical)
		runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
			return &ExitError{Name: "docker", Args: spec.Args, ExitCode: 1}
		})
		s := newOpsSupervisor(t, seed, runner, nil)

		res, err := s.Stop(context.Background(), RunOptions{Checkout: seed.Checkout, RunID: seed.RunID})
		require.NoError(t, err)
		require.Equal(t, StateFailed, res.State)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureBuild, res.Failure.Category)
		m, err := readManifest(seed.Manifest.Artifacts.Manifest)
		require.NoError(t, err)
		require.Equal(t, StateFailed, m.State)
	})
}

func TestStopIssuesNoPruneOrCacheDestroyingCommand(t *testing.T) {
	seed := seedRun(t, "run-stop-nocache", StateReady, true)
	runner := newLifecycleRunner(seed.Canonical)
	scriptMatchingResources(runner, seed.Manifest, false)
	runner.on(" logs ", func(ctx context.Context, spec CommandSpec) error { return nil })
	runner.on(" kill ", func(ctx context.Context, spec CommandSpec) error { return nil })
	runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error { return nil })
	runner.on(" rm --force ", func(ctx context.Context, spec CommandSpec) error { return nil })
	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error { return nil })
	s := newOpsSupervisor(t, seed, runner, newReadinessFakeClock(seed.Now))

	_, err := s.Stop(context.Background(), RunOptions{Checkout: seed.Checkout, RunID: seed.RunID})
	require.NoError(t, err)

	for _, c := range runner.Calls() {
		if c.Spec.Name != "docker" {
			continue
		}
		for _, arg := range c.Spec.Args {
			a := strings.ToLower(arg)
			require.NotEqual(t, "prune", a)
			require.NotEqual(t, "buildx", a)
			require.False(t, a == "system" && containsArg(c.Spec.Args, "prune"))
		}
		joined := strings.ToLower(strings.Join(c.Spec.Args, " "))
		require.NotContains(t, joined, "builder cache")
	}
}

func containsArg(args []string, want string) bool {
	for _, a := range args {
		if strings.EqualFold(a, want) {
			return true
		}
	}
	return false
}
