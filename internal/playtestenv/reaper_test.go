package playtestenv

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// reaperSafetyRunner wraps lifecycleRunner and fails the test on unsafe Docker
// usage: wildcard deletes, unfiltered list queries, system prune, or destructive
// targets not sourced from a validated manifest allow-list.
type reaperSafetyRunner struct {
	t *testing.T
	*lifecycleRunner

	mu      sync.Mutex
	allowed map[string]struct{} // exact resource IDs/names from validated manifests
}

func newReaperSafetyRunner(t *testing.T, checkout string) *reaperSafetyRunner {
	t.Helper()
	return &reaperSafetyRunner{
		t:               t,
		lifecycleRunner: newLifecycleRunner(checkout),
		allowed:         map[string]struct{}{},
	}
}

func (r *reaperSafetyRunner) allowManifest(m *Manifest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range []string{m.ContainerID, m.Network, m.Volume, m.Image, m.Project} {
		if id != "" {
			r.allowed[id] = struct{}{}
		}
	}
}

func (r *reaperSafetyRunner) Run(ctx context.Context, spec CommandSpec) error {
	r.assertSafe(spec)
	return r.lifecycleRunner.Run(ctx, spec)
}

func (r *reaperSafetyRunner) assertSafe(spec CommandSpec) {
	r.t.Helper()
	if spec.Name != "docker" && spec.Name != "git" {
		return
	}
	if spec.Name == "git" {
		return
	}
	joined := strings.Join(spec.Args, " ")
	lower := strings.ToLower(joined)

	if strings.Contains(lower, "system prune") ||
		strings.Contains(lower, "builder prune") ||
		containsArg(spec.Args, "prune") {
		r.t.Fatalf("reaper safety: forbidden prune command: docker %s", joined)
	}
	if strings.Contains(joined, "*") || strings.Contains(joined, "?") {
		r.t.Fatalf("reaper safety: wildcard in docker args: docker %s", joined)
	}

	// Listing queries must carry the managed=true filter.
	if isDockerListQuery(spec.Args) && !hasManagedFilter(spec.Args) {
		r.t.Fatalf("reaper safety: unfiltered Docker list query: docker %s", joined)
	}

	// Destructive targets must come from validated manifests.
	if targets := destructiveTargets(spec.Args); len(targets) > 0 {
		r.mu.Lock()
		allowed := r.allowed
		r.mu.Unlock()
		for _, target := range targets {
			if _, ok := allowed[target]; !ok {
				r.t.Fatalf("reaper safety: destructive target %q not from validated manifest: docker %s", target, joined)
			}
		}
	}
}

func isDockerListQuery(args []string) bool {
	joined := " " + strings.Join(args, " ") + " "
	switch {
	case strings.Contains(joined, " ps "),
		strings.Contains(joined, " network ls "),
		strings.Contains(joined, " volume ls "),
		strings.Contains(joined, " images "),
		strings.Contains(joined, " image ls "):
		return true
	default:
		return false
	}
}

func hasManagedFilter(args []string) bool {
	want := "label=" + labelManaged + "=" + labelManagedValue
	for i, a := range args {
		if a == "--filter" && i+1 < len(args) && args[i+1] == want {
			return true
		}
		if strings.HasPrefix(a, "--filter=") && strings.Contains(a, want) {
			return true
		}
	}
	return false
}

func destructiveTargets(args []string) []string {
	joined := " " + strings.Join(args, " ") + " "
	var targets []string
	switch {
	case strings.Contains(joined, " kill "),
		strings.Contains(joined, " rm "),
		strings.Contains(joined, " image rm "),
		strings.Contains(joined, " rmi "):
		skip := map[string]bool{
			"docker": true, "--context": true, "kill": true, "rm": true,
			"image": true, "rmi": true, "--force": true, "--signal=TERM": true,
			"-f": true, "--volumes": true, "--remove-orphans": true,
		}
		for i := 0; i < len(args); i++ {
			a := args[i]
			if a == "--context" {
				i++
				continue
			}
			if strings.HasPrefix(a, "-") || skip[a] {
				continue
			}
			// Skip compose project flag value pairing handled below.
			targets = append(targets, a)
		}
	case strings.Contains(joined, " compose ") && strings.Contains(joined, " down "):
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "-p" || args[i] == "--project-name" {
				targets = append(targets, args[i+1])
			}
		}
	}
	return targets
}

func seedExpired(t *testing.T, seed *seededRun) {
	t.Helper()
	seed.Manifest.LeaseExpiresAt = seed.Now.Add(-time.Minute)
	require.NoError(t, writeManifest(seed.Manifest.Artifacts.Manifest, seed.Manifest))
}

func seedRunAt(t *testing.T, checkout, runID string, state State, withResources bool, now time.Time) seededRun {
	t.Helper()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
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

func newReapSupervisor(t *testing.T, seed seededRun, runner Runner, clock *readinessFakeClock) *Supervisor {
	t.Helper()
	if lr, ok := runner.(*lifecycleRunner); ok {
		scriptGitValidateAndBaseline(lr, seed.Canonical)
	}
	if sr, ok := runner.(*reaperSafetyRunner); ok {
		scriptGitValidateAndBaseline(sr.lifecycleRunner, seed.Canonical)
	}
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
		lockWait: ReaperLockWait,
	})
}

func scriptReapCleanupHappy(r *lifecycleRunner, m *Manifest) {
	labels := identityLabels(m)
	termSent := false
	var mu sync.Mutex
	r.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		if strings.Contains(joined, m.ContainerID) {
			mu.Lock()
			running := m.State == StateReady || m.State == StateBuilding || m.State == StateStarting || m.State == StateStopping
			if termSent {
				running = false
			}
			mu.Unlock()
			writeStdout(spec, labelledInspectJSON(running, "127.0.0.1", "54321", labels)+"\n")
			return nil
		}
		for _, id := range []string{m.Network, m.Volume, m.Image} {
			if id != "" && strings.Contains(joined, id) {
				writeStdout(spec, labelledInspectJSON(false, "", "", labels)+"\n")
				return nil
			}
		}
		if spec.Stderr != nil {
			_, _ = io.WriteString(spec.Stderr, "Error: No such object\n")
		}
		return &ExitError{Name: "docker", Args: spec.Args, ExitCode: 1}
	})
	r.on(" logs ", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, "reap final log\n")
		return nil
	})
	r.on(" kill ", func(ctx context.Context, spec CommandSpec) error {
		mu.Lock()
		termSent = true
		mu.Unlock()
		return nil
	})
	r.on(" image rm ", func(ctx context.Context, spec CommandSpec) error { return nil })
	r.on(" rm --force ", func(ctx context.Context, spec CommandSpec) error {
		return nil
	})
	r.on(" compose ", func(ctx context.Context, spec CommandSpec) error { return nil })
}

func scriptEmptyOrphanLists(r *lifecycleRunner) {
	r.on(" ps ", func(ctx context.Context, spec CommandSpec) error { return nil })
	r.on(" network ls ", func(ctx context.Context, spec CommandSpec) error { return nil })
	r.on(" volume ls ", func(ctx context.Context, spec CommandSpec) error { return nil })
	r.on(" images ", func(ctx context.Context, spec CommandSpec) error { return nil })
}

func dockerListJSON(id string, labels map[string]string) string {
	// docker ps/network/volume/image --format {{json .}} emits Labels as a
	// comma-separated key=value string on many Docker versions.
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, k+"="+v)
	}
	doc := map[string]any{
		"ID":     id,
		"Labels": strings.Join(parts, ","),
		"Names":  id,
		"Name":   id,
	}
	b, _ := json.Marshal(doc)
	return string(b)
}

func findReapResult(results []Result, runID string) (Result, bool) {
	for _, r := range results {
		if r.RunID == runID {
			return r, true
		}
	}
	return Result{}, false
}

func TestReapLeavesActiveUnexpiredRunUntouched(t *testing.T) {
	seed := seedRun(t, "run-reap-active", StateReady, true)
	runner := newReaperSafetyRunner(t, seed.Canonical)
	runner.allowManifest(seed.Manifest)
	scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
	scriptMatchingResources(runner.lifecycleRunner, seed.Manifest, true)
	scriptEmptyOrphanLists(runner.lifecycleRunner)

	var deleted bool
	runner.on(" kill ", func(ctx context.Context, spec CommandSpec) error {
		deleted = true
		return nil
	})
	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		deleted = true
		return nil
	})
	runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error {
		deleted = true
		return nil
	})

	s := newReapSupervisor(t, seed, runner, nil)
	results, err := s.Reap(context.Background(), seed.Checkout)
	require.NoError(t, err)
	require.False(t, deleted, "active unexpired run must not be cleaned")
	m, err := readManifest(seed.Manifest.Artifacts.Manifest)
	require.NoError(t, err)
	require.Equal(t, StateReady, m.State)
	require.Equal(t, seed.Manifest.LeaseExpiresAt, m.LeaseExpiresAt)
	res, ok := findReapResult(results, seed.RunID)
	require.True(t, ok)
	require.Equal(t, "reap", res.Operation)
	require.Equal(t, StateReady, res.State)
	require.Nil(t, res.Cleanup)
}

func TestReapRemovesExpiredMatchingRun(t *testing.T) {
	seed := seedRun(t, "run-reap-expired", StateReady, true)
	seedExpired(t, &seed)
	runner := newReaperSafetyRunner(t, seed.Canonical)
	runner.allowManifest(seed.Manifest)
	scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
	scriptReapCleanupHappy(runner.lifecycleRunner, seed.Manifest)
	scriptEmptyOrphanLists(runner.lifecycleRunner)

	clock := newReadinessFakeClock(seed.Now)
	s := newReapSupervisor(t, seed, runner, clock)
	results, err := s.Reap(context.Background(), seed.Checkout)
	require.NoError(t, err)

	res, ok := findReapResult(results, seed.RunID)
	require.True(t, ok)
	require.Equal(t, StateStopped, res.State)
	require.NotNil(t, res.Cleanup)
	require.True(t, res.Cleanup.Complete)

	m, err := readManifest(seed.Manifest.Artifacts.Manifest)
	require.NoError(t, err)
	require.Equal(t, StateStopped, m.State)
	require.NoFileExists(t, seed.Manifest.Artifacts.Compose)
	require.NoDirExists(t, filepath.Join(seed.RunDir, controlDirName))
}

func TestReapCleansExpiredAbandonedBuildingAndStopping(t *testing.T) {
	for _, state := range []State{StateBuilding, StateStopping} {
		state := state
		t.Run(string(state), func(t *testing.T) {
			seed := seedRun(t, "run-reap-abandon-"+string(state), state, true)
			seedExpired(t, &seed)
			runner := newReaperSafetyRunner(t, seed.Canonical)
			runner.allowManifest(seed.Manifest)
			scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
			scriptReapCleanupHappy(runner.lifecycleRunner, seed.Manifest)
			scriptEmptyOrphanLists(runner.lifecycleRunner)

			s := newReapSupervisor(t, seed, runner, newReadinessFakeClock(seed.Now))
			results, err := s.Reap(context.Background(), seed.Checkout)
			require.NoError(t, err)

			res, ok := findReapResult(results, seed.RunID)
			require.True(t, ok)
			require.Equal(t, StateFailed, res.State)
			require.NotNil(t, res.Failure)
			require.Equal(t, FailureAbandonedRun, res.Failure.Category)
			require.NotNil(t, res.Cleanup)
			require.True(t, res.Cleanup.Complete)

			m, err := readManifest(seed.Manifest.Artifacts.Manifest)
			require.NoError(t, err)
			require.Equal(t, StateFailed, m.State)
			require.Equal(t, FailureAbandonedRun, m.Failure.Category)
		})
	}
}

func TestReapRespectsLeaseRenewedWhileWaitingForLock(t *testing.T) {
	seed := seedRun(t, "run-reap-renew-race", StateReady, true)
	seedExpired(t, &seed)

	runner := newReaperSafetyRunner(t, seed.Canonical)
	runner.allowManifest(seed.Manifest)
	scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
	scriptMatchingResources(runner.lifecycleRunner, seed.Manifest, true)
	scriptEmptyOrphanLists(runner.lifecycleRunner)

	var deleted bool
	runner.on(" kill ", func(ctx context.Context, spec CommandSpec) error {
		deleted = true
		return nil
	})
	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		deleted = true
		return nil
	})
	runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error {
		deleted = true
		return nil
	})

	// While the reaper waits for the lock, renew the lease so the final
	// under-lock reread refuses deletion.
	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    func() time.Time { return seed.Now },
		after:  time.After,
		dial:   (&fakeDialer{}).DialContext,
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			return seed.DC, nil
		},
		acquireLock: func(ctx context.Context, path string, wait time.Duration) (*runLock, error) {
			require.Equal(t, ReaperLockWait, wait)
			seed.Manifest.LeaseExpiresAt = seed.Now.Add(2 * time.Hour)
			require.NoError(t, writeManifest(seed.Manifest.Artifacts.Manifest, seed.Manifest))
			return acquireRunLock(ctx, path, wait)
		},
		lockWait: ReaperLockWait,
	})

	results, err := s.Reap(context.Background(), seed.Checkout)
	require.NoError(t, err)
	require.False(t, deleted, "renewed lease must prevent reaping")
	res, ok := findReapResult(results, seed.RunID)
	require.True(t, ok)
	require.Equal(t, StateReady, res.State)
	require.Nil(t, res.Cleanup)
}

func TestReapFinalLeaseRereadPreventsDeletion(t *testing.T) {
	// Distinct from the wait-for-lock case: lock acquired immediately, but the
	// second under-lock reread still sees a fresh lease written after the first
	// read that discovered expiry.
	seed := seedRun(t, "run-reap-reread", StateReady, true)
	seedExpired(t, &seed)

	runner := newReaperSafetyRunner(t, seed.Canonical)
	runner.allowManifest(seed.Manifest)
	scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
	scriptMatchingResources(runner.lifecycleRunner, seed.Manifest, true)
	scriptEmptyOrphanLists(runner.lifecycleRunner)

	var deleted bool
	runner.on(" kill ", func(ctx context.Context, spec CommandSpec) error {
		deleted = true
		return nil
	})
	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		deleted = true
		return nil
	})
	runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error {
		deleted = true
		return nil
	})

	reads := 0
	var mu sync.Mutex
	origAcquire := acquireRunLock
	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    func() time.Time { return seed.Now },
		after:  time.After,
		dial:   (&fakeDialer{}).DialContext,
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			return seed.DC, nil
		},
		acquireLock: func(ctx context.Context, path string, wait time.Duration) (*runLock, error) {
			lock, err := origAcquire(ctx, path, wait)
			if err != nil {
				return nil, err
			}
			// After lock is held, rewrite lease before reaper's mandatory reread.
			mu.Lock()
			reads++
			mu.Unlock()
			seed.Manifest.LeaseExpiresAt = seed.Now.Add(time.Hour)
			require.NoError(t, writeManifest(seed.Manifest.Artifacts.Manifest, seed.Manifest))
			return lock, nil
		},
		lockWait: ReaperLockWait,
	})

	results, err := s.Reap(context.Background(), seed.Checkout)
	require.NoError(t, err)
	require.False(t, deleted)
	res, ok := findReapResult(results, seed.RunID)
	require.True(t, ok)
	require.Nil(t, res.Cleanup)
	require.Equal(t, StateReady, res.State)
}

func TestReapReportsMalformedAndAbsentManifest(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		seed := seedRun(t, "run-reap-malformed", StateReady, true)
		require.NoError(t, os.WriteFile(seed.Manifest.Artifacts.Manifest, []byte("{not-json"), 0o644))
		runner := newReaperSafetyRunner(t, seed.Canonical)
		scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
		scriptEmptyOrphanLists(runner.lifecycleRunner)
		runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
			t.Fatal("inspect must not run for malformed manifest")
			return nil
		})
		s := newReapSupervisor(t, seed, runner, nil)
		results, err := s.Reap(context.Background(), seed.Checkout)
		require.NoError(t, err)
		res, ok := findReapResult(results, seed.RunID)
		require.True(t, ok)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureManifest, res.Failure.Category)
		require.Nil(t, res.Cleanup)
	})

	t.Run("absent", func(t *testing.T) {
		seed := seedRun(t, "run-reap-absent", StateReady, true)
		require.NoError(t, os.Remove(seed.Manifest.Artifacts.Manifest))
		runner := newReaperSafetyRunner(t, seed.Canonical)
		scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
		scriptEmptyOrphanLists(runner.lifecycleRunner)
		s := newReapSupervisor(t, seed, runner, nil)
		results, err := s.Reap(context.Background(), seed.Checkout)
		require.NoError(t, err)
		res, ok := findReapResult(results, seed.RunID)
		require.True(t, ok)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureManifest, res.Failure.Category)
		require.Nil(t, res.Cleanup)
	})
}

func TestReapDiagnosticsPartialAndMismatchedLabels(t *testing.T) {
	t.Run("partial labels", func(t *testing.T) {
		seed := seedRun(t, "run-reap-partial", StateReady, true)
		seedExpired(t, &seed)
		runner := newReaperSafetyRunner(t, seed.Canonical)
		runner.allowManifest(seed.Manifest)
		scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
		scriptEmptyOrphanLists(runner.lifecycleRunner)

		partial := identityLabels(seed.Manifest)
		delete(partial, labelSchema)
		runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
			writeStdout(spec, labelledInspectJSON(true, "127.0.0.1", "54321", partial)+"\n")
			return nil
		})
		runner.on(" kill ", func(ctx context.Context, spec CommandSpec) error {
			t.Fatal("must not delete on partial labels")
			return nil
		})
		runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
			t.Fatal("must not delete on partial labels")
			return nil
		})
		runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error {
			t.Fatal("must not delete on partial labels")
			return nil
		})

		s := newReapSupervisor(t, seed, runner, nil)
		results, err := s.Reap(context.Background(), seed.Checkout)
		require.NoError(t, err)
		res, ok := findReapResult(results, seed.RunID)
		require.True(t, ok)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureIdentityMismatch, res.Failure.Category)
		require.Nil(t, res.Cleanup)
		m, err := readManifest(seed.Manifest.Artifacts.Manifest)
		require.NoError(t, err)
		require.Equal(t, StateReady, m.State)
	})

	t.Run("mismatched identity labels", func(t *testing.T) {
		seed := seedRun(t, "run-reap-mismatch", StateReady, true)
		seedExpired(t, &seed)
		runner := newReaperSafetyRunner(t, seed.Canonical)
		runner.allowManifest(seed.Manifest)
		scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
		scriptEmptyOrphanLists(runner.lifecycleRunner)

		bad := identityLabels(seed.Manifest)
		bad[labelProject] = "other-project"
		bad[labelCheckout] = "other-fp"
		bad[labelSchema] = "9"
		bad[labelCreatedAt] = "2000-01-01T00:00:00Z"
		runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
			writeStdout(spec, labelledInspectJSON(true, "127.0.0.1", "54321", bad)+"\n")
			return nil
		})
		runner.on(" kill ", func(ctx context.Context, spec CommandSpec) error {
			t.Fatal("must not delete on mismatched labels")
			return nil
		})
		runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
			t.Fatal("must not delete on mismatched labels")
			return nil
		})
		runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error {
			t.Fatal("must not delete on mismatched labels")
			return nil
		})

		s := newReapSupervisor(t, seed, runner, nil)
		results, err := s.Reap(context.Background(), seed.Checkout)
		require.NoError(t, err)
		res, ok := findReapResult(results, seed.RunID)
		require.True(t, ok)
		require.NotNil(t, res.Failure)
		require.Equal(t, FailureIdentityMismatch, res.Failure.Category)
		require.Nil(t, res.Cleanup)
	})
}

func TestReapDiagnosticsLabelledDecoyAndOtherCheckout(t *testing.T) {
	seed := seedRun(t, "run-reap-decoy-base", StateReady, true)
	seedExpired(t, &seed)
	// Also seed a matching expired run that WILL be cleaned so allow-list is used.
	runner := newReaperSafetyRunner(t, seed.Canonical)
	runner.allowManifest(seed.Manifest)
	scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
	scriptReapCleanupHappy(runner.lifecycleRunner, seed.Manifest)

	decoyLabels := map[string]string{
		labelManaged:   labelManagedValue,
		labelRunID:     "decoy-run",
		labelProject:   "dogmud-playtest-decoy-run",
		labelCheckout:  "decoy-fingerprint",
		labelSchema:    labelSchemaValue,
		labelCreatedAt: seed.Now.UTC().Format(time.RFC3339),
	}
	otherCheckoutLabels := identityLabels(seed.Manifest)
	otherCheckoutLabels[labelCheckout] = "other-checkout-fingerprint"
	otherCheckoutLabels[labelRunID] = "other-checkout-run"

	runner.on(" ps ", func(ctx context.Context, spec CommandSpec) error {
		require.True(t, hasManagedFilter(spec.Args))
		writeStdout(spec, dockerListJSON("decoy-container", decoyLabels)+"\n")
		writeStdout(spec, dockerListJSON("other-checkout-cid", otherCheckoutLabels)+"\n")
		return nil
	})
	runner.on(" network ls ", func(ctx context.Context, spec CommandSpec) error {
		require.True(t, hasManagedFilter(spec.Args))
		return nil
	})
	runner.on(" volume ls ", func(ctx context.Context, spec CommandSpec) error {
		require.True(t, hasManagedFilter(spec.Args))
		return nil
	})
	runner.on(" images ", func(ctx context.Context, spec CommandSpec) error {
		require.True(t, hasManagedFilter(spec.Args))
		writeStdout(spec, dockerListJSON("decoy-image", decoyLabels)+"\n")
		return nil
	})

	s := newReapSupervisor(t, seed, runner, newReadinessFakeClock(seed.Now))
	results, err := s.Reap(context.Background(), seed.Checkout)
	require.NoError(t, err)

	// Expired matching run cleaned.
	cleaned, ok := findReapResult(results, seed.RunID)
	require.True(t, ok)
	require.NotNil(t, cleaned.Cleanup)
	require.True(t, cleaned.Cleanup.Complete)

	// Decoy / other-checkout only reported — never deleted (safety runner would
	// Fatal if decoy IDs were passed to destructive commands).
	var sawDecoyDiag, sawOtherDiag bool
	for _, r := range results {
		if r.Failure == nil {
			continue
		}
		sum := strings.ToLower(r.Failure.Summary)
		if strings.Contains(sum, "decoy") || strings.Contains(sum, "without matching manifest") || strings.Contains(sum, "no matching manifest") {
			sawDecoyDiag = true
		}
		if strings.Contains(sum, "other-checkout") || strings.Contains(sum, "other checkout") || strings.Contains(sum, "checkout") {
			sawOtherDiag = true
		}
	}
	require.True(t, sawDecoyDiag || sawOtherDiag, "expected orphan diagnostics in results: %+v", results)
}

func TestReapReportsLockBusyWithoutTouchingResources(t *testing.T) {
	seed := seedRun(t, "run-reap-lockbusy", StateReady, true)
	seedExpired(t, &seed)
	runner := newReaperSafetyRunner(t, seed.Canonical)
	runner.allowManifest(seed.Manifest)
	scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
	scriptEmptyOrphanLists(runner.lifecycleRunner)
	runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
		t.Fatal("inspect must not run when lock is busy")
		return nil
	})
	runner.on(" kill ", func(ctx context.Context, spec CommandSpec) error {
		t.Fatal("must not delete when lock busy")
		return nil
	})

	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    func() time.Time { return seed.Now },
		after:  time.After,
		dial:   (&fakeDialer{}).DialContext,
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			return seed.DC, nil
		},
		acquireLock: func(ctx context.Context, path string, wait time.Duration) (*runLock, error) {
			require.Equal(t, ReaperLockWait, wait)
			return nil, ErrLockBusy
		},
		lockWait: ReaperLockWait,
	})

	results, err := s.Reap(context.Background(), seed.Checkout)
	require.NoError(t, err)
	res, ok := findReapResult(results, seed.RunID)
	require.True(t, ok)
	require.NotNil(t, res.Failure)
	require.Equal(t, FailureLockBusy, res.Failure.Category)
	require.True(t, res.Failure.Retryable)
	require.Nil(t, res.Cleanup)
	m, err := readManifest(seed.Manifest.Artifacts.Manifest)
	require.NoError(t, err)
	require.Equal(t, StateReady, m.State)
}

func TestReapOneCleanupFailureDoesNotHideLaterDiagnostics(t *testing.T) {
	checkout := t.TempDir()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
	writeMinimalCheckout(t, checkout, `const VERSION = "1.2.3"`)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

	failSeed := seedRunAt(t, checkout, "run-reap-fail-first", StateReady, true, now)
	seedExpired(t, &failSeed)
	okSeed := seedRunAt(t, checkout, "run-reap-ok-second", StateReady, true, now)
	seedExpired(t, &okSeed)

	// Manifest-less directory for later diagnostic.
	ghostDir := filepath.Join(canonical, filepath.FromSlash(runsDirName), "run-reap-ghost")
	require.NoError(t, os.MkdirAll(ghostDir, 0o755))

	runner := newReaperSafetyRunner(t, canonical)
	runner.allowManifest(failSeed.Manifest)
	runner.allowManifest(okSeed.Manifest)
	scriptGitValidateAndBaseline(runner.lifecycleRunner, canonical)

	failLabels := identityLabels(failSeed.Manifest)
	okLabels := identityLabels(okSeed.Manifest)
	termOK := false
	var mu sync.Mutex

	runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		switch {
		case strings.Contains(joined, failSeed.Manifest.ContainerID):
			writeStdout(spec, labelledInspectJSON(true, "127.0.0.1", "54321", failLabels)+"\n")
			return nil
		case strings.Contains(joined, okSeed.Manifest.ContainerID):
			mu.Lock()
			running := !termOK
			mu.Unlock()
			writeStdout(spec, labelledInspectJSON(running, "127.0.0.1", "54321", okLabels)+"\n")
			return nil
		case strings.Contains(joined, failSeed.Manifest.Network),
			strings.Contains(joined, failSeed.Manifest.Volume),
			strings.Contains(joined, failSeed.Manifest.Image):
			writeStdout(spec, labelledInspectJSON(false, "", "", failLabels)+"\n")
			return nil
		case strings.Contains(joined, okSeed.Manifest.Network),
			strings.Contains(joined, okSeed.Manifest.Volume),
			strings.Contains(joined, okSeed.Manifest.Image):
			writeStdout(spec, labelledInspectJSON(false, "", "", okLabels)+"\n")
			return nil
		default:
			if spec.Stderr != nil {
				_, _ = io.WriteString(spec.Stderr, "Error: No such object\n")
			}
			return &ExitError{Name: "docker", Args: spec.Args, ExitCode: 1}
		}
	})
	runner.on(" logs ", func(ctx context.Context, spec CommandSpec) error {
		return nil
	})
	runner.on(" kill ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		if strings.Contains(joined, failSeed.Manifest.ContainerID) {
			return errors.New("simulated kill failure")
		}
		mu.Lock()
		termOK = true
		mu.Unlock()
		return nil
	})
	runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error { return nil })
	runner.on(" rm --force ", func(ctx context.Context, spec CommandSpec) error { return nil })
	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		if strings.Contains(joined, failSeed.Manifest.Project) {
			return errors.New("simulated compose down failure")
		}
		return nil
	})
	scriptEmptyOrphanLists(runner.lifecycleRunner)

	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    func() time.Time { return now },
		after:  time.After,
		dial:   (&fakeDialer{}).DialContext,
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			return failSeed.DC, nil
		},
		lockWait: ReaperLockWait,
	})

	results, err := s.Reap(context.Background(), checkout)
	// Aggregate may surface cleanup failure, but later candidates must still appear.
	_ = err

	failRes, ok := findReapResult(results, failSeed.RunID)
	require.True(t, ok, "failed cleanup result must be present")
	require.NotNil(t, failRes.Cleanup)
	require.False(t, failRes.Cleanup.Complete)

	okRes, ok := findReapResult(results, okSeed.RunID)
	require.True(t, ok, "later candidate must still be processed")
	require.NotNil(t, okRes.Cleanup)
	require.True(t, okRes.Cleanup.Complete)

	ghostRes, ok := findReapResult(results, "run-reap-ghost")
	require.True(t, ok, "manifest-less diagnostic must not be hidden")
	require.NotNil(t, ghostRes.Failure)
	require.Equal(t, FailureManifest, ghostRes.Failure.Category)
}

func TestReapSkipsLaterCandidatesAfterCallerCancel(t *testing.T) {
	checkout := t.TempDir()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
	writeMinimalCheckout(t, checkout, `const VERSION = "1.2.3"`)
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

	first := seedRunAt(t, checkout, "run-reap-cancel-a", StateReady, true, now)
	seedExpired(t, &first)
	second := seedRunAt(t, checkout, "run-reap-cancel-b", StateReady, true, now)
	seedExpired(t, &second)

	runner := newReaperSafetyRunner(t, canonical)
	runner.allowManifest(first.Manifest)
	runner.allowManifest(second.Manifest)
	scriptGitValidateAndBaseline(runner.lifecycleRunner, canonical)
	scriptEmptyOrphanLists(runner.lifecycleRunner)

	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	var cleanedSecond bool
	termFirst := false

	runner.on(" inspect ", func(c context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		labels := identityLabels(first.Manifest)
		if strings.Contains(joined, second.Manifest.ContainerID) ||
			strings.Contains(joined, second.Manifest.Network) ||
			strings.Contains(joined, second.Manifest.Volume) ||
			strings.Contains(joined, second.Manifest.Image) {
			labels = identityLabels(second.Manifest)
		}
		running := true
		mu.Lock()
		if strings.Contains(joined, first.Manifest.ContainerID) && termFirst {
			running = false
		}
		mu.Unlock()
		if strings.Contains(joined, first.Manifest.ContainerID) || strings.Contains(joined, second.Manifest.ContainerID) {
			writeStdout(spec, labelledInspectJSON(running, "127.0.0.1", "54321", labels)+"\n")
			return nil
		}
		writeStdout(spec, labelledInspectJSON(false, "", "", labels)+"\n")
		return nil
	})
	runner.on(" logs ", func(c context.Context, spec CommandSpec) error { return nil })
	runner.on(" kill ", func(c context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		if strings.Contains(joined, first.Manifest.ContainerID) {
			cancel()
			mu.Lock()
			termFirst = true
			mu.Unlock()
			require.NoError(t, c.Err(), "in-flight cleanup must ignore caller cancel")
			return nil
		}
		if strings.Contains(joined, second.Manifest.ContainerID) {
			mu.Lock()
			cleanedSecond = true
			mu.Unlock()
		}
		return nil
	})
	runner.on(" image rm ", func(c context.Context, spec CommandSpec) error {
		require.NoError(t, c.Err())
		return nil
	})
	runner.on(" rm --force ", func(c context.Context, spec CommandSpec) error { return nil })
	runner.on(" compose ", func(c context.Context, spec CommandSpec) error {
		require.NoError(t, c.Err())
		return nil
	})

	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    func() time.Time { return now },
		after:  time.After,
		dial:   (&fakeDialer{}).DialContext,
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			return first.DC, nil
		},
		lockWait: ReaperLockWait,
	})

	results, err := s.Reap(ctx, checkout)
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "canceled"))
	require.False(t, cleanedSecond, "later candidates must not start after caller cancel")

	firstRes, ok := findReapResult(results, first.RunID)
	require.True(t, ok)
	require.NotNil(t, firstRes.Cleanup)
	require.True(t, firstRes.Cleanup.Complete)
}

func TestReapValidatesDockerOnceBeforeCandidates(t *testing.T) {
	seed := seedRun(t, "run-reap-docker-order", StateReady, true)
	seedExpired(t, &seed)
	runner := newReaperSafetyRunner(t, seed.Canonical)
	runner.allowManifest(seed.Manifest)
	scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
	scriptReapCleanupHappy(runner.lifecycleRunner, seed.Manifest)
	scriptEmptyOrphanLists(runner.lifecycleRunner)

	resolveCalls := 0
	var sawInspectBeforeDocker bool
	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    func() time.Time { return seed.Now },
		after:  time.After,
		dial:   (&fakeDialer{}).DialContext,
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			resolveCalls++
			for _, c := range runner.Calls() {
				if strings.Contains(strings.Join(c.Spec.Args, " "), "inspect") {
					sawInspectBeforeDocker = true
				}
			}
			return seed.DC, nil
		},
		lockWait: ReaperLockWait,
	})

	_, err := s.Reap(context.Background(), seed.Checkout)
	require.NoError(t, err)
	require.Equal(t, 1, resolveCalls, "docker context must be validated once")
	require.False(t, sawInspectBeforeDocker)
}

func TestReapSafetyRunnerRejectsUnfilteredQuery(t *testing.T) {
	require.False(t, hasManagedFilter([]string{"ps", "-a"}))
	require.True(t, hasManagedFilter([]string{"ps", "-a", "--filter", "label=dogmud.playtest.managed=true"}))
	require.True(t, isDockerListQuery([]string{"--context", "x", "ps", "-a"}))
	require.Equal(t, []string{"cid-x"}, destructiveTargets([]string{"--context", "c", "kill", "--signal=TERM", "cid-x"}))
}

func TestReapIgnoresInvalidRunIDDirectoryNames(t *testing.T) {
	seed := seedRun(t, "run-reap-valid-only", StateReady, true)
	seedExpired(t, &seed)
	badDir := filepath.Join(seed.Canonical, filepath.FromSlash(runsDirName), "BAD_ID")
	require.NoError(t, os.MkdirAll(badDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(badDir, manifestFileName), []byte(`{"run_id":"BAD_ID"}`), 0o644))

	runner := newReaperSafetyRunner(t, seed.Canonical)
	runner.allowManifest(seed.Manifest)
	scriptGitValidateAndBaseline(runner.lifecycleRunner, seed.Canonical)
	scriptReapCleanupHappy(runner.lifecycleRunner, seed.Manifest)
	scriptEmptyOrphanLists(runner.lifecycleRunner)

	s := newReapSupervisor(t, seed, runner, newReadinessFakeClock(seed.Now))
	results, err := s.Reap(context.Background(), seed.Checkout)
	require.NoError(t, err)
	_, ok := findReapResult(results, "BAD_ID")
	require.False(t, ok, "invalid run id dirs must not become deletion candidates")
	_, ok = findReapResult(results, seed.RunID)
	require.True(t, ok)
}
