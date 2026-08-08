package playtestenv

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const integrationEnv = "DOGMUD_PLAYTESTENV_INTEGRATION"

// TestDockerIntegration exercises the real Docker lifecycle against the host
// daemon. Opt in with DOGMUD_PLAYTESTENV_INTEGRATION=1.
func TestDockerIntegration(t *testing.T) {
	if os.Getenv(integrationEnv) != "1" {
		t.Skip("set DOGMUD_PLAYTESTENV_INTEGRATION=1")
	}

	checkout := integrationRepoRoot(t)
	s := New()
	ctx := context.Background()

	sweepExistingRuns(t, s, checkout)
	forceRemoveManagedLeftovers(t)
	require.Empty(t, listManagedIDs(t), "pre-existing dogmud.playtest.managed resources must be cleared before integration")
	beforeGit := snapshotGit(t, checkout)
	trackedIDs := &sync.Map{}

	t.Cleanup(func() {
		sweepTracked(t, s, checkout, trackedIDs)
		assertNoManagedLeftovers(t, trackedIDs)
		afterGit := snapshotGit(t, checkout)
		require.Equal(t, beforeGit, afterGit, "host Git status changed outside ignored run/report artifacts")
	})

	t.Run("lifecycle_status_logs_renew_stop", func(t *testing.T) {
		res, readyFor := startReady(t, s, checkout, StartOptions{Checkout: checkout}, trackedIDs)
		t.Logf("cold boot-to-ready after compose up window recorded via Start success; total Start elapsed=%s", readyFor)
		require.LessOrEqual(t, readyFor, DefaultReadinessTimeout,
			"boot-to-ready exceeded DefaultReadinessTimeout; stop and revise default rather than weakening readiness")

		st, err := s.Status(ctx, RunOptions{Checkout: checkout, RunID: res.RunID})
		require.NoError(t, err)
		require.Equal(t, StateReady, st.State)
		require.NotNil(t, st.Endpoint)
		require.True(t, isLoopbackHost(st.Endpoint.Host), "endpoint host %q", st.Endpoint.Host)
		require.Greater(t, st.Endpoint.Port, 0)

		var logBuf strings.Builder
		lr, err := s.Logs(ctx, LogsOptions{Checkout: checkout, RunID: res.RunID, Output: &logBuf})
		require.NoError(t, err)
		require.NotEmpty(t, lr.ServerLog)
		logBytes, err := os.ReadFile(lr.ServerLog)
		require.NoError(t, err)
		require.Contains(t, string(logBytes), serverReadyMarker)
		require.NotContains(t, string(logBytes), "migration", "CONFIG_PATH migration/config-save errors")
		require.NotContains(t, strings.ToLower(string(logBytes)), "failed to save config")

		renewed, err := s.Renew(ctx, RenewOptions{Checkout: checkout, RunID: res.RunID, Lease: time.Hour})
		require.NoError(t, err)
		require.Equal(t, StateReady, renewed.State)

		// Authored _datafiles present in disposable volume.
		out := dockerExec(t, res.RunID, "test", "-f", "/app/_datafiles/config.yaml")
		require.Empty(t, out)

		hostCfg := filepath.Join(checkout, "_datafiles", "config.yaml")
		hostHashBefore := fileSHA256(t, hostCfg)
		gitBeforeMut := snapshotGit(t, checkout)

		dockerExec(t, res.RunID, "sh", "-c", "echo playtestenv-volume-mutate > /app/_datafiles/playtestenv-volume-probe.txt")
		require.Equal(t, hostHashBefore, fileSHA256(t, hostCfg), "host authored config hash changed after volume mutation")
		require.Equal(t, gitBeforeMut, snapshotGit(t, checkout), "host Git changed after volume mutation")
		_, err = os.Stat(filepath.Join(checkout, "_datafiles", "playtestenv-volume-probe.txt"))
		require.True(t, errors.Is(err, os.ErrNotExist), "volume probe leaked onto host checkout")

		stop1, err := s.Stop(ctx, RunOptions{Checkout: checkout, RunID: res.RunID})
		require.NoError(t, err)
		require.NotNil(t, stop1.Cleanup)
		require.True(t, stop1.Cleanup.Complete, stop1.Cleanup.Summary)
		require.Equal(t, StateStopped, stop1.State)

		assertImageGone(t, res.RunID)
		assertControlGone(t, checkout, res.RunID)

		// Repeated stop is idempotent.
		stop2, err := s.Stop(ctx, RunOptions{Checkout: checkout, RunID: res.RunID})
		require.NoError(t, err)
		require.NotNil(t, stop2.Cleanup)
		require.True(t, stop2.Cleanup.Complete)

		// Second build shows BuildKit cache reuse.
		cachedStart := time.Now()
		res2, cachedFor := startReady(t, s, checkout, StartOptions{Checkout: checkout}, trackedIDs)
		t.Logf("cached Start elapsed=%s readiness-bound=%s", time.Since(cachedStart), cachedFor)
		require.LessOrEqual(t, cachedFor, DefaultReadinessTimeout,
			"cached boot-to-ready exceeded DefaultReadinessTimeout; stop and revise default")
		buildLog, err := os.ReadFile(res2.Artifacts.BuildLog)
		require.NoError(t, err)
		bl := string(buildLog)
		require.True(t,
			strings.Contains(bl, "CACHED") || strings.Contains(bl, "cache") || strings.Contains(strings.ToLower(bl), "using cache"),
			"second build log should show BuildKit cache reuse; log excerpt:\n%s", truncate(bl, 2000))
		require.NoError(t, mustStop(t, s, checkout, res2.RunID))
	})

	t.Run("concurrent_starts", func(t *testing.T) {
		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			errs []error
			got  []Result
		)
		wg.Add(2)
		for i := 0; i < 2; i++ {
			go func() {
				defer wg.Done()
				start := time.Now()
				res, err := s.Start(ctx, StartOptions{Checkout: checkout, Lease: time.Hour})
				registerPartialCleanup(t, s, checkout, res, trackedIDs)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					errs = append(errs, fmt.Errorf("start after %s: %w (failure=%v)", time.Since(start), err, res.Failure))
					return
				}
				got = append(got, res)
			}()
		}
		wg.Wait()
		require.Empty(t, errs)
		require.Len(t, got, 2)
		require.NotEqual(t, got[0].RunID, got[1].RunID)
		require.NotEqual(t, got[0].Project, got[1].Project)
		require.NotNil(t, got[0].Endpoint)
		require.NotNil(t, got[1].Endpoint)
		require.NotEqual(t, got[0].Endpoint.Port, got[1].Endpoint.Port)
		require.NotEqual(t, got[0].Manifest, got[1].Manifest)
		if got[0].Report != got[1].Report && got[0].Artifacts != nil && got[1].Artifacts != nil {
			require.NotEqual(t, got[0].Artifacts.BuildLog, got[1].Artifacts.BuildLog)
		}
		m0 := mustReadManifest(t, got[0].Manifest)
		m1 := mustReadManifest(t, got[1].Manifest)
		require.NotEqual(t, m0.Image, m1.Image)
		require.NotEqual(t, m0.Volume, m1.Volume)
		require.NotEqual(t, m0.Network, m1.Network)
		for _, res := range got {
			require.NoError(t, mustStop(t, s, checkout, res.RunID))
		}
	})

	t.Run("worktree_checkout_isolation", func(t *testing.T) {
		markerA := "playtestenv-tracked-marker-" + integrationID()
		markerB := "playtestenv-untracked-marker-" + integrationID()
		probeName := "playtestenv-probe-" + integrationID() + ".txt"

		wtA := addDetachedWorktree(t, checkout, "a")
		wtB := addDetachedWorktree(t, checkout, "b")

		zonesPath := filepath.Join(wtA, "_datafiles", "html", "public", "static", "js", "zones.js")
		appendFile(t, zonesPath, "\n// "+markerA+"\n")

		probePath := filepath.Join(wtB, "_datafiles", "html", "public", "static", probeName)
		require.NoError(t, os.WriteFile(probePath, []byte(markerB+"\n"), 0o644))

		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			errs []error
			got  = map[string]Result{}
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			res, err := s.Start(ctx, StartOptions{Checkout: wtA, Lease: time.Hour})
			registerPartialCleanup(t, s, wtA, res, trackedIDs)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			got["a"] = res
		}()
		go func() {
			defer wg.Done()
			res, err := s.Start(ctx, StartOptions{Checkout: wtB, Lease: time.Hour})
			registerPartialCleanup(t, s, wtB, res, trackedIDs)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			got["b"] = res
		}()
		wg.Wait()
		require.Empty(t, errs)
		require.Len(t, got, 2)

		aOut := dockerExecOutput(t, got["a"].RunID, "cat", "/app/_datafiles/html/public/static/js/zones.js")
		require.Contains(t, aOut, markerA)
		bOut := dockerExecOutput(t, got["b"].RunID, "cat", "/app/_datafiles/html/public/static/"+probeName)
		require.Contains(t, bOut, markerB)
		// Cross-check isolation: A must not contain B's probe path.
		cross := dockerExecCombined(t, got["a"].RunID, "cat", "/app/_datafiles/html/public/static/"+probeName)
		require.True(t, strings.Contains(cross, "No such file") || strings.Contains(cross, "can't open") || cross == "",
			"worktree A unexpectedly has B probe: %q", cross)

		require.NoError(t, mustStop(t, s, wtA, got["a"].RunID))
		require.NoError(t, mustStop(t, s, wtB, got["b"].RunID))
	})

	t.Run("failure_invalid_dockerfile", func(t *testing.T) {
		wt := addDetachedWorktree(t, checkout, "bd")
		require.NoError(t, os.WriteFile(filepath.Join(wt, "provisioning", "Dockerfile"), []byte("FROM totally-invalid-base-image-playtestenv\n"), 0o644))
		res, err := s.Start(ctx, StartOptions{Checkout: wt, Lease: time.Hour})
		registerPartialCleanup(t, s, wt, res, trackedIDs)
		require.Error(t, err)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureBuild, res.Failure.Category)
		require.NotEmpty(t, res.RunID)
		require.FileExists(t, res.Artifacts.BuildLog)
		require.NotEmpty(t, res.Report)
		require.FileExists(t, res.Report)
		assertImageGone(t, res.RunID)
	})

	t.Run("failure_boot_panic_yaml", func(t *testing.T) {
		wt := addDetachedWorktree(t, checkout, "bp")
		mob := filepath.Join(wt, "_datafiles", "world", "dogmud", "mobs", "thornwall_city", "102-market_merchant.yaml")
		raw, err := os.ReadFile(mob)
		require.NoError(t, err)
		updated := strings.Replace(string(raw), "schedule_id: thornwall_market_merchant", "schedule_id: nonexistent_schedule_playtestenv_integration", 1)
		require.NotEqual(t, string(raw), updated)
		require.NoError(t, os.WriteFile(mob, []byte(updated), 0o644))

		res, err := s.Start(ctx, StartOptions{Checkout: wt, Lease: time.Hour, ReadinessTimeout: 60 * time.Second})
		registerPartialCleanup(t, s, wt, res, trackedIDs)
		require.Error(t, err)
		require.NotNil(t, res.Failure)
		// Compound readiness classifies a fast exit as container_exited when the
		// process is already gone before panic markers can be observed; both
		// match the plan's "boot panic/exit" failure case.
		require.Contains(t, []FailureCategory{FailureBootPanic, FailureContainerExited}, res.Failure.Category)
		require.FileExists(t, res.Artifacts.ServerLog)
		logBytes, err := os.ReadFile(res.Artifacts.ServerLog)
		require.NoError(t, err)
		low := strings.ToLower(string(logBytes))
		require.True(t,
			strings.Contains(low, "panic") ||
				strings.Contains(low, "schedule") ||
				strings.Contains(low, "nonexistent_schedule_playtestenv_integration") ||
				res.Failure.Category == FailureContainerExited,
			"server log should evidence YAML boot failure; category=%s log=%s",
			res.Failure.Category, truncate(string(logBytes), 1500))
		require.FileExists(t, res.Report)
		assertImageGone(t, res.RunID)
	})

	t.Run("failure_tiny_readiness_timeout", func(t *testing.T) {
		res, err := s.Start(ctx, StartOptions{
			Checkout:         checkout,
			Lease:            time.Hour,
			ReadinessTimeout: 50 * time.Millisecond,
		})
		registerPartialCleanup(t, s, checkout, res, trackedIDs)
		require.Error(t, err)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureReadinessTimeout, res.Failure.Category)
		require.FileExists(t, res.Report)
		assertImageGone(t, res.RunID)
	})

	t.Run("failure_no_port_policy", func(t *testing.T) {
		withTestComposePolicy(t, testComposePolicyNoPort(t))
		res, err := s.Start(ctx, StartOptions{Checkout: checkout, Lease: time.Hour, ReadinessTimeout: 30 * time.Second})
		registerPartialCleanup(t, s, checkout, res, trackedIDs)
		require.Error(t, err)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailurePortPublication, res.Failure.Category)
		assertImageGone(t, res.RunID)
	})

	t.Run("failure_non_loopback_policy", func(t *testing.T) {
		withTestComposePolicy(t, testComposePolicyNonLoopback(t))
		res, err := s.Start(ctx, StartOptions{Checkout: checkout, Lease: time.Hour, ReadinessTimeout: 45 * time.Second})
		registerPartialCleanup(t, s, checkout, res, trackedIDs)
		require.Error(t, err)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureNonLoopback, res.Failure.Category)
		assertImageGone(t, res.RunID)
	})

	t.Run("reject_hostile_docker_host", func(t *testing.T) {
		t.Setenv("DOCKER_HOST", "tcp://203.0.113.10:2375")
		res, err := s.Start(ctx, StartOptions{Checkout: checkout, Lease: time.Hour})
		registerPartialCleanup(t, s, checkout, res, trackedIDs)
		// Clear overrides before Stop/Reap cleanups (LIFO: this runs first).
		clearDockerOverridesForCleanup(t)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDockerHostOverride)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureDockerUnavailable, res.Failure.Category)
		require.NotEmpty(t, res.RunID, "post-reservation Result must identify the run for cleanup")
		require.NotNil(t, res.Cleanup)
		require.True(t, res.Cleanup.Complete)
		// Must fail before compose build (no build.log content from a real build).
		if res.Artifacts != nil && res.Artifacts.BuildLog != "" {
			if b, err := os.ReadFile(res.Artifacts.BuildLog); err == nil {
				require.NotContains(t, string(b), "DONE", "must reject before build")
			}
		}
	})

	t.Run("reject_remote_context_fixture", func(t *testing.T) {
		name := integrationID()
		createRemoteContextFixture(t, name, "tcp://203.0.113.11:2375")
		t.Setenv("DOCKER_CONTEXT", name)
		// Ensure we never flip the user's active context.
		activeBefore := strings.TrimSpace(runCmdOutput(t, "docker", "context", "show"))
		res, err := s.Start(ctx, StartOptions{Checkout: checkout, Lease: time.Hour})
		registerPartialCleanup(t, s, checkout, res, trackedIDs)
		clearDockerOverridesForCleanup(t)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrDockerContextNotLocal)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureDockerUnavailable, res.Failure.Category)
		activeAfter := strings.TrimSpace(runCmdOutput(t, "docker", "context", "show"))
		require.Equal(t, activeBefore, activeAfter, "active Docker context must not change")
	})

	t.Run("honor_named_local_context", func(t *testing.T) {
		endpoint := localDockerEndpoint(t)
		name := integrationID()
		createLocalContextFixture(t, name, endpoint)
		t.Setenv("DOCKER_CONTEXT", name)
		wt := addDetachedWorktree(t, checkout, "lctx")
		require.NoError(t, os.WriteFile(filepath.Join(wt, "provisioning", "Dockerfile"), []byte("FROM totally-invalid-base-image-localctx\n"), 0o644))
		res, err := s.Start(ctx, StartOptions{Checkout: wt, Lease: time.Hour})
		registerPartialCleanup(t, s, wt, res, trackedIDs)
		clearDockerOverridesForCleanup(t)
		require.Error(t, err)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureBuild, res.Failure.Category, "local named context must pass preflight and reach build")
		require.NotEqual(t, FailureDockerUnavailable, res.Failure.Category)
	})

	t.Run("cancel_after_container_create", func(t *testing.T) {
		ctx2, cancel := context.WithCancel(context.Background())
		defer cancel()
		sup := newSupervisor(supervisorDeps{
			onEvent: func(name string) {
				if name == "readiness" {
					cancel()
				}
			},
		})
		res, err := sup.Start(ctx2, StartOptions{Checkout: checkout, Lease: time.Hour})
		registerPartialCleanup(t, sup, checkout, res, trackedIDs)
		require.Error(t, err)
		require.NotEmpty(t, res.RunID)
		require.NotNil(t, res.Failure)
		require.True(t, res.Cleanup == nil || res.Cleanup.Complete, "cleanup after cancel: %+v", res.Cleanup)
		assertImageGone(t, res.RunID)
		require.FileExists(t, res.Report)
	})

	t.Run("graceful_stop_and_force_term_ignore", func(t *testing.T) {
		res, _ := startReady(t, s, checkout, StartOptions{Checkout: checkout, Lease: time.Hour}, trackedIDs)
		require.NoError(t, mustStop(t, s, checkout, res.RunID))
		assertImageGone(t, res.RunID)

		withTestComposePolicy(t, testComposePolicyIgnoreTERM(t))
		res2, _ := startReady(t, s, checkout, StartOptions{Checkout: checkout, Lease: time.Hour}, trackedIDs)
		stopRes, err := s.Stop(ctx, RunOptions{Checkout: checkout, RunID: res2.RunID})
		require.NoError(t, err)
		require.NotNil(t, stopRes.Cleanup)
		require.True(t, stopRes.Cleanup.Complete, stopRes.Cleanup.Summary)
		assertImageGone(t, res2.RunID)
		assertControlGone(t, checkout, res2.RunID)
	})

	t.Run("expired_reap_and_decoy_untouched", func(t *testing.T) {
		res, _ := startReady(t, s, checkout, StartOptions{Checkout: checkout, Lease: 2 * time.Second}, trackedIDs)
		m := mustReadManifest(t, res.Manifest)
		m.LeaseExpiresAt = time.Now().Add(-time.Minute)
		require.NoError(t, writeManifest(m.Artifacts.Manifest, m))

		decoyName := "dogmud-playtest-decoy-" + integrationID()
		decoyRun := "decoy" + integrationID()[:12]
		runCmd(t, "docker", "run", "-d", "--name", decoyName,
			"--label", labelManaged+"="+labelManagedValue,
			"--label", labelRunID+"="+decoyRun,
			"--label", labelProject+"=dogmud-playtest-"+decoyRun,
			"--label", labelSchema+"="+labelSchemaValue,
			"alpine:3.20", "sleep", "3600")
		t.Cleanup(func() {
			_ = exec.Command("docker", "rm", "-f", decoyName).Run()
		})

		results, err := s.Reap(ctx, checkout)
		require.NoError(t, err)
		var reaped bool
		for _, r := range results {
			if r.RunID == res.RunID && r.Cleanup != nil && r.Cleanup.Complete {
				reaped = true
			}
		}
		require.True(t, reaped, "expired run not reaped; results=%+v", results)
		assertImageGone(t, res.RunID)
		require.FileExists(t, res.Artifacts.ServerLog)

		inspect := runCmdCombined(t, "docker", "inspect", decoyName)
		require.NotContains(t, inspect, "No such object")
		require.Contains(t, inspect, decoyName)
	})
}

func startReady(t *testing.T, s *Supervisor, checkout string, opts StartOptions, tracked *sync.Map) (Result, time.Duration) {
	t.Helper()
	if opts.Lease <= 0 {
		opts.Lease = time.Hour
	}
	if opts.Checkout == "" {
		opts.Checkout = checkout
	}
	var (
		mu            sync.Mutex
		readyStarted  time.Time
		readyElapsed  time.Duration
		sawReadiness  bool
	)
	sup := newSupervisor(supervisorDeps{
		onEvent: func(name string) {
			mu.Lock()
			defer mu.Unlock()
			switch name {
			case "readiness":
				readyStarted = time.Now()
				sawReadiness = true
			case "transition ready":
				if sawReadiness && !readyStarted.IsZero() {
					readyElapsed = time.Since(readyStarted)
				}
			}
		},
	})
	res, err := sup.Start(context.Background(), opts)
	registerPartialCleanup(t, sup, opts.Checkout, res, tracked)
	require.NoError(t, err, "Start failure=%+v", res.Failure)
	require.Equal(t, StateReady, res.State)
	require.NotNil(t, res.Endpoint)
	require.True(t, isLoopbackHost(res.Endpoint.Host))
	mu.Lock()
	elapsed := readyElapsed
	mu.Unlock()
	require.Greater(t, elapsed, time.Duration(0), "readiness timing not recorded")
	require.LessOrEqual(t, elapsed, DefaultReadinessTimeout,
		"boot-to-ready %s exceeded DefaultReadinessTimeout; stop and revise default rather than weakening", elapsed)
	return res, elapsed
}

func registerPartialCleanup(t *testing.T, s *Supervisor, checkout string, res Result, tracked *sync.Map) {
	t.Helper()
	if res.RunID == "" {
		return
	}
	tracked.Store(res.RunID, checkout)
	runID := res.RunID
	t.Cleanup(func() {
		// Ambient hostile DOCKER_* from failure cases must not poison Stop/Reap.
		_ = os.Unsetenv("DOCKER_HOST")
		_ = os.Unsetenv("DOCKER_CONTEXT")
		_ = os.Unsetenv("DOCKER_TLS_VERIFY")
		_ = os.Unsetenv("DOCKER_CERT_PATH")
		ctx, cancel := context.WithTimeout(context.Background(), CleanupTimeout+30*time.Second)
		defer cancel()
		stopRes, err := s.Stop(ctx, RunOptions{Checkout: checkout, RunID: runID})
		if err == nil && (stopRes.Cleanup == nil || stopRes.Cleanup.Complete) {
			return
		}
		_, reapErr := s.Reap(ctx, checkout)
		leftovers := listManagedIDs(t)
		if err != nil || reapErr != nil || len(leftovers) > 0 {
			t.Errorf("cleanup incomplete for run %s stopErr=%v reapErr=%v leftovers=%v stopCleanup=%+v",
				runID, err, reapErr, leftovers, stopRes.Cleanup)
		}
	})
}

func mustStop(t *testing.T, s *Supervisor, checkout, runID string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), CleanupTimeout+30*time.Second)
	defer cancel()
	res, err := s.Stop(ctx, RunOptions{Checkout: checkout, RunID: runID})
	if err != nil {
		return err
	}
	if res.Cleanup != nil && !res.Cleanup.Complete {
		return fmt.Errorf("incomplete cleanup: %s leftovers=%v", res.Cleanup.Summary, res.Cleanup.Leftovers)
	}
	return nil
}

func sweepExistingRuns(t *testing.T, s *Supervisor, checkout string) {
	t.Helper()
	ids, err := listRunCandidates(checkout)
	require.NoError(t, err)
	for _, id := range ids {
		_ = mustStop(t, s, checkout, id)
	}
	_, _ = s.Reap(context.Background(), checkout)
}

func sweepTracked(t *testing.T, s *Supervisor, _ string, tracked *sync.Map) {
	t.Helper()
	tracked.Range(func(key, value any) bool {
		runID, _ := key.(string)
		checkout, _ := value.(string)
		_ = mustStop(t, s, checkout, runID)
		return true
	})
}

func assertNoManagedLeftovers(t *testing.T, tracked *sync.Map) {
	t.Helper()
	leftovers := listManagedIDs(t)
	var unexpected []string
	for _, id := range leftovers {
		// Decoy containers use alpine and names dogmud-playtest-decoy-*; allow only if already removed.
		unexpected = append(unexpected, id)
	}
	// Filter out nothing — decoys must be removed by their own Cleanup.
	// Also check dogmud-playtest images for tracked run IDs.
	tracked.Range(func(key, _ any) bool {
		runID, _ := key.(string)
		out := runCmdCombined(t, "docker", "image", "inspect", imageNamePrefix+runID)
		if !strings.Contains(strings.ToLower(out), "no such image") && !strings.Contains(strings.ToLower(out), "no such object") {
			unexpected = append(unexpected, "image:"+imageNamePrefix+runID)
		}
		return true
	})
	require.Empty(t, unexpected, "leftover dogmud-playtest resources: %v", unexpected)
}

func forceRemoveManagedLeftovers(t *testing.T) {
	t.Helper()
	// Integration tests may delete only dogmud.playtest.managed resources.
	for _, id := range strings.Fields(runCmdCombined(t, "docker", "ps", "-aq", "--filter", managedLabelFilter)) {
		_ = exec.Command("docker", "rm", "-f", id).Run()
	}
	for _, id := range strings.Fields(runCmdCombined(t, "docker", "network", "ls", "-q", "--filter", managedLabelFilter)) {
		_ = exec.Command("docker", "network", "rm", id).Run()
	}
	for _, id := range strings.Fields(runCmdCombined(t, "docker", "volume", "ls", "-q", "--filter", managedLabelFilter)) {
		_ = exec.Command("docker", "volume", "rm", "-f", id).Run()
	}
	for _, id := range strings.Fields(runCmdCombined(t, "docker", "images", "-q", "--filter", managedLabelFilter)) {
		_ = exec.Command("docker", "image", "rm", "-f", id).Run()
	}
	// Also drop any dangling dogmud-playtest: tags without the label filter match.
	imgs := runCmdCombined(t, "docker", "images", "--format", "{{.Repository}}:{{.Tag}}")
	for _, line := range strings.Split(imgs, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, imageNamePrefix) {
			_ = exec.Command("docker", "image", "rm", "-f", line).Run()
		}
	}
}

func listManagedIDs(t *testing.T) []string {
	t.Helper()
	var ids []string
	for _, args := range [][]string{
		{"ps", "-aq", "--filter", managedLabelFilter},
		{"network", "ls", "-q", "--filter", managedLabelFilter},
		{"volume", "ls", "-q", "--filter", managedLabelFilter},
		{"images", "-q", "--filter", managedLabelFilter},
	} {
		out := strings.TrimSpace(runCmdCombined(t, "docker", args...))
		if out == "" {
			continue
		}
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				ids = append(ids, line)
			}
		}
	}
	return ids
}

func integrationID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func assertImageGone(t *testing.T, runID string) {
	t.Helper()
	if runID == "" {
		return
	}
	out := runCmdCombined(t, "docker", "image", "inspect", imageNamePrefix+runID)
	low := strings.ToLower(out)
	require.True(t, strings.Contains(low, "no such image") || strings.Contains(low, "no such object"),
		"image still present for %s: %s", runID, truncate(out, 500))
}

func assertControlGone(t *testing.T, checkout, runID string) {
	t.Helper()
	control := filepath.Join(checkout, filepath.FromSlash(runsDirName), runID, controlDirName)
	_, err := os.Stat(control)
	require.True(t, errors.Is(err, os.ErrNotExist), "control/ still present at %s", control)
	compose := filepath.Join(checkout, filepath.FromSlash(runsDirName), runID, composeResolvedFileName)
	_, err = os.Stat(compose)
	require.True(t, errors.Is(err, os.ErrNotExist), "compose.resolved.yml still present at %s", compose)
}

func integrationRepoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "--no-optional-locks", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	root := strings.TrimSpace(string(out))
	require.DirExists(t, root)
	return filepath.Clean(root)
}

func snapshotGit(t *testing.T, checkout string) GitBaseline {
	t.Helper()
	baseline, err := collectGitBaseline(context.Background(), execRunner{}, checkout)
	require.NoError(t, err)
	return baseline
}

func addDetachedWorktree(t *testing.T, repo, name string) string {
	t.Helper()
	// Keep paths short on Windows: Temp + long _archive report names exceed MAX_PATH.
	root := filepath.Join(filepath.Dir(repo), "pwt")
	require.NoError(t, os.MkdirAll(root, 0o755))
	path := filepath.Join(root, name+"-"+integrationID()[:8])
	runCmd(t, "git", "-c", "core.longpaths=true", "--no-optional-locks", "-C", repo,
		"worktree", "add", "--detach", path, "HEAD")
	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	listBefore := runCmdOutput(t, "git", "--no-optional-locks", "-C", repo, "worktree", "list", "--porcelain")
	require.True(t, worktreeListHas(listBefore, abs), "worktree admin entry missing for %s in:\n%s", abs, listBefore)
	t.Cleanup(func() {
		runCmd(t, "git", "--no-optional-locks", "-C", repo, "worktree", "remove", "--force", abs)
		listAfter := runCmdOutput(t, "git", "--no-optional-locks", "-C", repo, "worktree", "list", "--porcelain")
		require.False(t, worktreeListHas(listAfter, abs), "worktree admin entry still present for %s", abs)
	})
	return abs
}

func clearDockerOverridesForCleanup(t *testing.T) {
	t.Helper()
	// Registered after Stop cleanup so LIFO clears ambient overrides first.
	t.Cleanup(func() {
		_ = os.Unsetenv("DOCKER_HOST")
		_ = os.Unsetenv("DOCKER_CONTEXT")
		_ = os.Unsetenv("DOCKER_TLS_VERIFY")
		_ = os.Unsetenv("DOCKER_CERT_PATH")
	})
}

func worktreeListHas(list, path string) bool {
	norm := strings.ToLower(filepath.ToSlash(path))
	for _, line := range strings.Split(list, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		entry := strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
		if strings.ToLower(filepath.ToSlash(entry)) == norm {
			return true
		}
	}
	return false
}

func appendFile(t *testing.T, path, suffix string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(suffix)
	require.NoError(t, err)
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func mustReadManifest(t *testing.T, path string) *Manifest {
	t.Helper()
	m, err := readManifest(path)
	require.NoError(t, err)
	return m
}

func dockerExec(t *testing.T, runID string, args ...string) string {
	t.Helper()
	cid := containerIDForRun(t, runID)
	cmdArgs := append([]string{"exec", cid}, args...)
	return runCmdOutput(t, "docker", cmdArgs...)
}

func dockerExecOutput(t *testing.T, runID string, args ...string) string {
	t.Helper()
	return dockerExec(t, runID, args...)
}

func dockerExecCombined(t *testing.T, runID string, args ...string) string {
	t.Helper()
	cid := containerIDForRun(t, runID)
	cmdArgs := append([]string{"exec", cid}, args...)
	return runCmdCombined(t, "docker", cmdArgs...)
}

func containerIDForRun(t *testing.T, runID string) string {
	t.Helper()
	out := runCmdOutput(t, "docker", "ps", "-q",
		"--filter", "label="+labelRunID+"="+runID,
		"--filter", managedLabelFilter,
	)
	id := strings.TrimSpace(out)
	require.NotEmpty(t, id, "no running container for run %s", runID)
	return strings.Split(id, "\n")[0]
}

func createRemoteContextFixture(t *testing.T, name, host string) {
	t.Helper()
	runCmd(t, "docker", "context", "create", name, "--docker", "host="+host)
	t.Cleanup(func() {
		_ = exec.Command("docker", "context", "rm", name).Run()
	})
}

func createLocalContextFixture(t *testing.T, name, endpoint string) {
	t.Helper()
	runCmd(t, "docker", "context", "create", name, "--docker", "host="+endpoint)
	t.Cleanup(func() {
		_ = exec.Command("docker", "context", "rm", name).Run()
	})
}

func localDockerEndpoint(t *testing.T) string {
	t.Helper()
	ctxName := strings.TrimSpace(runCmdOutput(t, "docker", "context", "show"))
	raw := strings.TrimSpace(runCmdOutput(t, "docker", "context", "inspect", ctxName, "--format", "{{.Endpoints.docker.Host}}"))
	require.NotEmpty(t, raw)
	return raw
}

func runCmd(t *testing.T, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s %v: %s", name, args, string(out))
}

func runCmdOutput(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	out, err := cmd.Output()
	require.NoError(t, err, "%s %v: %s", name, args, string(out))
	return string(out)
}

func runCmdCombined(t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	out, _ := cmd.CombinedOutput()
	return string(out)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
