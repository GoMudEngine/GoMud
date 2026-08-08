package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/GoMudEngine/GoMud/internal/playtestenv"
	"github.com/GoMudEngine/GoMud/internal/playtestrun"
)

type fakeEnv struct {
	startCalled bool
	startOpts   playtestenv.StartOptions
	startRes    playtestenv.Result
	startErr    error
	stopCalled  bool
}

func (f *fakeEnv) Start(_ context.Context, opts playtestenv.StartOptions) (playtestenv.Result, error) {
	f.startCalled = true
	f.startOpts = opts
	return f.startRes, f.startErr
}

func (f *fakeEnv) Stop(context.Context, playtestenv.RunOptions) (playtestenv.Result, error) {
	f.stopCalled = true
	return playtestenv.Result{Operation: "stop"}, nil
}

func (f *fakeEnv) Status(context.Context, playtestenv.RunOptions) (playtestenv.Result, error) {
	return playtestenv.Result{Operation: "status"}, nil
}

func TestCLI_StopWritesSignalIdempotent(t *testing.T) {
	checkout := t.TempDir()
	var stderr bytes.Buffer
	code := run(context.Background(), []string{"stop", "--checkout", checkout, "--run", "r1"}, ioDiscard(), &stderr, &fakeEnv{})
	require.Equal(t, exitOK, code)
	stopPath := filepath.Join(playtestrun.BridgeDirPath(checkout, "r1"), "stop")
	require.FileExists(t, stopPath)
	code = run(context.Background(), []string{"stop", "--checkout", checkout, "--run", "r1"}, ioDiscard(), &stderr, &fakeEnv{})
	require.Equal(t, exitOK, code)
}

func TestCLI_StatusPrintsSidecar(t *testing.T) {
	checkout := t.TempDir()
	_, err := playtestrun.WriteSidecar(checkout, playtestrun.SessionSidecar{
		RunID:     "r2",
		Checkout:  checkout,
		Budgets:   playtestrun.SessionBudgets{WallClock: "30m"},
		Status:    playtestrun.StatusReady,
		BridgeDir: playtestrun.BridgeDirPath(checkout, "r2"),
	})
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"status", "--checkout", checkout, "--run", "r2"}, &stdout, &stderr, &fakeEnv{})
	require.Equal(t, exitOK, code)
	var sc playtestrun.SessionSidecar
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &sc))
	require.Equal(t, "r2", sc.RunID)
	require.Equal(t, "30m", sc.Budgets.WallClock)
}

func TestCLI_RunMissingPersonality(t *testing.T) {
	env := &fakeEnv{}
	var stderr bytes.Buffer
	code := run(context.Background(), []string{
		"run", "--checkout", t.TempDir(), "--goals", "g.yaml",
	}, ioDiscard(), &stderr, env)
	require.Equal(t, exitUsage, code)
	require.False(t, env.startCalled)
}

func TestCLI_RunMissingCheckout(t *testing.T) {
	env := &fakeEnv{}
	var stderr bytes.Buffer
	code := run(context.Background(), []string{
		"run", "--goals", "g.yaml", "--personality", "bug-finder",
	}, ioDiscard(), &stderr, env)
	require.Equal(t, exitUsage, code)
	require.False(t, env.startCalled)
}

func ioDiscard() *bytes.Buffer { return &bytes.Buffer{} }
