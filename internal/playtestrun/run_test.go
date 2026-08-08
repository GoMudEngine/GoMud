package playtestrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/GoMudEngine/GoMud/internal/playtestenv"
)

type fakeEnv struct {
	mu sync.Mutex

	startOpts playtestenv.StartOptions
	startRes  playtestenv.Result
	startErr  error
	started   bool

	stopCalled bool
	stopOpts   playtestenv.RunOptions
	stopRes    playtestenv.Result
	stopErr    error
}

func (f *fakeEnv) Start(_ context.Context, opts playtestenv.StartOptions) (playtestenv.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = true
	f.startOpts = opts
	return f.startRes, f.startErr
}

func (f *fakeEnv) Stop(_ context.Context, opts playtestenv.RunOptions) (playtestenv.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalled = true
	f.stopOpts = opts
	return f.stopRes, f.stopErr
}

func (f *fakeEnv) Status(_ context.Context, _ playtestenv.RunOptions) (playtestenv.Result, error) {
	return playtestenv.Result{Operation: "status"}, nil
}

// manualClock advances only when Advance is called; After channels fire when
// Now reaches the requested deadline.
type manualClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []clockWaiter
}

type clockWaiter struct {
	at time.Time
	ch chan time.Time
}

func newManualClock(t time.Time) *manualClock {
	return &manualClock{now: t}
}

func (c *manualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *manualClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan time.Time, 1)
	at := c.now.Add(d)
	if !c.now.Before(at) {
		ch <- c.now
		return ch
	}
	c.waiters = append(c.waiters, clockWaiter{at: at, ch: ch})
	return ch
}

func (c *manualClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	var keep []clockWaiter
	for _, w := range c.waiters {
		if !c.now.Before(w.at) {
			w.ch <- c.now
		} else {
			keep = append(keep, w)
		}
	}
	c.waiters = keep
}

func goalsProfile(t *testing.T) string {
	t.Helper()
	return writeGoals(t, `
ephemeral:
  profile: fresh
  start_room: 5200
  budgets:
    wall_clock: 30m
`)
}

// pumpClock advances a manual clock so wait loops that sleep via Clock.After
// can observe stop files and deadlines. Cancel the context to stop pumping.
func pumpClock(t *testing.T, ctx context.Context, clock *manualClock) {
	t.Helper()
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				clock.Advance(time.Millisecond)
			}
		}
	}()
}

func TestRun_MissingCheckout(t *testing.T) {
	env := &fakeEnv{}
	err := Run(context.Background(), RunParams{
		GoalsPath:   goalsProfile(t),
		Personality: "bug-finder",
		Env:         env,
	})
	require.Error(t, err)
	require.False(t, env.started)
}

func TestRun_MissingPersonality(t *testing.T) {
	env := &fakeEnv{}
	err := Run(context.Background(), RunParams{
		Checkout:  t.TempDir(),
		GoalsPath: goalsProfile(t),
		Env:       env,
	})
	require.Error(t, err)
	require.False(t, env.started)
}

func TestRun_BindingErrorNoStart(t *testing.T) {
	env := &fakeEnv{}
	err := Run(context.Background(), RunParams{
		Checkout:    t.TempDir(),
		GoalsPath:   writeGoals(t, "goals:\n  - x\n"),
		Personality: "bug-finder",
		Env:         env,
	})
	require.Error(t, err)
	require.False(t, env.started)
}

func TestRun_StartFailureEnvironmentFailed(t *testing.T) {
	checkout := t.TempDir()
	env := &fakeEnv{
		startRes: playtestenv.Result{
			RunID:  "run-fail",
			Report: filepath.Join(checkout, "report.md"),
		},
		startErr: errors.New("docker boom"),
	}
	dirty := false
	err := Run(context.Background(), RunParams{
		Checkout:    checkout,
		GoalsPath:   goalsProfile(t),
		Personality: "bug-finder",
		Env:         env,
		Commit:      "abc",
		Dirty:       &dirty,
	})
	require.Error(t, err)
	sc, readErr := ReadSidecar(checkout, "run-fail")
	require.NoError(t, readErr)
	require.Equal(t, StatusEnvironmentFailed, sc.Status)
	require.Equal(t, "bug-finder", sc.Personality)
}

func TestRun_ReadyPathLeaseAndJSON(t *testing.T) {
	checkout := t.TempDir()
	clock := newManualClock(time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC))
	credsPath := filepath.Join(checkout, "creds.json")
	env := &fakeEnv{
		startRes: playtestenv.Result{
			RunID:    "run-ok",
			Endpoint: &playtestenv.Endpoint{Host: "127.0.0.1", Port: 55555},
			Artifacts: &playtestenv.ArtifactPaths{Creds: credsPath},
		},
	}
	var stdout bytes.Buffer
	dirty := true
	done := make(chan error, 1)
	pumpCtx, pumpCancel := context.WithCancel(context.Background())
	defer pumpCancel()
	go func() {
		done <- Run(context.Background(), RunParams{
			Checkout:         checkout,
			GoalsPath:        goalsProfile(t),
			Personality:      "feature-tester",
			Env:              env,
			Clock:            clock,
			Stdout:           &stdout,
			Commit:           "deadbeef",
			Dirty:            &dirty,
			StopPollInterval: time.Millisecond,
		})
	}()
	pumpClock(t, pumpCtx, clock)

	// Wait until ready sidecar exists.
	require.Eventually(t, func() bool {
		_, err := ReadSidecar(checkout, "run-ok")
		return err == nil
	}, 2*time.Second, 10*time.Millisecond)

	require.True(t, env.started)
	require.Equal(t, 30*time.Minute+leaseBuffer, env.startOpts.Lease)
	require.Len(t, env.startOpts.Profiles, 1)
	require.Equal(t, "fresh", env.startOpts.Profiles[0].Profile)

	sc, err := ReadSidecar(checkout, "run-ok")
	require.NoError(t, err)
	require.Equal(t, StatusReady, sc.Status)
	require.Equal(t, "feature-tester", sc.Personality)
	require.Equal(t, "30m0s", sc.Budgets.WallClock)
	require.Equal(t, BridgeDirPath(checkout, "run-ok"), sc.BridgeDir)

	var ready ReadyPayload
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &ready))
	require.Equal(t, "run-ok", ready.RunID)
	require.NotNil(t, ready.Endpoint)
	require.NotNil(t, ready.Creds)
	require.Equal(t, credsPath, *ready.Creds)
	require.Equal(t, "deadbeef", ready.Commit)
	require.True(t, ready.Dirty)
	require.Equal(t, SidecarPath(checkout, "run-ok"), ready.Sidecar)
	require.Equal(t, BridgeDirPath(checkout, "run-ok"), ready.BridgeDir)
	require.Contains(t, ready.BridgeDir, filepath.Join(".run", "run-ok", "bridge"))

	require.NoError(t, WriteStopSignal(checkout, "run-ok"))
	require.NoError(t, <-done)
	require.True(t, env.stopCalled)
	sc, err = ReadSidecar(checkout, "run-ok")
	require.NoError(t, err)
	require.Equal(t, StatusStopped, sc.Status)
}

func TestRun_WallClockOverridePrecedence(t *testing.T) {
	checkout := t.TempDir()
	clock := newManualClock(time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC))
	env := &fakeEnv{startRes: playtestenv.Result{RunID: "run-wc"}}
	dirty := false
	done := make(chan error, 1)
	pumpCtx, pumpCancel := context.WithCancel(context.Background())
	defer pumpCancel()
	go func() {
		done <- Run(context.Background(), RunParams{
			Checkout:          checkout,
			GoalsPath:         goalsProfile(t),
			Personality:       "bug-finder",
			WallClockOverride: 2 * time.Minute,
			Env:               env,
			Clock:             clock,
			Commit:            "c",
			Dirty:             &dirty,
			StopPollInterval:  time.Millisecond,
		})
	}()
	pumpClock(t, pumpCtx, clock)
	require.Eventually(t, func() bool {
		_, err := ReadSidecar(checkout, "run-wc")
		return err == nil
	}, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, 2*time.Minute+leaseBuffer, env.startOpts.Lease)
	sc, err := ReadSidecar(checkout, "run-wc")
	require.NoError(t, err)
	require.Equal(t, (2 * time.Minute).String(), sc.Budgets.WallClock)
	require.NoError(t, WriteStopSignal(checkout, "run-wc"))
	require.NoError(t, <-done)
}

func TestRun_IncompleteWallclockNonZero(t *testing.T) {
	checkout := t.TempDir()
	start := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	clock := newManualClock(start)
	env := &fakeEnv{startRes: playtestenv.Result{RunID: "run-late"}}
	dirty := false
	done := make(chan error, 1)
	go func() {
		done <- Run(context.Background(), RunParams{
			Checkout: checkout,
			GoalsPath: writeGoals(t, `
ephemeral:
  profile: fresh
  start_room: 1
  budgets:
    wall_clock: 1m
`),
			Personality:      "bug-finder",
			Env:              env,
			Clock:            clock,
			Commit:           "c",
			Dirty:            &dirty,
			StopPollInterval: time.Millisecond,
		})
	}()
	require.Eventually(t, func() bool {
		sc, err := ReadSidecar(checkout, "run-late")
		return err == nil && sc.Status == StatusReady
	}, 2*time.Second, 10*time.Millisecond)

	// Jump past the 1m wall-clock; pump remaining After waiters.
	clock.Advance(2 * time.Minute)
	pumpCtx, pumpCancel := context.WithCancel(context.Background())
	defer pumpCancel()
	pumpClock(t, pumpCtx, clock)
	err := <-done
	require.Error(t, err)
	require.True(t, env.stopCalled)
	sc, readErr := ReadSidecar(checkout, "run-late")
	require.NoError(t, readErr)
	require.Equal(t, StatusIncompleteWallclock, sc.Status)
}

func TestWriteStopSignal_Idempotent(t *testing.T) {
	checkout := t.TempDir()
	require.NoError(t, WriteStopSignal(checkout, "r1"))
	require.NoError(t, WriteStopSignal(checkout, "r1"))
	_, err := os.Stat(filepath.Join(BridgeDirPath(checkout, "r1"), "stop"))
	require.NoError(t, err)
}
