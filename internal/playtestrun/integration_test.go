package playtestrun

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/GoMudEngine/GoMud/internal/playtestenv"
)

const (
	integrationEnvPlaytestrun = "DOGMUD_PLAYTESTRUN_INTEGRATION"
	integrationEnvPlaytestenv = "DOGMUD_PLAYTESTENV_INTEGRATION"
)

// TestDockerPlaytestrun exercises playtestrun.Run against a real Docker
// playtestenv. Opt in with DOGMUD_PLAYTESTRUN_INTEGRATION=1 (or
// DOGMUD_PLAYTESTENV_INTEGRATION=1).
func TestDockerPlaytestrun(t *testing.T) {
	if os.Getenv(integrationEnvPlaytestrun) != "1" && os.Getenv(integrationEnvPlaytestenv) != "1" {
		t.Skip("set DOGMUD_PLAYTESTRUN_INTEGRATION=1 (or DOGMUD_PLAYTESTENV_INTEGRATION=1)")
	}

	checkout := repoRoot(t)
	goalsProfilePath := writeGoals(t, `
ephemeral:
  profile: fresh
  start_room: 5200
  budgets:
    wall_clock: 30m
`)
	goalsCreationPath := writeGoals(t, `
ephemeral:
  creation_flow: true
  creation_rationale: >
    Docker integration creation-flow exemplar for playtestrun.
  budgets:
    wall_clock: 30m
`)

	t.Run("profile_ready_stop", func(t *testing.T) {
		runDockerSession(t, checkout, goalsProfilePath, true)
	})
	t.Run("creation_flow_ready_stop", func(t *testing.T) {
		runDockerSession(t, checkout, goalsCreationPath, false)
	})
}

// TestDockerPlaytestrunScenario exercises playtestrun.RunScenario (two-actor
// mixed profiles) ready+stop through real Docker. Same opt-in env as
// TestDockerPlaytestrun.
func TestDockerPlaytestrunScenario(t *testing.T) {
	if os.Getenv(integrationEnvPlaytestrun) != "1" && os.Getenv(integrationEnvPlaytestenv) != "1" {
		t.Skip("set DOGMUD_PLAYTESTRUN_INTEGRATION=1 (or DOGMUD_PLAYTESTENV_INTEGRATION=1)")
	}

	checkout := repoRoot(t)
	playtestRoot := filepath.Join(checkout, "tools", "playtest")
	scenarioPath := filepath.Join(playtestRoot, "scenarios", "party-formation.yaml")

	var stdout bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- RunScenario(ctx, ScenarioParams{
			Checkout:          checkout,
			ScenarioPath:      scenarioPath,
			PlaytestRoot:      playtestRoot,
			WallClockOverride: 25 * time.Minute,
			Env:               playtestenv.New(),
			Stdout:            &stdout,
			StopPollInterval:  500 * time.Millisecond,
		})
	}()

	var ready ScenarioReadyPayload
	require.Eventually(t, func() bool {
		if stdout.Len() == 0 {
			return false
		}
		return json.Unmarshal(stdout.Bytes(), &ready) == nil && ready.RunID != ""
	}, 20*time.Minute, 2*time.Second, "waiting for playtestrun scenario ready JSON")

	require.NotNil(t, ready.Endpoint)
	require.NotEmpty(t, ready.RunID)
	require.Equal(t, BlackboardDirPath(checkout, ready.RunID), ready.BlackboardDir)
	require.Equal(t, onActorStopContinue, ready.OnActorStop)
	require.Len(t, ready.Actors, 2)

	for _, actor := range ready.Actors {
		require.DirExists(t, actor.BridgeDir)
		require.Contains(t, actor.BridgeDir, filepath.Join("actors", actor.ID, "bridge"))
		require.NotNil(t, actor.Creds)
		require.FileExists(t, *actor.Creds)
		require.NotEmpty(t, actor.Username)
		user, _, err := SelectCredsByActorID(*actor.Creds, actor.ID)
		require.NoError(t, err)
		require.Equal(t, actor.Username, user)
	}
	require.DirExists(t, ready.BlackboardDir)

	sc, err := ReadSidecar(checkout, ready.RunID)
	require.NoError(t, err)
	require.Equal(t, StatusReady, sc.Status)
	require.NotEmpty(t, sc.ScenarioPath)
	require.Len(t, sc.Actors, 2)

	require.NoError(t, WriteStopSignal(checkout, ready.RunID))
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Minute):
		cancel()
		t.Fatal("timed out waiting for playtestrun scenario stop after signal")
	}

	sc, err = ReadSidecar(checkout, ready.RunID)
	require.NoError(t, err)
	require.Equal(t, StatusStopped, sc.Status)
}

func runDockerSession(t *testing.T, checkout, goalsPath string, expectCreds bool) {
	t.Helper()
	var stdout bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, RunParams{
			Checkout:          checkout,
			GoalsPath:         goalsPath,
			Personality:       "feature-tester",
			WallClockOverride: 25 * time.Minute,
			Env:               playtestenv.New(),
			Stdout:            &stdout,
			StopPollInterval:  500 * time.Millisecond,
		})
	}()

	var ready ReadyPayload
	require.Eventually(t, func() bool {
		if stdout.Len() == 0 {
			return false
		}
		return json.Unmarshal(stdout.Bytes(), &ready) == nil && ready.RunID != ""
	}, 20*time.Minute, 2*time.Second, "waiting for playtestrun ready JSON")

	require.NotNil(t, ready.Endpoint)
	require.NotEmpty(t, ready.RunID)
	require.Equal(t, BridgeDirPath(checkout, ready.RunID), ready.BridgeDir)
	require.Contains(t, ready.BridgeDir, filepath.Join(".run", ready.RunID, "bridge"))

	sc, err := ReadSidecar(checkout, ready.RunID)
	require.NoError(t, err)
	require.Equal(t, StatusReady, sc.Status)
	require.Equal(t, "feature-tester", sc.Personality)

	if expectCreds {
		require.NotNil(t, ready.Creds)
		require.FileExists(t, *ready.Creds)
		require.NotEmpty(t, sc.Creds)
	} else {
		require.Nil(t, ready.Creds)
		require.Empty(t, sc.Creds)
	}

	require.NoError(t, WriteStopSignal(checkout, ready.RunID))
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(10 * time.Minute):
		cancel()
		t.Fatal("timed out waiting for playtestrun stop after signal")
	}

	sc, err = ReadSidecar(checkout, ready.RunID)
	require.NoError(t, err)
	require.Equal(t, StatusStopped, sc.Status)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from", wd)
		}
		dir = parent
	}
}
