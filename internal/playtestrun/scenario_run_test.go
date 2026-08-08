package playtestrun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/GoMudEngine/GoMud/internal/playtestenv"
	"github.com/GoMudEngine/GoMud/internal/playtestprofiles"
)

func writeScenarioCreds(t *testing.T, checkout string) string {
	t.Helper()
	path := filepath.Join(checkout, "creds.json")
	file := playtestprofiles.CredsFile{
		RunID: "scenario-run",
		Players: []playtestprofiles.PlayerCreds{
			{Profile: "early", ActorID: "leader", Username: "pt_early_aaa", Password: "secret1", UserID: 1, RoomID: 5200},
			{Profile: "fresh", ActorID: "joiner", Username: "pt_fresh_bbb", Password: "secret2", UserID: 2, RoomID: 5200},
		},
	}
	raw, err := json.Marshal(file)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0o600))
	return path
}

func setupRunnableScenario(t *testing.T) (checkout, scenarioPath, playtestRoot string) {
	t.Helper()
	checkout = t.TempDir()
	playtestRoot, _, _ = setupScenarioPlaytestRoot(t)
	scenarioPath = writeScenario(t, playtestRoot, `
name: party-formation
on_actor_stop: continue
budgets:
  wall_clock: 45m
roster:
  - id: leader
    personality: feature-tester
    goals: goals/scenarios/party-formation/leader.yaml
  - id: joiner
    personality: feel-tester
    goals: goals/scenarios/party-formation/joiner.yaml
`)
	return checkout, scenarioPath, playtestRoot
}

func TestRunScenario_MissingCheckout(t *testing.T) {
	env := &fakeEnv{}
	err := RunScenario(context.Background(), ScenarioParams{
		ScenarioPath: "x.yaml",
		Env:          env,
	})
	require.Error(t, err)
	require.False(t, env.started)
}

func TestRunScenario_BindingErrorNoStart(t *testing.T) {
	checkout, _, playtestRoot := setupRunnableScenario(t)
	bad := writeScenario(t, playtestRoot, `
name: bad
roster:
  - id: leader
    personality: feature-tester
    goals: goals/scenarios/missing.yaml
`)
	env := &fakeEnv{}
	err := RunScenario(context.Background(), ScenarioParams{
		Checkout:     checkout,
		ScenarioPath: bad,
		PlaytestRoot: playtestRoot,
		Env:          env,
	})
	require.Error(t, err)
	require.False(t, env.started)
}

func TestRunScenario_StartFailureEnvironmentFailed(t *testing.T) {
	checkout, scenarioPath, playtestRoot := setupRunnableScenario(t)
	env := &fakeEnv{
		startRes: playtestenv.Result{RunID: "sc-fail", Report: filepath.Join(checkout, "report.md")},
		startErr: errors.New("docker boom"),
	}
	dirty := false
	err := RunScenario(context.Background(), ScenarioParams{
		Checkout:     checkout,
		ScenarioPath: scenarioPath,
		PlaytestRoot: playtestRoot,
		Env:          env,
		Commit:       "abc",
		Dirty:        &dirty,
	})
	require.Error(t, err)
	sc, readErr := ReadSidecar(checkout, "sc-fail")
	require.NoError(t, readErr)
	require.Equal(t, StatusEnvironmentFailed, sc.Status)
	require.NotEmpty(t, sc.ScenarioPath)
	require.Equal(t, onActorStopContinue, sc.OnActorStop)
	require.Len(t, sc.Actors, 2)
}

func TestRunScenario_ReadyJSONActorsAndBlackboard(t *testing.T) {
	checkout, scenarioPath, playtestRoot := setupRunnableScenario(t)
	credsPath := writeScenarioCreds(t, checkout)
	clock := newManualClock(time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC))
	env := &fakeEnv{
		startRes: playtestenv.Result{
			RunID:     "sc-ok",
			Endpoint:  &playtestenv.Endpoint{Host: "127.0.0.1", Port: 55555},
			Artifacts: &playtestenv.ArtifactPaths{Creds: credsPath},
		},
	}
	var stdout bytes.Buffer
	dirty := true
	done := make(chan error, 1)
	pumpCtx, pumpCancel := context.WithCancel(context.Background())
	defer pumpCancel()
	go func() {
		done <- RunScenario(context.Background(), ScenarioParams{
			Checkout:         checkout,
			ScenarioPath:     scenarioPath,
			PlaytestRoot:     playtestRoot,
			Env:              env,
			Clock:            clock,
			Stdout:           &stdout,
			Commit:           "deadbeef",
			Dirty:            &dirty,
			StopPollInterval: time.Millisecond,
		})
	}()
	pumpClock(t, pumpCtx, clock)

	require.Eventually(t, func() bool {
		_, err := ReadSidecar(checkout, "sc-ok")
		return err == nil
	}, 2*time.Second, 10*time.Millisecond)

	require.True(t, env.started)
	require.Equal(t, 45*time.Minute+leaseBuffer, env.startOpts.Lease)
	require.Len(t, env.startOpts.Profiles, 2)
	require.Equal(t, "leader", env.startOpts.Profiles[0].ActorID)
	require.Equal(t, "joiner", env.startOpts.Profiles[1].ActorID)

	sc, err := ReadSidecar(checkout, "sc-ok")
	require.NoError(t, err)
	require.Equal(t, StatusReady, sc.Status)
	require.Equal(t, "45m0s", sc.Budgets.WallClock)
	require.Equal(t, BlackboardDirPath(checkout, "sc-ok"), sc.BlackboardDir)
	require.Equal(t, onActorStopContinue, sc.OnActorStop)
	require.Len(t, sc.Actors, 2)
	require.Equal(t, ActorStatusReady, sc.Actors[0].Status)
	require.Equal(t, "pt_early_aaa", sc.Actors[0].Username)
	require.NotNil(t, sc.Actors[0].Creds)
	require.Equal(t, credsPath, *sc.Actors[0].Creds)
	require.DirExists(t, ActorBridgeDirPath(checkout, "sc-ok", "leader"))
	require.DirExists(t, ActorBridgeDirPath(checkout, "sc-ok", "joiner"))
	require.DirExists(t, BlackboardDirPath(checkout, "sc-ok"))

	var ready ScenarioReadyPayload
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &ready))
	require.Equal(t, "sc-ok", ready.RunID)
	require.Equal(t, BlackboardDirPath(checkout, "sc-ok"), ready.BlackboardDir)
	require.Equal(t, onActorStopContinue, ready.OnActorStop)
	require.Len(t, ready.Actors, 2)
	require.Equal(t, ActorBridgeDirPath(checkout, "sc-ok", "leader"), ready.Actors[0].BridgeDir)
	require.Equal(t, "feature-tester", ready.Actors[0].Personality)
	require.NotNil(t, ready.Actors[0].Creds)

	require.NoError(t, WriteStopSignal(checkout, "sc-ok"))
	require.NoError(t, <-done)
	require.True(t, env.stopCalled)
	sc, err = ReadSidecar(checkout, "sc-ok")
	require.NoError(t, err)
	require.Equal(t, StatusStopped, sc.Status)
}

func TestRunScenario_CLIWallClockOverride(t *testing.T) {
	checkout, scenarioPath, playtestRoot := setupRunnableScenario(t)
	credsPath := writeScenarioCreds(t, checkout)
	clock := newManualClock(time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC))
	env := &fakeEnv{startRes: playtestenv.Result{
		RunID:     "sc-wc",
		Artifacts: &playtestenv.ArtifactPaths{Creds: credsPath},
	}}
	dirty := false
	done := make(chan error, 1)
	pumpCtx, pumpCancel := context.WithCancel(context.Background())
	defer pumpCancel()
	go func() {
		done <- RunScenario(context.Background(), ScenarioParams{
			Checkout:          checkout,
			ScenarioPath:      scenarioPath,
			PlaytestRoot:      playtestRoot,
			WallClockOverride: 10 * time.Minute,
			Env:               env,
			Clock:             clock,
			Commit:            "c",
			Dirty:             &dirty,
			StopPollInterval:  time.Millisecond,
		})
	}()
	pumpClock(t, pumpCtx, clock)
	require.Eventually(t, func() bool {
		_, err := ReadSidecar(checkout, "sc-wc")
		return err == nil
	}, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, 10*time.Minute+leaseBuffer, env.startOpts.Lease)
	sc, err := ReadSidecar(checkout, "sc-wc")
	require.NoError(t, err)
	require.Equal(t, "10m0s", sc.Budgets.WallClock)
	require.NoError(t, WriteStopSignal(checkout, "sc-wc"))
	require.NoError(t, <-done)
}

func TestRunScenario_WallClockIncomplete(t *testing.T) {
	checkout, scenarioPath, playtestRoot := setupRunnableScenario(t)
	// Tiny wall clock via override.
	credsPath := writeScenarioCreds(t, checkout)
	clock := newManualClock(time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC))
	env := &fakeEnv{startRes: playtestenv.Result{
		RunID:     "sc-deadline",
		Artifacts: &playtestenv.ArtifactPaths{Creds: credsPath},
	}}
	dirty := false
	done := make(chan error, 1)
	pumpCtx, pumpCancel := context.WithCancel(context.Background())
	defer pumpCancel()
	go func() {
		done <- RunScenario(context.Background(), ScenarioParams{
			Checkout:          checkout,
			ScenarioPath:      scenarioPath,
			PlaytestRoot:      playtestRoot,
			WallClockOverride: 20 * time.Millisecond,
			Env:               env,
			Clock:             clock,
			Commit:            "c",
			Dirty:             &dirty,
			StopPollInterval:  time.Millisecond,
		})
	}()
	// Advance past deadline quickly.
	go func() {
		for i := 0; i < 50; i++ {
			clock.Advance(time.Millisecond)
			time.Sleep(time.Millisecond)
		}
	}()
	_ = pumpCtx
	err := <-done
	pumpCancel()
	require.Error(t, err)
	require.Contains(t, err.Error(), "wall-clock")
	require.True(t, env.stopCalled)
	sc, readErr := ReadSidecar(checkout, "sc-deadline")
	require.NoError(t, readErr)
	require.Equal(t, StatusIncompleteWallclock, sc.Status)
}

func TestRunScenario_Interrupt(t *testing.T) {
	checkout, scenarioPath, playtestRoot := setupRunnableScenario(t)
	credsPath := writeScenarioCreds(t, checkout)
	clock := newManualClock(time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC))
	env := &fakeEnv{startRes: playtestenv.Result{
		RunID:     "sc-int",
		Artifacts: &playtestenv.ArtifactPaths{Creds: credsPath},
	}}
	dirty := false
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	pumpCtx, pumpCancel := context.WithCancel(context.Background())
	defer pumpCancel()
	go func() {
		done <- RunScenario(ctx, ScenarioParams{
			Checkout:         checkout,
			ScenarioPath:     scenarioPath,
			PlaytestRoot:     playtestRoot,
			Env:              env,
			Clock:            clock,
			Commit:           "c",
			Dirty:            &dirty,
			StopPollInterval: time.Millisecond,
		})
	}()
	pumpClock(t, pumpCtx, clock)
	require.Eventually(t, func() bool {
		_, err := ReadSidecar(checkout, "sc-int")
		return err == nil
	}, 2*time.Second, 10*time.Millisecond)
	cancel()
	err := <-done
	require.Error(t, err)
	require.Contains(t, err.Error(), "interrupted")
	sc, readErr := ReadSidecar(checkout, "sc-int")
	require.NoError(t, readErr)
	require.Equal(t, StatusInterrupted, sc.Status)
}

func TestMarkScenarioAbort_DriverContract(t *testing.T) {
	checkout := t.TempDir()
	sc := SessionSidecar{
		RunID:       "sc-abort",
		Checkout:    checkout,
		OnActorStop: onActorStopAbort,
		Status:      StatusReady,
		Actors: []ActorSidecar{
			{ID: "leader", Status: ActorStatusFailed},
			{ID: "joiner", Status: ActorStatusReady},
		},
	}
	_, err := WriteSidecar(checkout, sc)
	require.NoError(t, err)
	require.NoError(t, MarkScenarioAbort(checkout, "sc-abort", "leader"))
	got, err := ReadSidecar(checkout, "sc-abort")
	require.NoError(t, err)
	require.Equal(t, StatusIncompleteAbort, got.Status)
	require.Equal(t, ActorStatusFailed, got.Actors[0].Status)
	require.Equal(t, ActorStatusAbortedPeer, got.Actors[1].Status)
}
