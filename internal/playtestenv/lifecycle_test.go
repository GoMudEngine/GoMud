package playtestenv

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// lifecycleRunner is a scripted Runner that records semantic startup events
// derived from command argv and filesystem side effects.
type lifecycleRunner struct {
	mu     sync.Mutex
	calls  []recordedCall
	events []string
	// handlers match by substring of joined args (after Name).
	handlers []lifecycleHandler
	checkout string
}

type recordedCall struct {
	Ctx  context.Context
	Spec CommandSpec
}

type lifecycleHandler struct {
	match string
	fn    func(ctx context.Context, spec CommandSpec) error
}

func newLifecycleRunner(checkout string) *lifecycleRunner {
	return &lifecycleRunner{checkout: checkout}
}

func (r *lifecycleRunner) on(match string, fn func(ctx context.Context, spec CommandSpec) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, lifecycleHandler{match: match, fn: fn})
}

func (r *lifecycleRunner) Events() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := append([]string(nil), r.events...)
	return out
}

func (r *lifecycleRunner) Calls() []recordedCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := append([]recordedCall(nil), r.calls...)
	return out
}

func (r *lifecycleRunner) addEvent(e string) {
	r.events = append(r.events, e)
}

func (r *lifecycleRunner) Run(ctx context.Context, spec CommandSpec) error {
	r.mu.Lock()
	recorded := recordedCall{Ctx: ctx, Spec: spec}
	recorded.Spec.Args = append([]string(nil), spec.Args...)
	recorded.Spec.Env = append([]string(nil), spec.Env...)
	r.calls = append(r.calls, recorded)

	joined := spec.Name + " " + strings.Join(spec.Args, " ")
	r.classify(joined, spec)

	var fn func(ctx context.Context, spec CommandSpec) error
	for i := range r.handlers {
		if strings.Contains(joined, r.handlers[i].match) {
			fn = r.handlers[i].fn
			break
		}
	}
	r.mu.Unlock()

	if fn == nil {
		return fmt.Errorf("lifecycleRunner: no handler for %q", joined)
	}
	return fn(ctx, spec)
}

func (r *lifecycleRunner) classify(joined string, spec CommandSpec) {
	switch {
	case strings.Contains(joined, "rev-parse --show-toplevel"),
		strings.Contains(joined, "check-ignore"):
		if !containsEvent(r.events, "validate checkout") {
			r.addEvent("validate checkout")
		}
	case strings.Contains(joined, "context show"),
		strings.Contains(joined, "context inspect"),
		strings.Contains(joined, "compose version"),
		(strings.Contains(joined, " version ") && strings.Contains(joined, "Server.Version")):
		if !containsEvent(r.events, "local Docker preflight") {
			// Prove validating manifest already exists before Docker.
			manifestGlob := filepath.Join(r.checkout, "tools", "playtest", ".run", "*", "manifest.json")
			matches, _ := filepath.Glob(manifestGlob)
			if len(matches) > 0 {
				if !containsEvent(r.events, "write validating manifest") {
					r.addEvent("write validating manifest")
				}
				m, err := readManifest(matches[0])
				if err == nil && m.State == StateValidating {
					r.addEvent("local Docker preflight")
				}
			}
		}
	case strings.Contains(joined, " compose ") && strings.Contains(joined, " build ") && strings.Contains(joined, " server"):
		if !containsEvent(r.events, "write control files") {
			r.addEvent("write control files")
		}
		if !containsEvent(r.events, "transition building") {
			r.addEvent("transition building")
		}
		r.addEvent("compose build server")
	case strings.Contains(joined, " compose ") && strings.Contains(joined, " up ") && strings.Contains(joined, "--no-build") && strings.Contains(joined, " server"):
		if !containsEvent(r.events, "transition starting") {
			r.addEvent("transition starting")
		}
		r.addEvent("compose up -d --no-build server")
	case strings.Contains(joined, " compose ") && strings.Contains(joined, " ps ") && strings.Contains(joined, " server"):
		r.addEvent("resolve container")
	case strings.Contains(joined, " logs ") && !strings.Contains(joined, " compose "):
		if !containsEvent(r.events, "readiness") {
			r.addEvent("readiness")
		}
	case strings.Contains(joined, " inspect "):
		if !containsEvent(r.events, "readiness") {
			r.addEvent("readiness")
		}
	}
	_ = spec
}

func containsEvent(events []string, want string) bool {
	for _, e := range events {
		if e == want {
			return true
		}
	}
	return false
}

func writeStdout(spec CommandSpec, s string) {
	if spec.Stdout != nil && s != "" {
		_, _ = io.WriteString(spec.Stdout, s)
	}
}

func scriptGitValidateAndBaseline(r *lifecycleRunner, canonical string) {
	r.on("rev-parse --show-toplevel", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, canonical+"\n")
		return nil
	})
	r.on("check-ignore", func(ctx context.Context, spec CommandSpec) error {
		return nil
	})
	r.on("rev-parse HEAD", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, "abc123deadbeef\n")
		return nil
	})
	r.on("status --short -z", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, " M file.yaml\x00")
		return nil
	})
}

func scriptDockerPreflight(r *lifecycleRunner, ctxName string) {
	r.on("context show", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, ctxName+"\n")
		return nil
	})
	r.on("context inspect", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, `"npipe://./pipe/docker_engine"`+"\n")
		return nil
	})
	r.on("compose version --short", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, "2.29.0\n")
		return nil
	})
	r.on("version --format", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, "24.0.5\n")
		return nil
	})
}

func TestStartHappyPathStrictOrder(t *testing.T) {
	checkout := t.TempDir()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
	writeMinimalCheckout(t, checkout, `const VERSION = "1.2.3"`)

	runner := newLifecycleRunner(canonical)
	scriptGitValidateAndBaseline(runner, canonical)

	// Inject docker resolver so we do not depend on host GOOS endpoint rules.
	dc := dockerContext{name: "desktop-linux", env: []string{"PATH=/usr/bin"}}
	containerID := "containerhappy01"
	inspect := inspectJSON(true, "127.0.0.1", "54321")

	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		switch {
		case strings.Contains(joined, " build ") && strings.Contains(joined, " server"):
			writeStdout(spec, "building...\n")
			return nil
		case strings.Contains(joined, " up ") && strings.Contains(joined, "--no-build"):
			require.Contains(t, joined, "-d")
			require.Contains(t, joined, "server")
			return nil
		case strings.Contains(joined, " ps ") && strings.Contains(joined, "server"):
			writeStdout(spec, containerID+"\n")
			return nil
		case strings.Contains(joined, " down "):
			return nil
		default:
			return fmt.Errorf("unexpected compose: %s", joined)
		}
	})
	runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, inspect+"\n")
		return nil
	})
	runner.on(" logs ", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, "Server Ready\n")
		return nil
	})

	var events []string
	fixedNow := time.Date(2026, 8, 8, 15, 0, 0, 0, time.UTC)
	clock := newReadinessFakeClock(fixedNow)
	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    clock.Now,
		genID:  func() (string, error) { return "run-happy", nil },
		dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			c1, c2 := net.Pipe()
			_ = c2.Close()
			return c1, nil
		},
		after: clock.After,
		onEvent: func(name string) {
			events = append(events, name)
		},
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			// Validating manifest must exist before any Docker resource work.
			manifestPath := filepath.Join(canonical, "tools", "playtest", ".run", "run-happy", "manifest.json")
			require.FileExists(t, manifestPath)
			m, err := readManifest(manifestPath)
			require.NoError(t, err)
			require.Equal(t, StateValidating, m.State)
			require.NotEmpty(t, m.CheckoutFingerprint)
			require.FileExists(t, filepath.Join(canonical, "tools", "playtest", ".run", "run-happy", ".lock"))
			return dc, nil
		},
	})

	res, err := s.Start(context.Background(), StartOptions{Checkout: checkout})
	require.NoError(t, err)
	require.Equal(t, "start", res.Operation)
	require.Equal(t, "run-happy", res.RunID)
	require.Equal(t, projectName("run-happy"), res.Project)
	require.Equal(t, StateReady, res.State)
	require.NotNil(t, res.Endpoint)
	require.Equal(t, "127.0.0.1", res.Endpoint.Host)
	require.Equal(t, 54321, res.Endpoint.Port)
	require.FileExists(t, res.Manifest)
	require.NotNil(t, res.Artifacts)
	require.FileExists(t, res.Artifacts.Compose)
	require.FileExists(t, res.Artifacts.Config)
	require.FileExists(t, res.Artifacts.BuildLog)

	wantOrder := []string{
		"validate checkout",
		"reserve and lock run",
		"write validating manifest",
		"local Docker preflight",
		"write control files",
		"transition building",
		"compose build server",
		"transition starting",
		"compose up -d --no-build server",
		"resolve container",
		"readiness",
		"transition ready",
		"unlock",
	}
	require.Equal(t, wantOrder, events)

	m, err := readManifest(res.Manifest)
	require.NoError(t, err)
	require.Equal(t, StateReady, m.State)
	require.Equal(t, containerID, m.ContainerID)

	// Lock must be released: a second acquire should succeed quickly.
	lock, err := acquireRunLock(context.Background(), filepath.Join(filepath.Dir(res.Manifest), ".lock"), time.Second)
	require.NoError(t, err)
	require.NoError(t, lock.Close())
}

func TestStartValidationFailureWritesNoManifestOrReport(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	runner := newLifecycleRunner(missing)
	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    time.Now,
		genID:  func() (string, error) { return "should-not-run", nil },
		dial:   (&fakeDialer{}).DialContext,
		after:  time.After,
		resolveDocker: func(ctx context.Context, runner Runner) (dockerContext, error) {
			t.Fatal("docker must not be consulted for invalid checkout")
			return dockerContext{}, nil
		},
	})

	res, err := s.Start(context.Background(), StartOptions{Checkout: missing})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrCheckoutNotDirectory)
	require.Equal(t, "start", res.Operation)
	require.Empty(t, res.RunID)
	require.Empty(t, res.Manifest)
	require.Nil(t, res.Artifacts)
	require.NotNil(t, res.Failure)
	require.Equal(t, FailureInvalidCheckout, res.Failure.Category)
	require.Empty(t, runner.Calls(), "no git/docker commands against an invalid path that fails before git")
}

func TestStartDockerUnavailableAfterReservationReturnsPopulatedResult(t *testing.T) {
	checkout := t.TempDir()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
	writeMinimalCheckout(t, checkout, `const VERSION = "1.2.3"`)

	runner := newLifecycleRunner(canonical)
	scriptGitValidateAndBaseline(runner, canonical)
	// Cleanup may invoke compose down / image rm with live ctx.
	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		require.NoError(t, ctx.Err(), "cleanup must use a live context")
		deadline, ok := ctx.Deadline()
		require.True(t, ok, "cleanup context must be bounded")
		require.True(t, time.Until(deadline) <= 45*time.Second+time.Second)
		return nil
	})
	runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error {
		require.NoError(t, ctx.Err())
		_, ok := ctx.Deadline()
		require.True(t, ok)
		return nil
	})
	runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error { return nil })
	runner.on(" logs ", func(ctx context.Context, spec CommandSpec) error { return nil })

	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    func() time.Time { return time.Date(2026, 8, 8, 16, 0, 0, 0, time.UTC) },
		genID:  func() (string, error) { return "run-nodocker", nil },
		dial:   (&fakeDialer{}).DialContext,
		after:  time.After,
		resolveDocker: func(ctx context.Context, runner Runner) (dockerContext, error) {
			return dockerContext{}, ErrDockerHostOverride
		},
	})

	res, err := s.Start(context.Background(), StartOptions{Checkout: checkout})
	require.Error(t, err)
	require.Equal(t, "run-nodocker", res.RunID)
	require.Equal(t, projectName("run-nodocker"), res.Project)
	require.NotEmpty(t, res.Manifest)
	require.NotNil(t, res.Artifacts)
	require.NotEmpty(t, res.Artifacts.Manifest)
	require.NotNil(t, res.Failure)
	require.Equal(t, FailureDockerUnavailable, res.Failure.Category)
	require.Equal(t, StateFailed, res.State)
	require.NotEmpty(t, res.Report)
	require.FileExists(t, res.Report)
}

func TestStartCancelDuringBuildCleansUpWithLiveBoundedContext(t *testing.T) {
	checkout := t.TempDir()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
	writeMinimalCheckout(t, checkout, `const VERSION = "1.2.3"`)

	runner := newLifecycleRunner(canonical)
	scriptGitValidateAndBaseline(runner, canonical)

	buildStarted := make(chan struct{})
	var cleanupCtxs []context.Context

	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		if strings.Contains(joined, " build ") {
			close(buildStarted)
			<-ctx.Done()
			return ctx.Err()
		}
		// cleanup down
		cleanupCtxs = append(cleanupCtxs, ctx)
		require.NoError(t, ctx.Err(), "cleanup must not see the cancelled caller context")
		_, ok := ctx.Deadline()
		require.True(t, ok, "cleanup must be bounded")
		return nil
	})
	runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error {
		cleanupCtxs = append(cleanupCtxs, ctx)
		require.NoError(t, ctx.Err())
		_, ok := ctx.Deadline()
		require.True(t, ok)
		return nil
	})
	runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
		cleanupCtxs = append(cleanupCtxs, ctx)
		require.NoError(t, ctx.Err())
		return nil
	})
	runner.on(" logs ", func(ctx context.Context, spec CommandSpec) error {
		cleanupCtxs = append(cleanupCtxs, ctx)
		require.NoError(t, ctx.Err())
		return nil
	})

	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    func() time.Time { return time.Date(2026, 8, 8, 17, 0, 0, 0, time.UTC) },
		genID:  func() (string, error) { return "run-cancel", nil },
		dial:   (&fakeDialer{}).DialContext,
		after:  time.After,
		resolveDocker: func(ctx context.Context, runner Runner) (dockerContext, error) {
			return dockerContext{name: "desktop-linux", env: []string{}}, nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	var res Result
	go func() {
		var startErr error
		res, startErr = s.Start(ctx, StartOptions{Checkout: checkout})
		errCh <- startErr
	}()

	select {
	case <-buildStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for compose build")
	}
	cancel()

	startErr := <-errCh
	require.Error(t, startErr)
	require.True(t, errors.Is(startErr, context.Canceled) || strings.Contains(startErr.Error(), "canceled") || strings.Contains(startErr.Error(), "cancelled"))
	require.Equal(t, "run-cancel", res.RunID)
	require.NotNil(t, res.Failure)
	require.Equal(t, StateFailed, res.State)
	require.NotEmpty(t, res.Report)
	require.NotEmpty(t, cleanupCtxs, "cleanup must invoke docker with a live bounded context")

	m, err := readManifest(res.Manifest)
	require.NoError(t, err)
	require.Equal(t, StateFailed, m.State)
	require.NotNil(t, m.Failure)
}

func TestStartMapsLockBusyToRetryableFailure(t *testing.T) {
	checkout := t.TempDir()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
	writeMinimalCheckout(t, checkout, `const VERSION = "1.2.3"`)

	runner := newLifecycleRunner(canonical)
	scriptGitValidateAndBaseline(runner, canonical)

	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    time.Now,
		genID:  func() (string, error) { return "run-busy", nil },
		dial:   (&fakeDialer{}).DialContext,
		after:  time.After,
		acquireLock: func(ctx context.Context, path string, wait time.Duration) (*runLock, error) {
			return nil, ErrLockBusy
		},
		resolveDocker: func(ctx context.Context, runner Runner) (dockerContext, error) {
			t.Fatal("docker must not run when lock is busy")
			return dockerContext{}, nil
		},
		lockWait: 100 * time.Millisecond,
	})

	res, err := s.Start(context.Background(), StartOptions{Checkout: checkout})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrLockBusy)
	require.NotNil(t, res.Failure)
	require.Equal(t, FailureLockBusy, res.Failure.Category)
	require.True(t, res.Failure.Retryable)
}

func TestComposeBuildAndNarrowUpHelpers(t *testing.T) {
	dc := dockerContext{name: "ctx", env: []string{}}
	vars := composeRunVars{RunID: "r", Project: "p", Checkout: "c", CheckoutFingerprint: "f", ControlDir: "d"}
	composeFile := "compose.resolved.yml"

	buildSpec := composeBuildCommand(dc, vars, composeFile, "", nil, nil)
	require.Equal(t, []string{"build", "server"}, buildSpec.Args[len(buildSpec.Args)-2:])

	upSpec := composeUpNoBuildServerCommand(dc, vars, composeFile, "", nil, nil)
	require.Equal(t, []string{"up", "-d", "--no-build", "server"}, upSpec.Args[len(upSpec.Args)-4:])

	// Existing Task 3 helper remains unchanged.
	legacy := composeUpCommand(dc, vars, composeFile, "", nil, nil)
	require.Equal(t, []string{"up", "-d"}, legacy.Args[len(legacy.Args)-2:])
}

func TestStartBuildFailureCategory(t *testing.T) {
	checkout := t.TempDir()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
	writeMinimalCheckout(t, checkout, `const VERSION = "9.9.9"`)

	runner := newLifecycleRunner(canonical)
	scriptGitValidateAndBaseline(runner, canonical)
	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		if strings.Contains(joined, " build ") {
			return &ExitError{Name: "docker", Args: spec.Args, ExitCode: 1}
		}
		require.NoError(t, ctx.Err())
		return nil
	})
	runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error {
		require.NoError(t, ctx.Err())
		return nil
	})

	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    func() time.Time { return time.Date(2026, 8, 8, 18, 0, 0, 0, time.UTC) },
		genID:  func() (string, error) { return "run-buildfail", nil },
		dial:   (&fakeDialer{}).DialContext,
		after:  time.After,
		resolveDocker: func(ctx context.Context, runner Runner) (dockerContext, error) {
			return dockerContext{name: "desktop-linux", env: []string{}}, nil
		},
	})

	res, err := s.Start(context.Background(), StartOptions{Checkout: checkout})
	require.Error(t, err)
	require.Equal(t, FailureBuild, res.Failure.Category)
	require.Equal(t, "run-buildfail", res.RunID)
	require.NotNil(t, res.Artifacts)
}

func TestStartFailurePersistsInspectEvidenceBeforeCleanup(t *testing.T) {
	checkout := t.TempDir()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
	writeMinimalCheckout(t, checkout, `const VERSION = "1.2.3"`)

	runner := newLifecycleRunner(canonical)
	scriptGitValidateAndBaseline(runner, canonical)
	dc := dockerContext{name: "desktop-linux", env: []string{"PATH=/usr/bin"}}
	containerID := "containerevidence01"
	// Keep the container Running so readiness can classify the log marker
	// (Error creating server) rather than short-circuiting as container_exited.
	inspectPayload := inspectJSON(true, "127.0.0.1", "54323")

	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		switch {
		case strings.Contains(joined, " build ") && strings.Contains(joined, " server"):
			return nil
		case strings.Contains(joined, " up ") && strings.Contains(joined, "--no-build"):
			return nil
		case strings.Contains(joined, " ps ") && strings.Contains(joined, "server"):
			writeStdout(spec, containerID+"\n")
			return nil
		case strings.Contains(joined, " down "):
			return nil
		default:
			return fmt.Errorf("unexpected compose: %s", joined)
		}
	})
	runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, inspectPayload+"\n")
		return nil
	})
	runner.on(" logs ", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, "Error creating server\n")
		return nil
	})
	runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error {
		return nil
	})

	fixedNow := time.Date(2026, 8, 8, 20, 0, 0, 0, time.UTC)
	clock := newReadinessFakeClock(fixedNow)
	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    clock.Now,
		genID:  func() (string, error) { return "run-inspect", nil },
		dial:   (&fakeDialer{}).DialContext,
		after:  clock.After,
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			return dc, nil
		},
	})

	res, err := s.Start(context.Background(), StartOptions{Checkout: checkout, ReadinessTimeout: time.Second})
	require.Error(t, err)
	require.Equal(t, FailureListenerCreation, res.Failure.Category)
	require.NotNil(t, res.Artifacts)
	require.NotEmpty(t, res.Artifacts.Inspect)
	require.FileExists(t, res.Artifacts.Inspect)
	body, err := os.ReadFile(res.Artifacts.Inspect)
	require.NoError(t, err)
	require.Contains(t, string(body), `"Running":true`)
	require.Contains(t, string(body), "55555/tcp")
	require.FileExists(t, res.Artifacts.ServerLog)
	reportBody, err := os.ReadFile(res.Report)
	require.NoError(t, err)
	require.Contains(t, string(reportBody), res.Artifacts.Inspect)
}

func TestStartIncompleteCleanupPreservesComposeAndControl(t *testing.T) {
	checkout := t.TempDir()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
	writeMinimalCheckout(t, checkout, `const VERSION = "1.2.3"`)

	runner := newLifecycleRunner(canonical)
	scriptGitValidateAndBaseline(runner, canonical)
	dc := dockerContext{name: "desktop-linux", env: []string{"PATH=/usr/bin"}}

	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		switch {
		case strings.Contains(joined, " build ") && strings.Contains(joined, " server"):
			return &ExitError{Name: "docker", Args: spec.Args, ExitCode: 1}
		case strings.Contains(joined, " down "):
			return errors.New("compose down refused")
		default:
			return fmt.Errorf("unexpected compose: %s", joined)
		}
	})
	runner.on(" image rm ", func(ctx context.Context, spec CommandSpec) error {
		return errors.New("image still in use")
	})

	s := newSupervisor(supervisorDeps{
		runner: runner,
		now:    func() time.Time { return time.Date(2026, 8, 8, 21, 0, 0, 0, time.UTC) },
		genID:  func() (string, error) { return "run-leftover", nil },
		dial:   (&fakeDialer{}).DialContext,
		after:  time.After,
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			return dc, nil
		},
	})

	res, err := s.Start(context.Background(), StartOptions{Checkout: checkout})
	require.Error(t, err)
	require.Equal(t, FailureBuild, res.Failure.Category)
	require.NotNil(t, res.Cleanup)
	require.False(t, res.Cleanup.Complete)
	require.FileExists(t, res.Artifacts.Compose, "compose.resolved.yml must remain when leftovers exist")
	require.DirExists(t, filepath.Join(filepath.Dir(res.Artifacts.Compose), "control"), "control/ must remain when leftovers exist")
	require.FileExists(t, res.Artifacts.Config)
}

func TestStartBuildOutputTeesToDiagnosticsAndBuildLog(t *testing.T) {
	checkout := t.TempDir()
	canonical, err := canonicalizeCheckoutPath(checkout)
	require.NoError(t, err)
	writeMinimalCheckout(t, checkout, `const VERSION = "1.2.3"`)

	runner := newLifecycleRunner(canonical)
	scriptGitValidateAndBaseline(runner, canonical)
	dc := dockerContext{name: "desktop-linux", env: []string{"PATH=/usr/bin"}}
	containerID := "containerbuildtee01"
	inspect := inspectJSON(true, "127.0.0.1", "54322")
	const buildLine = "STEP 1/2: building image layers\n"

	runner.on(" compose ", func(ctx context.Context, spec CommandSpec) error {
		joined := strings.Join(spec.Args, " ")
		switch {
		case strings.Contains(joined, " build ") && strings.Contains(joined, " server"):
			writeStdout(spec, buildLine)
			if spec.Stderr != nil {
				_, _ = io.WriteString(spec.Stderr, "build warning on stderr\n")
			}
			return nil
		case strings.Contains(joined, " up ") && strings.Contains(joined, "--no-build"):
			return nil
		case strings.Contains(joined, " ps ") && strings.Contains(joined, "server"):
			writeStdout(spec, containerID+"\n")
			return nil
		default:
			return fmt.Errorf("unexpected compose: %s", joined)
		}
	})
	runner.on(" inspect ", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, inspect+"\n")
		return nil
	})
	runner.on(" logs ", func(ctx context.Context, spec CommandSpec) error {
		writeStdout(spec, "Server Ready\n")
		return nil
	})

	var diag bytes.Buffer
	fixedNow := time.Date(2026, 8, 8, 19, 0, 0, 0, time.UTC)
	clock := newReadinessFakeClock(fixedNow)
	s := newSupervisor(supervisorDeps{
		runner:      runner,
		now:         clock.Now,
		genID:       func() (string, error) { return "run-buildtee", nil },
		diagnostics: &diag,
		dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			c1, c2 := net.Pipe()
			_ = c2.Close()
			return c1, nil
		},
		after: clock.After,
		resolveDocker: func(ctx context.Context, _ Runner) (dockerContext, error) {
			return dc, nil
		},
	})

	res, err := s.Start(context.Background(), StartOptions{Checkout: checkout})
	require.NoError(t, err)
	require.FileExists(t, res.Artifacts.BuildLog)

	logBytes, err := os.ReadFile(res.Artifacts.BuildLog)
	require.NoError(t, err)
	require.Contains(t, string(logBytes), buildLine)
	require.Contains(t, string(logBytes), "build warning on stderr\n")
	require.Contains(t, diag.String(), buildLine, "build stdout must also reach terminal diagnostics")
	require.Contains(t, diag.String(), "build warning on stderr\n", "build stderr must also reach terminal diagnostics")
}

// Ensure Result JSON keeps failure categories stable.
func TestFailureCategoryJSONValues(t *testing.T) {
	type wrap struct {
		C FailureCategory `json:"c"`
	}
	for _, cat := range []FailureCategory{
		FailureInvalidCheckout, FailureDockerUnavailable, FailureBuild,
		FailureContainerExited, FailureBootPanic, FailureListenerCreation,
		FailurePortPublication, FailureNonLoopback, FailureReadinessTimeout,
		FailureConnectionProbe, FailureManifest, FailureCleanup,
		FailureLockBusy, FailureAbandonedRun,
	} {
		b, err := json.Marshal(wrap{C: cat})
		require.NoError(t, err)
		require.Contains(t, string(b), `"`+string(cat)+`"`)
	}
}
